package kcell

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/providers"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}
func record(id int) string {
	return `{"jobId":` + strconv.Itoa(id) + `,"nameRu":"Инженер","nameKk":"Инженер KZ","descRu":"Описание","city":{"nameRu":"Алматы"},"team":{"nameRu":"Technology"},"jobType":{"nameRu":"Полный день"},"createdDate":"2026-09-20T10:00:00","updatedDate":"2026-09-24T11:00:00","isPublic":true,"statusJob":"PUBLISHED"}`
}

func TestCollectPaginationAndMapping(t *testing.T) {
	calls := 0
	c, _ := NewClient(5, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if r.Method != "GET" || r.URL.Query().Get("size") != "50" || r.Header.Get("Authorization") != "" {
			t.Fatal("unexpected request")
		}
		if page == 0 {
			return response(r, 200, `{"content":[`+record(900719925)+`],"number":0,"size":50,"totalElements":2,"totalPages":2,"numberOfElements":1,"first":true,"last":false}`), nil
		}
		return response(r, 200, `{"content":[`+record(900719926)+`],"number":1,"size":50,"totalElements":2,"totalPages":2,"numberOfElements":1,"first":false,"last":true}`), nil
	})})
	got, err := c.Collect(context.Background())
	if err != nil || calls != 2 || got.Requests != 2 || got.ListRequests != 2 || got.PagesFetched != 2 || got.Fetched != 2 || len(got.Items) != 2 || !got.Complete {
		t.Fatalf("%+v %v", got, err)
	}
	j := got.Items[0]
	if j.ExternalID != "900719925" || j.Source != Source || j.SourceURL != "https://jobs.kcell.kz/job/900719925" || j.CompanyNameRaw != "Kcell" || j.Title != "Инженер" || j.LocationRaw != "Алматы" || j.Category != "Technology" || j.EmploymentTypeRaw != "Полный день" || j.ExternalPublishedAt == nil || j.ExternalUpdatedAt == nil || j.SalaryRaw != "" {
		t.Fatalf("%+v", j)
	}
}

func TestEmptyMalformedAndPageFailure(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		c, _ := NewClient(1, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			return response(r, 200, `{"content":[],"number":0,"size":50,"totalElements":0,"totalPages":0,"numberOfElements":0,"first":true,"last":true}`), nil
		})})
		got, err := c.Collect(context.Background())
		if err != nil || !got.Complete || got.Fetched != 0 {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		c, _ := NewClient(1, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			return response(r, 200, `{"content":[{"jobId":"bad"}],"number":0,"size":50,"totalElements":1,"totalPages":1,"numberOfElements":1,"last":true}`), nil
		})})
		got, err := c.Collect(context.Background())
		if err != nil || got.Malformed != 1 || got.Complete {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		c, _ := NewClient(1, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return response(r, 200, `{"content":`), nil })})
		got, err := c.Collect(context.Background())
		if providers.Category(err) != "INVALID_RESPONSE" || got.Requests != 1 {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("second page failure", func(t *testing.T) {
		c, _ := NewClient(3, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Query().Get("page") == "0" {
				return response(r, 200, `{"content":[`+record(1)+`],"number":0,"size":50,"totalElements":2,"totalPages":2,"numberOfElements":1,"last":false}`), nil
			}
			return response(r, 400, `{}`), nil
		})})
		got, err := c.Collect(context.Background())
		if providers.Category(err) != "HTTP_FAILED" || got.Complete || got.PagesFetched != 1 || got.Requests != 2 {
			t.Fatalf("%+v %v", got, err)
		}
	})
}

func TestTransientRetryCounted(t *testing.T) {
	calls := 0
	c, _ := NewClient(2, 200, &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(r, 503, `{}`), nil
		}
		return response(r, 200, `{"content":[],"number":0,"size":50,"totalElements":0,"totalPages":0,"numberOfElements":0,"last":true}`), nil
	})})
	got, err := c.Collect(context.Background())
	if err != nil || got.Requests != 2 || got.ListRequests != 2 {
		t.Fatalf("%+v %v", got, err)
	}
}
