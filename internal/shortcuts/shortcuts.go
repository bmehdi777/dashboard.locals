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
	ReorderShortcuts(ctx context.Context, order []store.ShortcutOrder) error
	ListShortcutFolders(ctx context.Context) ([]store.ShortcutFolder, error)
	GetShortcutFolder(ctx context.Context, id string) (store.ShortcutFolder, error)
	CreateShortcutFolder(ctx context.Context, folder store.ShortcutFolder) (store.ShortcutFolder, error)
	UpdateShortcutFolder(ctx context.Context, folder store.ShortcutFolder) (store.ShortcutFolder, error)
	DeleteShortcutFolder(ctx context.Context, id string) error
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
	if sortBy != "recent" && sortBy != "popular" && sortBy != "custom" {
		return nil, fmt.Errorf("%w: sort must be recent, popular, or custom", ErrInvalidShortcut)
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

func (s *Service) Create(ctx context.Context, title, rawURL, description string, folderID *string) (store.Shortcut, error) {
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
	validatedFolderID, err := s.validateFolderID(ctx, folderID)
	if err != nil {
		return store.Shortcut{}, err
	}
	return s.repo.CreateShortcut(ctx, store.Shortcut{
		Title: validatedTitle, URL: validatedURL, Description: validatedDescription, FolderID: validatedFolderID,
	})
}

func (s *Service) Update(ctx context.Context, id string, title, rawURL, description, folderID *string) (store.Shortcut, error) {
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
	if folderID != nil {
		shortcut.FolderID, err = s.validateFolderID(ctx, folderID)
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

func (s *Service) Reorder(ctx context.Context, order []store.ShortcutOrder) error {
	if s == nil || s.repo == nil {
		return errors.New("shortcut service is unavailable")
	}
	if len(order) == 0 {
		return fmt.Errorf("%w: shortcut order cannot be empty", ErrInvalidShortcut)
	}

	current, err := s.repo.ListShortcuts(ctx, "custom", 0)
	if err != nil {
		return err
	}
	if len(order) != len(current) {
		return fmt.Errorf("%w: shortcut order must contain every shortcut exactly once", ErrInvalidShortcut)
	}
	existing := make(map[string]struct{}, len(current))
	for _, shortcut := range current {
		existing[shortcut.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(order))
	normalizedOrder := make([]store.ShortcutOrder, len(order))
	for index, item := range order {
		id := item.ID
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("%w: shortcut id at position %d is empty", ErrInvalidShortcut, index)
		}
		if _, ok := existing[id]; !ok {
			return fmt.Errorf("%w: shortcut %q is not part of the current collection", ErrInvalidShortcut, id)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: shortcut order contains duplicate id %q", ErrInvalidShortcut, id)
		}
		seen[id] = struct{}{}
		folderID, err := s.validateFolderID(ctx, item.FolderID)
		if err != nil {
			return err
		}
		normalizedOrder[index] = store.ShortcutOrder{ID: id, FolderID: folderID}
	}
	return s.repo.ReorderShortcuts(ctx, normalizedOrder)
}

func (s *Service) ListFolders(ctx context.Context) ([]store.ShortcutFolder, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("shortcut service is unavailable")
	}
	return s.repo.ListShortcutFolders(ctx)
}

func (s *Service) CreateFolder(ctx context.Context, name string) (store.ShortcutFolder, error) {
	if s == nil || s.repo == nil {
		return store.ShortcutFolder{}, errors.New("shortcut service is unavailable")
	}
	validatedName, err := validateFolderName(name)
	if err != nil {
		return store.ShortcutFolder{}, err
	}
	return s.repo.CreateShortcutFolder(ctx, store.ShortcutFolder{Name: validatedName})
}

func (s *Service) UpdateFolder(ctx context.Context, id, name string) (store.ShortcutFolder, error) {
	if s == nil || s.repo == nil {
		return store.ShortcutFolder{}, errors.New("shortcut service is unavailable")
	}
	folder, err := s.repo.GetShortcutFolder(ctx, strings.TrimSpace(id))
	if err != nil {
		return store.ShortcutFolder{}, err
	}
	folder.Name, err = validateFolderName(name)
	if err != nil {
		return store.ShortcutFolder{}, err
	}
	return s.repo.UpdateShortcutFolder(ctx, folder)
}

func (s *Service) DeleteFolder(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return errors.New("shortcut service is unavailable")
	}
	if _, err := s.repo.GetShortcutFolder(ctx, strings.TrimSpace(id)); err != nil {
		return err
	}
	return s.repo.DeleteShortcutFolder(ctx, strings.TrimSpace(id))
}

func (s *Service) validateFolderID(ctx context.Context, folderID *string) (*string, error) {
	if folderID == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*folderID)
	if normalized == "" {
		return nil, nil
	}
	if _, err := s.repo.GetShortcutFolder(ctx, normalized); err != nil {
		return nil, err
	}
	return &normalized, nil
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

func validateFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return "", fmt.Errorf("%w: folder name must contain between 1 and 100 characters", ErrInvalidShortcut)
	}
	return name, nil
}
