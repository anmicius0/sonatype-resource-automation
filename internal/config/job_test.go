package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJobStore(t *testing.T) {
	store := NewJobStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.jobs)
}

func TestCreateJob(t *testing.T) {
	store := NewJobStore()
	job := store.CreateJob("job-1", "create", 10)

	assert.Equal(t, "job-1", job.ID)
	assert.Equal(t, "create", job.Action)
	assert.Equal(t, 10, job.TotalRequests)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, 10, job.NotProcessedOperations)
	assert.WithinDuration(t, time.Now(), job.CreatedAt, time.Second)
}

func TestGetJob(t *testing.T) {
	store := NewJobStore()
	store.CreateJob("job-1", "create", 10)

	// Test existing job
	job, exists := store.GetJob("job-1")
	assert.True(t, exists)
	assert.Equal(t, "job-1", job.ID)

	// Test non-existing job
	job, exists = store.GetJob("job-2")
	assert.False(t, exists)
	assert.Nil(t, job)
}

func TestUpdateJob(t *testing.T) {
	store := NewJobStore()
	store.CreateJob("job-1", "create", 10)

	// Update job
	err := store.UpdateJob("job-1", func(j *Job) {
		j.Status = JobStatusProcessing
		j.SuccessfulOperations = 5
	})

	assert.NoError(t, err)

	job, _ := store.GetJob("job-1")
	assert.Equal(t, JobStatusProcessing, job.Status)
	assert.Equal(t, 5, job.SuccessfulOperations)

	// Update non-existing job
	err = store.UpdateJob("job-2", func(j *Job) {})
	assert.Error(t, err)
}
