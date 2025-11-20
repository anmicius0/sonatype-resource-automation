// internal/service/progress_tracker.go
package service

import (
	"fmt"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

// JobProgressTracker manages job progress tracking and updates.
type JobProgressTracker struct {
	jobStore *config.JobStore
	jobID    string
}

// NewJobProgressTracker creates a new job progress tracker.
func NewJobProgressTracker(jobStore *config.JobStore, jobID string) *JobProgressTracker {
	return &JobProgressTracker{
		jobStore: jobStore,
		jobID:    jobID,
	}
}

// SetProcessing marks the job as processing.
func (jpt *JobProgressTracker) SetProcessing() {
	_ = jpt.jobStore.UpdateJob(jpt.jobID, func(job *config.Job) {
		job.Status = config.JobStatusProcessing
		job.Message = "Processing requests"
	})
}

// Finalize marks a job as completed or failed with appropriate status and message.
func (jpt *JobProgressTracker) Finalize(successful, failed, notProcessed, total int, failedRequests []config.FailedRequest) {
	_ = jpt.jobStore.UpdateJob(jpt.jobID, func(job *config.Job) {
		job.SuccessfulOperations = successful
		job.FailedOperations = failed
		job.NotProcessedOperations = notProcessed
		job.FailedRequests = failedRequests

		// Determine final status and message
		if failed == 0 {
			job.Status = config.JobStatusCompleted
			job.Message = fmt.Sprintf("Successfully processed all %d requests", successful)
		} else if successful == 0 {
			job.Status = config.JobStatusFailed
			job.Message = fmt.Sprintf("All %d requests failed", failed)
		} else {
			job.Status = config.JobStatusCompleted
			job.Message = fmt.Sprintf("Processed %d of %d requests with %d errors", successful, total, failed)
		}
	})

	utils.Logger.Info("Job finalized",
		zap.String("job_id", jpt.jobID),
		zap.Int("successful", successful),
		zap.Int("failed", failed),
		zap.Int("total", total))
}

// MarkFailed marks a job as failed when no valid requests exist.
func (jpt *JobProgressTracker) MarkFailed(totalRequests int) {
	_ = jpt.jobStore.UpdateJob(jpt.jobID, func(job *config.Job) {
		job.Status = config.JobStatusFailed
		job.TotalRequests = totalRequests
		job.FailedOperations = totalRequests
		job.NotProcessedOperations = 0
		job.Message = fmt.Sprintf("All %d requests failed", totalRequests)
	})

	utils.Logger.Info("Job marked as failed",
		zap.String("job_id", jpt.jobID),
		zap.Int("total_requests", totalRequests))
}
