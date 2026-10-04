package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dashboard.locals/internal/config"
	"dashboard.locals/internal/launcher"
	"dashboard.locals/internal/opencode"
	"dashboard.locals/internal/search"
	"dashboard.locals/internal/shortcuts"
	"dashboard.locals/internal/stats"
	"dashboard.locals/internal/store"
)

type OpenCodeDetector interface {
	Detect(ctx context.Context) opencode.Detection
}

type Dependencies struct {
	Settings  *config.Service
	Roots     *search.RootService
	Search    *search.Service
	Launcher  *launcher.Service
	Stats     *stats.Service
	History   *search.HistoryService
	Shortcuts *shortcuts.Service
	OpenCode  OpenCodeDetector
	Version   string
}

type Handler struct {
	dependencies Dependencies
	mux          *http.ServeMux
}

func NewHandler(dependencies Dependencies) *Handler {
	handler := &Handler{dependencies: dependencies, mux: http.NewServeMux()}
	handler.mux.HandleFunc("/api/v1", handler.notFound)
	handler.mux.HandleFunc("/api/v1/", handler.notFound)
	handler.mux.HandleFunc("/api/v1/health", handler.health)
	handler.mux.HandleFunc("/api/v1/settings", handler.settings)
	handler.mux.HandleFunc("/api/v1/shortcuts/{shortcutID}/favicon", handler.shortcutFavicon)
	handler.mux.HandleFunc("/api/v1/shortcuts/{shortcutID}/use", handler.useShortcut)
	handler.mux.HandleFunc("/api/v1/shortcuts/order", handler.reorderShortcuts)
	handler.mux.HandleFunc("/api/v1/shortcuts/{shortcutID}", handler.shortcut)
	handler.mux.HandleFunc("/api/v1/shortcuts", handler.shortcuts)
	handler.mux.HandleFunc("/api/v1/shortcut-folders/{folderID}", handler.shortcutFolder)
	handler.mux.HandleFunc("/api/v1/shortcut-folders", handler.shortcutFolders)
	handler.mux.HandleFunc("/api/v1/search-roots", handler.searchRoots)
	handler.mux.HandleFunc("/api/v1/search-roots/{rootID}", handler.searchRoot)
	handler.mux.HandleFunc("/api/v1/search/stream", handler.searchStream)
	handler.mux.HandleFunc("/api/v1/search", handler.search)
	handler.mux.HandleFunc("/api/v1/search-history/{historyID}", handler.searchHistoryItem)
	handler.mux.HandleFunc("/api/v1/search-history", handler.searchHistory)
	handler.mux.HandleFunc("/api/v1/files/open", handler.openFile)
	handler.mux.HandleFunc("/api/v1/stats", handler.stats)
	handler.mux.HandleFunc("/api/v1/stats/sync", handler.syncStats)
	handler.mux.HandleFunc("/api/v1/stats/compact", handler.compactStats)
	return handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAPIPath(r.URL.Path) {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}
	h.mux.ServeHTTP(w, r)
}

func isAPIPath(value string) bool {
	return value == "/api/v1" || strings.HasPrefix(value, "/api/v1/")
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response := map[string]any{
		"status":  "ok",
		"version": h.dependencies.Version,
	}
	if h.dependencies.OpenCode != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		defer cancel()
		response["opencode"] = h.dependencies.OpenCode.Detect(ctx)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Settings == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "settings service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := h.dependencies.Settings.Get(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, config.PublicSettings(settings))
	case http.MethodPatch:
		var request SettingsPatch
		if !decodeJSON(w, r, &request) {
			return
		}
		settings, err := h.dependencies.Settings.Get(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if err := request.Apply(&settings); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
			return
		}
		settings, err = h.dependencies.Settings.Save(r.Context(), settings)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, config.PublicSettings(settings))
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPatch)
	}
}

type SettingsPatch struct {
	Search   *SearchSettingsPatch   `json:"search"`
	Editor   *EditorSettingsPatch   `json:"editor"`
	OpenCode *OpenCodeSettingsPatch `json:"opencode"`
	Stats    *StatsSettingsPatch    `json:"stats"`
}

type SearchSettingsPatch struct {
	RespectGitignore *bool  `json:"respectGitignore"`
	IncludeBinary    *bool  `json:"includeBinary"`
	Literal          *bool  `json:"literal"`
	MaxResults       *int   `json:"maxResults"`
	Timeout          string `json:"timeout"`
}

type EditorSettingsPatch struct {
	Name      *string   `json:"name"`
	Command   *string   `json:"command"`
	Arguments *[]string `json:"arguments"`
}

type OpenCodeSettingsPatch struct {
	Enabled     *bool   `json:"enabled"`
	ServiceFile *string `json:"serviceFile"`
}

type StatsSettingsPatch struct {
	Timezone    *string `json:"timezone"`
	Tools       *string `json:"tools"`
	Granularity *string `json:"granularity"`
}

func (p SettingsPatch) Apply(settings *config.Settings) error {
	if p.Search != nil {
		if p.Search.RespectGitignore != nil {
			settings.Search.RespectGitignore = *p.Search.RespectGitignore
		}
		if p.Search.IncludeBinary != nil {
			settings.Search.IncludeBinary = *p.Search.IncludeBinary
		}
		if p.Search.Literal != nil {
			settings.Search.Literal = *p.Search.Literal
		}
		if p.Search.MaxResults != nil {
			settings.Search.MaxResults = *p.Search.MaxResults
		}
		if p.Search.Timeout != "" {
			value, err := time.ParseDuration(p.Search.Timeout)
			if err != nil {
				return fmt.Errorf("invalid search timeout: %w", err)
			}
			settings.Search.Timeout = value
		}
	}
	if p.Editor != nil {
		if p.Editor.Name != nil {
			settings.Editor.Name = strings.TrimSpace(*p.Editor.Name)
		}
		if p.Editor.Command != nil {
			settings.Editor.Command = strings.TrimSpace(*p.Editor.Command)
		}
		if p.Editor.Arguments != nil {
			settings.Editor.Arguments = append([]string(nil), (*p.Editor.Arguments)...)
		}
	}
	if p.OpenCode != nil {
		if p.OpenCode.Enabled != nil {
			settings.OpenCode.Enabled = *p.OpenCode.Enabled
		}
		if p.OpenCode.ServiceFile != nil {
			settings.OpenCode.ServiceFile = strings.TrimSpace(*p.OpenCode.ServiceFile)
		}
	}
	if p.Stats != nil {
		if p.Stats.Timezone != nil {
			settings.Stats.Timezone = strings.TrimSpace(*p.Stats.Timezone)
		}
		if p.Stats.Tools != nil {
			settings.Stats.Tools = strings.TrimSpace(*p.Stats.Tools)
		}
		if p.Stats.Granularity != nil {
			settings.Stats.Granularity = strings.TrimSpace(*p.Stats.Granularity)
		}
	}
	return nil
}

type CreateShortcutRequest struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	FolderID    *string `json:"folderId"`
}

type UpdateShortcutRequest struct {
	Title       *string `json:"title"`
	URL         *string `json:"url"`
	Description *string `json:"description"`
	FolderID    *string `json:"folderId"`
}

type ReorderShortcutsRequest struct {
	IDs   []string                 `json:"ids"`
	Items []reorderShortcutRequest `json:"items"`
}

type reorderShortcutRequest struct {
	ID       string  `json:"id"`
	FolderID *string `json:"folderId"`
}

type CreateShortcutFolderRequest struct {
	Name string `json:"name"`
}

type UpdateShortcutFolderRequest struct {
	Name string `json:"name"`
}

type shortcutsResponse struct {
	Data []store.Shortcut `json:"data"`
}

type shortcutFoldersResponse struct {
	Data []store.ShortcutFolder `json:"data"`
}

func (h *Handler) shortcuts(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
		if sortBy == "" {
			sortBy = "recent"
		}
		limit := 0
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "invalid_request", "limit must be a positive integer")
				return
			}
			limit = parsed
		}
		items, err := h.dependencies.Shortcuts.List(r.Context(), sortBy, limit)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, shortcutsResponse{Data: items})
	case http.MethodPost:
		var request CreateShortcutRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		shortcut, err := h.dependencies.Shortcuts.Create(r.Context(), request.Title, request.URL, request.Description, request.FolderID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, shortcut)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) shortcut(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	id := r.PathValue("shortcutID")
	switch r.Method {
	case http.MethodPatch:
		var request UpdateShortcutRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		shortcut, err := h.dependencies.Shortcuts.Update(r.Context(), id, request.Title, request.URL, request.Description, request.FolderID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, shortcut)
	case http.MethodDelete:
		if err := h.dependencies.Shortcuts.Delete(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPatch, http.MethodDelete)
	}
}

func (h *Handler) reorderShortcuts(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	var request ReorderShortcutsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	order := make([]store.ShortcutOrder, 0, len(request.Items)+len(request.IDs))
	if len(request.Items) > 0 {
		for _, item := range request.Items {
			order = append(order, store.ShortcutOrder{ID: item.ID, FolderID: item.FolderID})
		}
	} else {
		current, err := h.dependencies.Shortcuts.List(r.Context(), "custom", 0)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		foldersByID := make(map[string]*string, len(current))
		for _, shortcut := range current {
			foldersByID[shortcut.ID] = shortcut.FolderID
		}
		for _, id := range request.IDs {
			order = append(order, store.ShortcutOrder{ID: id, FolderID: foldersByID[id]})
		}
	}
	if err := h.dependencies.Shortcuts.Reorder(r.Context(), order); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) shortcutFolders(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		folders, err := h.dependencies.Shortcuts.ListFolders(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, shortcutFoldersResponse{Data: folders})
	case http.MethodPost:
		var request CreateShortcutFolderRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		folder, err := h.dependencies.Shortcuts.CreateFolder(r.Context(), request.Name)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, folder)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) shortcutFolder(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	id := r.PathValue("folderID")
	switch r.Method {
	case http.MethodPatch:
		var request UpdateShortcutFolderRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		folder, err := h.dependencies.Shortcuts.UpdateFolder(r.Context(), id, request.Name)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, folder)
	case http.MethodDelete:
		if err := h.dependencies.Shortcuts.DeleteFolder(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPatch, http.MethodDelete)
	}
}

func (h *Handler) useShortcut(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	shortcut, err := h.dependencies.Shortcuts.Use(r.Context(), r.PathValue("shortcutID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shortcut)
}

func (h *Handler) shortcutFavicon(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.dependencies.Shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "shortcut service is unavailable")
		return
	}
	data, contentType, err := h.dependencies.Shortcuts.Favicon(r.Context(), r.PathValue("shortcutID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

type CreateRootRequest struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled *bool  `json:"enabled"`
}

type UpdateRootRequest struct {
	Name    *string `json:"name"`
	Path    *string `json:"path"`
	Enabled *bool   `json:"enabled"`
}

func (h *Handler) searchRoots(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Roots == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search root service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		includeDisabled := r.URL.Query().Get("includeDisabled") == "true"
		roots, err := h.dependencies.Roots.List(r.Context(), includeDisabled)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rootsResponse{Data: roots})
	case http.MethodPost:
		var request CreateRootRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		enabled := true
		if request.Enabled != nil {
			enabled = *request.Enabled
		}
		root, err := h.dependencies.Roots.Create(r.Context(), request.Name, request.Path, enabled)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, root)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

type rootsResponse struct {
	Data []store.SearchRoot `json:"data"`
}

func (h *Handler) searchRoot(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.Roots == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search root service is unavailable")
		return
	}
	id := r.PathValue("rootID")
	switch r.Method {
	case http.MethodPatch:
		var request UpdateRootRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		root, err := h.dependencies.Roots.Update(r.Context(), id, request.Name, request.Path, request.Enabled)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, root)
	case http.MethodDelete:
		if err := h.dependencies.Roots.Delete(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPatch, http.MethodDelete)
	}
}

type SearchRequest struct {
	RootID           string  `json:"rootId"`
	Query            string  `json:"query"`
	SearchIn         *string `json:"searchIn"`
	Literal          *bool   `json:"literal"`
	RespectGitignore *bool   `json:"respectGitignore"`
	IncludeBinary    *bool   `json:"includeBinary"`
	MaxResults       *int    `json:"maxResults"`
	TimeoutMS        *int    `json:"timeoutMs"`
}

type SearchResponse struct {
	RootID  string          `json:"rootId"`
	Count   int             `json:"count"`
	Results []search.Result `json:"results"`
}

type searchHistoryResponse struct {
	Data []store.SearchHistory `json:"data"`
}

type searchStreamEvent struct {
	Type      string         `json:"type"`
	Result    *search.Result `json:"result,omitempty"`
	Count     int            `json:"count,omitempty"`
	Truncated bool           `json:"truncated,omitempty"`
	Code      string         `json:"code,omitempty"`
	Message   string         `json:"message,omitempty"`
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Search == nil || h.dependencies.Settings == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search service is unavailable")
		return
	}
	var request SearchRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	settings, err := h.dependencies.Settings.Get(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	options := search.Options{
		Query: request.Query, Literal: settings.Search.Literal,
		RespectGitignore: settings.Search.RespectGitignore,
		IncludeBinary:    settings.Search.IncludeBinary,
		MaxResults:       settings.Search.MaxResults, Timeout: settings.Search.Timeout,
	}
	target, targetErr := parseSearchTarget(request.SearchIn)
	if targetErr != nil {
		writeServiceError(w, targetErr)
		return
	}
	options.Target = target
	if request.Literal != nil {
		options.Literal = *request.Literal
	}
	if request.RespectGitignore != nil {
		options.RespectGitignore = *request.RespectGitignore
	}
	if request.IncludeBinary != nil {
		options.IncludeBinary = *request.IncludeBinary
	}
	if request.MaxResults != nil {
		options.MaxResults = *request.MaxResults
	}
	if request.TimeoutMS != nil {
		if *request.TimeoutMS <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "timeoutMs must be positive")
			return
		}
		options.Timeout = time.Duration(*request.TimeoutMS) * time.Millisecond
	}
	results, err := h.dependencies.Search.Search(r.Context(), request.RootID, options)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	truncated := options.MaxResults > 0 && len(results) >= options.MaxResults
	if err := h.recordSearchHistory(r.Context(), request, options, len(results), truncated); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SearchResponse{RootID: request.RootID, Count: len(results), Results: results})
}

func (h *Handler) searchStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Search == nil || h.dependencies.Settings == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search service is unavailable")
		return
	}
	var request SearchRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	settings, err := h.dependencies.Settings.Get(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	options := search.Options{
		Query: request.Query, Literal: settings.Search.Literal,
		RespectGitignore: settings.Search.RespectGitignore,
		IncludeBinary:    settings.Search.IncludeBinary,
		MaxResults:       settings.Search.MaxResults, Timeout: settings.Search.Timeout,
	}
	target, targetErr := parseSearchTarget(request.SearchIn)
	if targetErr != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "searchIn must be content or filename")
		return
	}
	options.Target = target
	if request.Literal != nil {
		options.Literal = *request.Literal
	}
	if request.RespectGitignore != nil {
		options.RespectGitignore = *request.RespectGitignore
	}
	if request.IncludeBinary != nil {
		options.IncludeBinary = *request.IncludeBinary
	}
	if request.MaxResults != nil {
		options.MaxResults = *request.MaxResults
	}
	if request.TimeoutMS != nil {
		if *request.TimeoutMS <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "timeoutMs must be positive")
			return
		}
		options.Timeout = time.Duration(*request.TimeoutMS) * time.Millisecond
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_not_supported", "streaming responses are unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	encoder := json.NewEncoder(w)

	streamResult, err := h.dependencies.Search.SearchStream(r.Context(), request.RootID, options, func(result search.Result) error {
		if encodeErr := encoder.Encode(searchStreamEvent{Type: "result", Result: &result}); encodeErr != nil {
			return encodeErr
		}
		flusher.Flush()
		return nil
	})
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		_, code, message := classifyError(err)
		_ = encoder.Encode(searchStreamEvent{Type: "error", Code: code, Message: message})
		flusher.Flush()
		return
	}
	if err := h.recordSearchHistory(r.Context(), request, options, streamResult.Count, streamResult.Truncated); err != nil {
		_, code, message := classifyError(err)
		_ = encoder.Encode(searchStreamEvent{Type: "error", Code: code, Message: message})
		flusher.Flush()
		return
	}
	_ = encoder.Encode(searchStreamEvent{Type: "done", Count: streamResult.Count, Truncated: streamResult.Truncated})
	flusher.Flush()
}

func (h *Handler) recordSearchHistory(ctx context.Context, request SearchRequest, options search.Options, resultCount int, truncated bool) error {
	if h.dependencies.History == nil {
		return nil
	}
	rootName := request.RootID
	if h.dependencies.Roots != nil {
		if root, err := h.dependencies.Roots.Get(ctx, request.RootID); err == nil && root.Name != "" {
			rootName = root.Name
		}
	}
	_, err := h.dependencies.History.Record(ctx, store.SearchHistory{
		RootID: request.RootID, RootName: rootName, Query: request.Query,
		SearchIn: string(options.Target),
		Literal:  options.Literal, RespectGitignore: options.RespectGitignore,
		IncludeBinary: options.IncludeBinary, ResultCount: resultCount,
		Truncated: truncated, Status: "completed",
	})
	return err
}

func parseSearchTarget(value *string) (search.SearchTarget, error) {
	if value == nil {
		return search.SearchTargetContent, nil
	}
	return search.ParseSearchTarget(*value)
}

func (h *Handler) searchHistory(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.History == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search history service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "invalid_request", "limit must be a positive integer")
				return
			}
			limit = parsed
		}
		history, err := h.dependencies.History.List(r.Context(), limit)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, searchHistoryResponse{Data: history})
	case http.MethodDelete:
		if err := h.dependencies.History.Clear(r.Context()); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodDelete)
	}
}

func (h *Handler) searchHistoryItem(w http.ResponseWriter, r *http.Request) {
	if h.dependencies.History == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "search history service is unavailable")
		return
	}
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}
	if err := h.dependencies.History.Delete(r.Context(), r.PathValue("historyID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type OpenFileRequest struct {
	RootID string `json:"rootId"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

func (h *Handler) openFile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Launcher == nil || h.dependencies.Settings == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "launcher service is unavailable")
		return
	}
	var request OpenFileRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	settings, err := h.dependencies.Settings.Get(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	err = h.dependencies.Launcher.Open(r.Context(), request.RootID, request.Path, request.Line, request.Column, launcher.Config{
		Name: settings.Editor.Name, Command: settings.Editor.Command, Arguments: settings.Editor.Arguments,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.dependencies.Stats == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "statistics service is unavailable")
		return
	}
	query, err := parseStatsQuery(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	view, err := h.dependencies.Stats.Get(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

type StatsRequest struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Project     string `json:"project"`
	Timezone    string `json:"timezone"`
	Tools       string `json:"tools"`
	Granularity string `json:"granularity"`
}

func (h *Handler) syncStats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Stats == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "statistics service is unavailable")
		return
	}
	var request StatsRequest
	if !decodeJSONAllowEmpty(w, r, &request) {
		return
	}
	query, err := request.toQuery()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.dependencies.Stats.Sync(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type CompactStatsRequest struct {
	Before      string `json:"before"`
	Project     string `json:"project"`
	Granularity string `json:"granularity"`
	DryRun      bool   `json:"dryRun"`
	DropRaw     bool   `json:"dropRaw"`
}

func (h *Handler) compactStats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.dependencies.Stats == nil {
		writeError(w, http.StatusInternalServerError, "server_not_configured", "statistics service is unavailable")
		return
	}
	var request CompactStatsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	before, err := parseTime(request.Before)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "before must be an RFC3339 timestamp")
		return
	}
	result, err := h.dependencies.Stats.Compact(r.Context(), stats.CompactRequest{
		Before: before, Project: request.Project, Granularity: request.Granularity,
		DryRun: request.DryRun, DropRaw: request.DropRaw,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseStatsQuery(values map[string][]string) (stats.Query, error) {
	get := func(key string) string {
		if value := values[key]; len(value) > 0 {
			return value[0]
		}
		return ""
	}
	from, err := optionalTime(get("from"))
	if err != nil {
		return stats.Query{}, errors.New("from must be an RFC3339 timestamp")
	}
	to, err := optionalTime(get("to"))
	if err != nil {
		return stats.Query{}, errors.New("to must be an RFC3339 timestamp")
	}
	return stats.Query{From: from, To: to, Project: get("project"), Granularity: get("granularity"), Timezone: get("timezone"), Tools: get("tools")}, nil
}

func (request StatsRequest) toQuery() (stats.Query, error) {
	from, err := optionalTime(request.From)
	if err != nil {
		return stats.Query{}, errors.New("from must be an RFC3339 timestamp")
	}
	to, err := optionalTime(request.To)
	if err != nil {
		return stats.Query{}, errors.New("to must be an RFC3339 timestamp")
	}
	return stats.Query{From: from, To: to, Project: request.Project, Granularity: request.Granularity, Timezone: request.Timezone, Tools: request.Tools}, nil
}

func optionalTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "API route not found")
}

func requireMethod(w http.ResponseWriter, r *http.Request, expected string) bool {
	if r.Method == expected {
		return true
	}
	methodNotAllowed(w, expected)
	return false
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP method is not allowed")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain valid JSON")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON value")
		return false
	}
	return true
}

func decodeJSONAllowEmpty(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if errors.Is(err, io.EOF) {
		return true
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain valid JSON")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON value")
		return false
	}
	return true
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}

func writeServiceError(w http.ResponseWriter, err error) {
	status, code, message := classifyError(err)
	writeError(w, status, code, message)
}

func classifyError(err error) (int, string, string) {
	switch {
	case errors.Is(err, search.ErrInvalidQuery), errors.Is(err, search.ErrInvalidSearchTarget), errors.Is(err, search.ErrInvalidRoot), errors.Is(err, search.ErrPathOutside), errors.Is(err, search.ErrSymlink):
		return http.StatusBadRequest, "invalid_request", "request validation failed"
	case errors.Is(err, shortcuts.ErrInvalidShortcut):
		return http.StatusBadRequest, "invalid_shortcut", "shortcut validation failed"
	case errors.Is(err, shortcuts.ErrFaviconNotFound):
		return http.StatusNotFound, "favicon_not_found", "shortcut favicon was not found"
	case errors.Is(err, shortcuts.ErrFaviconUnavailable):
		return http.StatusBadGateway, "favicon_unavailable", "shortcut favicon is unavailable"
	case errors.Is(err, search.ErrInvalidHistory):
		return http.StatusBadRequest, "invalid_history", "search history entry is invalid"
	case errors.Is(err, search.ErrRootNotFound), errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, search.ErrRootDisabled):
		return http.StatusConflict, "root_disabled", "search root is disabled"
	case errors.Is(err, store.ErrConflict):
		return http.StatusConflict, "conflict", "resource already exists"
	case errors.Is(err, search.ErrSearchTimeout):
		return http.StatusGatewayTimeout, "search_timeout", "search timed out"
	case errors.Is(err, search.ErrSearchExecution):
		return http.StatusInternalServerError, "search_failed", "search could not be completed"
	case errors.Is(err, launcher.ErrNotConfigured):
		return http.StatusConflict, "editor_not_configured", "editor is not configured"
	case errors.Is(err, launcher.ErrInvalidEditor):
		return http.StatusBadRequest, "invalid_editor", "editor configuration is invalid"
	case errors.Is(err, opencode.ErrUnauthenticated):
		return http.StatusServiceUnavailable, "opencode_unauthenticated", "OpenCode is not authenticated"
	case errors.Is(err, opencode.ErrIncompatible), errors.Is(err, opencode.ErrInvalidResponse):
		return http.StatusBadGateway, "opencode_incompatible", "OpenCode returned an incompatible response"
	case errors.Is(err, opencode.ErrUnavailable):
		return http.StatusServiceUnavailable, "opencode_unavailable", "OpenCode is unavailable"
	default:
		return http.StatusInternalServerError, "internal_error", "an internal server error occurred"
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
