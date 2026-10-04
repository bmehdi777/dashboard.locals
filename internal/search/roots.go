package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dashboard.locals/internal/store"
)

var (
	ErrInvalidRoot  = errors.New("invalid search root")
	ErrRootNotFound = errors.New("search root not found")
	ErrRootDisabled = errors.New("search root is disabled")
	ErrPathOutside  = errors.New("path is outside the search root")
	ErrSymlink      = errors.New("symbolic links are not allowed")
)

type RootRepository interface {
	ListSearchRoots(ctx context.Context, includeDisabled bool) ([]store.SearchRoot, error)
	GetSearchRoot(ctx context.Context, id string) (store.SearchRoot, error)
	CreateSearchRoot(ctx context.Context, root store.SearchRoot) (store.SearchRoot, error)
	UpdateSearchRoot(ctx context.Context, root store.SearchRoot) (store.SearchRoot, error)
	DeleteSearchRoot(ctx context.Context, id string) error
}

type RootService struct {
	repo RootRepository
}

func NewRootService(repo RootRepository) *RootService {
	return &RootService{repo: repo}
}

func (s *RootService) List(ctx context.Context, includeDisabled bool) ([]store.SearchRoot, error) {
	return s.repo.ListSearchRoots(ctx, includeDisabled)
}

func (s *RootService) Get(ctx context.Context, id string) (store.SearchRoot, error) {
	if strings.TrimSpace(id) == "" {
		return store.SearchRoot{}, fmt.Errorf("%w: id is empty", ErrInvalidRoot)
	}
	root, err := s.repo.GetSearchRoot(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.SearchRoot{}, ErrRootNotFound
		}
		return store.SearchRoot{}, err
	}
	return root, nil
}

func (s *RootService) Create(ctx context.Context, name, path string, enabled bool) (store.SearchRoot, error) {
	name, err := validateName(name)
	if err != nil {
		return store.SearchRoot{}, err
	}
	normalized, err := NormalizeRootPath(path)
	if err != nil {
		return store.SearchRoot{}, err
	}
	return s.repo.CreateSearchRoot(ctx, store.SearchRoot{Name: name, Path: normalized, Enabled: enabled})
}

func (s *RootService) Update(ctx context.Context, id string, name, path *string, enabled *bool) (store.SearchRoot, error) {
	root, err := s.Get(ctx, id)
	if err != nil {
		return store.SearchRoot{}, err
	}
	if name != nil {
		root.Name, err = validateName(*name)
		if err != nil {
			return store.SearchRoot{}, err
		}
	}
	if path != nil {
		root.Path, err = NormalizeRootPath(*path)
		if err != nil {
			return store.SearchRoot{}, err
		}
	}
	if enabled != nil {
		root.Enabled = *enabled
	}
	return s.repo.UpdateSearchRoot(ctx, root)
}

func (s *RootService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteSearchRoot(ctx, id)
}

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 200 {
		return "", fmt.Errorf("%w: name must contain between 1 and 200 characters", ErrInvalidRoot)
	}
	return name, nil
}

// NormalizeRootPath resolves a user supplied root to an absolute canonical
// directory. Canonicalizing the root once means subsequent containment checks
// can reject links which would otherwise escape it.
func NormalizeRootPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%w: path is empty", ErrInvalidRoot)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w: resolve path", ErrInvalidRoot)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: path is not accessible", ErrInvalidRoot)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory", ErrInvalidRoot)
	}
	directory, err := os.Open(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: path is not accessible", ErrInvalidRoot)
	}
	_ = directory.Close()
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: canonicalize path", ErrInvalidRoot)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: resolve canonical path", ErrInvalidRoot)
	}
	return filepath.Clean(canonical), nil
}

// ResolveResultPath validates a relative result path and returns its absolute
// path. Every component is checked with Lstat so a symlink cannot be followed.
func ResolveResultPath(root, result string) (string, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(result) == "" {
		return "", ErrPathOutside
	}
	root, err := NormalizeRootPath(root)
	if err != nil {
		return "", err
	}
	result = filepath.FromSlash(result)
	if filepath.IsAbs(result) {
		rel, relErr := filepath.Rel(root, filepath.Clean(result))
		if relErr != nil {
			return "", ErrPathOutside
		}
		result = rel
	}
	clean := filepath.Clean(result)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrPathOutside
	}
	absolute := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathOutside
	}

	current := root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return "", os.ErrNotExist
			}
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", ErrSymlink
		}
	}
	return absolute, nil
}
