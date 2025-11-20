package service

import (
	"os"
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	// Initialize a no-op logger for testing to prevent panics
	utils.Logger = zap.NewNop()

	os.Exit(m.Run())
}
