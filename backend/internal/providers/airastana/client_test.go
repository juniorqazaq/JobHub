package airastana

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/providers"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

const id = "b2d5c717-6172-90c7-5149-6b82786addcc"
const full = `{"ID":"` + id + `","Code":"R0001","Job":"Engineer","Description":"Build aircraft","CityName":"Almaty","CountryName":"Kazakhstan","CategoryName":"Engineering","StartDate":"2026-09-24T00:00:00","FinalDate":"2026-10-08T00:00:00"}`

func TestListMappingWithoutUnneededDetail(t *testing.T) {
	calls := 0
	c, _ := NewClient(10, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/api/v1/vacancies/get" || r.Header.Get("Authorization") != "" {
			t.Fatal("unexpected request")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "{}" {
			t.Fatal("unexpected body")
		}
		return response(r, 200, "["+full+"]"), nil
	})})
	got, err := c.Collect(context.Background())
	if err != nil || calls != 1 || got.Requests != 1 || got.MaxRequests != 10 || got.RequestsUsed != 1 || got.RemainingRequests != 9 || got.ListRequests != 1 || got.DetailRequests != 0 || got.Fetched != 1 || len(got.Items) != 1 || got.Complete {
		t.Fatalf("%+v %v", got, err)
	}
	j := got.Items[0]
	if j.ExternalID != id || j.SourceURL != "https://job.airastana.com/vacancies/detail/"+id || j.CompanyNameRaw != "Air Astana" || j.LocationRaw != "Almaty, Kazakhstan" || j.Category != "Engineering" || j.Description != "Build aircraft" || j.ExternalPublishedAt == nil || j.ExternalExpiresAt == nil || j.SalaryRaw != "" || j.EmploymentTypeRaw != "" {
		t.Fatalf("%+v", j)
	}
}

func TestMissingDescriptionHydratesDetail(t *testing.T) {
	calls := 0
	list := strings.Replace(full, `"Description":"Build aircraft"`, `"Description":""`, 1)
	c, _ := NewClientWithDetailLimit(3, 200, 1, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method == "POST" {
			return response(r, 200, "["+list+"]"), nil
		}
		if r.Method != "GET" || r.URL.Path != "/api/v1/vacancies/"+id {
			t.Fatal("wrong detail")
		}
		return response(r, 200, full), nil
	})})
	got, err := c.Collect(context.Background())
	if err != nil || calls != 2 || got.DetailRequests != 1 || got.DetailUnavailable != 0 || got.Items[0].Description != "Build aircraft" || got.Items[0].DescriptionKind != "full" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestDetailFailureIsNonFatalAndDistinct(t *testing.T) {
	list := strings.Replace(full, `"Description":"Build aircraft"`, `"Description":""`, 1)
	c, _ := NewClientWithDetailLimit(2, 200, 1, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == "POST" {
			return response(r, 200, "["+list+"]"), nil
		}
		return response(r, 404, `{}`), nil
	})})
	got, err := c.Collect(context.Background())
	if err != nil || got.DetailUnavailable != 1 || got.DetailRequests != 1 || len(got.Items) != 1 || got.Items[0].DescriptionKind != "snippet" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestEmptyMalformedAndListFailure(t *testing.T) {
	for _, tc := range []struct {
		name, body, category string
		status               int
		fetched              int
	}{{"empty", "[]", "", 200, 0}, {"malformed", "{", "INVALID_RESPONSE", 200, 0}, {"failure", `{}`, "HTTP_FAILED", 400, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := NewClient(2, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return response(r, tc.status, tc.body), nil })})
			got, err := c.Collect(context.Background())
			if (tc.category == "" && err != nil) || (tc.category != "" && providers.Category(err) != tc.category) || got.Requests != 1 || got.Fetched != tc.fetched {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
	c, _ := NewClient(2, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return response(r, 200, `[{"ID":"bad"}]`), nil })})
	got, err := c.Collect(context.Background())
	if err != nil || got.Malformed != 1 || len(got.Items) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}
