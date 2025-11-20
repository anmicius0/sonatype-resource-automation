package server

import (
	"fmt"
	"net/http"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler bundles request-time dependencies for the API routes.
type Handler struct {
	cfg          *config.Config
	jobStore     *config.JobStore
	batchManager *BatchManager
}

// newHandler constructs a Handler with attached dependencies.
func newHandler(cfg *config.Config, jobStore *config.JobStore, batchManager *BatchManager) *Handler {
	return &Handler{
		cfg:          cfg,
		jobStore:     jobStore,
		batchManager: batchManager,
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "status": StatusHealthy})
}

func (h *Handler) createBatch(c *gin.Context) {
	h.processBatch(c, MethodCreate)
}

func (h *Handler) deleteBatch(c *gin.Context) {
	h.processBatch(c, MethodDelete)
}

func (h *Handler) processBatch(c *gin.Context, action string) {
	// Validate and parse the incoming batch request
	var batch batchRepositoryRequest
	if err := c.ShouldBindJSON(&batch); err != nil {
		utils.Logger.Error("Invalid request body",
			zap.Error(err))
		respBuilder := newResponseBuilder()
		c.JSON(http.StatusUnprocessableEntity, respBuilder.BuildErrorResponse(
			ErrorCodeInvalidRequestBody,
			MessageInvalidRequestBody,
			err.Error(),
		))
		return
	}

	// Ensure at least one request is present
	if len(batch.Requests) == 0 {
		respBuilder := newResponseBuilder()
		c.JSON(http.StatusUnprocessableEntity, respBuilder.BuildErrorResponse(
			ErrorCodeValidationFailed,
			MessageBatchEmpty,
			nil,
		))
		return
	}

	// Validate the request body format
	validationResult := h.validateBatchRequest(batch, action)

	// If all requests are invalid, return a validation failed response
	if len(validationResult.ValidRequests) == 0 {
		respBuilder := newResponseBuilder()
		utils.Logger.Info("All requests failed validation",
			zap.Int("invalid_count", len(validationResult.InvalidRequests)))
		c.JSON(http.StatusUnprocessableEntity, respBuilder.BuildValidationFailedResponse(validationResult))
		return
	}

	// Process the valid requests asynchronously
	jobID, totalRequests, validCount, invalidCount := h.batchManager.ProcessBatchAsync(validationResult, batch, action)
	respBuilder := newResponseBuilder()
	c.JSON(http.StatusAccepted, respBuilder.BuildAcceptedResponse(jobID, totalRequests, validCount, invalidCount, validationResult))
}

func (h *Handler) getJobStatus(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := h.jobStore.GetJob(jobID)
	if !exists {
		utils.Logger.Debug("Job not found",
			zap.String(utils.FieldJobID, jobID))
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf(JobNotFoundMessageFmt, jobID)})
		return
	}

	respBuilder := newResponseBuilder()
	c.JSON(http.StatusOK, respBuilder.BuildJobResponse(job))
}

func authMiddleware(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		expectedAuth := fmt.Sprintf("Bearer %s", expectedToken)
		if authHeader != expectedAuth {
			utils.Logger.Warn("Unauthorized access attempt",
				zap.String(utils.FieldPath, c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{"error": MessageInvalidToken})
			c.Abort()
			return
		}
		c.Next()
	}
}

// validateBatchRequest validates the individual requests in a batch.
func (h *Handler) validateBatchRequest(batch batchRepositoryRequest, action string) *ValidationResult {
	validationResult := &ValidationResult{
		ValidRequests:   make([]config.RepositoryRequest, 0, len(batch.Requests)),
		InvalidRequests: make([]ValidationError, 0, len(batch.Requests)),
	}
	for _, req := range batch.Requests {
		// 1. Validate PackageManager
		// Case A: Delete + Shared = Offboarding. PackageManager MUST be empty.
		// Case B: All other cases. PackageManager MUST be present.
		if action == MethodDelete && req.Shared {
			if req.PackageManager != "" {
				validationResult.InvalidRequests = append(validationResult.InvalidRequests, ValidationError{
					Request: req,
					Reasons: []string{"packageManager must be empty for shared delete operations"},
				})
				continue
			}
		} else {
			if req.PackageManager == "" {
				validationResult.InvalidRequests = append(validationResult.InvalidRequests, ValidationError{
					Request: req,
					Reasons: []string{"packageManager is required for this operation type"},
				})
				continue
			}
		}

		// 2. Validate AppID/Shared Combinations
		// If Action is Create: Shared=true MUST have Empty AppID.
		// If Action is Delete: Shared=true MUST have AppID (Offboarding Mode).
		if action == MethodCreate {
			if req.Shared && req.AppID != "" {
				validationResult.InvalidRequests = append(validationResult.InvalidRequests, ValidationError{
					Request: req,
					Reasons: []string{"appid not allowed for shared repos on create"},
				})
				continue
			}
		} else if action == MethodDelete {
			if req.Shared && req.AppID == "" {
				validationResult.InvalidRequests = append(validationResult.InvalidRequests, ValidationError{
					Request: req,
					Reasons: []string{"appid required for shared repos on delete (offboarding)"},
				})
				continue
			}
		}

		if !req.Shared && req.AppID == "" {
			validationResult.InvalidRequests = append(validationResult.InvalidRequests, ValidationError{
				Request: req,
				Reasons: []string{"appid required for non-shared repos"},
			})
			continue
		}

		validationResult.ValidRequests = append(validationResult.ValidRequests, req)
	}
	return validationResult
}
