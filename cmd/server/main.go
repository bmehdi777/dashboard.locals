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
	"dashboard.locals/internal/opencode"
	"dashboard.locals/internal/search"
	"dashboard.locals/internal/stats"
	"dashboard.locals/internal/store"
)

const version = "dev"

func main() {
	logger := log.New(os.Stderr, "dashboard.locals: ", log.LstdFlags)
	if err := run(logger); err != nil {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
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
			Settings: settingsService,
			Roots:    rootService,
			Search:   searchService,
			Launcher: launcherService,
			Stats:    statsService,
			OpenCode: detector,
			Version:  version,
		},
		Logger: logger,
	})
	server := httpserver.New(runtimeConfig.ListenAddr, handler, runtimeConfig.HTTPReadTimeout, runtimeConfig.HTTPWriteTimeout, runtimeConfig.HTTPIdleTimeout)

	// OpenCode is optional. Detection is deliberately asynchronous so a broken
	// or stopped local instance cannot delay the local search server.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), runtimeConfig.OpenCodeTimeout)
		defer cancel()
		detection := detector.Detect(ctx)
		logger.Printf("opencode state=%s executable=%t", detection.State, detection.ExecutableAvailable)
	}()

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", runtimeConfig.ListenAddr)
	if err != nil {
		return err
	}
	logger.Printf("server started at http://%s", listener.Addr().String())
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

func minDuration(settingsValue, operationalValue time.Duration) time.Duration {
	if settingsValue <= 0 {
		return operationalValue
	}
	if operationalValue <= 0 || settingsValue < operationalValue {
		return settingsValue
	}
	return operationalValue
}
