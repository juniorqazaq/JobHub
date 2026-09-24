package telegramcollector

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"jobhub-ai/backend/internal/jobs"
	telegramprovider "jobhub-ai/backend/internal/providers/telegram"
)

type memoryStore struct {
	items map[string]jobs.ImportedJob
	runs  int
}

func (s *memoryStore) BeginIngestionRun(context.Context, string, int) (string, error) {
	s.runs++
	return string(rune('0' + s.runs)), nil
}
func (s *memoryStore) FailIngestionRun(context.Context, string, jobs.ImportStats, string, string) error {
	return nil
}
func (s *memoryStore) CompleteIngestionRun(_ context.Context, _ string, source string, imported []jobs.ImportedJob, _ time.Time, _ time.Time, stats jobs.ImportStats) (jobs.ImportStats, error) {
	for _, item := range imported {
		key := source + "/" + item.ExternalID
		if _, exists := s.items[key]; exists {
			stats.UpdatedCount++
		} else {
			stats.InsertedCount++
		}
		s.items[key] = item
	}
	return stats, nil
}
func (s *memoryStore) Search(context.Context, jobs.SearchParams) (jobs.SearchResult, error) {
	return jobs.SearchResult{}, nil
}
func (s *memoryStore) Get(context.Context, string) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}
func (s *memoryStore) RegisterDevelopmentSource(context.Context, string, string, string) error {
	return nil
}

func decodeUpdate(t *testing.T, raw string) telegramprovider.Update {
	t.Helper()
	var update telegramprovider.Update
	if err := json.Unmarshal([]byte(raw), &update); err != nil {
		t.Fatal(err)
	}
	return update
}

func TestProcessorDuplicateAndEditUpdateInPlace(t *testing.T) {
	store := &memoryStore{items: map[string]jobs.ImportedJob{}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processor := NewProcessor(map[int64]struct{}{-10042: {}}, store, store, logger)
	firstUpdate := decodeUpdate(t, `{"update_id":1,"channel_post":{"message_id":7,"date":1700000000,"text":"Вакансия: Go developer","chat":{"id":-10042,"type":"channel","title":"JobHub Test","username":"jobhub_test"}}}`)
	first, err := processor.Process(context.Background(), firstUpdate)
	if err != nil || !first.Accepted || first.Report.Stats.InsertedCount != 1 {
		t.Fatalf("first process failed: %#v %v", first, err)
	}
	second, err := processor.Process(context.Background(), firstUpdate)
	if err != nil || second.Report.Stats.UpdatedCount != 1 || len(store.items) != 1 {
		t.Fatalf("duplicate was not idempotent: %#v %v", second, err)
	}
	editedUpdate := decodeUpdate(t, `{"update_id":2,"edited_channel_post":{"message_id":7,"date":1700000000,"edit_date":1700000100,"text":"Вакансия: Senior Go developer","chat":{"id":-10042,"type":"channel","title":"JobHub Test","username":"jobhub_test"}}}`)
	edited, err := processor.Process(context.Background(), editedUpdate)
	stored := store.items["telegram:-10042/telegram:-10042:7"]
	if err != nil || edited.Report.Stats.UpdatedCount != 1 || len(store.items) != 1 || stored.Title != "Senior Go developer" || stored.SourceURL != "https://t.me/jobhub_test/7" {
		t.Fatalf("edit did not update in place: %#v stored=%#v err=%v", edited, stored, err)
	}
	if stored.CompanyNameRaw != "" || stored.LocationRaw != "" || stored.EmploymentTypeRaw != "" {
		t.Fatalf("unknown values were invented: %#v", stored)
	}
}

func TestProcessorRejectsUnknownAndMalformedWithoutWrites(t *testing.T) {
	store := &memoryStore{items: map[string]jobs.ImportedJob{}}
	processor := NewProcessor(map[int64]struct{}{-10042: {}}, store, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	unknown := decodeUpdate(t, `{"update_id":1,"channel_post":{"message_id":7,"date":1700000000,"text":"Вакансия: Go developer","chat":{"id":-10099,"type":"channel","username":"other_channel"}}}`)
	out, err := processor.Process(context.Background(), unknown)
	if err != nil || out.SkipReason != telegramprovider.SkipSourceNotAllowed || store.runs != 0 {
		t.Fatalf("unknown source result: %#v %v", out, err)
	}
	malformed := decodeUpdate(t, `{"update_id":2,"channel_post":{"message_id":8,"date":1700000000,"text":"обычный пост","chat":{"id":-10042,"type":"channel","username":"jobhub_test"}}}`)
	out, err = processor.Process(context.Background(), malformed)
	if err != nil || out.SkipReason != telegramprovider.SkipMissingTitle || store.runs != 0 {
		t.Fatalf("malformed result: %#v %v", out, err)
	}
}
