package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	ErrNotFound = gorm.ErrRecordNotFound
	ErrConflict = errors.New("storage conflict")
)

const currentSchemaVersion = 5

type Store struct {
	db *gorm.DB
}

type migrationModel struct {
	Version   int       `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

func (migrationModel) TableName() string {
	return "schema_migrations"
}

type searchRootModel struct {
	ID        string    `gorm:"primaryKey;size:32"`
	Name      string    `gorm:"not null;size:200"`
	Path      string    `gorm:"not null;uniqueIndex;size:4096"`
	Enabled   bool      `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type settingModel struct {
	Key       string    `gorm:"primaryKey;size:100"`
	Value     string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type shortcutModel struct {
	ID          string     `gorm:"primaryKey;size:32"`
	Title       string     `gorm:"not null;size:200"`
	URL         string     `gorm:"not null;size:2048"`
	Description string     `gorm:"type:text;not null"`
	UsageCount  int64      `gorm:"not null;default:0;index"`
	LastUsedAt  *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

type rawStatModel struct {
	ID            string    `gorm:"primaryKey;size:32"`
	Source        string    `gorm:"not null;size:100;uniqueIndex:idx_raw_identity"`
	Project       string    `gorm:"not null;size:4096;uniqueIndex:idx_raw_identity"`
	PeriodStart   time.Time `gorm:"not null;uniqueIndex:idx_raw_identity"`
	PeriodEnd     time.Time `gorm:"not null"`
	Granularity   string    `gorm:"not null;size:20;uniqueIndex:idx_raw_identity"`
	FormatVersion int       `gorm:"not null;uniqueIndex:idx_raw_identity"`
	ContentHash   string    `gorm:"not null;size:64"`
	Payload       string    `gorm:"type:text;not null"`
	FetchedAt     time.Time `gorm:"not null;index"`
	CreatedAt     time.Time `gorm:"not null"`
}

type aggregateModel struct {
	ID            string    `gorm:"primaryKey;size:32"`
	Source        string    `gorm:"not null;size:100;uniqueIndex:idx_aggregate_identity"`
	Project       string    `gorm:"not null;size:4096;uniqueIndex:idx_aggregate_identity"`
	PeriodStart   time.Time `gorm:"not null;uniqueIndex:idx_aggregate_identity"`
	PeriodEnd     time.Time `gorm:"not null"`
	Granularity   string    `gorm:"not null;size:20;uniqueIndex:idx_aggregate_identity"`
	FormatVersion int       `gorm:"not null;uniqueIndex:idx_aggregate_identity"`
	Sessions      int64     `gorm:"not null"`
	Subagents     int64     `gorm:"not null;default:0"`
	Prompts       int64     `gorm:"not null"`
	Steps         int64     `gorm:"not null"`
	InputTokens   int64     `gorm:"not null"`
	OutputTokens  int64     `gorm:"not null"`
	Reasoning     int64     `gorm:"not null"`
	CacheRead     int64     `gorm:"not null"`
	CacheWrite    int64     `gorm:"not null"`
	Cost          float64   `gorm:"not null"`
	Models        string    `gorm:"type:text;not null"`
	Tools         string    `gorm:"type:text;not null"`
	Activity      string    `gorm:"type:text;not null;default:'[]'"`
	ActiveDays    int64     `gorm:"not null;default:0"`
	Streak        int64     `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

type syncMetadataModel struct {
	Source     string `gorm:"primaryKey;size:100"`
	Status     string `gorm:"not null;size:30"`
	LastSyncAt *time.Time
	LastFrom   *time.Time
	LastTo     *time.Time
	LastError  string    `gorm:"type:text"`
	UpdatedAt  time.Time `gorm:"not null"`
}

type searchHistoryModel struct {
	ID               string    `gorm:"primaryKey;size:32"`
	RootID           string    `gorm:"not null;index;size:32"`
	RootName         string    `gorm:"not null;size:200"`
	Query            string    `gorm:"not null;size:4096"`
	Literal          bool      `gorm:"not null"`
	RespectGitignore bool      `gorm:"not null"`
	IncludeBinary    bool      `gorm:"not null"`
	ResultCount      int       `gorm:"not null"`
	Truncated        bool      `gorm:"not null"`
	Status           string    `gorm:"not null;size:20;index"`
	CreatedAt        time.Time `gorm:"not null;index"`
}

type SearchRoot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Shortcut struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	UsageCount  int64      `json:"usageCount"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type RawStatRecord struct {
	ID            string
	Source        string
	Project       string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Granularity   string
	FormatVersion int
	ContentHash   string
	Payload       json.RawMessage
	FetchedAt     time.Time
	CreatedAt     time.Time
}

type Aggregate struct {
	ID            string
	Source        string
	Project       string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Granularity   string
	FormatVersion int
	Sessions      int64
	Subagents     int64
	Prompts       int64
	Steps         int64
	InputTokens   int64
	OutputTokens  int64
	Reasoning     int64
	CacheRead     int64
	CacheWrite    int64
	Cost          float64
	Models        json.RawMessage
	Tools         json.RawMessage
	Activity      json.RawMessage
	ActiveDays    int64
	Streak        int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SyncMetadata struct {
	Source     string
	Status     string
	LastSyncAt *time.Time
	LastFrom   *time.Time
	LastTo     *time.Time
	LastError  string
	UpdatedAt  time.Time
}

type SearchHistory struct {
	ID               string    `json:"id"`
	RootID           string    `json:"rootId"`
	RootName         string    `json:"rootName"`
	Query            string    `json:"query"`
	Literal          bool      `json:"literal"`
	RespectGitignore bool      `json:"respectGitignore"`
	IncludeBinary    bool      `json:"includeBinary"`
	ResultCount      int       `json:"resultCount"`
	Truncated        bool      `json:"truncated"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path is empty")
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		TranslateError: true,
		Logger:         gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sqlite handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)
	if _, err := sqlDB.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	if _, err := sqlDB.ExecContext(context.Background(), "PRAGMA busy_timeout = 5000"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("configure sqlite busy timeout: %w", err)
	}
	if _, err := sqlDB.ExecContext(context.Background(), "PRAGMA journal_mode = WAL"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("configure sqlite journal mode: %w", err)
	}
	return &Store{db: db}, nil
}

func OpenInMemory(name string) (*Store, error) {
	if name == "" {
		name = newID()
	}
	return Open("file:" + name + "?mode=memory&cache=shared")
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Migrate(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&migrationModel{}); err != nil {
			return fmt.Errorf("migrate schema version table: %w", err)
		}
		var applied migrationModel
		err := tx.Where("version = ?", currentSchemaVersion).First(&applied).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read schema version: %w", err)
		}
		var latest int
		if err := tx.Model(&migrationModel{}).Select("COALESCE(MAX(version), 0)").Scan(&latest).Error; err != nil {
			return fmt.Errorf("read latest schema version: %w", err)
		}
		for version := latest + 1; version <= currentSchemaVersion; version++ {
			switch version {
			case 1:
				if err := tx.AutoMigrate(
					&searchRootModel{},
					&settingModel{},
					&rawStatModel{},
					&aggregateModel{},
					&syncMetadataModel{},
				); err != nil {
					return fmt.Errorf("migrate application schema: %w", err)
				}
			case 2:
				if err := tx.AutoMigrate(&aggregateModel{}); err != nil {
					return fmt.Errorf("migrate aggregate schema: %w", err)
				}
			case 3:
				if err := tx.AutoMigrate(&searchHistoryModel{}); err != nil {
					return fmt.Errorf("migrate search history schema: %w", err)
				}
			case 4:
				if err := migrateIdempotentStatistics(tx); err != nil {
					return fmt.Errorf("migrate idempotent statistics schema: %w", err)
				}
			case 5:
				if err := tx.AutoMigrate(&shortcutModel{}); err != nil {
					return fmt.Errorf("migrate shortcuts schema: %w", err)
				}
			}
			if err := tx.Create(&migrationModel{Version: version, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return fmt.Errorf("record schema version: %w", err)
			}
		}
		return nil
	})
}

// migrateIdempotentStatistics repairs the identity indexes introduced by the
// initial statistics implementation. That implementation included the
// response end time and payload hash in the identity, although OpenCode
// legitimately changes both values for the same in-progress calendar period.
// Keep the most recently fetched row for each logical period before creating
// the new indexes, so existing dashboards do not keep displaying duplicates.
func migrateIdempotentStatistics(tx *gorm.DB) error {
	if err := tx.Exec("DROP INDEX IF EXISTS idx_raw_identity").Error; err != nil {
		return err
	}
	if err := tx.Exec("DROP INDEX IF EXISTS idx_aggregate_identity").Error; err != nil {
		return err
	}

	var rawRows []rawStatModel
	if err := tx.Order("source ASC, project ASC, period_start ASC, granularity ASC, format_version ASC, fetched_at DESC, created_at DESC, id DESC").Find(&rawRows).Error; err != nil {
		return err
	}
	seenRaw := make(map[string]struct{}, len(rawRows))
	for _, row := range rawRows {
		key := statisticsIdentity(row.Source, row.Project, row.PeriodStart, row.Granularity, row.FormatVersion)
		if _, exists := seenRaw[key]; exists {
			if err := tx.Delete(&rawStatModel{}, "id = ?", row.ID).Error; err != nil {
				return err
			}
			continue
		}
		seenRaw[key] = struct{}{}
	}

	var aggregateRows []aggregateModel
	if err := tx.Order("source ASC, project ASC, period_start ASC, granularity ASC, format_version ASC, updated_at DESC, created_at DESC, id DESC").Find(&aggregateRows).Error; err != nil {
		return err
	}
	seenAggregates := make(map[string]struct{}, len(aggregateRows))
	for _, row := range aggregateRows {
		key := statisticsIdentity(row.Source, row.Project, row.PeriodStart, row.Granularity, row.FormatVersion)
		if _, exists := seenAggregates[key]; exists {
			if err := tx.Delete(&aggregateModel{}, "id = ?", row.ID).Error; err != nil {
				return err
			}
			continue
		}
		seenAggregates[key] = struct{}{}
	}

	if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_raw_identity ON raw_stat_models (source, project, period_start, granularity, format_version)").Error; err != nil {
		return err
	}
	return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_aggregate_identity ON aggregate_models (source, project, period_start, granularity, format_version)").Error
}

func statisticsIdentity(source, project string, periodStart time.Time, granularity string, formatVersion int) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", source, project, periodStart.UTC().Format(time.RFC3339Nano), granularity, formatVersion)
}

func newID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func (s *Store) GetSetting(ctx context.Context, key string) ([]byte, error) {
	var model settingModel
	if err := s.db.WithContext(ctx).First(&model, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get setting: %w", err)
	}
	return []byte(model.Value), nil
}

func (s *Store) PutSetting(ctx context.Context, key string, value []byte) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("setting key is empty")
	}
	if !json.Valid(value) {
		return errors.New("setting value must be valid JSON")
	}
	model := settingModel{Key: key, Value: string(value), UpdatedAt: time.Now().UTC()}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing settingModel
		err := tx.First(&existing, "key = ?", key).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&model).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&existing).Updates(map[string]any{
			"value":      model.Value,
			"updated_at": model.UpdatedAt,
		}).Error
	})
}

func (s *Store) ListSearchRoots(ctx context.Context, includeDisabled bool) ([]SearchRoot, error) {
	query := s.db.WithContext(ctx).Model(&searchRootModel{}).Order("name ASC")
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	var models []searchRootModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list search roots: %w", err)
	}
	result := make([]SearchRoot, 0, len(models))
	for _, model := range models {
		result = append(result, toSearchRoot(model))
	}
	return result, nil
}

func (s *Store) GetSearchRoot(ctx context.Context, id string) (SearchRoot, error) {
	var model searchRootModel
	if err := s.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return SearchRoot{}, err
	}
	return toSearchRoot(model), nil
}

func (s *Store) CreateSearchRoot(ctx context.Context, root SearchRoot) (SearchRoot, error) {
	if root.ID == "" {
		root.ID = newID()
	}
	now := time.Now().UTC()
	if root.CreatedAt.IsZero() {
		root.CreatedAt = now
	}
	if root.UpdatedAt.IsZero() {
		root.UpdatedAt = now
	}
	model := searchRootModel{
		ID: root.ID, Name: root.Name, Path: root.Path, Enabled: root.Enabled,
		CreatedAt: root.CreatedAt, UpdatedAt: root.UpdatedAt,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return SearchRoot{}, fmt.Errorf("%w: search root already exists", ErrConflict)
		}
		return SearchRoot{}, fmt.Errorf("create search root: %w", err)
	}
	return toSearchRoot(model), nil
}

func (s *Store) UpdateSearchRoot(ctx context.Context, root SearchRoot) (SearchRoot, error) {
	if root.ID == "" {
		return SearchRoot{}, errors.New("search root id is empty")
	}
	root.UpdatedAt = time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&searchRootModel{}).Where("id = ?", root.ID).Updates(map[string]any{
		"name":       root.Name,
		"path":       root.Path,
		"enabled":    root.Enabled,
		"updated_at": root.UpdatedAt,
	})
	if result.Error != nil {
		return SearchRoot{}, fmt.Errorf("update search root: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return SearchRoot{}, gorm.ErrRecordNotFound
	}
	return s.GetSearchRoot(ctx, root.ID)
}

func (s *Store) DeleteSearchRoot(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&searchRootModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete search root: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) CreateSearchHistory(ctx context.Context, history SearchHistory) (SearchHistory, error) {
	if history.ID == "" {
		history.ID = newID()
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now().UTC()
	}
	model := searchHistoryModel{
		ID: history.ID, RootID: history.RootID, RootName: history.RootName,
		Query: history.Query, Literal: history.Literal,
		RespectGitignore: history.RespectGitignore, IncludeBinary: history.IncludeBinary,
		ResultCount: history.ResultCount, Truncated: history.Truncated,
		Status: history.Status, CreatedAt: history.CreatedAt,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return SearchHistory{}, fmt.Errorf("create search history: %w", err)
	}
	return toSearchHistory(model), nil
}

func (s *Store) ListSearchHistory(ctx context.Context, limit int) ([]SearchHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var models []searchHistoryModel
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list search history: %w", err)
	}
	result := make([]SearchHistory, 0, len(models))
	for _, model := range models {
		result = append(result, toSearchHistory(model))
	}
	return result, nil
}

func (s *Store) DeleteSearchHistory(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&searchHistoryModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete search history: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) ClearSearchHistory(ctx context.Context) error {
	return s.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&searchHistoryModel{}).Error
}

func toSearchHistory(model searchHistoryModel) SearchHistory {
	return SearchHistory{
		ID: model.ID, RootID: model.RootID, RootName: model.RootName,
		Query: model.Query, Literal: model.Literal,
		RespectGitignore: model.RespectGitignore, IncludeBinary: model.IncludeBinary,
		ResultCount: model.ResultCount, Truncated: model.Truncated,
		Status: model.Status, CreatedAt: model.CreatedAt,
	}
}

func toSearchRoot(model searchRootModel) SearchRoot {
	return SearchRoot{
		ID: model.ID, Name: model.Name, Path: model.Path, Enabled: model.Enabled,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func (s *Store) ListShortcuts(ctx context.Context, sortBy string, limit int) ([]Shortcut, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 200 {
		limit = 200
	}

	query := s.db.WithContext(ctx).Model(&shortcutModel{})
	switch sortBy {
	case "popular":
		query = query.Order("usage_count DESC").Order("CASE WHEN last_used_at IS NULL THEN 1 ELSE 0 END ASC").Order("last_used_at DESC").Order("title ASC")
	default:
		query = query.Order("created_at DESC").Order("title ASC")
	}

	var models []shortcutModel
	if err := query.Limit(limit).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list shortcuts: %w", err)
	}
	result := make([]Shortcut, 0, len(models))
	for _, model := range models {
		result = append(result, toShortcut(model))
	}
	return result, nil
}

func (s *Store) GetShortcut(ctx context.Context, id string) (Shortcut, error) {
	var model shortcutModel
	if err := s.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return Shortcut{}, err
	}
	return toShortcut(model), nil
}

func (s *Store) CreateShortcut(ctx context.Context, shortcut Shortcut) (Shortcut, error) {
	if shortcut.ID == "" {
		shortcut.ID = newID()
	}
	now := time.Now().UTC()
	if shortcut.CreatedAt.IsZero() {
		shortcut.CreatedAt = now
	}
	if shortcut.UpdatedAt.IsZero() {
		shortcut.UpdatedAt = now
	}
	model := shortcutModel{
		ID: shortcut.ID, Title: shortcut.Title, URL: shortcut.URL,
		Description: shortcut.Description, UsageCount: shortcut.UsageCount,
		LastUsedAt: shortcut.LastUsedAt, CreatedAt: shortcut.CreatedAt,
		UpdatedAt: shortcut.UpdatedAt,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Shortcut{}, fmt.Errorf("%w: shortcut already exists", ErrConflict)
		}
		return Shortcut{}, fmt.Errorf("create shortcut: %w", err)
	}
	return toShortcut(model), nil
}

func (s *Store) UpdateShortcut(ctx context.Context, shortcut Shortcut) (Shortcut, error) {
	if shortcut.ID == "" {
		return Shortcut{}, errors.New("shortcut id is empty")
	}
	shortcut.UpdatedAt = time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&shortcutModel{}).Where("id = ?", shortcut.ID).Updates(map[string]any{
		"title":       shortcut.Title,
		"url":         shortcut.URL,
		"description": shortcut.Description,
		"updated_at":  shortcut.UpdatedAt,
	})
	if result.Error != nil {
		return Shortcut{}, fmt.Errorf("update shortcut: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return Shortcut{}, gorm.ErrRecordNotFound
	}
	return s.GetShortcut(ctx, shortcut.ID)
}

func (s *Store) DeleteShortcut(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&shortcutModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete shortcut: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) RecordShortcutUse(ctx context.Context, id string) (Shortcut, error) {
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&shortcutModel{}).Where("id = ?", id).Updates(map[string]any{
		"usage_count":  gorm.Expr("usage_count + ?", 1),
		"last_used_at": now,
		"updated_at":   now,
	})
	if result.Error != nil {
		return Shortcut{}, fmt.Errorf("record shortcut use: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return Shortcut{}, gorm.ErrRecordNotFound
	}
	return s.GetShortcut(ctx, id)
}

func toShortcut(model shortcutModel) Shortcut {
	return Shortcut{
		ID: model.ID, Title: model.Title, URL: model.URL,
		Description: model.Description, UsageCount: model.UsageCount,
		LastUsedAt: model.LastUsedAt, CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func (s *Store) UpsertRawStat(ctx context.Context, record RawStatRecord) (RawStatRecord, bool, error) {
	if record.ID == "" {
		record.ID = newID()
	}
	if record.FetchedAt.IsZero() {
		record.FetchedAt = time.Now().UTC()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = record.FetchedAt
	}
	model := rawStatModel{
		ID: record.ID, Source: record.Source, Project: record.Project,
		PeriodStart: record.PeriodStart, PeriodEnd: record.PeriodEnd,
		Granularity: record.Granularity, FormatVersion: record.FormatVersion,
		ContentHash: record.ContentHash, Payload: string(record.Payload),
		FetchedAt: record.FetchedAt, CreatedAt: record.CreatedAt,
	}
	var existing rawStatModel
	query := s.db.WithContext(ctx).Where(
		"source = ? AND project = ? AND period_start = ? AND granularity = ? AND format_version = ?",
		record.Source, record.Project, record.PeriodStart, record.Granularity, record.FormatVersion,
	).First(&existing)
	if query.Error == nil {
		model.ID = existing.ID
		model.CreatedAt = existing.CreatedAt
		if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
			return RawStatRecord{}, false, fmt.Errorf("update raw statistics: %w", err)
		}
		return toRawStat(model), false, nil
	}
	if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return RawStatRecord{}, false, query.Error
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		// A concurrent insert is idempotent too; return the row that won.
		if existingErr := s.db.WithContext(ctx).Where(
			"source = ? AND project = ? AND period_start = ? AND granularity = ? AND format_version = ?",
			record.Source, record.Project, record.PeriodStart, record.Granularity, record.FormatVersion,
		).First(&existing).Error; existingErr == nil {
			return toRawStat(existing), false, nil
		}
		return RawStatRecord{}, false, fmt.Errorf("upsert raw statistics: %w", err)
	}
	return record, true, nil
}

func (s *Store) ListRawStats(ctx context.Context, source, project string, before *time.Time) ([]RawStatRecord, error) {
	query := s.db.WithContext(ctx).Model(&rawStatModel{}).Where("source = ?", source).Order("period_start ASC")
	if project != "" {
		query = query.Where("project = ?", project)
	}
	if before != nil {
		query = query.Where("period_end <= ?", *before)
	}
	var models []rawStatModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list raw statistics: %w", err)
	}
	result := make([]RawStatRecord, 0, len(models))
	for _, model := range models {
		result = append(result, toRawStat(model))
	}
	return result, nil
}

func (s *Store) DeleteRawStats(ctx context.Context, source, project string, before time.Time) (int64, error) {
	query := s.db.WithContext(ctx).Where("source = ? AND period_end <= ?", source, before)
	if project != "" {
		query = query.Where("project = ?", project)
	}
	result := query.Delete(&rawStatModel{})
	return result.RowsAffected, result.Error
}

func toRawStat(model rawStatModel) RawStatRecord {
	return RawStatRecord{
		ID: model.ID, Source: model.Source, Project: model.Project,
		PeriodStart: model.PeriodStart, PeriodEnd: model.PeriodEnd,
		Granularity: model.Granularity, FormatVersion: model.FormatVersion,
		ContentHash: model.ContentHash, Payload: json.RawMessage(model.Payload),
		FetchedAt: model.FetchedAt, CreatedAt: model.CreatedAt,
	}
}

func (s *Store) UpsertAggregate(ctx context.Context, aggregate Aggregate) (Aggregate, bool, error) {
	if aggregate.ID == "" {
		aggregate.ID = newID()
	}
	now := time.Now().UTC()
	if aggregate.CreatedAt.IsZero() {
		aggregate.CreatedAt = now
	}
	aggregate.UpdatedAt = now
	model := aggregateModel{
		ID: aggregate.ID, Source: aggregate.Source, Project: aggregate.Project,
		PeriodStart: aggregate.PeriodStart, PeriodEnd: aggregate.PeriodEnd,
		Granularity: aggregate.Granularity, FormatVersion: aggregate.FormatVersion,
		Sessions: aggregate.Sessions, Prompts: aggregate.Prompts, Steps: aggregate.Steps,
		Subagents:   aggregate.Subagents,
		InputTokens: aggregate.InputTokens, OutputTokens: aggregate.OutputTokens,
		Reasoning: aggregate.Reasoning, CacheRead: aggregate.CacheRead,
		CacheWrite: aggregate.CacheWrite, Cost: aggregate.Cost,
		Models: string(defaultJSON(aggregate.Models)), Tools: string(defaultJSON(aggregate.Tools)),
		Activity: string(defaultJSON(aggregate.Activity)), ActiveDays: aggregate.ActiveDays, Streak: aggregate.Streak,
		CreatedAt: aggregate.CreatedAt, UpdatedAt: aggregate.UpdatedAt,
	}
	var existing aggregateModel
	query := s.db.WithContext(ctx).Where(
		"source = ? AND project = ? AND period_start = ? AND granularity = ? AND format_version = ?",
		aggregate.Source, aggregate.Project, aggregate.PeriodStart, aggregate.Granularity, aggregate.FormatVersion,
	).First(&existing)
	if query.Error == nil {
		model.ID = existing.ID
		model.CreatedAt = existing.CreatedAt
		if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
			return Aggregate{}, false, err
		}
		return toAggregate(model), false, nil
	}
	if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return Aggregate{}, false, query.Error
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return Aggregate{}, false, fmt.Errorf("upsert aggregate: %w", err)
	}
	return toAggregate(model), true, nil
}

func (s *Store) ListAggregates(ctx context.Context, source, project, granularity string, from, to *time.Time) ([]Aggregate, error) {
	query := s.db.WithContext(ctx).Model(&aggregateModel{}).Where("source = ?", source).Order("period_start ASC")
	if project != "" {
		query = query.Where("project = ?", project)
	}
	if granularity != "" {
		query = query.Where("granularity = ?", granularity)
	}
	if from != nil {
		query = query.Where("period_end > ?", *from)
	}
	if to != nil {
		query = query.Where("period_start < ?", *to)
	}
	var models []aggregateModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list aggregates: %w", err)
	}
	result := make([]Aggregate, 0, len(models))
	for _, model := range models {
		result = append(result, toAggregate(model))
	}
	return result, nil
}

func toAggregate(model aggregateModel) Aggregate {
	return Aggregate{
		ID: model.ID, Source: model.Source, Project: model.Project,
		PeriodStart: model.PeriodStart, PeriodEnd: model.PeriodEnd,
		Granularity: model.Granularity, FormatVersion: model.FormatVersion,
		Sessions: model.Sessions, Prompts: model.Prompts, Steps: model.Steps,
		Subagents:   model.Subagents,
		InputTokens: model.InputTokens, OutputTokens: model.OutputTokens,
		Reasoning: model.Reasoning, CacheRead: model.CacheRead,
		CacheWrite: model.CacheWrite, Cost: model.Cost,
		Models: json.RawMessage(model.Models), Tools: json.RawMessage(model.Tools),
		Activity: json.RawMessage(model.Activity), ActiveDays: model.ActiveDays, Streak: model.Streak,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func defaultJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 || !json.Valid(value) {
		return json.RawMessage(`{}`)
	}
	return value
}

func (s *Store) GetSyncMetadata(ctx context.Context, source string) (SyncMetadata, error) {
	var model syncMetadataModel
	if err := s.db.WithContext(ctx).First(&model, "source = ?", source).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SyncMetadata{Source: source, Status: "never"}, nil
		}
		return SyncMetadata{}, err
	}
	return toSyncMetadata(model), nil
}

func (s *Store) PutSyncMetadata(ctx context.Context, metadata SyncMetadata) error {
	metadata.UpdatedAt = time.Now().UTC()
	model := syncMetadataModel{
		Source: metadata.Source, Status: metadata.Status,
		LastSyncAt: metadata.LastSyncAt, LastFrom: metadata.LastFrom,
		LastTo: metadata.LastTo, LastError: metadata.LastError,
		UpdatedAt: metadata.UpdatedAt,
	}
	return s.db.WithContext(ctx).Save(&model).Error
}

func toSyncMetadata(model syncMetadataModel) SyncMetadata {
	return SyncMetadata{
		Source: model.Source, Status: model.Status,
		LastSyncAt: model.LastSyncAt, LastFrom: model.LastFrom,
		LastTo: model.LastTo, LastError: model.LastError,
		UpdatedAt: model.UpdatedAt,
	}
}

// WithTransaction exposes a transaction to higher-level services without
// exposing the database handle to HTTP handlers. The callback receives a
// store facade bound to the transaction.
func (s *Store) WithTransaction(ctx context.Context, fn func(*Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}
