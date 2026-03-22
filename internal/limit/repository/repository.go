package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type AppLimit struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	GroupID           string    `json:"group_id"`
	PackageName       string    `json:"package_name"`
	AppLabel          string    `json:"app_label"`
	DailyLimitMinutes int       `json:"daily_limit_minutes"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, l *AppLimit) error
	GetByID(ctx context.Context, id string) (*AppLimit, error)
	ListByUserAndGroup(ctx context.Context, userID, groupID string) ([]AppLimit, error)
	Update(ctx context.Context, id string, dailyMinutes *int, isActive *bool) (*AppLimit, error)
	SoftDelete(ctx context.Context, id string) error
}

type repo struct{ db *database.Pool }

func New(db *pgxpool.Pool) Repository { return &repo{db: db} }

func (r *repo) Create(ctx context.Context, l *AppLimit) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO app_limits (id, user_id, group_id, package_name, app_label, daily_limit_minutes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, l.ID, l.UserID, l.GroupID, l.PackageName, l.AppLabel, l.DailyLimitMinutes, l.IsActive)
	return err
}

func (r *repo) GetByID(ctx context.Context, id string) (*AppLimit, error) {
	var l AppLimit
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, group_id, package_name, app_label,
		       daily_limit_minutes, is_active, created_at, updated_at
		FROM app_limits WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&l.ID, &l.UserID, &l.GroupID, &l.PackageName, &l.AppLabel,
		&l.DailyLimitMinutes, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *repo) ListByUserAndGroup(ctx context.Context, userID, groupID string) ([]AppLimit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, group_id, package_name, app_label,
		       daily_limit_minutes, is_active, created_at, updated_at
		FROM app_limits
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		ORDER BY app_label
	`, userID, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AppLimit
	for rows.Next() {
		var l AppLimit
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.GroupID, &l.PackageName, &l.AppLabel,
			&l.DailyLimitMinutes, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *repo) Update(ctx context.Context, id string, dailyMinutes *int, isActive *bool) (*AppLimit, error) {
	var l AppLimit
	err := r.db.QueryRow(ctx, `
		UPDATE app_limits
		SET daily_limit_minutes = COALESCE($2, daily_limit_minutes),
		    is_active           = COALESCE($3, is_active),
		    updated_at          = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, group_id, package_name, app_label,
		          daily_limit_minutes, is_active, created_at, updated_at
	`, id, dailyMinutes, isActive).Scan(
		&l.ID, &l.UserID, &l.GroupID, &l.PackageName, &l.AppLabel,
		&l.DailyLimitMinutes, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *repo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_limits SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}
