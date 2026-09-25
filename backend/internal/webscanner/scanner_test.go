package webscanner

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"jobhub-ai/backend/internal/jobs"
)

type resolverFunc func(context.Context, string, string) ([]net.IP, error)

func (f resolverFunc) LookupIP(c context.Context, n, h string) ([]net.IP, error) { return f(c, n, h) }

var publicResolver = resolverFunc(func(context.Context, string, string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
})

func TestValidateURLBlocksSSRF(t *testing.T) {
	blocked := []string{"file:///etc/passwd", "ftp://example.com/a", "http://localhost/a", "http://127.0.0.1/a", "http://[::1]/a", "http://169.254.169.254/latest/meta-data", "http://10.1.2.3/a", "http://192.168.1.2/a"}
	for _, raw := range blocked {
		if _, err := ValidateURL(context.Background(), raw, publicResolver); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	privateDNS := resolverFunc(func(context.Context, string, string) ([]net.IP, error) { return []net.IP{net.ParseIP("10.0.0.2")}, nil })
	if _, err := ValidateURL(context.Background(), "https://example.test", privateDNS); err == nil {
		t.Fatal("accepted private DNS result")
	}
	if u, err := ValidateURL(context.Background(), "https://example.com/jobs#x", publicResolver); err != nil || u.Fragment != "" {
		t.Fatalf("safe URL failed: %v %#v", err, u)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, body, contentType string, req *http.Request) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": []string{contentType}}, Request: req}
}

type fakeBrowser struct {
	calls  int
	result BrowserResult
	err    error
}

func (f *fakeBrowser) Scan(context.Context, *url.URL, string) (BrowserResult, error) {
	f.calls++
	return f.result, f.err
}

func TestRedirectSafetyAndLimits(t *testing.T) {
	t.Run("redirect to private address", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			res := response(302, "", "text/html", r)
			res.Header.Set("Location", "http://127.0.0.1/private")
			return res, nil
		})}
		s := newScanner(client, publicResolver, 3, 100)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), newFetchBudget(3))
		if category(err) != "UNSAFE_URL" {
			t.Fatalf("unsafe redirect accepted: %v", err)
		}
	})
	t.Run("redirect limit", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			res := response(302, "", "text/html", r)
			res.Header.Set("Location", "https://example.com/again")
			return res, nil
		})}
		s := newScanner(client, publicResolver, 20, 100)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), newFetchBudget(20))
		if category(err) != "TOO_MANY_REDIRECTS" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("response size", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return response(200, strings.Repeat("x", 11), "text/html", r), nil
		})}
		s := newScanner(client, publicResolver, 2, 10)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), newFetchBudget(2))
		if category(err) != "RESPONSE_TOO_LARGE" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("request limit", func(t *testing.T) {
		s := newScanner(&http.Client{}, publicResolver, 1, 10)
		budget := newFetchBudget(1)
		_ = budget.value.Acquire()
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), budget)
		if category(err) != "REQUEST_LIMIT" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })}
		s := newScanner(client, publicResolver, 2, 10)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), newFetchBudget(2))
		if category(err) != "TRANSPORT_FAILED" {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCareerDiscoveryAndJSONLD(t *testing.T) {
	base := mustURL(t, "https://www.example.kz/")
	links, detection := discover(base, []byte(`<nav><a href="/about">About</a><a href="/careers">Работа у нас</a><script src="https://boards.greenhouse.io/acme"></script></nav>`))
	if len(links) != 1 || links[0].Path != "/careers" || detection.Provider != "greenhouse" || detection.ProviderKey != "acme" || detection.Confidence != "high" || detection.Source != "src" {
		t.Fatalf("unexpected discovery: %#v %#v", links, detection)
	}
	atsURL, _ := url.Parse(detection.URL)
	if atsURL == nil || atsURL.Hostname() != "boards.greenhouse.io" {
		t.Fatal("ATS URL missing")
	}
	data := []byte(`{"@context":"https://schema.org","@type":"JobPosting","title":"Go Developer","hiringOrganization":{"name":"Example"},"jobLocation":{"address":{"addressLocality":"Алматы","addressCountry":"KZ"}},"employmentType":"FULL_TIME","description":"<p>Build APIs</p>","url":"/jobs/go"}`)
	items, err := parseJSONLD(data, base, "example.kz")
	if err != nil || len(items) != 1 {
		t.Fatalf("parse: %#v %v", items, err)
	}
	job := items[0]
	if job.ExternalID != "https://www.example.kz/jobs/go" || job.SourceURL != job.ExternalID || job.Source != "website:example.kz" || job.CompanyNameRaw != "Example" || job.LocationRaw != "Алматы, KZ" || job.SalaryRaw != "" || job.Description != "Build APIs" {
		t.Fatalf("bad job: %#v", job)
	}
	items2, _ := parseJSONLD(data, base, "example.kz")
	if items2[0].ExternalID != job.ExternalID {
		t.Fatal("identity is not deterministic")
	}
	if _, err := parseJSONLD([]byte(`{"@type":`), base, "example.kz"); err == nil {
		t.Fatal("malformed json-ld accepted")
	}
}

func TestCareerListingDoesNotTreatFiltersAsJobs(t *testing.T) {
	base := mustURL(t, "https://job.example.kz/")
	links, _ := discover(base, []byte(`<a href="/search?categories=33">66 вакансий</a><a href="/search?cities=1">Алматы</a>`))
	if len(links) != 1 || links[0].String() != "https://job.example.kz/search" {
		t.Fatalf("listing links were not canonicalized: %#v", links)
	}
	_, jobs, _ := extractPage(mustURL(t, "https://job.example.kz/search"), []byte(`<a href="/search?categories=8">IT вакансии</a><a href="/vacancy/backend-engineer">Backend Engineer</a>`), "job.example.kz")
	if len(jobs) != 1 || jobs[0].Path != "/vacancy/backend-engineer" {
		t.Fatalf("unexpected detail links: %#v", jobs)
	}
}

func TestATSDetectionProvidersAndKeys(t *testing.T) {
	cases := map[string]string{"https://jobs.lever.co/acme": "lever", "https://tenant.wd3.myworkdayjobs.com/en-US/jobs": "workday", "https://careers.smartrecruiters.com/Acme": "smartrecruiters", "https://jobs.ashbyhq.com/acme": "ashby", "https://acme.recruitee.com": "recruitee", "https://apply.workable.com/acme": "workable", "https://acme.bamboohr.com/careers": "bamboohr", "https://acme.teamtailor.com/jobs": "teamtailor", "https://acme.jobs.personio.com": "personio"}
	for raw, provider := range cases {
		d, ok := detectATSURL(raw, "href")
		if !ok || d.Provider != provider || d.ProviderKey == "" {
			t.Errorf("%s => %#v", raw, d)
		}
	}
}

func TestBoundedCareerCandidates(t *testing.T) {
	rows := commonCareerCandidates(mustURL(t, "https://example.kz"))
	if len(rows) > 13 || rows[0].Path != "/career" {
		t.Fatalf("unexpected candidates: %d %#v", len(rows), rows)
	}
}

func TestDeterministicHTMLFallback(t *testing.T) {
	item, ok := extractHTMLJob(mustURL(t, "https://example.kz/jobs/7#apply"), []byte(`<html><head><meta property="og:site_name" content="Example"></head><body><h1>Backend Engineer</h1><main><p>Build reliable APIs.</p></main></body></html>`), "example.kz")
	if !ok || item.Title != "Backend Engineer" || item.CompanyNameRaw != "Example" || item.ExternalID != "https://example.kz/jobs/7" || item.LocationRaw != "" || item.SalaryRaw != "" {
		t.Fatalf("unexpected fallback: %#v", item)
	}
}

func TestScanCountsActualRequests(t *testing.T) {
	count := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		count++
		switch r.URL.Path {
		case "", "/":
			return response(200, `<a href="/careers">Карьера</a>`, "text/html", r), nil
		case "/careers":
			return response(200, `<script type="application/ld+json">{"@type":"JobPosting","title":"Engineer","url":"/jobs/1"}</script>`, "text/html", r), nil
		}
		return response(404, "", "text/html", r), nil
	})}
	s := newScanner(client, publicResolver, MaxRequests, MaxResponseBytes)
	result, err := s.Scan(context.Background(), "https://example.kz")
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestCount != 2 || result.MaxRequests != MaxRequests || result.RemainingRequests != MaxRequests-2 || count != 2 || result.Parsed != 1 || result.Items[0].CompanyNameRaw != "" || result.Items[0].SalaryRaw != "" {
		t.Fatalf("unexpected result: %#v count=%d", result, count)
	}
}

func TestBrowserFallbackOnlyForUnproductiveSPA(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(200, `<html><body><div id="root"></div><script type="module" src="/app.js"></script></body></html>`, "text/html", r), nil
	})}
	browser := &fakeBrowser{result: BrowserResult{FinalURL: "https://example.kz/careers", PagesLoaded: 2, RequestCount: 7, VacancyURLs: 1, Items: []jobs.ImportedJob{{Source: "website:example.kz", ExternalID: "https://example.kz/jobs/1", SourceURL: "https://example.kz/jobs/1", Title: "Engineer"}}}}
	s := newScanner(client, publicResolver, 2, MaxResponseBytes)
	s.browser = browser
	result, err := s.Scan(context.Background(), "https://example.kz")
	if err != nil || browser.calls != 1 || result.ScanMethod != "browser" || result.PagesLoaded != 2 || result.BrowserRequests != 7 || result.Parsed != 1 {
		t.Fatalf("fallback failed: result=%#v calls=%d err=%v", result, browser.calls, err)
	}
}

func TestBrowserFallbackSkippedWhenStaticDataExists(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(200, `<div id="root"></div><script type="module"></script><script type="application/ld+json">{"@type":"JobPosting","title":"Engineer","url":"/jobs/1"}</script>`, "text/html", r), nil
	})}
	browser := &fakeBrowser{}
	s := newScanner(client, publicResolver, 2, MaxResponseBytes)
	s.browser = browser
	result, err := s.Scan(context.Background(), "https://example.kz")
	if err != nil || browser.calls != 0 || result.ScanMethod != "static" || result.Parsed != 1 {
		t.Fatalf("static scan invoked browser: result=%#v calls=%d err=%v", result, browser.calls, err)
	}
}

func TestBrowserLimits(t *testing.T) {
	r := NewChromiumRenderer("")
	if r.maxPages != MaxBrowserPages || r.maxBytes != MaxRenderedBytes || r.timeout != BrowserScanTimeout || r.maxPages > 8 || r.timeout > 30*time.Second || r.maxRequests != MaxBrowserRequests || MaxBrowserRequests != 100 {
		t.Fatalf("unsafe browser limits: %#v", r)
	}
}

func TestChromiumRendererRequestLimitIsOnlyLowerable(t *testing.T) {
	tests := []struct {
		requested int
		want      int
	}{
		{requested: 40, want: 40},
		{requested: 100, want: 100},
		{requested: 200, want: 100},
		{requested: 0, want: 100},
		{requested: -1, want: 100},
	}
	for _, tt := range tests {
		r := NewChromiumRendererWithRequestLimit("", tt.requested)
		if r.maxRequests != tt.want {
			t.Fatalf("requested %d: maxRequests=%d, want %d", tt.requested, r.maxRequests, tt.want)
		}
	}
}

func TestKnownCareerProvidersBypassBrowser(t *testing.T) {
	cases := []struct{ raw, provider, body string }{
		{"https://jobs.kcell.kz", "kcell", `{"content":[],"number":0,"size":50,"totalElements":0,"totalPages":0,"numberOfElements":0,"last":true}`},
		{"https://job.airastana.com", "airastana", `[]`},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return response(200, tc.body, "application/json", r), nil
			})}
			browser := &fakeBrowser{}
			s := NewWithBrowser(client, browser)
			result, err := s.Scan(context.Background(), tc.raw)
			if err != nil || result.ATS != tc.provider || result.ScanMethod != "ats" || result.ListRequests != 1 || browser.calls != 0 {
				t.Fatalf("result=%+v calls=%d err=%v", result, browser.calls, err)
			}
		})
	}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

var _ = errors.Is
var _ = time.Second
