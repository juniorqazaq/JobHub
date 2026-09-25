package jobs

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
)

func TestPostgresSearchFiltersStayConsistent(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := config.ValidateLocalPOCDatabase("development", databaseURL); err != nil {
		t.Skip("integration test requires an isolated local POC database")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, databaseURL, "cache_statement")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	marker := fmt.Sprintf("filter-test-%d", time.Now().UnixNano())
	externalSource := "test:" + marker
	var userID, companyID string
	if _, err := pool.Exec(ctx, `INSERT INTO jobhub.job_sources (source,provider,market,display_name,enabled) VALUES ($1,'test','KZ','Filter Test',true)`, externalSource); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO jobhub.users (full_name,email,normalized_email,password_hash,role,status) VALUES ('Filter Test',$1,$1,'not-used','employer','active') RETURNING id::text`, marker+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO jobhub.companies (name,status) VALUES ($1,'active') RETURNING id::text`, marker).Scan(&companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO jobhub.company_memberships (company_id,user_id,role) VALUES ($1::uuid,$2::uuid,'owner')`, companyID, userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.jobs WHERE created_by_user_id=$1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.jobs WHERE source=$1`, externalSource)
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.company_memberships WHERE user_id=$1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.companies WHERE id=$1::uuid`, companyID)
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.users WHERE id=$1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM jobhub.job_sources WHERE source=$1`, externalSource)
	})

	insert := func(title, city, mode, experience, employment, status string, salary float64, publishedAt time.Time) {
		t.Helper()
		_, err := pool.Exec(ctx, `
			INSERT INTO jobhub.jobs (
				source,company_id,created_by_user_id,company_name_raw,title,location_raw,canonical_city_id,
				description,description_kind,skills,work_mode,employment_type,experience_level,
				salary_min,salary_max,salary_currency,salary_period,salary_visible,application_method,
				source_status,publication_status,moderation_status,first_seen_at,last_seen_at,last_synced_at,
				fresh_until,published_at
			) VALUES ('jobhub',$1::uuid,$2::uuid,$3,$4,$5,$6,'Go SQL services','full',ARRAY['Go','SQL'],$7,$8,$9,$10,$10,'KZT','month',true,'internal','active',$11,'approved',$12,$12,$12,'infinity'::timestamptz,$12)`,
			companyID, userID, marker, title, city, city, mode, employment, experience, salary, status, publishedAt)
		if err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Truncate(time.Second)
	insert(marker+" Go Remote", "almaty", "remote", "middle", "full_time", "published", 800000, now.Add(-time.Hour))
	insert(marker+" Design Hybrid", "almaty", "hybrid", "senior", "contract", "published", 400000, now.Add(-48*time.Hour))
	insert(marker+" Go Astana", "astana", "remote", "junior", "full_time", "published", 900000, now.Add(-10*24*time.Hour))
	insert(marker+" Hidden Go", "almaty", "remote", "middle", "full_time", "paused", 1000000, now)
	for externalID, city := range map[string]*string{"external-almaty": ptr("almaty"), "external-unknown": nil} {
		location := "Unknown"
		if city != nil {
			location = "Алматы"
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO jobhub.jobs (
				source,external_id,source_url,company_name_raw,title,location_raw,canonical_city_id,
				description,description_kind,skills,application_method,apply_url,source_status,
				moderation_status,first_seen_at,last_seen_at,last_synced_at,fresh_until,external_published_at
			) VALUES ($1,$2,'https://example.test/jobs/' || $2,$3,$4,$5,$6,
				'External vacancy','full','{}','external','https://example.test/jobs/' || $2,'active',
				'approved',$7,$7,$7,'infinity'::timestamptz,$7)`,
			externalSource, externalID, marker, marker+" Go External "+externalID, location, city, now.Add(-2*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
	}

	store := NewPostgresStore(pool)
	assert := func(name string, params SearchParams, want int64) SearchResult {
		t.Helper()
		params.Query = marker + params.Query
		if params.Page == 0 {
			params.Page = 1
		}
		if params.PageSize == 0 {
			params.PageSize = 20
		}
		if params.Sort == "" {
			params.Sort = "newest"
		}
		result, err := store.Search(ctx, params)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if result.Total != want || int64(len(result.Items)) > want {
			t.Fatalf("%s: total/list mismatch: %#v", name, result)
		}
		return result
	}

	assert("reset-equivalent unfiltered", SearchParams{}, 5)
	assert("city includes native and imported", SearchParams{City: "almaty"}, 3)
	assert("keyword and city", SearchParams{Query: " Go", City: "almaty"}, 2)
	assert("remote and city", SearchParams{City: "almaty", WorkModes: []string{"remote"}}, 1)
	minimum := 500000.0
	assert("salary and currency", SearchParams{City: "almaty", SalaryMin: &minimum, Currency: "KZT"}, 1)
	assert("experience", SearchParams{ExperienceLevel: "senior"}, 1)
	assert("employment", SearchParams{EmploymentType: "contract"}, 1)
	postedAfter := now.Add(-7 * 24 * time.Hour)
	assert("date posted", SearchParams{PostedAfter: &postedAfter}, 4)
	assert("multiple modes", SearchParams{City: "almaty", WorkModes: []string{"remote", "hybrid"}}, 2)
	assert("unknown work mode does not match", SearchParams{Query: " Go External external-unknown", WorkModes: []string{"remote"}}, 0)
	paged := assert("pagination", SearchParams{City: "almaty", Page: 2, PageSize: 1}, 3)
	if len(paged.Items) != 1 {
		t.Fatalf("pagination returned %d items", len(paged.Items))
	}
	oldest := assert("sort", SearchParams{City: "almaty", Sort: "oldest"}, 3)
	if len(oldest.Items) != 3 || oldest.Items[0].Title != marker+" Design Hybrid" {
		t.Fatalf("oldest sort is incorrect: %#v", oldest.Items)
	}
	assert("opaque injection-like keyword", SearchParams{Query: "'; DELETE FROM jobhub.jobs; --"}, 0)
	assert("hidden vacancy", SearchParams{Query: " Hidden Go"}, 0)
}

func ptr(value string) *string { return &value }
