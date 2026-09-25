package kcell

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
)

const (
	Source           = "kcell:careers"
	baseURL          = "https://jobs.kcell.kz"
	pageSize         = 50
	maxResponseBytes = 5 << 20
)

type Client struct {
	http                 *http.Client
	maxRequests, maxJobs int
}

func NewClient(maxRequests, maxJobs int, client *http.Client) (*Client, error) {
	if maxRequests < 1 || maxRequests > 5 || maxJobs < 1 || maxJobs > 200 {
		return nil, providers.Failure("INVALID_BUDGET")
	}
	if client == nil {
		client = &http.Client{}
	}
	c := *client
	c.Timeout = 20 * time.Second
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{http: &c, maxRequests: maxRequests, maxJobs: maxJobs}, nil
}
func (*Client) Source() string { return Source }

type pageResponse struct {
	Content          []json.RawMessage `json:"content"`
	Number           int               `json:"number"`
	Size             int               `json:"size"`
	TotalElements    int               `json:"totalElements"`
	TotalPages       int               `json:"totalPages"`
	NumberOfElements int               `json:"numberOfElements"`
	First            bool              `json:"first"`
	Last             bool              `json:"last"`
}
type vacancy struct {
	JobID       json.Number `json:"jobId"`
	NameRU      string      `json:"nameRu"`
	NameKK      string      `json:"nameKk"`
	NameEN      string      `json:"nameEn"`
	DescRU      string      `json:"descRu"`
	DescKK      string      `json:"descKk"`
	DescEN      string      `json:"descEn"`
	CreatedDate string      `json:"createdDate"`
	UpdatedDate string      `json:"updatedDate"`
	IsPublic    bool        `json:"isPublic"`
	Status      string      `json:"statusJob"`
	City        named       `json:"city"`
	Team        named       `json:"team"`
	JobType     named       `json:"jobType"`
}
type named struct {
	NameRU string `json:"nameRu"`
	NameKK string `json:"nameKk"`
	NameEN string `json:"nameEn"`
}

func (c *Client) Collect(ctx context.Context) (providers.Result, error) {
	result := providers.Result{}
	for page := 0; ; page++ {
		if result.Requests >= c.maxRequests {
			return result, providers.Failure("REQUEST_BUDGET_EXHAUSTED")
		}
		var decoded pageResponse
		attempts := 0
		for {
			attempts++
			result.Requests++
			result.ListRequests++
			rawURL := baseURL + "/api/jobs?isPublic=true&statusJob=PUBLISHED&page=" + strconv.Itoa(page) + "&size=" + strconv.Itoa(pageSize)
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("User-Agent", "JobHub-Career-POC/1.0")
			res, err := c.http.Do(req)
			if err != nil {
				return result, providers.Failure("TRANSPORT_FAILED")
			}
			if (res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500) && attempts < 2 && result.Requests < c.maxRequests {
				res.Body.Close()
				continue
			}
			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				return result, providers.Failure(httpCategory(res.StatusCode))
			}
			body, readErr := readJSON(res, maxResponseBytes)
			res.Body.Close()
			if readErr != nil {
				return result, readErr
			}
			dec := json.NewDecoder(bytes.NewReader(body))
			dec.UseNumber()
			if dec.Decode(&decoded) != nil || dec.Decode(new(any)) != io.EOF {
				return result, providers.Failure("INVALID_RESPONSE")
			}
			break
		}
		result.PagesFetched++
		if decoded.Number != page || decoded.NumberOfElements != len(decoded.Content) || decoded.TotalPages < 0 || decoded.TotalElements < 0 || (decoded.TotalPages == 0 && decoded.TotalElements != 0) || (decoded.TotalPages > 0 && decoded.Number >= decoded.TotalPages) {
			return result, providers.Failure("INCOMPLETE_PAGINATION")
		}
		result.Fetched += len(decoded.Content)
		if result.Fetched > c.maxJobs {
			return result, providers.Failure("JOB_BUDGET_EXHAUSTED")
		}
		for _, raw := range decoded.Content {
			var v vacancy
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			if dec.Decode(&v) != nil {
				result.Malformed++
				continue
			}
			item, err := normalize(v)
			if err != nil {
				result.Malformed++
				continue
			}
			result.Items = append(result.Items, item)
		}
		if decoded.Last {
			paginationComplete := decoded.TotalPages == page+1 || (page == 0 && decoded.TotalPages == 0 && decoded.TotalElements == 0)
			result.Complete = decoded.TotalElements == result.Fetched && paginationComplete && result.Malformed == 0
			return result, nil
		}
		if decoded.TotalPages == 0 || page+1 >= decoded.TotalPages {
			return result, providers.Failure("INCOMPLETE_PAGINATION")
		}
	}
}

func normalize(v vacancy) (jobs.ImportedJob, error) {
	id := v.JobID.String()
	if _, err := strconv.ParseUint(id, 10, 64); err != nil || id == "0" {
		return jobs.ImportedJob{}, providers.Failure("INVALID_ID")
	}
	title := pick(v.NameRU, v.NameKK, v.NameEN)
	if title == "" || !v.IsPublic || v.Status != "PUBLISHED" {
		return jobs.ImportedJob{}, providers.Failure("INVALID_JOB")
	}
	created, err := parseOptional(v.CreatedDate)
	if err != nil {
		return jobs.ImportedJob{}, err
	}
	updated, err := parseOptional(v.UpdatedDate)
	if err != nil {
		return jobs.ImportedJob{}, err
	}
	return jobs.ImportedJob{Source: Source, ExternalID: id, SourceURL: baseURL + "/job/" + url.PathEscape(id), CompanyNameRaw: "Kcell", Title: title, LocationRaw: pick(v.City.NameRU, v.City.NameKK, v.City.NameEN), Description: pick(v.DescRU, v.DescKK, v.DescEN), DescriptionKind: "full", EmploymentTypeRaw: pick(v.JobType.NameRU, v.JobType.NameKK, v.JobType.NameEN), Category: pick(v.Team.NameRU, v.Team.NameKK, v.Team.NameEN), ExternalPublishedAt: created, ExternalUpdatedAt: updated, ExternalUpdatedRaw: v.UpdatedDate}, nil
}
func pick(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func parseOptional(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if v, e := time.Parse(layout, raw); e == nil {
			v = v.UTC()
			return &v, nil
		}
	}
	return nil, providers.Failure("INVALID_TIMESTAMP")
}
func readJSON(res *http.Response, limit int64) ([]byte, error) {
	ct, _, e := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if e != nil || ct != "application/json" {
		return nil, providers.Failure("INVALID_CONTENT_TYPE")
	}
	body, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil {
		return nil, providers.Failure("RESPONSE_READ_FAILED")
	}
	if int64(len(body)) > limit {
		return nil, providers.Failure("RESPONSE_TOO_LARGE")
	}
	return body, nil
}
func httpCategory(status int) string {
	switch {
	case status == 401 || status == 403:
		return "ACCESS_DENIED"
	case status == 429:
		return "RATE_LIMITED"
	case status >= 300 && status < 400:
		return "UNEXPECTED_REDIRECT"
	default:
		return "HTTP_FAILED"
	}
}
