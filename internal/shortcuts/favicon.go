package shortcuts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrFaviconNotFound    = errors.New("favicon not found")
	ErrFaviconUnavailable = errors.New("favicon unavailable")
)

const (
	maxFaviconSize  = 512 << 10
	faviconCacheTTL = time.Hour
)

type faviconCacheEntry struct {
	data        []byte
	contentType string
	expiresAt   time.Time
}

type faviconFetcher struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]faviconCacheEntry
}

func newFaviconFetcher(client *http.Client) *faviconFetcher {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &faviconFetcher{client: client, cache: make(map[string]faviconCacheEntry)}
}

func (f *faviconFetcher) fetch(ctx context.Context, rawURL string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", fmt.Errorf("%w: invalid shortcut URL", ErrFaviconUnavailable)
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if cached, ok := f.get(origin); ok {
		return cached.data, cached.contentType, nil
	}

	iconURL := origin + "/favicon.ico"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("%w: create favicon request", ErrFaviconUnavailable)
	}
	request.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	client := *f.client
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			initial := via[0].URL
			if !sameHTTPOrigin(initial, next.URL) {
				return http.ErrUseLastResponse
			}
		}
		if originalCheckRedirect != nil {
			return originalCheckRedirect(next, via)
		}
		return nil
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("%w: fetch favicon", ErrFaviconUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", ErrFaviconNotFound
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxFaviconSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read favicon", ErrFaviconUnavailable)
	}
	if len(data) == 0 || len(data) > maxFaviconSize {
		return nil, "", ErrFaviconNotFound
	}

	contentType := faviconContentType(response.Header.Get("Content-Type"), data)
	if contentType == "" {
		return nil, "", ErrFaviconNotFound
	}
	f.put(origin, faviconCacheEntry{data: data, contentType: contentType, expiresAt: time.Now().Add(faviconCacheTTL)})
	return data, contentType, nil
}

func sameHTTPOrigin(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	leftScheme := strings.ToLower(left.Scheme)
	rightScheme := strings.ToLower(right.Scheme)
	return (leftScheme == "http" || leftScheme == "https") &&
		(leftScheme == rightScheme || (leftScheme == "http" && rightScheme == "https") || (leftScheme == "https" && rightScheme == "http")) &&
		strings.EqualFold(left.Host, right.Host)
}

func (f *faviconFetcher) get(key string) (faviconCacheEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry, ok := f.cache[key]
	if !ok {
		return faviconCacheEntry{}, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(f.cache, key)
		return faviconCacheEntry{}, false
	}
	return entry, true
}

func (f *faviconFetcher) put(key string, entry faviconCacheEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cache[key] = entry
}

func faviconContentType(header string, data []byte) string {
	if mediaType, _, err := mime.ParseMediaType(header); err == nil && strings.HasPrefix(strings.ToLower(mediaType), "image/") {
		return mediaType
	}
	detected := http.DetectContentType(data)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	// Browsers commonly identify ICO files as application/octet-stream even
	// though the format has a stable magic header.
	if len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00 {
		return "image/x-icon"
	}
	return ""
}

// Favicon returns the icon fetched from the origin of a stored shortcut. The
// URL is read from the repository instead of being accepted from the client,
// so callers can only fetch favicons for shortcuts that actually exist.
func (s *Service) Favicon(ctx context.Context, id string) ([]byte, string, error) {
	if s == nil || s.repo == nil || s.favicon == nil {
		return nil, "", errors.New("shortcut service is unavailable")
	}
	shortcut, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return s.favicon.fetch(ctx, shortcut.URL)
}
