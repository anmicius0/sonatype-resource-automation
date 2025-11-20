// main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/server"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

func main() {
	// Initialize logging first
	if err := utils.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	defer utils.Sync()

	// Load configuration
	appConfig, err := config.Load()
	if err != nil {
		utils.Logger.Fatal("Failed to load configuration", zap.Error(err))
	}
	utils.Logger.Info("Configuration loaded successfully")

	// Initialize job store
	jobStore := config.NewJobStore()

	// Initialize clients and batch manager
	nexusClient := client.NewNexusClient(appConfig.NexusURL, appConfig.NexusUsername, appConfig.NexusPassword, appConfig.PackageManagers)
	iqClient := client.NewIQServerClient(appConfig.IQServerURL, appConfig.IQServerUsername, appConfig.IQServerPassword)
	batchManager := server.NewBatchManager(appConfig, jobStore, nexusClient, iqClient)

	// Setup HTTP server
	router := server.NewRouter(appConfig, jobStore, batchManager)
	startServer(router, appConfig)
}

// startServer binds the HTTP server and handles graceful shutdown signals.
func startServer(router http.Handler, appConfig *config.Config) {
	portStr := strconv.Itoa(appConfig.Port)
	addr := fmt.Sprintf("%s:%s", appConfig.APIHost, portStr)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  config.DefaultReadTimeout,
		WriteTimeout: config.DefaultWriteTimeout,
		IdleTimeout:  config.DefaultIdleTimeout,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		utils.Logger.Info("Shutdown signal received", zap.String(utils.FieldSignal, sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			utils.Logger.Error("Server shutdown error", zap.Error(err))
		}
	}()

	utils.Logger.Info("Server starting",
		zap.String(utils.FieldHost, appConfig.APIHost),
		zap.String(utils.FieldPort, portStr))

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		utils.Logger.Fatal("Server failed to start", zap.Error(err))
	}

	utils.Logger.Info("Server stopped")
}
