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

func TestRedirectSafetyAndLimits(t *testing.T) {
	t.Run("redirect to private address", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return response(302, "", "text/html", r), nil })}
		s := newScanner(client, publicResolver, 3, 100)
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return nil }
		// Exercise the scanner-owned redirect validator directly through a transport
		// that returns a redirect location.
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			res := response(302, "", "text/html", r)
			res.Header.Set("Location", "http://127.0.0.1/private")
			return res, nil
		})
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), &fetchBudget{remaining: 3})
		if err == nil {
			t.Fatal("unsafe redirect accepted")
		}
	})
	t.Run("response size", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return response(200, strings.Repeat("x", 11), "text/html", r), nil
		})}
		s := newScanner(client, publicResolver, 2, 10)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), &fetchBudget{remaining: 2})
		if category(err) != "RESPONSE_TOO_LARGE" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("request limit", func(t *testing.T) {
		s := newScanner(&http.Client{}, publicResolver, 1, 10)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), &fetchBudget{})
		if category(err) != "REQUEST_LIMIT" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })}
		s := newScanner(client, publicResolver, 2, 10)
		_, _, err := s.fetch(context.Background(), mustURL(t, "https://example.com"), &fetchBudget{remaining: 2})
		if category(err) != "TRANSPORT_FAILED" {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCareerDiscoveryAndJSONLD(t *testing.T) {
	base := mustURL(t, "https://www.example.kz/")
	links, ats, atsURL := discover(base, []byte(`<nav><a href="/about">About</a><a href="/careers">Работа у нас</a><a href="https://boards.greenhouse.io/acme">Jobs</a></nav>`))
	if len(links) != 1 || links[0].Path != "/careers" || ats != "Greenhouse" {
		t.Fatalf("unexpected discovery: %#v %q", links, ats)
	}
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
	if result.RequestCount != 2 || count != 2 || result.Parsed != 1 || result.Items[0].CompanyNameRaw != "" || result.Items[0].SalaryRaw != "" {
		t.Fatalf("unexpected result: %#v count=%d", result, count)
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
