package greenhouse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"jobhub-ai/backend/internal/providers"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixtureClient(t *testing.T, body string, status int, budget *Budget) *Client {
	t.Helper()
	c, err := NewClient("fixture-board", budget, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.String() != endpoint+"fixture-board/jobs?content=true" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected request")
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const record = `{"id":900719925474099312345,"internal_job_id":8,"title":"Go Engineer","absolute_url":"https://careers.example/jobs/42?gh_src=original","content":"&amp;lt;p&amp;gt;Build &amp;amp; maintain.&amp;lt;/p&amp;gt;<script>bad()</script>","updated_at":"2026-09-24T10:00:00+06:00","location":{"name":"Almaty"}}`

func TestFetchAndDeterministicLosslessNormalization(t *testing.T) {
	b, _ := NewBudget(2, 200)
	c := fixtureClient(t, `{"jobs":[`+record+`],"meta":{"total":1}}`, 200, b)
	first, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Collect(context.Background())
	if err != nil || !reflect.DeepEqual(first.Items, second.Items) {
		t.Fatalf("nondeterministic: %v", err)
	}
	if first.Requests != 1 || first.Fetched != 1 || len(first.Items) != 1 || !first.Complete {
		t.Fatalf("bad collection: %+v", first)
	}
	j := first.Items[0]
	if j.ExternalID != "900719925474099312345" || j.Source != "greenhouse:fixture-board" || j.SourceURL != "https://careers.example/jobs/42?gh_src=original" || j.Description != "Build & maintain." || j.ExternalUpdatedAt.Format("15:04") != "04:00" {
		t.Fatalf("bad mapping: %+v", j)
	}
	if j.CompanyNameRaw != "" || j.SalaryRaw != "" || j.EmploymentTypeRaw != "" || j.CanonicalCityID != "" || j.UpstreamSourceName != "" {
		t.Fatalf("invented data: %+v", j)
	}
}
func TestInvalidResponsesAndBudgets(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		category   string
	}{
		{"html", "<html>token=secret</html>", 200, "INVALID_RESPONSE"},
		{"missing jobs", `{"meta":{"total":0}}`, 200, "INVALID_RESPONSE"},
		{"null jobs", `{"jobs":null}`, 200, "INVALID_RESPONSE"},
		{"trailing", `{"jobs":[]}extra`, 200, "INVALID_RESPONSE"},
		{"denied", "secret", 403, "ACCESS_DENIED"},
		{"rate limit", "secret", 429, "RATE_LIMITED"},
		{"redirect", "secret", 302, "UNEXPECTED_REDIRECT"},
		{"too large", strings.Repeat("x", maxBytes+1), 200, "RESPONSE_TOO_LARGE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := NewBudget(1, 200)
			r, err := fixtureClient(t, tc.body, tc.status, b).Collect(context.Background())
			if providers.Category(err) != tc.category || r.Requests != 1 || r.Complete || len(r.Items) != 0 {
				t.Fatalf("%+v %v", r, err)
			}
		})
	}
	b, _ := NewBudget(1, 1)
	c := fixtureClient(t, `{"jobs":[`+record+`,`+record+`],"meta":{"total":2}}`, 200, b)
	r, err := c.Collect(context.Background())
	if providers.Category(err) != "JOB_BUDGET_EXHAUSTED" || len(r.Items) != 0 || r.Fetched != 2 {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = c.Collect(context.Background())
	if providers.Category(err) != "REQUEST_BUDGET_EXHAUSTED" || r.Requests != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestEmptyPartialAndMalformedAreDistinct(t *testing.T) {
	for _, tc := range []struct {
		body                     string
		complete                 bool
		malformed, skipped, jobs int
	}{
		{`{"jobs":[],"meta":{"total":0}}`, true, 0, 0, 0},
		{`{"jobs":[]}`, false, 0, 0, 0},
		{`{"jobs":[` + record + `],"meta":{"total":2}}`, false, 0, 0, 1},
		{`{"jobs":[{"id":null},` + record + `],"meta":{"total":2}}`, false, 1, 0, 1},
		{`{"jobs":[{"id":1,"internal_job_id":null}],"meta":{"total":1}}`, true, 0, 1, 0},
	} {
		b, _ := NewBudget(1, 200)
		r, err := fixtureClient(t, tc.body, 200, b).Collect(context.Background())
		if err != nil || r.Complete != tc.complete || r.Malformed != tc.malformed || r.Skipped != tc.skipped || len(r.Items) != tc.jobs {
			t.Fatalf("%+v %v", r, err)
		}
	}
}
func TestMalformedIDsURLsAndTimes(t *testing.T) {
	for _, patch := range []string{`"id":null`, `"id":0`, `"id":1.5`, `"id":1e3`, `"id":" 42 "`, `"id":{}`, `"id":true`} {
		body := strings.Replace(record, `"id":900719925474099312345`, patch, 1)
		b, _ := NewBudget(1, 200)
		r, _ := fixtureClient(t, `{"jobs":[`+body+`]}`, 200, b).Collect(context.Background())
		if r.Malformed != 1 || len(r.Items) != 0 {
			t.Fatalf("accepted %s", patch)
		}
	}
	for _, body := range []string{strings.Replace(record, "https://careers.example", "javascript:alert", 1), strings.Replace(record, "https://careers.example", "https://user:secret@careers.example", 1), strings.Replace(record, "2026-09-24T10:00:00+06:00", "invalid", 1)} {
		b, _ := NewBudget(1, 200)
		r, _ := fixtureClient(t, `{"jobs":[`+body+`]}`, 200, b).Collect(context.Background())
		if r.Malformed != 1 {
			t.Fatal("accepted malformed job")
		}
	}
}
func TestSharedBudgetIsRaceSafeAndErrorsAreSanitized(t *testing.T) {
	b, _ := NewBudget(2, 200)
	var calls atomic.Int32
	c, _ := NewClient("fixture-board", b, &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, fmt.Errorf("https://user:SECRET@host/?token=SECRET")
	})})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Collect(context.Background())
			if strings.Contains(err.Error(), "SECRET") {
				t.Error("leaked transport secret")
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("budget violated: %d", calls.Load())
	}
}
