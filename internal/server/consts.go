package server

const (
	HealthEndpoint   = "/health"
	RepositoriesPath = "/repositories"
	JobsPath         = "/jobs"
)

// keys constants were intentionally removed. Responses are generated via structs
// and use lowerCamelCase JSON fields.

const (
	StatusHealthy = "healthy"
	StatusPending = "pending"
)

const (
	MessageJobQueued          = "Job queued for processing"
	MessageValidationFailed   = "All requests failed validation"
	MessageInvalidRequestBody = "Invalid request body"
	MessageBatchEmpty         = "Batch must contain at least one request"
	MessageInvalidToken       = "Invalid token"
)

const (
	ErrorCodeInvalidRequestBody = "invalid_request_body"
	ErrorCodeValidationFailed   = "validation_failed"
)

const (
	JobNotFoundMessageFmt = "Job %s not found"
)

const (
	MethodCreate = "create"
	MethodDelete = "delete"
)
