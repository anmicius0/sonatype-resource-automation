package server

import "github.com/anmicius0/sonatype-resource-automation/internal/config"

// ValidationError represents validation errors for a single request with detailed context.
type ValidationError struct {
	Request config.RepositoryRequest
	Reasons []string // List of all validation error messages
}

// ValidationResult contains validation results for an entire batch.
type ValidationResult struct {
	ValidRequests   []config.RepositoryRequest
	InvalidRequests []ValidationError
}

// batchRepositoryRequest holds a batch of repository requests for bulk processing.
type batchRepositoryRequest struct {
	// Requests is the list of repository operation requests to process
	Requests []config.RepositoryRequest `binding:"required,dive"`
}
