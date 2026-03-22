package service

import (
	"context"
	"fmt"
	"time"

	"github.com/doomslock/backend/internal/reward/repository"
	usageRepo "github.com/doomslock/backend/internal/usage/repository"
)

type Service interface {
	GetStreak(ctx context.Context, userID string) (*repository.UserStreak, error)
	UpdateStreak(ctx context.Context, userID string) (*repository.UserStreak, error)
	GetBadges(ctx context.Context, userID string) ([]repository.UserBadge, error)
}

type service struct {
	repo     repository.Repository
	usageRep usageRepo.Repository
}

func New(repo repository.Repository, ur usageRepo.Repository) Service {
	return &service{repo: repo, usageRep: ur}
}

func (s *service) GetStreak(ctx context.Context, userID string) (*repository.UserStreak, error) {
	streak, err := s.repo.GetStreak(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get streak: %w", err)
	}
	return streak, nil
}

// UpdateStreak dipanggil setelah usage sync — cek apakah hari ini "clean day"
// (total usage < threshold, misal 60 menit total)
func (s *service) UpdateStreak(ctx context.Context, userID string) (*repository.UserStreak, error) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	totalToday, err := s.usageRep.GetDailyTotal(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get daily total: %w", err)
	}

	// Clean day = total usage kurang dari 1 jam (3600 detik)
	cleanToday := totalToday < 3600

	existing, err := s.repo.GetStreak(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get streak: %w", err)
	}

	currentDays := existing.CurrentDays
	longestDays := existing.LongestDays

	if cleanToday {
		// Kalau kemarin juga clean, tambah streak
		if existing.LastClean != nil && *existing.LastClean == yesterday {
			currentDays++
		} else if existing.LastClean != nil && *existing.LastClean == today {
			// Sudah update hari ini, skip
		} else {
			currentDays = 1
		}

		if currentDays > longestDays {
			longestDays = currentDays
		}

		if err := s.repo.UpsertStreak(ctx, userID, currentDays, longestDays, today); err != nil {
			return nil, fmt.Errorf("upsert streak: %w", err)
		}

		// Auto badge berdasarkan streak
		s.checkBadges(ctx, userID, currentDays)
	}

	return s.repo.GetStreak(ctx, userID)
}

func (s *service) GetBadges(ctx context.Context, userID string) ([]repository.UserBadge, error) {
	badges, err := s.repo.ListBadges(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list badges: %w", err)
	}
	if badges == nil {
		badges = []repository.UserBadge{}
	}
	return badges, nil
}

func (s *service) checkBadges(ctx context.Context, userID string, streak int) {
	badges := map[int]string{
		3:  "3_day_streak",
		7:  "week_warrior",
		14: "two_week_champion",
		30: "monthly_master",
	}

	for threshold, badge := range badges {
		if streak >= threshold {
			_ = s.repo.AwardBadge(ctx, userID, badge)
		}
	}
}
