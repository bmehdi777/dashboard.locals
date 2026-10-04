package config

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRuntimeUsesXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "config")
	runtime, err := LoadRuntimeFromEnvironment(home, func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return xdg
		}
		return ""
	})
	if err != nil {
		t.Fatalf("LoadRuntimeFromEnvironment() error = %v", err)
	}
	wantDir := filepath.Join(xdg, "dashboard.locals")
	if runtime.ConfigDir != wantDir || runtime.DatabasePath != filepath.Join(wantDir, "database.sqlite") {
		t.Fatalf("unexpected paths: %+v", runtime)
	}
}

func TestLoadRuntimeRejectsRelativeXDGPath(t *testing.T) {
	_, err := LoadRuntimeFromEnvironment(t.TempDir(), func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return "relative"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected relative XDG_CONFIG_HOME to be rejected")
	}
}

func TestServicePersistsSettings(t *testing.T) {
	repository := &memorySettingsRepository{}
	service := NewService(repository)
	settings, err := service.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	settings.Search.Timeout = 2 * time.Second
	settings.Editor.Command = "code"
	if _, err := service.Save(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Search.Timeout != 2*time.Second || loaded.Editor.Command != "code" {
		t.Fatalf("settings were not persisted: %+v", loaded)
	}
	public := PublicSettings(loaded)
	if public.OpenCode.ServiceFile != "" {
		t.Fatalf("unexpected public service file: %q", public.OpenCode.ServiceFile)
	}
}

type memorySettingsRepository struct {
	value []byte
}

func (m *memorySettingsRepository) GetSetting(_ context.Context, _ string) ([]byte, error) {
	return append([]byte(nil), m.value...), nil
}

func (m *memorySettingsRepository) PutSetting(_ context.Context, _ string, value []byte) error {
	m.value = append([]byte(nil), value...)
	return nil
}
