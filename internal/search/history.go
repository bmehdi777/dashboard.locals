package search

import (
	"context"
	"errors"
	"strings"

	"dashboard.locals/internal/store"
)

var ErrInvalidHistory = errors.New("invalid search history entry")

type HistoryRepository interface {
	CreateSearchHistory(ctx context.Context, history store.SearchHistory) (store.SearchHistory, error)
	ListSearchHistory(ctx context.Context, limit int) ([]store.SearchHistory, error)
	DeleteSearchHistory(ctx context.Context, id string) error
	ClearSearchHistory(ctx context.Context) error
}

type HistoryService struct {
	repo HistoryRepository
}

func NewHistoryService(repo HistoryRepository) *HistoryService {
	return &HistoryService{repo: repo}
}

func (s *HistoryService) Record(ctx context.Context, history store.SearchHistory) (store.SearchHistory, error) {
	if s == nil || s.repo == nil {
		return store.SearchHistory{}, errors.New("search history service is unavailable")
	}
	if strings.TrimSpace(history.RootID) == "" || strings.TrimSpace(history.Query) == "" {
		return store.SearchHistory{}, ErrInvalidHistory
	}
	if len([]rune(history.Query)) > 4096 || history.ResultCount < 0 {
		return store.SearchHistory{}, ErrInvalidHistory
	}
	if history.Status == "" {
		history.Status = "completed"
	}
	if history.Status != "completed" {
		return store.SearchHistory{}, ErrInvalidHistory
	}
	return s.repo.CreateSearchHistory(ctx, history)
}

func (s *HistoryService) List(ctx context.Context, limit int) ([]store.SearchHistory, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("search history service is unavailable")
	}
	return s.repo.ListSearchHistory(ctx, limit)
}

func (s *HistoryService) Delete(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return errors.New("search history service is unavailable")
	}
	if strings.TrimSpace(id) == "" {
		return ErrInvalidHistory
	}
	return s.repo.DeleteSearchHistory(ctx, id)
}

func (s *HistoryService) Clear(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return errors.New("search history service is unavailable")
	}
	return s.repo.ClearSearchHistory(ctx)
}
