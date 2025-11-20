// internal/server/sync.go
// Package server provides orchestration logic for Sonatype resource automation.
package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/service"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BatchManager encapsulates async job execution for repository requests.
type BatchManager struct {
	cfg      *config.Config
	jobStore *config.JobStore
	nexus    client.NexusClient
	iq       client.IQClient
}

type operationResult struct {
	Success bool
	Error   string
}

// NewBatchManager constructs a BatchManager with the required dependencies.
func NewBatchManager(cfg *config.Config, jobStore *config.JobStore, nexus client.NexusClient, iq client.IQClient) *BatchManager {
	return &BatchManager{cfg, jobStore, nexus, iq}
}

// ProcessBatchAsync creates a job and processes the valid requests in the background.
// This function combines the logic of the previous QueueJob and processBatch.
func (bm *BatchManager) ProcessBatchAsync(validationResult *ValidationResult, batchRequest batchRepositoryRequest, action string) (string, int, int, int) {
	totalRequests := len(batchRequest.Requests)
	validCount := len(validationResult.ValidRequests)
	invalidCount := len(validationResult.InvalidRequests)
	jobID := uuid.New().String()

	// 1. Create the job in the store.
	bm.jobStore.CreateJob(jobID, action, validCount)

	utils.Logger.Debug("Queued job",
		zap.String(utils.FieldJobID, jobID),
		zap.String(utils.FieldAction, action),
		zap.Int("total_requests", totalRequests),
		zap.Int("valid_count", validCount),
		zap.Int("invalid_count", invalidCount))

	// 2. Launch the background processor.
	go func() {
		ctx := context.Background()
		requests := validationResult.ValidRequests
		tracker := service.NewJobProgressTracker(bm.jobStore, jobID)

		utils.Logger.Debug("Starting batch processing",
			zap.String(utils.FieldJobID, jobID),
			zap.Int("request_count", len(requests)),
			zap.String(utils.FieldAction, action))
		tracker.SetProcessing()

		type batchResult struct {
			request config.RepositoryRequest
			result  operationResult
		}

		results := make(chan batchResult, len(requests))
		var wg sync.WaitGroup

		// 3. Fan out: Start a worker goroutine for each request.
		for _, repositoryRequest := range requests {
			wg.Add(1)
			go func(req config.RepositoryRequest) {
				defer wg.Done()
				utils.Logger.Debug("Attempting operation for repository",
					zap.String("ldap_username", req.LdapUsername),
					zap.String("package_manager", req.PackageManager),
					zap.String("organization_name", req.OrganizationName),
					zap.String(utils.FieldAction, action))
				opResult := bm.attemptOperation(ctx, action, req)
				results <- batchResult{request: req, result: opResult}
			}(repositoryRequest)
		}

		// Wait for all workers to finish, then close the results channel.
		wg.Wait()
		close(results)

		// 4. Fan in: Aggregate results and finalize the job.
		successfulOps := 0
		failedOps := 0
		failedRequests := make([]config.FailedRequest, 0, len(requests))

		for res := range results {
			if res.result.Success {
				successfulOps++
			} else {
				failedOps++
				failedRequests = append(failedRequests, config.FailedRequest{
					Request: res.request,
					Reason:  res.result.Error,
				})
			}
		}

		tracker.Finalize(successfulOps, failedOps, 0, len(requests), failedRequests)
		utils.Logger.Debug("Finished batch processing",
			zap.String(utils.FieldJobID, jobID),
			zap.Int("successful_ops", successfulOps),
			zap.Int("failed_ops", failedOps))
	}()

	return jobID, totalRequests, validCount, invalidCount
}

// attemptOperation performs the actual create/delete logic for a single request.
// This function accepts a context for cancellation support.
func (bm *BatchManager) attemptOperation(ctx context.Context, action string, req config.RepositoryRequest) operationResult {
	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return operationResult{Success: false, Error: fmt.Sprintf("request cancelled: %v", ctx.Err())}
	default:
	}

	opConfig, err := bm.cfg.CreateOpConfig(req, action)
	if err != nil {
		utils.Logger.Error("Failed to create operation config",
			zap.Error(err),
			zap.String(utils.FieldAction, action))
		return operationResult{Success: false, Error: err.Error()}
	}

	utils.Logger.Debug("Created operation config",
		zap.String(utils.FieldRepo, opConfig.RepositoryName),
		zap.String(utils.FieldAction, opConfig.Action),
		zap.String("package_manager", opConfig.PackageManager))

	var opErr error

	switch action {
	case MethodCreate:
		// Step 1: Create Nexus resources. If it fails, stop.
		repoManager := service.NewCreationManager(opConfig, bm.nexus)
		if _, opErr = repoManager.Run(); opErr != nil {
			break
		}

		// Step 2: If the first step succeeded, add owner role in IQ Server.
		if opConfig.OrganizationID != "" {
			opErr = bm.iq.AddOwnerRoleToUser(opConfig)
			if opErr != nil {
				utils.Logger.Error("Failed to assign Owner role in IQ Server",
					zap.String("ldap_username", opConfig.LdapUsername),
					zap.String("organization_id", opConfig.OrganizationID),
					zap.Error(opErr))
			} else {
				utils.Logger.Info("Successfully assigned Owner role in IQ Server",
					zap.String("ldap_username", opConfig.LdapUsername),
					zap.String("organization_id", opConfig.OrganizationID))
			}
		} else {
			utils.Logger.Warn("No organization_id; skipping IQ Server role assignment",
				zap.String("ldap_username", opConfig.LdapUsername))
		}

	case MethodDelete:
		// Step 1: Delete Nexus resources. If it fails, stop.
		repoManager := service.NewDeletionManager(opConfig, bm.nexus)
		if _, opErr = repoManager.Run(); opErr != nil {
			break
		}

		// Step 2: If the first step succeeded, clean up from IQ Server.
		iqManager := service.NewIQDeletionManager(opConfig, bm.iq, bm.nexus)
		_, opErr = iqManager.Run()

	default:
		opErr = fmt.Errorf("unsupported action: %s", action)
	}

	// Centralized error handling for the entire operation
	if opErr != nil {
		utils.Logger.Error("Operation failed",
			zap.Error(opErr),
			zap.String(utils.FieldAction, action),
			zap.String(utils.FieldRepo, opConfig.RepositoryName))
		return operationResult{Success: false, Error: opErr.Error()}
	}

	utils.Logger.Info("Operation succeeded",
		zap.String(utils.FieldAction, action),
		zap.String(utils.FieldRepo, opConfig.RepositoryName))
	return operationResult{Success: true}
}
