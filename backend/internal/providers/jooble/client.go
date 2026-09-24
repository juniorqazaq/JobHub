package jooble

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/locations"
)

const Source = "jooble:kz"

var spacedEntityPattern = regexp.MustCompile(`&\s+(#?[[:alnum:]]+)\s*;`)
var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

type Search struct {
	Keywords       string
	Location       string
	Page           int
	ResultsPerPage int
}

type ExternalID string

func (id *ExternalID) UnmarshalJSON(data []byte) error {
	var value string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
	} else {
		value = string(data)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty job id")
	}
	*id = ExternalID(value)
	return nil
}

type Vacancy struct {
	ID       ExternalID `json:"id"`
	Title    string     `json:"title"`
	Location string     `json:"location"`
	Snippet  string     `json:"snippet"`
	Salary   string     `json:"salary"`
	Source   string     `json:"source"`
	Type     string     `json:"type"`
	Link     string     `json:"link"`
	Company  string     `json:"company"`
	Updated  string     `json:"updated"`
}

type searchResponse struct {
	TotalCount int       `json:"totalCount"`
	Jobs       []Vacancy `json:"jobs"`
}

type Client struct {
	baseURL     string
	apiKey      string
	maxRequests int
	httpClient  *http.Client
	logger      *slog.Logger
	mu          sync.Mutex
	requests    int
}

func NewClient(baseURL, apiKey string, maxRequests int, httpClient *http.Client, logger *slog.Logger) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("Jooble API key is required")
	}
	if maxRequests < 1 || maxRequests > 50 {
		return nil, errors.New("Jooble request budget must be between 1 and 50")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, maxRequests: maxRequests, httpClient: httpClient, logger: logger}, nil
}

func (c *Client) Search(ctx context.Context, search Search) ([]Vacancy, error) {
	requestNumber, err := c.reserveRequest()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{
		"keywords": search.Keywords, "location": search.Location, "page": search.Page,
		"ResultOnPage": search.ResultsPerPage, "SearchMode": 0, "companysearch": false,
	})
	if err != nil {
		return nil, errors.New("encode Jooble request")
	}
	endpoint := c.baseURL + "/" + url.PathEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("build Jooble request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.logger.Info("Jooble request", "request_count", requestNumber, "request_budget", c.maxRequests, "keywords", search.Keywords, "location", search.Location)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("Jooble request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
		return nil, fmt.Errorf("Jooble returned HTTP %d", res.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(res.Body, 2<<20))
	decoder.UseNumber()
	var response searchResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, errors.New("decode Jooble response")
	}
	return response.Jobs, nil
}

func (c *Client) RequestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

func (c *Client) reserveRequest() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.requests >= c.maxRequests {
		return c.requests, errors.New("Jooble request budget exhausted")
	}
	c.requests++
	return c.requests, nil
}

func Normalize(vacancy Vacancy) (jobs.ImportedJob, error) {
	title := cleanText(vacancy.Title)
	link := strings.TrimSpace(vacancy.Link)
	if title == "" {
		return jobs.ImportedJob{}, errors.New("Jooble vacancy title is empty")
	}
	parsedLink, err := url.Parse(link)
	if err != nil || parsedLink.Host == "" || (parsedLink.Scheme != "http" && parsedLink.Scheme != "https") {
		return jobs.ImportedJob{}, errors.New("Jooble vacancy link is invalid")
	}
	updatedRaw := strings.TrimSpace(vacancy.Updated)
	var updatedAt *time.Time
	if parsed, err := time.Parse(time.RFC3339Nano, updatedRaw); err == nil {
		value := parsed.UTC()
		updatedAt = &value
	}
	return jobs.ImportedJob{
		Source: Source, ExternalID: string(vacancy.ID), SourceURL: link,
		UpstreamSourceName: cleanText(vacancy.Source), CompanyNameRaw: cleanText(vacancy.Company),
		Title: title, LocationRaw: cleanText(vacancy.Location), Description: cleanText(vacancy.Snippet),
		CanonicalCityID: locations.Normalize(vacancy.Location),
		DescriptionKind: "snippet", EmploymentTypeRaw: cleanText(vacancy.Type), SalaryRaw: cleanText(vacancy.Salary),
		ExternalUpdatedAt: updatedAt, ExternalUpdatedRaw: updatedRaw,
	}, nil
}

func cleanText(value string) string {
	value = spacedEntityPattern.ReplaceAllString(value, `&$1;`)
	value = html.UnescapeString(value)
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer("« ", "«", " »", "»").Replace(value)
}
