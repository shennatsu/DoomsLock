package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type UserStreak struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CurrentDays int       `json:"current_days"`
	LongestDays int       `json:"longest_days"`
	LastClean   *string   `json:"last_clean,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserBadge struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Badge    string    `json:"badge"`
	EarnedAt time.Time `json:"earned_at"`
}

type Repository interface {
	GetStreak(ctx context.Context, userID string) (*UserStreak, error)
	UpsertStreak(ctx context.Context, userID string, currentDays, longestDays int, lastClean string) error
	ListBadges(ctx context.Context, userID string) ([]UserBadge, error)
	AwardBadge(ctx context.Context, userID, badge string) error
}

type repo struct{ db *database.Pool }

func New(db *pgxpool.Pool) Repository { return &repo{db: db} }

func (r *repo) GetStreak(ctx context.Context, userID string) (*UserStreak, error) {
	var s UserStreak
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, current_days, longest_days, last_clean::text, updated_at
		FROM user_streaks WHERE user_id = $1
	`, userID).Scan(&s.ID, &s.UserID, &s.CurrentDays, &s.LongestDays, &s.LastClean, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &UserStreak{UserID: userID}, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *repo) UpsertStreak(ctx context.Context, userID string, currentDays, longestDays int, lastClean string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_streaks (user_id, current_days, longest_days, last_clean, updated_at)
		VALUES ($1, $2, $3, $4::date, NOW())
		ON CONFLICT (user_id) DO UPDATE
			SET current_days = $2, longest_days = $3, last_clean = $4::date, updated_at = NOW()
	`, userID, currentDays, longestDays, lastClean)
	return err
}

func (r *repo) ListBadges(ctx context.Context, userID string) ([]UserBadge, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, badge, earned_at
		FROM user_badges WHERE user_id = $1
		ORDER BY earned_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserBadge
	for rows.Next() {
		var b UserBadge
		if err := rows.Scan(&b.ID, &b.UserID, &b.Badge, &b.EarnedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *repo) AwardBadge(ctx context.Context, userID, badge string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_badges (user_id, badge)
		VALUES ($1, $2)
		ON CONFLICT (user_id, badge) DO NOTHING
	`, userID, badge)
	return err
}
