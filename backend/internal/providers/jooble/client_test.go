package jooble

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientFetchesAndNormalizesVacancyWithoutLoggingKey(t *testing.T) {
	const secret = "test-secret-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+secret {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCount":1,"jobs":[{"id":987654321,"title":"Go Developer","location":"Алматы","snippet":"Build APIs","salary":"700 000 KZT","source":"partner","type":"Full-time","link":"https://kz.jooble.org/jdp/987654321","company":"Example KZ","updated":"2026-09-23T10:00:00Z"}]}`))
	}))
	defer server.Close()
	var logs bytes.Buffer
	client, err := NewClient(server.URL+"/api", secret, 1, server.Client(), slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	vacancies, err := client.Search(context.Background(), Search{Keywords: "Go", Location: "Казахстан", Page: 1, ResultsPerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(vacancies) != 1 || client.RequestCount() != 1 {
		t.Fatalf("unexpected response: %#v", vacancies)
	}
	job, err := Normalize(vacancies[0])
	if err != nil {
		t.Fatal(err)
	}
	if job.ExternalID != "987654321" || job.Title != "Go Developer" || job.ExternalUpdatedAt == nil || job.LocationRaw != "Алматы" || job.CanonicalCityID != "almaty" {
		t.Fatalf("unexpected normalized job: %#v", job)
	}
	if strings.Contains(logs.String(), secret) {
		t.Fatal("API key was written to logs")
	}
	if _, err := client.Search(context.Background(), Search{}); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatal("expected strict request budget")
	}
}

func TestNormalizeDoesNotGuessAmbiguousProviderLocation(t *testing.T) {
	job, err := Normalize(Vacancy{ID: "3", Title: "Developer", Location: "Алматы, Астана", Link: "https://kz.jooble.org/jdp/3"})
	if err != nil {
		t.Fatal(err)
	}
	if job.LocationRaw != "Алматы, Астана" || job.CanonicalCityID != "" {
		t.Fatalf("ambiguous provider location was mislabeled: %#v", job)
	}
}

func TestNormalizeKeepsOffsetlessTimestampRaw(t *testing.T) {
	job, err := Normalize(Vacancy{ID: "1", Title: "Developer", Link: "https://kz.jooble.org/jdp/1", Updated: "2026-09-23T10:00:00.0000000"})
	if err != nil {
		t.Fatal(err)
	}
	if job.ExternalUpdatedAt != nil || job.ExternalUpdatedRaw == "" {
		t.Fatal("offsetless provider time must not be guessed")
	}
}

func TestNormalizeCleansProviderMarkupAndEntities(t *testing.T) {
	job, err := Normalize(Vacancy{
		ID:      "2",
		Title:   "<b>Программист</b>",
		Snippet: "Работа в & laquo; 1С & raquo; <strong>команде</strong>",
		Link:    "https://kz.jooble.org/jdp/2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Title != "Программист" || job.Description != "Работа в «1С» команде" {
		t.Fatalf("unexpected cleaned content: %#v", job)
	}
}
