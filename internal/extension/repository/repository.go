package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type LimitExtension struct {
	ID           string     `json:"id"`
	LimitID      string     `json:"limit_id"`
	RequestedBy  string     `json:"requested_by"`
	ExtraMinutes int        `json:"extra_minutes"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	VotesNeeded  int        `json:"votes_needed"`
	VotesYes     int        `json:"votes_yes"`
	VotesNo      int        `json:"votes_no"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

type ExtensionVote struct {
	ID          string    `json:"id"`
	ExtensionID string    `json:"extension_id"`
	VoterID     string    `json:"voter_id"`
	Vote        string    `json:"vote"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, ext *LimitExtension) error
	GetByID(ctx context.Context, id string) (*LimitExtension, error)
	ListByLimit(ctx context.Context, limitID string) ([]LimitExtension, error)
	CastVote(ctx context.Context, v *ExtensionVote) error
	HasVoted(ctx context.Context, extensionID, voterID string) (bool, error)
	IncrVotesYes(ctx context.Context, id string) error
	IncrVotesNo(ctx context.Context, id string) error
	Resolve(ctx context.Context, id, status string) error
	GetVotes(ctx context.Context, extensionID string) ([]ExtensionVote, error)
}

type repo struct{ db *database.Pool }

func New(db *pgxpool.Pool) Repository { return &repo{db: db} }

func (r *repo) Create(ctx context.Context, ext *LimitExtension) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO limit_extensions (id, limit_id, requested_by, extra_minutes, reason, status, votes_needed, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, ext.ID, ext.LimitID, ext.RequestedBy, ext.ExtraMinutes, ext.Reason, ext.Status, ext.VotesNeeded, ext.ExpiresAt)
	return err
}

func (r *repo) GetByID(ctx context.Context, id string) (*LimitExtension, error) {
	var e LimitExtension
	err := r.db.QueryRow(ctx, `
		SELECT id, limit_id, requested_by, extra_minutes, reason, status,
		       votes_needed, votes_yes, votes_no, expires_at, created_at, resolved_at
		FROM limit_extensions WHERE id = $1
	`, id).Scan(
		&e.ID, &e.LimitID, &e.RequestedBy, &e.ExtraMinutes, &e.Reason, &e.Status,
		&e.VotesNeeded, &e.VotesYes, &e.VotesNo, &e.ExpiresAt, &e.CreatedAt, &e.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *repo) ListByLimit(ctx context.Context, limitID string) ([]LimitExtension, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, limit_id, requested_by, extra_minutes, reason, status,
		       votes_needed, votes_yes, votes_no, expires_at, created_at, resolved_at
		FROM limit_extensions WHERE limit_id = $1
		ORDER BY created_at DESC
	`, limitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LimitExtension
	for rows.Next() {
		var e LimitExtension
		if err := rows.Scan(
			&e.ID, &e.LimitID, &e.RequestedBy, &e.ExtraMinutes, &e.Reason, &e.Status,
			&e.VotesNeeded, &e.VotesYes, &e.VotesNo, &e.ExpiresAt, &e.CreatedAt, &e.ResolvedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *repo) CastVote(ctx context.Context, v *ExtensionVote) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO extension_votes (id, extension_id, voter_id, vote)
		VALUES ($1, $2, $3, $4)
	`, v.ID, v.ExtensionID, v.VoterID, v.Vote)
	return err
}

func (r *repo) HasVoted(ctx context.Context, extensionID, voterID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM extension_votes WHERE extension_id = $1 AND voter_id = $2)
	`, extensionID, voterID).Scan(&exists)
	return exists, err
}

func (r *repo) IncrVotesYes(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE limit_extensions SET votes_yes = votes_yes + 1 WHERE id = $1`, id)
	return err
}

func (r *repo) IncrVotesNo(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE limit_extensions SET votes_no = votes_no + 1 WHERE id = $1`, id)
	return err
}

func (r *repo) Resolve(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE limit_extensions SET status = $2, resolved_at = NOW() WHERE id = $1
	`, id, status)
	return err
}

func (r *repo) GetVotes(ctx context.Context, extensionID string) ([]ExtensionVote, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, extension_id, voter_id, vote, created_at
		FROM extension_votes WHERE extension_id = $1
		ORDER BY created_at
	`, extensionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExtensionVote
	for rows.Next() {
		var v ExtensionVote
		if err := rows.Scan(&v.ID, &v.ExtensionID, &v.VoterID, &v.Vote, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
