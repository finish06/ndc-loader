//	@title						ndc-loader API
//	@version					1.0
//	@description				FDA NDC Directory bulk loader and REST API. Ingests NDC Directory and Drugs@FDA datasets daily, serves via REST with full-text search and NDC format normalization.
//	@host						localhost:8081
//	@BasePath					/
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key

//go:generate swag init -g cmd/ndc-loader/main.go -o docs/swagger -d ../..

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/calebdunn/ndc-loader/internal"
	"github.com/calebdunn/ndc-loader/internal/api"
	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/store"

	_ "github.com/calebdunn/ndc-loader/docs/swagger" // swagger docs
)

// Build-time variables injected via ldflags.
var (
	version   = "dev"
	gitCommit = "unknown"
	gitBranch = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set build info for /version and /health endpoints.
	api.SetBuildInfo(version, gitCommit, gitBranch, buildTime)

	cfg, err := internal.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := internal.SetupLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	logger.Info("starting ndc-loader",
		"version", version,
		"commit", gitCommit,
		"branch", gitBranch,
	)

	datasetsCfg, err := loader.LoadDatasetsConfig(cfg.DatasetsFile)
	if err != nil {
		logger.Error("failed to load datasets config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := store.RunMigrations(ctx, db); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	checkpointStore := store.NewCheckpointStore(db)
	dataLoader := store.NewDataLoader(db, cfg.RowCountDropThreshold)

	downloader := loader.NewDownloader(cfg.DownloadDir, cfg.MaxRetryAttempts)
	orchestrator := loader.NewOrchestrator(
		logger,
		downloader,
		dataLoader,
		checkpointStore,
		datasetsCfg,
	)

	queryStore := store.NewQueryStore(db)
	router := api.NewRouter(logger, cfg.APIKeys, orchestrator, checkpointStore, queryStore, db)

	scheduler, err := loader.NewScheduler(logger, cfg.LoadSchedule, orchestrator)
	if err != nil {
		logger.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}
	scheduler.Start()
	defer scheduler.Stop()

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}
