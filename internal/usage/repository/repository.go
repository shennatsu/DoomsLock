package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type UsageLog struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PackageName string    `json:"package_name"`
	DurationSec int       `json:"duration_sec"`
	RecordedAt  time.Time `json:"recorded_at"`
	SyncedAt    time.Time `json:"synced_at"`
}

type DailySummary struct {
	PackageName string `json:"package_name"`
	TotalSec    int    `json:"total_seconds"`
}

type Repository interface {
	BatchInsert(ctx context.Context, logs []UsageLog) error
	GetDailySummary(ctx context.Context, userID, date string) ([]DailySummary, error)
	GetDailyTotal(ctx context.Context, userID, date string) (int, error)
}

type repo struct{ db *database.Pool }

func New(db *pgxpool.Pool) Repository { return &repo{db: db} }

func (r *repo) BatchInsert(ctx context.Context, logs []UsageLog) error {
	for _, l := range logs {
		_, err := r.db.Exec(ctx, `
			INSERT INTO usage_logs (id, user_id, package_name, duration_sec, recorded_at)
			VALUES ($1, $2, $3, $4, $5)
		`, l.ID, l.UserID, l.PackageName, l.DurationSec, l.RecordedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) GetDailySummary(ctx context.Context, userID, date string) ([]DailySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT package_name, SUM(duration_sec) as total
		FROM usage_logs
		WHERE user_id = $1 AND recorded_at::date = $2::date
		GROUP BY package_name
		ORDER BY total DESC
	`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailySummary
	for rows.Next() {
		var s DailySummary
		if err := rows.Scan(&s.PackageName, &s.TotalSec); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *repo) GetDailyTotal(ctx context.Context, userID, date string) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(duration_sec), 0)
		FROM usage_logs
		WHERE user_id = $1 AND recorded_at::date = $2::date
	`, userID, date).Scan(&total)
	return total, err
}
