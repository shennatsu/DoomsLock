package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FCMToken     string
	Timezone     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	UpdateFCMToken(ctx context.Context, userID, token string) error
	GetFCMTokensByUserIDs(ctx context.Context, userIDs []string) ([]string, error)
}

type repo struct {
	db *database.Pool
}

func New(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}

func (r *repo) Create(ctx context.Context, u *User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, fcm_token, timezone)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, u.ID, u.Username, u.Email, u.PasswordHash, u.FCMToken, u.Timezone)
	return err
}

func (r *repo) FindByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, fcm_token, timezone, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.FCMToken, &u.Timezone, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *repo) FindByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, fcm_token, timezone, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.FCMToken, &u.Timezone, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *repo) UpdateFCMToken(ctx context.Context, userID, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET fcm_token = $1, updated_at = NOW() WHERE id = $2`,
		token, userID,
	)
	return err
}

func (r *repo) GetFCMTokensByUserIDs(ctx context.Context, userIDs []string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fcm_token FROM users
		WHERE id = ANY($1) AND fcm_token != ''
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
