package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	extRepo "github.com/doomslock/backend/internal/extension/repository"
	groupRepo "github.com/doomslock/backend/internal/group/repository"
	limitRepo "github.com/doomslock/backend/internal/limit/repository"
)

var (
	ErrExtNotFound   = errors.New("extension not found")
	ErrAlreadyVoted  = errors.New("you already voted")
	ErrCannotVoteOwn = errors.New("cannot vote on your own request")
	ErrExtExpired    = errors.New("extension request has expired")
	ErrExtResolved   = errors.New("extension already resolved")
	ErrNotInGroup    = errors.New("you are not a member of this group")
)

type RequestInput struct {
	LimitID      string `json:"limit_id"      validate:"required,uuid"`
	ExtraMinutes int    `json:"extra_minutes" validate:"required,min=5,max=120"`
	Reason       string `json:"reason"`
}

type VoteInput struct {
	Vote string `json:"vote" validate:"required,oneof=yes no"`
}

type ExtensionDetail struct {
	Extension extRepo.LimitExtension `json:"extension"`
	Votes     []extRepo.ExtensionVote `json:"votes"`
}

type Service interface {
	Request(ctx context.Context, userID string, req RequestInput) (*extRepo.LimitExtension, error)
	CastVote(ctx context.Context, voterID, extensionID string, req VoteInput) (*extRepo.LimitExtension, error)
	GetByID(ctx context.Context, extensionID string) (*ExtensionDetail, error)
	ListByLimit(ctx context.Context, limitID string) ([]extRepo.LimitExtension, error)
}

type service struct {
	repo      extRepo.Repository
	limitRepo limitRepo.Repository
	groupRepo groupRepo.Repository
}

func New(r extRepo.Repository, lr limitRepo.Repository, gr groupRepo.Repository) Service {
	return &service{repo: r, limitRepo: lr, groupRepo: gr}
}

func (s *service) Request(ctx context.Context, userID string, req RequestInput) (*extRepo.LimitExtension, error) {
	// Ambil limit buat tahu group-nya
	limit, err := s.limitRepo.GetByID(ctx, req.LimitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("limit not found")
		}
		return nil, fmt.Errorf("get limit: %w", err)
	}

	ok, err := s.groupRepo.IsMember(ctx, limit.GroupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrNotInGroup
	}

	// Hitung votes yang dibutuhkan = majority dari group members (selain requester)
	memberCount, err := s.groupRepo.CountMembers(ctx, limit.GroupID)
	if err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}
	votesNeeded := (memberCount - 1 + 1) / 2 // majority dari members selain requester
	if votesNeeded < 1 {
		votesNeeded = 1
	}

	ext := &extRepo.LimitExtension{
		ID:           uuid.New().String(),
		LimitID:      req.LimitID,
		RequestedBy:  userID,
		ExtraMinutes: req.ExtraMinutes,
		Reason:       req.Reason,
		Status:       "pending",
		VotesNeeded:  votesNeeded,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := s.repo.Create(ctx, ext); err != nil {
		return nil, fmt.Errorf("create extension: %w", err)
	}

	return ext, nil
}

func (s *service) CastVote(ctx context.Context, voterID, extensionID string, req VoteInput) (*extRepo.LimitExtension, error) {
	ext, err := s.repo.GetByID(ctx, extensionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExtNotFound
		}
		return nil, fmt.Errorf("get extension: %w", err)
	}

	if ext.Status != "pending" {
		return nil, ErrExtResolved
	}
	if time.Now().After(ext.ExpiresAt) {
		_ = s.repo.Resolve(ctx, ext.ID, "expired")
		return nil, ErrExtExpired
	}
	if ext.RequestedBy == voterID {
		return nil, ErrCannotVoteOwn
	}

	already, err := s.repo.HasVoted(ctx, extensionID, voterID)
	if err != nil {
		return nil, fmt.Errorf("check voted: %w", err)
	}
	if already {
		return nil, ErrAlreadyVoted
	}

	vote := &extRepo.ExtensionVote{
		ID:          uuid.New().String(),
		ExtensionID: extensionID,
		VoterID:     voterID,
		Vote:        req.Vote,
	}

	if err := s.repo.CastVote(ctx, vote); err != nil {
		return nil, fmt.Errorf("cast vote: %w", err)
	}

	if req.Vote == "yes" {
		_ = s.repo.IncrVotesYes(ctx, extensionID)
		ext.VotesYes++
	} else {
		_ = s.repo.IncrVotesNo(ctx, extensionID)
		ext.VotesNo++
	}

	// Auto-resolve kalau quorum terpenuhi
	if ext.VotesYes >= ext.VotesNeeded {
		_ = s.repo.Resolve(ctx, ext.ID, "approved")
		ext.Status = "approved"
	} else if ext.VotesNo >= ext.VotesNeeded {
		_ = s.repo.Resolve(ctx, ext.ID, "rejected")
		ext.Status = "rejected"
	}

	return ext, nil
}

func (s *service) GetByID(ctx context.Context, extensionID string) (*ExtensionDetail, error) {
	ext, err := s.repo.GetByID(ctx, extensionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExtNotFound
		}
		return nil, fmt.Errorf("get extension: %w", err)
	}

	votes, err := s.repo.GetVotes(ctx, extensionID)
	if err != nil {
		return nil, fmt.Errorf("get votes: %w", err)
	}
	if votes == nil {
		votes = []extRepo.ExtensionVote{}
	}

	return &ExtensionDetail{Extension: *ext, Votes: votes}, nil
}

func (s *service) ListByLimit(ctx context.Context, limitID string) ([]extRepo.LimitExtension, error) {
	exts, err := s.repo.ListByLimit(ctx, limitID)
	if err != nil {
		return nil, fmt.Errorf("list extensions: %w", err)
	}
	if exts == nil {
		exts = []extRepo.LimitExtension{}
	}
	return exts, nil
}
