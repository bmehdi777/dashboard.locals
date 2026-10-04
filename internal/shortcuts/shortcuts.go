package shortcuts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"dashboard.locals/internal/store"
)

var ErrInvalidShortcut = errors.New("invalid shortcut")

type Repository interface {
	ListShortcuts(ctx context.Context, sortBy string, limit int) ([]store.Shortcut, error)
	GetShortcut(ctx context.Context, id string) (store.Shortcut, error)
	CreateShortcut(ctx context.Context, shortcut store.Shortcut) (store.Shortcut, error)
	UpdateShortcut(ctx context.Context, shortcut store.Shortcut) (store.Shortcut, error)
	DeleteShortcut(ctx context.Context, id string) error
	RecordShortcutUse(ctx context.Context, id string) (store.Shortcut, error)
}

type Service struct {
	repo    Repository
	favicon *faviconFetcher
}

func NewService(repo Repository) *Service {
	return NewServiceWithHTTPClient(repo, nil)
}

func NewServiceWithHTTPClient(repo Repository, client *http.Client) *Service {
	return &Service{repo: repo, favicon: newFaviconFetcher(client)}
}

func (s *Service) List(ctx context.Context, sortBy string, limit int) ([]store.Shortcut, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("shortcut service is unavailable")
	}
	sortBy = strings.TrimSpace(strings.ToLower(sortBy))
	if sortBy == "" {
		sortBy = "recent"
	}
	if sortBy != "recent" && sortBy != "popular" {
		return nil, fmt.Errorf("%w: sort must be recent or popular", ErrInvalidShortcut)
	}
	if limit < 0 {
		return nil, fmt.Errorf("%w: limit must not be negative", ErrInvalidShortcut)
	}
	return s.repo.ListShortcuts(ctx, sortBy, limit)
}

func (s *Service) Get(ctx context.Context, id string) (store.Shortcut, error) {
	if s == nil || s.repo == nil {
		return store.Shortcut{}, errors.New("shortcut service is unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return store.Shortcut{}, fmt.Errorf("%w: id is empty", ErrInvalidShortcut)
	}
	return s.repo.GetShortcut(ctx, id)
}

func (s *Service) Create(ctx context.Context, title, rawURL, description string) (store.Shortcut, error) {
	if s == nil || s.repo == nil {
		return store.Shortcut{}, errors.New("shortcut service is unavailable")
	}
	validatedTitle, err := validateTitle(title)
	if err != nil {
		return store.Shortcut{}, err
	}
	validatedURL, err := validateURL(rawURL)
	if err != nil {
		return store.Shortcut{}, err
	}
	validatedDescription, err := validateDescription(description)
	if err != nil {
		return store.Shortcut{}, err
	}
	return s.repo.CreateShortcut(ctx, store.Shortcut{
		Title: validatedTitle, URL: validatedURL, Description: validatedDescription,
	})
}

func (s *Service) Update(ctx context.Context, id string, title, rawURL, description *string) (store.Shortcut, error) {
	shortcut, err := s.Get(ctx, id)
	if err != nil {
		return store.Shortcut{}, err
	}
	if title != nil {
		shortcut.Title, err = validateTitle(*title)
		if err != nil {
			return store.Shortcut{}, err
		}
	}
	if rawURL != nil {
		shortcut.URL, err = validateURL(*rawURL)
		if err != nil {
			return store.Shortcut{}, err
		}
	}
	if description != nil {
		shortcut.Description, err = validateDescription(*description)
		if err != nil {
			return store.Shortcut{}, err
		}
	}
	return s.repo.UpdateShortcut(ctx, shortcut)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteShortcut(ctx, id)
}

func (s *Service) Use(ctx context.Context, id string) (store.Shortcut, error) {
	if s == nil || s.repo == nil {
		return store.Shortcut{}, errors.New("shortcut service is unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return store.Shortcut{}, fmt.Errorf("%w: id is empty", ErrInvalidShortcut)
	}
	return s.repo.RecordShortcutUse(ctx, id)
}

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 200 {
		return "", fmt.Errorf("%w: title must contain between 1 and 200 characters", ErrInvalidShortcut)
	}
	return title, nil
}

func validateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len([]rune(rawURL)) > 2048 || strings.ContainsAny(rawURL, "\r\n") {
		return "", fmt.Errorf("%w: url must contain between 1 and 2048 characters", ErrInvalidShortcut)
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed == nil || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("%w: url must be an HTTP or HTTPS link", ErrInvalidShortcut)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: url must be an HTTP or HTTPS link", ErrInvalidShortcut)
	}
	return rawURL, nil
}

func validateDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if len([]rune(description)) > 500 {
		return "", fmt.Errorf("%w: description must contain at most 500 characters", ErrInvalidShortcut)
	}
	return description, nil
}
