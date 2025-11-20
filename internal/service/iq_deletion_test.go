package service

import (
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockIQClient is a mock implementation of client.IQClient for service tests.
type MockIQClient struct {
	mock.Mock
}

func (m *MockIQClient) GetRoles() ([]client.IQRole, error) {
	args := m.Called()
	return args.Get(0).([]client.IQRole), args.Error(1)
}

func (m *MockIQClient) FindOwnerRoleID() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockIQClient) AddOwnerRoleToUser(opConfig *config.OperationConfig) error {
	args := m.Called(opConfig)
	return args.Error(0)
}

func (m *MockIQClient) RemoveOwnerRoleFromUser(opConfig *config.OperationConfig) error {
	args := m.Called(opConfig)
	return args.Error(0)
}

func TestIQServerCleaner_RemovesOwnerDuringOffboarding(t *testing.T) {
	opConfig := &config.OperationConfig{
		Action:         "delete",
		LdapUsername:   "offboard-user",
		OrganizationID: "org-123",
		RoleName:       "offboard-user",
		Shared:         true,
		AppID:          "app-99",
		BaseRoles:      []string{"base-role"},
		ExtraRoles:     []string{},
	}

	mockNexus := new(MockNexusClient)
	mockNexus.On("GetUser", "offboard-user").Return(&client.User{Roles: []string{"offboard-user", "base-role"}}, nil)

	mockIQ := new(MockIQClient)
	mockIQ.On("RemoveOwnerRoleFromUser", opConfig).Return(nil)

	cleaner := NewIQServerCleaner(opConfig, mockIQ, mockNexus)
	err := cleaner.CleanupUserFromOrganization()

	assert.NoError(t, err)
	mockNexus.AssertExpectations(t)
	mockIQ.AssertExpectations(t)
}
