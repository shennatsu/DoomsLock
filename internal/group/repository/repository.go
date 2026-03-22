package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/doomslock/backend/pkg/database"
)

type Group struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedBy  string    `json:"created_by"`
	MaxMembers int       `json:"max_members"`
	CreatedAt  time.Time `json:"created_at"`
}

type GroupMember struct {
	ID       string    `json:"id"`
	GroupID  string    `json:"group_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
	Username string    `json:"username,omitempty"`
}

type GroupInvite struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	InvitedBy string   `json:"invited_by"`
	Token     string    `json:"token"`
	Status    string    `json:"status"`
	MaxUses   int       `json:"max_uses"`
	UsedCount int       `json:"used_count"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, g *Group) error
	GetByID(ctx context.Context, id string) (*Group, error)
	ListByUser(ctx context.Context, userID string) ([]Group, error)

	AddMember(ctx context.Context, groupID, userID, role string) error
	GetMembers(ctx context.Context, groupID string) ([]GroupMember, error)
	GetMemberRole(ctx context.Context, groupID, userID string) (string, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	CountMembers(ctx context.Context, groupID string) (int, error)
	RemoveMember(ctx context.Context, groupID, userID string) error
	GetMemberUserIDs(ctx context.Context, groupID string) ([]string, error)

	CreateInvite(ctx context.Context, inv *GroupInvite) error
	GetInviteByToken(ctx context.Context, token string) (*GroupInvite, error)
	UseInvite(ctx context.Context, id string) error
}

type repo struct{ db *database.Pool }

func New(db *pgxpool.Pool) Repository { return &repo{db: db} }

func (r *repo) Create(ctx context.Context, g *Group) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO groups (id, name, created_by, max_members)
		VALUES ($1, $2, $3, $4)
	`, g.ID, g.Name, g.CreatedBy, g.MaxMembers)
	return err
}

func (r *repo) GetByID(ctx context.Context, id string) (*Group, error) {
	var g Group
	err := r.db.QueryRow(ctx, `
		SELECT id, name, created_by, max_members, created_at
		FROM groups WHERE id = $1
	`, id).Scan(&g.ID, &g.Name, &g.CreatedBy, &g.MaxMembers, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repo) ListByUser(ctx context.Context, userID string) ([]Group, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.name, g.created_by, g.max_members, g.created_at
		FROM groups g
		JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = $1 AND gm.status = 'active'
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.MaxMembers, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *repo) AddMember(ctx context.Context, groupID, userID, role string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role, status)
		VALUES ($1, $2, $3, 'active')
		ON CONFLICT (group_id, user_id) DO UPDATE
			SET status = 'active', role = EXCLUDED.role, joined_at = NOW()
	`, groupID, userID, role)
	return err
}

func (r *repo) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.joined_at, u.username
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.status = 'active'
		ORDER BY gm.joined_at
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupMember
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.Role, &m.Status, &m.JoinedAt, &m.Username); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *repo) GetMemberRole(ctx context.Context, groupID, userID string) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `
		SELECT role FROM group_members
		WHERE group_id = $1 AND user_id = $2 AND status = 'active'
	`, groupID, userID).Scan(&role)
	return role, err
}

func (r *repo) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM group_members
			WHERE group_id = $1 AND user_id = $2 AND status = 'active'
		)
	`, groupID, userID).Scan(&exists)
	return exists, err
}

func (r *repo) CountMembers(ctx context.Context, groupID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM group_members
		WHERE group_id = $1 AND status = 'active'
	`, groupID).Scan(&count)
	return count, err
}

func (r *repo) RemoveMember(ctx context.Context, groupID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE group_members SET status = 'left'
		WHERE group_id = $1 AND user_id = $2 AND status = 'active'
	`, groupID, userID)
	return err
}

func (r *repo) GetMemberUserIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id FROM group_members
		WHERE group_id = $1 AND status = 'active'
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *repo) CreateInvite(ctx context.Context, inv *GroupInvite) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO group_invites (id, group_id, invited_by, token, status, max_uses, used_count, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, inv.ID, inv.GroupID, inv.InvitedBy, inv.Token, inv.Status, inv.MaxUses, inv.UsedCount, inv.ExpiresAt)
	return err
}

func (r *repo) GetInviteByToken(ctx context.Context, token string) (*GroupInvite, error) {
	var inv GroupInvite
	err := r.db.QueryRow(ctx, `
		SELECT id, group_id, invited_by, token, status, max_uses, used_count, expires_at, created_at
		FROM group_invites
		WHERE token = $1 AND status = 'pending' AND used_count < max_uses AND expires_at > NOW()
	`, token).Scan(
		&inv.ID, &inv.GroupID, &inv.InvitedBy, &inv.Token,
		&inv.Status, &inv.MaxUses, &inv.UsedCount, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *repo) UseInvite(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE group_invites
		SET used_count = used_count + 1,
		    status = CASE WHEN used_count + 1 >= max_uses THEN 'used' ELSE status END
		WHERE id = $1
	`, id)
	return err
}
