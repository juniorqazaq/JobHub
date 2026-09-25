package airastana

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/safety"
)

const (
	Source           = "airastana:careers"
	baseURL          = "https://job.airastana.com"
	maxResponseBytes = 5 << 20
)

var idPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Client struct {
	http              *http.Client
	budget            *safety.RequestBudget
	maxJobs           int
	maxDetailRequests int
}

func NewClient(maxRequests, maxJobs int, client *http.Client) (*Client, error) {
	return NewClientWithDetailLimit(maxRequests, maxJobs, 0, client)
}

func NewClientWithDetailLimit(maxRequests, maxJobs, maxDetails int, client *http.Client) (*Client, error) {
	if maxRequests < 1 || maxRequests > 10 || maxJobs < 1 || maxJobs > 200 {
		return nil, providers.Failure("INVALID_BUDGET")
	}
	if maxDetails < 0 || maxDetails >= maxRequests {
		return nil, providers.Failure("INVALID_BUDGET")
	}
	if client == nil {
		client = &http.Client{}
	}
	c := *client
	c.Timeout = 20 * time.Second
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	budget, _ := safety.NewRequestBudget(maxRequests)
	return &Client{&c, budget, maxJobs, maxDetails}, nil
}
func (*Client) Source() string { return Source }

type vacancy struct {
	ID           string `json:"ID"`
	Code         string `json:"Code"`
	Job          string `json:"Job"`
	Description  string `json:"Description"`
	CityName     string `json:"CityName"`
	CountryName  string `json:"CountryName"`
	CategoryName string `json:"CategoryName"`
	StartDate    string `json:"StartDate"`
	FinalDate    string `json:"FinalDate"`
}

func (c *Client) Collect(ctx context.Context) (result providers.Result, err error) {
	defer func() {
		m := c.budget.Metrics()
		result.MaxRequests = m.MaxRequests
		result.RequestsUsed = m.RequestsUsed
		result.RemainingRequests = m.Remaining
	}()
	body, status, err := c.request(ctx, http.MethodPost, baseURL+"/api/v1/vacancies/get", []byte("{}"), &result, true)
	if err != nil {
		return result, err
	}
	_ = status
	var raws []json.RawMessage
	if json.Unmarshal(body, &raws) != nil {
		return result, providers.Failure("INVALID_RESPONSE")
	}
	result.Fetched = len(raws)
	if result.Fetched > c.maxJobs {
		return result, providers.Failure("JOB_BUDGET_EXHAUSTED")
	}
	for _, raw := range raws {
		var v vacancy
		if json.Unmarshal(raw, &v) != nil {
			result.Malformed++
			continue
		}
		if strings.TrimSpace(v.Description) == "" {
			if result.DetailRequests < c.maxDetailRequests && c.budget.Metrics().Remaining > 0 {
				detailBody, _, detailErr := c.request(ctx, http.MethodGet, baseURL+"/api/v1/vacancies/"+url.PathEscape(v.ID), nil, &result, false)
				if detailErr == nil {
					var detail vacancy
					if json.Unmarshal(detailBody, &detail) == nil && detail.ID == v.ID {
						v = merge(v, detail)
					} else {
						result.DetailUnavailable++
					}
				} else {
					result.DetailUnavailable++
				}
			} else {
				result.DetailUnavailable++
			}
		}
		item, normErr := normalize(v)
		if normErr != nil {
			result.Malformed++
			continue
		}
		result.Items = append(result.Items, item)
	}
	// The list is an unpaginated array without a total/snapshot marker. A clean
	// response is useful but does not prove absence-based lifecycle completeness.
	result.Complete = false
	return result, nil
}
func (c *Client) request(ctx context.Context, method, rawURL string, payload []byte, result *providers.Result, list bool) ([]byte, int, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if c.budget.Acquire() != nil {
			return nil, 0, providers.Failure("REQUEST_BUDGET_EXHAUSTED")
		}
		result.Requests++
		if list {
			result.ListRequests++
		} else {
			result.DetailRequests++
		}
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, _ := http.NewRequestWithContext(ctx, method, rawURL, reader)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "JobHub-Career-POC/1.0")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, 0, providers.Failure("TRANSPORT_FAILED")
		}
		if (res.StatusCode == 429 || res.StatusCode >= 500) && attempt == 0 && c.budget.Metrics().Remaining > 0 {
			res.Body.Close()
			continue
		}
		if res.StatusCode != 200 {
			status := res.StatusCode
			res.Body.Close()
			return nil, status, providers.Failure(httpCategory(status))
		}
		body, readErr := readJSON(res, maxResponseBytes)
		res.Body.Close()
		if readErr != nil {
			return nil, res.StatusCode, readErr
		}
		return body, res.StatusCode, nil
	}
	return nil, 0, providers.Failure("HTTP_FAILED")
}
func merge(list, detail vacancy) vacancy {
	if detail.Code != "" {
		list.Code = detail.Code
	}
	if detail.Job != "" {
		list.Job = detail.Job
	}
	if detail.Description != "" {
		list.Description = detail.Description
	}
	if detail.CityName != "" {
		list.CityName = detail.CityName
	}
	if detail.CountryName != "" {
		list.CountryName = detail.CountryName
	}
	if detail.CategoryName != "" {
		list.CategoryName = detail.CategoryName
	}
	if detail.StartDate != "" {
		list.StartDate = detail.StartDate
	}
	if detail.FinalDate != "" {
		list.FinalDate = detail.FinalDate
	}
	return list
}
func normalize(v vacancy) (jobs.ImportedJob, error) {
	id := strings.TrimSpace(v.ID)
	title := strings.TrimSpace(v.Job)
	if !idPattern.MatchString(id) || title == "" {
		return jobs.ImportedJob{}, providers.Failure("INVALID_JOB")
	}
	published, err := parseOptional(v.StartDate)
	if err != nil {
		return jobs.ImportedJob{}, err
	}
	expires, err := parseOptional(v.FinalDate)
	if err != nil {
		return jobs.ImportedJob{}, err
	}
	location := strings.Trim(strings.Join(nonempty(v.CityName, v.CountryName), ", "), " ,")
	kind := "full"
	if strings.TrimSpace(v.Description) == "" {
		kind = "snippet"
	}
	return jobs.ImportedJob{Source: Source, ExternalID: id, SourceURL: baseURL + "/vacancies/detail/" + url.PathEscape(id), CompanyNameRaw: "Air Astana", Title: title, LocationRaw: location, Description: strings.TrimSpace(v.Description), DescriptionKind: kind, Category: strings.TrimSpace(v.CategoryName), ExternalPublishedAt: published, ExternalExpiresAt: expires}, nil
}
func nonempty(values ...string) []string {
	out := []string{}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
func parseOptional(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
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
