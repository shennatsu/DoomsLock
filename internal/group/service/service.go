package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	groupRepo "github.com/doomslock/backend/internal/group/repository"
)

var (
	ErrGroupNotFound    = errors.New("group not found")
	ErrNotMember        = errors.New("you are not a member of this group")
	ErrNotAdmin         = errors.New("only admin can perform this action")
	ErrGroupFull        = errors.New("group has reached maximum members")
	ErrAlreadyMember    = errors.New("user is already a member")
	ErrInviteInvalid    = errors.New("invite link is invalid or expired")
	ErrCannotRemoveSelf = errors.New("use leave endpoint instead")
)

type CreateRequest struct {
	Name string `json:"name" validate:"required,min=2,max=50"`
}

type InviteRequest struct {
	MaxUses int `json:"max_uses" validate:"omitempty,min=1,max=10"`
}

type JoinRequest struct {
	Token string `json:"token" validate:"required"`
}

type GroupDetail struct {
	Group   groupRepo.Group        `json:"group"`
	Members []groupRepo.GroupMember `json:"members"`
}

type InviteResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxUses   int       `json:"max_uses"`
}

type Service interface {
	Create(ctx context.Context, userID string, req CreateRequest) (*groupRepo.Group, error)
	GetByID(ctx context.Context, userID, groupID string) (*GroupDetail, error)
	ListMyGroups(ctx context.Context, userID string) ([]groupRepo.Group, error)
	CreateInvite(ctx context.Context, userID, groupID string, req InviteRequest) (*InviteResponse, error)
	AcceptInvite(ctx context.Context, userID string, req JoinRequest) (*groupRepo.Group, error)
	LeaveGroup(ctx context.Context, userID, groupID string) error
	RemoveMember(ctx context.Context, adminID, groupID, targetUserID string) error
}

type service struct {
	repo groupRepo.Repository
}

func New(repo groupRepo.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, userID string, req CreateRequest) (*groupRepo.Group, error) {
	g := &groupRepo.Group{
		ID:         uuid.New().String(),
		Name:       req.Name,
		CreatedBy:  userID,
		MaxMembers: 6,
	}

	if err := s.repo.Create(ctx, g); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	if err := s.repo.AddMember(ctx, g.ID, userID, "admin"); err != nil {
		return nil, fmt.Errorf("add creator as admin: %w", err)
	}

	return g, nil
}

func (s *service) GetByID(ctx context.Context, userID, groupID string) (*GroupDetail, error) {
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}

	ok, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrNotMember
	}

	members, err := s.repo.GetMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	return &GroupDetail{Group: *group, Members: members}, nil
}

func (s *service) ListMyGroups(ctx context.Context, userID string) ([]groupRepo.Group, error) {
	groups, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	if groups == nil {
		groups = []groupRepo.Group{}
	}
	return groups, nil
}

func (s *service) CreateInvite(ctx context.Context, userID, groupID string, req InviteRequest) (*InviteResponse, error) {
	ok, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrNotMember
	}

	maxUses := 1
	if req.MaxUses > 0 {
		maxUses = req.MaxUses
	}

	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	inv := &groupRepo.GroupInvite{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		InvitedBy: userID,
		Token:     token,
		Status:    "pending",
		MaxUses:   maxUses,
		UsedCount: 0,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.CreateInvite(ctx, inv); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	return &InviteResponse{Token: token, ExpiresAt: expiresAt, MaxUses: maxUses}, nil
}

func (s *service) AcceptInvite(ctx context.Context, userID string, req JoinRequest) (*groupRepo.Group, error) {
	inv, err := s.repo.GetInviteByToken(ctx, req.Token)
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	if inv == nil {
		return nil, ErrInviteInvalid
	}

	already, err := s.repo.IsMember(ctx, inv.GroupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if already {
		return nil, ErrAlreadyMember
	}

	count, err := s.repo.CountMembers(ctx, inv.GroupID)
	if err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}

	group, err := s.repo.GetByID(ctx, inv.GroupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if count >= group.MaxMembers {
		return nil, ErrGroupFull
	}

	if err := s.repo.AddMember(ctx, inv.GroupID, userID, "member"); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}
	_ = s.repo.UseInvite(ctx, inv.ID)

	return group, nil
}

func (s *service) LeaveGroup(ctx context.Context, userID, groupID string) error {
	ok, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return ErrNotMember
	}
	return s.repo.RemoveMember(ctx, groupID, userID)
}

func (s *service) RemoveMember(ctx context.Context, adminID, groupID, targetUserID string) error {
	if adminID == targetUserID {
		return ErrCannotRemoveSelf
	}

	role, err := s.repo.GetMemberRole(ctx, groupID, adminID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotMember
		}
		return fmt.Errorf("get role: %w", err)
	}
	if role != "admin" {
		return ErrNotAdmin
	}

	return s.repo.RemoveMember(ctx, groupID, targetUserID)
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
