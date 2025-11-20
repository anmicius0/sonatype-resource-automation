package server

import (
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildJobResponse(t *testing.T) {
	rb := newResponseBuilder()
	job := &config.Job{
		ID:            "job-123",
		Status:        config.JobStatusPending,
		TotalRequests: 10,
	}

	resp := rb.BuildJobResponse(job)
	respMap, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "job-123", respMap["id"])
	assert.Equal(t, config.JobStatusPending, respMap["status"])
	assert.Equal(t, 10, respMap["totalRequests"])
}

func TestBuildAcceptedResponse(t *testing.T) {
	rb := newResponseBuilder()
	validationResult := &ValidationResult{
		InvalidRequests: []ValidationError{},
	}

	resp := rb.BuildAcceptedResponse("job-123", 10, 10, 0, validationResult)
	respMap, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, respMap["success"])
	assert.Equal(t, "job-123", respMap["jobId"])

	validation, ok := respMap["validation"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 10, validation["totalRequests"])
}

func TestBuildErrorResponse(t *testing.T) {
	rb := newResponseBuilder()
	resp := rb.BuildErrorResponse("ERR_CODE", "Error message", nil)

	respMap, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, false, respMap["success"])
	assert.Equal(t, "ERR_CODE", respMap["error"])
	assert.Equal(t, "Error message", respMap["message"])
}

func TestToCamelCaseMap(t *testing.T) {
	input := struct {
		SimpleField  string
		ID           string
		JobID        string
		RepoURL      string
		NestedStruct struct {
			InnerField int
		}
	}{
		SimpleField: "value",
		ID:          "123",
		JobID:       "job-1",
		RepoURL:     "http://example.com",
		NestedStruct: struct{ InnerField int }{
			InnerField: 42,
		},
	}

	output := toCamelCaseMap(input)
	outMap, ok := output.(map[string]interface{})
	assert.True(t, ok)

	assert.Equal(t, "value", outMap["simpleField"])
	assert.Equal(t, "123", outMap["id"])
	assert.Equal(t, "job-1", outMap["jobId"])
	assert.Equal(t, "http://example.com", outMap["repoUrl"])

	nested, ok := outMap["nestedStruct"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 42, nested["innerField"])
}
