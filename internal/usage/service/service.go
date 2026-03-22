package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/doomslock/backend/internal/usage/repository"
)

type SyncEntry struct {
	PackageName string `json:"package_name" validate:"required"`
	DurationSec int    `json:"duration_sec" validate:"required,min=1"`
	RecordedAt  string `json:"recorded_at"  validate:"required"`
}

type SyncRequest struct {
	Entries []SyncEntry `json:"entries" validate:"required,min=1,dive"`
}

type Service interface {
	Sync(ctx context.Context, userID string, req SyncRequest) (int, error)
	GetDailySummary(ctx context.Context, userID, date string) ([]repository.DailySummary, error)
}

type service struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Sync(ctx context.Context, userID string, req SyncRequest) (int, error) {
	logs := make([]repository.UsageLog, 0, len(req.Entries))
	for _, e := range req.Entries {
		t, err := time.Parse(time.RFC3339, e.RecordedAt)
		if err != nil {
			t = time.Now()
		}
		logs = append(logs, repository.UsageLog{
			ID:          uuid.New().String(),
			UserID:      userID,
			PackageName: e.PackageName,
			DurationSec: e.DurationSec,
			RecordedAt:  t,
		})
	}

	if err := s.repo.BatchInsert(ctx, logs); err != nil {
		return 0, fmt.Errorf("batch insert: %w", err)
	}

	return len(logs), nil
}

func (s *service) GetDailySummary(ctx context.Context, userID, date string) ([]repository.DailySummary, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	summary, err := s.repo.GetDailySummary(ctx, userID, date)
	if err != nil {
		return nil, fmt.Errorf("daily summary: %w", err)
	}
	if summary == nil {
		summary = []repository.DailySummary{}
	}
	return summary, nil
}
