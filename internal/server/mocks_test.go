package server

import (
	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/stretchr/testify/mock"
)

type MockNexusClient struct {
	mock.Mock
}

func (m *MockNexusClient) GetRepository(name string) (*client.Repository, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockNexusClient) CreateProxyRepository(config *config.OperationConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockNexusClient) DeleteRepository(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockNexusClient) GetPrivilege(name string) (*client.Privilege, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Privilege), args.Error(1)
}

func (m *MockNexusClient) CreatePrivilege(config *config.OperationConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockNexusClient) DeletePrivilege(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockNexusClient) GetRole(name string) (*client.Role, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Role), args.Error(1)
}

func (m *MockNexusClient) CreateRole(config *config.OperationConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockNexusClient) UpdateRole(role *client.Role) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockNexusClient) DeleteRole(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockNexusClient) GetUser(username string) (*client.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.User), args.Error(1)
}

func (m *MockNexusClient) UpdateUser(user *client.User) error {
	args := m.Called(user)
	return args.Error(0)
}

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
