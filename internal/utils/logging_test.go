package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithComponent(t *testing.T) {
	// Setup observer to capture logs
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	Logger = zap.New(observedZapCore)

	// Test WithComponent
	componentLogger := WithComponent("test_component")
	assert.NotNil(t, componentLogger)

	componentLogger.Info("test message")

	// Verify log entry
	logs := observedLogs.All()
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "test message", logs[0].Message)

	// Check context fields
	contextMap := logs[0].ContextMap()
	assert.Equal(t, "test_component", contextMap["component"])
}

func TestWithComponent_NilLogger(t *testing.T) {
	// Temporarily set Logger to nil
	originalLogger := Logger
	Logger = nil
	defer func() { Logger = originalLogger }()

	componentLogger := WithComponent("test_component")
	assert.Nil(t, componentLogger)
}
