package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	groupRepo "github.com/doomslock/backend/internal/group/repository"
	"github.com/doomslock/backend/internal/limit/repository"
)

var (
	ErrLimitNotFound = errors.New("app limit not found")
	ErrNotOwner      = errors.New("you don't own this limit")
	ErrNotInGroup    = errors.New("you are not a member of this group")
)

type CreateRequest struct {
	GroupID           string `json:"group_id"      validate:"required,uuid"`
	PackageName       string `json:"package_name"  validate:"required"`
	AppLabel          string `json:"app_label"     validate:"required"`
	DailyLimitMinutes int    `json:"daily_limit_minutes" validate:"required,min=1"`
}

type UpdateRequest struct {
	DailyLimitMinutes *int  `json:"daily_limit_minutes" validate:"omitempty,min=1"`
	IsActive          *bool `json:"is_active"`
}

type Service interface {
	Create(ctx context.Context, userID string, req CreateRequest) (*repository.AppLimit, error)
	ListByGroup(ctx context.Context, userID, groupID string) ([]repository.AppLimit, error)
	Update(ctx context.Context, userID, limitID string, req UpdateRequest) (*repository.AppLimit, error)
	Delete(ctx context.Context, userID, limitID string) error
	GetByID(ctx context.Context, limitID string) (*repository.AppLimit, error)
}

type service struct {
	repo      repository.Repository
	groupRepo groupRepo.Repository
}

func New(repo repository.Repository, gr groupRepo.Repository) Service {
	return &service{repo: repo, groupRepo: gr}
}

func (s *service) Create(ctx context.Context, userID string, req CreateRequest) (*repository.AppLimit, error) {
	ok, err := s.groupRepo.IsMember(ctx, req.GroupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrNotInGroup
	}

	l := &repository.AppLimit{
		ID:                uuid.New().String(),
		UserID:            userID,
		GroupID:           req.GroupID,
		PackageName:       req.PackageName,
		AppLabel:          req.AppLabel,
		DailyLimitMinutes: req.DailyLimitMinutes,
		IsActive:          true,
	}

	if err := s.repo.Create(ctx, l); err != nil {
		return nil, fmt.Errorf("create limit: %w", err)
	}

	return l, nil
}

func (s *service) ListByGroup(ctx context.Context, userID, groupID string) ([]repository.AppLimit, error) {
	ok, err := s.groupRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrNotInGroup
	}

	limits, err := s.repo.ListByUserAndGroup(ctx, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("list limits: %w", err)
	}
	if limits == nil {
		limits = []repository.AppLimit{}
	}
	return limits, nil
}

func (s *service) Update(ctx context.Context, userID, limitID string, req UpdateRequest) (*repository.AppLimit, error) {
	limit, err := s.repo.GetByID(ctx, limitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLimitNotFound
		}
		return nil, fmt.Errorf("get limit: %w", err)
	}
	if limit.UserID != userID {
		return nil, ErrNotOwner
	}

	updated, err := s.repo.Update(ctx, limitID, req.DailyLimitMinutes, req.IsActive)
	if err != nil {
		return nil, fmt.Errorf("update limit: %w", err)
	}
	return updated, nil
}

func (s *service) Delete(ctx context.Context, userID, limitID string) error {
	limit, err := s.repo.GetByID(ctx, limitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLimitNotFound
		}
		return fmt.Errorf("get limit: %w", err)
	}
	if limit.UserID != userID {
		return ErrNotOwner
	}
	return s.repo.SoftDelete(ctx, limitID)
}

func (s *service) GetByID(ctx context.Context, limitID string) (*repository.AppLimit, error) {
	limit, err := s.repo.GetByID(ctx, limitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLimitNotFound
		}
		return nil, fmt.Errorf("get limit: %w", err)
	}
	return limit, nil
}
