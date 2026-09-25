package staticcareers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func htmlResponse(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func TestCollectUsesCanonicalURLHashAndExternalFields(t *testing.T) {
	client, err := New(Config{Source: "halyk:careers", Company: "Halyk Bank", ListingURL: "https://example.test/careers", DetailPrefix: "/jobs/"}, 3, 10, &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/careers" {
			return htmlResponse(r, `<a href="/jobs/123">job</a>`), nil
		}
		return htmlResponse(r, `<html><h1>Engineer</h1><main>Казахстан, Алматы, KZ. Description.</main></html>`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Collect(context.Background())
	if err != nil || result.Requests != 2 || !result.Complete || len(result.Items) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	item := result.Items[0]
	if !strings.HasPrefix(item.ExternalID, "url_") || item.SourceURL != "https://example.test/jobs/123" || item.CompanyNameRaw != "Halyk Bank" || item.LocationRaw != "Казахстан, Алматы, KZ" || item.Title != "Engineer" {
		t.Fatalf("item=%+v", item)
	}
}

func TestCollectStopsAtBudgetAndDoesNotClaimCompleteSnapshot(t *testing.T) {
	client, err := New(Config{Source: "technodom:careers", ListingURL: "https://example.test/careers", DetailPrefix: "/jobs/"}, 2, 10, &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/careers" {
			return htmlResponse(r, `<a href="/jobs/1">one</a><a href="/jobs/2">two</a>`), nil
		}
		return htmlResponse(r, `<h1>Job</h1>`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Collect(context.Background())
	if err != nil || result.Requests != 2 || result.Complete || result.Skipped != 1 || len(result.Items) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
