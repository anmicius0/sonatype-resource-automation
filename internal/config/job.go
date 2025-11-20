// Path: internal/config/job.go
package config

import (
	"fmt"
	"sync"
	"time"
)

// JobStatus represents the current state of a background job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// Job represents a background operation for repository creation or deletion.
type Job struct {
	// ID is the unique identifier for this job
	ID string
	// Status is the current state of the job (pending, processing, completed, or failed)
	Status JobStatus
	// Action is either "create" or "delete"
	Action string
	// CreatedAt is the time when the job was created
	CreatedAt time.Time
	// UpdatedAt is the time when the job was last updated
	UpdatedAt time.Time
	// TotalRequests is the count of valid requests accepted into the job
	TotalRequests int
	// SuccessfulOperations counts requests that completed without error
	SuccessfulOperations int
	// FailedOperations counts requests that encountered an error
	FailedOperations int
	// NotProcessedOperations counts requests not yet processed
	NotProcessedOperations int
	// FailedRequests contains details of requests that failed
	FailedRequests []FailedRequest
	// Message is a human-readable status message
	Message string
}

// JobStore manages in-memory job tracking (use database for production)
type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewJobStore creates a new job store instance
func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job with pending status
func (js *JobStore) CreateJob(id, action string, totalRequests int) *Job {
	js.mu.Lock()
	defer js.mu.Unlock()

	job := &Job{
		ID:                     id,
		Status:                 JobStatusPending,
		Action:                 action,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		TotalRequests:          totalRequests,
		SuccessfulOperations:   0,
		FailedOperations:       0,
		NotProcessedOperations: totalRequests,
		FailedRequests:         make([]FailedRequest, 0),
		Message:                "Job queued",
	}
	js.jobs[id] = job
	return job
}

// GetJob retrieves a job by ID
func (js *JobStore) GetJob(id string) (*Job, bool) {
	js.mu.RLock()
	defer js.mu.RUnlock()
	job, exists := js.jobs[id]
	return job, exists
}

// UpdateJob updates a job's status and data
func (js *JobStore) UpdateJob(id string, updateFn func(*Job)) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}
	updateFn(job)
	job.UpdatedAt = time.Now()
	return nil
}
