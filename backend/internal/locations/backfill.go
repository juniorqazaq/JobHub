package locations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackfillStats struct {
	Matched   int `json:"matched"`
	Unchanged int `json:"unchanged"`
	Ambiguous int `json:"ambiguous"`
	Unknown   int `json:"unknown"`
}

// BackfillImportedJobs fills only missing canonical city IDs. Raw provider
// locations and existing canonical values are never changed.
func BackfillImportedJobs(ctx context.Context, pool *pgxpool.Pool) (BackfillStats, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return BackfillStats{}, fmt.Errorf("begin city backfill: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(location_raw, ''), COALESCE(canonical_city_id, '')
		FROM jobhub.jobs
		WHERE source <> 'jobhub'
		ORDER BY id
		FOR UPDATE`)
	if err != nil {
		return BackfillStats{}, fmt.Errorf("list imported jobs: %w", err)
	}
	type row struct{ id, location, city string }
	items := []row{}
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.location, &item.city); err != nil {
			rows.Close()
			return BackfillStats{}, fmt.Errorf("scan imported job: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return BackfillStats{}, fmt.Errorf("iterate imported jobs: %w", err)
	}
	rows.Close()

	stats := BackfillStats{}
	for _, item := range items {
		if item.city != "" {
			stats.Unchanged++
			continue
		}
		city, status := Classify(item.location)
		switch status {
		case MatchMatched:
			tag, err := tx.Exec(ctx, `UPDATE jobhub.jobs SET canonical_city_id=$1 WHERE id=$2::uuid AND canonical_city_id IS NULL`, city, item.id)
			if err != nil {
				return BackfillStats{}, fmt.Errorf("update imported city: %w", err)
			}
			if tag.RowsAffected() == 1 {
				stats.Matched++
			} else {
				stats.Unchanged++
			}
		case MatchAmbiguous:
			stats.Ambiguous++
		default:
			stats.Unknown++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return BackfillStats{}, fmt.Errorf("commit city backfill: %w", err)
	}
	return stats, nil
}
