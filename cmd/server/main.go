package main

import (
	"context"
	"errors"
	"log"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dashboard.locals/internal/config"
	httpserver "dashboard.locals/internal/http"
	"dashboard.locals/internal/http/api/v1"
	"dashboard.locals/internal/launcher"
	"dashboard.locals/internal/logging"
	"dashboard.locals/internal/opencode"
	"dashboard.locals/internal/search"
	"dashboard.locals/internal/shortcuts"
	"dashboard.locals/internal/stats"
	"dashboard.locals/internal/store"
)

const version = "dev"

func main() {
	bootstrapLogger := log.New(os.Stderr, "dashboard.locals: ", log.LstdFlags)
	logFiles, err := logging.Open()
	if err != nil {
		bootstrapLogger.Printf("open log files: %v", err)
		os.Exit(1)
	}
	defer logFiles.Close()
	if warning := logFiles.Warning(); warning != "" {
		bootstrapLogger.Printf("%s", warning)
		logFiles.Error.Printf("%s", warning)
	}
	if err := run(logFiles.Error, logFiles.Access); err != nil {
		logFiles.Error.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

func run(logger, accessLogger *log.Logger) error {
	runtimeConfig, err := config.LoadRuntime()
	if err != nil {
		return err
	}
	if err := runtimeConfig.EnsureDirectories(); err != nil {
		return err
	}

	database, err := store.Open(runtimeConfig.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		return err
	}

	settingsService := config.NewService(database)
	settings, err := settingsService.Get(context.Background())
	if err != nil {
		return err
	}
	if raw, err := database.GetSetting(context.Background(), "application"); err != nil {
		return err
	} else if len(raw) == 0 {
		if _, err := settingsService.Save(context.Background(), settings); err != nil {
			return err
		}
	}

	rootService := search.NewRootService(database)
	historyService := search.NewHistoryService(database)
	shortcutService := shortcuts.NewService(database)
	searchService := search.NewService(database, nil, settings.Search.MaxResults, minDuration(settings.Search.Timeout, runtimeConfig.SearchTimeout))
	launcherService := launcher.NewService(database, nil)

	home, _ := os.UserHomeDir()
	serviceFile := settings.OpenCode.ServiceFile
	if serviceFile == "" {
		serviceFile = config.DefaultServiceFile(home)
	}
	detector := opencode.NewDetector(serviceFile, runtimeConfig.OpenCodeTimeout)
	detector.Enabled = settings.OpenCode.Enabled
	statsService := stats.NewService(database, detector, settings.Stats.Timezone, settings.Stats.Tools, settings.Stats.Granularity)

	handler := httpserver.NewHandler(httpserver.Config{
		API: v1.Dependencies{
			Settings:  settingsService,
			Roots:     rootService,
			Search:    searchService,
			History:   historyService,
			Shortcuts: shortcutService,
			Launcher:  launcherService,
			Stats:     statsService,
			OpenCode:  detector,
			Version:   version,
		},
		Logger:       logger,
		AccessLogger: accessLogger,
	})
	server := httpserver.New(runtimeConfig.ListenAddr, handler, runtimeConfig.HTTPReadTimeout, runtimeConfig.HTTPWriteTimeout, runtimeConfig.HTTPIdleTimeout)

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", runtimeConfig.ListenAddr)
	if err != nil {
		return err
	}
	logger.Printf("server started at http://%s", listener.Addr().String())
	// OpenCode is optional. The initial synchronization runs in the background
	// so the HTTP server is immediately available; the stats service serializes
	// it with any manual synchronization started from the UI or CLI.
	go synchronizeStatsAtStartup(shutdownContext, statsService, settings, logger)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	select {
	case err := <-serverErrors:
		if errors.Is(err, stdhttp.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownContext.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

func synchronizeStatsAtStartup(ctx context.Context, service *stats.Service, settings config.Settings, logger *log.Logger) {
	if !settings.OpenCode.Enabled {
		logger.Printf("startup opencode sync skipped: integration disabled")
		return
	}
	startupContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	result, err := service.Sync(startupContext, stats.Query{
		Timezone:    settings.Stats.Timezone,
		Tools:       settings.Stats.Tools,
		Granularity: settings.Stats.Granularity,
	})
	if err != nil {
		state := result.Detection.State
		if state == "" {
			state = "unknown"
		}
		logger.Printf("startup opencode sync skipped: state=%s", state)
		return
	}
	logger.Printf("startup opencode sync completed: raw_created=%d raw_existing=%d aggregates_created=%d aggregates_updated=%d", result.RawCreated, result.RawExisting, result.AggregatesCreated, result.AggregatesUpdated)
}

func minDuration(settingsValue, operationalValue time.Duration) time.Duration {
	if settingsValue <= 0 {
		return operationalValue
	}
	if operationalValue <= 0 || settingsValue < operationalValue {
		return settingsValue
	}
	return operationalValue
}
