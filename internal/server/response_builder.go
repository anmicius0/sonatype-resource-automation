// internal/server/response_builder.go
package server

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
)

// ResponseBuilder provides utilities for constructing consistent API responses.
type ResponseBuilder struct{}

// newResponseBuilder creates a new response builder instance.
func newResponseBuilder() *ResponseBuilder { return &ResponseBuilder{} }

// AcceptedResponse is the payload returned for accepted batch requests.
type AcceptedResponse struct {
	Success    bool
	Message    string
	JobID      string
	Status     string
	Validation ValidationSummary
}

// ValidationSummary contains batch validation counts and details.
type ValidationSummary struct {
	TotalRequests     int
	ValidRequests     int
	InvalidRequests   int
	FailedValidations []InvalidRequestResponse
}

// InvalidRequestResponse holds a single invalid batch request's details.
type InvalidRequestResponse struct {
	OrganizationName string
	LdapUsername     string
	PackageManager   string
	Shared           bool
	AppID            string
	ValidationErrors []string
}

// ErrorResponse standardizes error responses.
type ErrorResponse struct {
	Success bool
	Error   string
	Message string
	Details any
}

// ValidationFailedResponse is returned when all requests are invalid.
type ValidationFailedResponse struct {
	Success         bool
	Message         string
	Error           string
	InvalidRequests ValidationFailedResponseDetails
}

// ValidationFailedResponseDetails contains the invalid requests summary and details.
type ValidationFailedResponseDetails struct {
	Count   int
	Details []InvalidRequestResponse
}

// BuildJobResponse constructs the job status response with all metrics, converting keys to camelCase.
func (rb *ResponseBuilder) BuildJobResponse(job *config.Job) any {
	return toCamelCaseMap(job)
}

// BuildAcceptedResponse constructs an AcceptedResponse with validation details, converting keys to camelCase.
func (rb *ResponseBuilder) BuildAcceptedResponse(jobID string, totalRequests, validCount, invalidCount int, validationResult *ValidationResult) any {
	response := AcceptedResponse{
		Success: true,
		Message: MessageJobQueued,
		JobID:   jobID,
		Status:  StatusPending,
		Validation: ValidationSummary{
			TotalRequests:     totalRequests,
			ValidRequests:     validCount,
			InvalidRequests:   invalidCount,
			FailedValidations: rb.ConvertValidationErrorsToResponse(validationResult.InvalidRequests),
		},
	}
	return toCamelCaseMap(response)
}

// BuildErrorResponse constructs a standardized error response, converting keys to camelCase.
func (rb *ResponseBuilder) BuildErrorResponse(errorCode, errorMessage string, details any) any {
	response := ErrorResponse{
		Success: false,
		Error:   errorCode,
		Message: errorMessage,
		Details: details,
	}
	return toCamelCaseMap(response)
}

// BuildValidationFailedResponse constructs a response for validation failures, converting keys to camelCase.
func (rb *ResponseBuilder) BuildValidationFailedResponse(validationResult *ValidationResult) any {
	response := ValidationFailedResponse{
		Success: false,
		Message: MessageValidationFailed,
		Error:   ErrorCodeValidationFailed,
		InvalidRequests: ValidationFailedResponseDetails{
			Count:   len(validationResult.InvalidRequests),
			Details: rb.ConvertValidationErrorsToResponse(validationResult.InvalidRequests),
		},
	}
	return toCamelCaseMap(response)
}

// ConvertValidationErrorsToResponse transforms validation errors to response format.
func (rb *ResponseBuilder) ConvertValidationErrorsToResponse(validationErrors []ValidationError) []InvalidRequestResponse {
	response := make([]InvalidRequestResponse, 0, len(validationErrors))
	for _, ve := range validationErrors {
		response = append(response, InvalidRequestResponse{
			OrganizationName: ve.Request.OrganizationName,
			LdapUsername:     ve.Request.LdapUsername,
			PackageManager:   ve.Request.PackageManager,
			Shared:           ve.Request.Shared,
			AppID:            ve.Request.AppID,
			ValidationErrors: ve.Reasons,
		})
	}
	return response
}

func toCamelCaseMap(data any) any {
	val := reflect.ValueOf(data)

	// Handle Pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Handle Slices/Arrays
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		out := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			out[i] = toCamelCaseMap(val.Index(i).Interface())
		}
		return out
	}

	// Handle Structs
	if val.Kind() == reflect.Struct {
		out := make(map[string]any)
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			// Skip unexported fields
			if field.PkgPath != "" {
				continue
			}

			// Recursively convert the field value
			fieldVal := toCamelCaseMap(val.Field(i).Interface())

			// Determine the new key name
			key := field.Name

			// Handle common acronyms manually for cleaner API design
			if key == "ID" || strings.HasSuffix(key, "ID") {
				// e.g., "ID" -> "id", "JobID" -> "jobId", "AppID" -> "appId"
				if key == "ID" {
					key = "id"
				} else {
					// Convert "JobID" -> "jobId"
					prefix := key[:len(key)-2]
					key = lowerFirst(prefix) + "Id"
				}
			} else if key == "URL" || strings.HasSuffix(key, "URL") {
				if key == "URL" {
					key = "url"
				} else {
					prefix := key[:len(key)-3]
					key = lowerFirst(prefix) + "Url"
				}
			} else {
				// Default camelCase conversion (lower first letter)
				key = lowerFirst(key)
			}

			out[key] = fieldVal
		}
		return out
	}

	// Return primitives as-is
	return data
}

// lowerFirst lowers the first rune of a string
func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}
