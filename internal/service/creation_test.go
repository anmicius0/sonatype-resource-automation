package service

import (
	"errors"
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNexusClient is a mock implementation of client.NexusClient
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

func (m *MockNexusClient) GetRepositories() ([]client.Repository, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Repository), args.Error(1)
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

func (m *MockNexusClient) GetPrivileges() ([]client.Privilege, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Privilege), args.Error(1)
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

func TestCreateRepository(t *testing.T) {
	opConfig := &config.OperationConfig{
		RepositoryName: "test-repo",
		PackageManager: "npm",
		RemoteURL:      "http://example.com",
		Action:         "create",
	}

	t.Run("Repository already exists", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetRepository", "test-repo").Return(&client.Repository{Name: "test-repo"}, nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreateRepository()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Repository does not exist, create success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetRepository", "test-repo").Return(nil, &client.HTTPError{StatusCode: 404, Body: "not found"})
		mockClient.On("CreateProxyRepository", opConfig).Return(nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreateRepository()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Create failure", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetRepository", "test-repo").Return(nil, &client.HTTPError{StatusCode: 404, Body: "not found"})
		mockClient.On("CreateProxyRepository", opConfig).Return(errors.New("create error"))

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreateRepository()

		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestCreatePrivilege(t *testing.T) {
	opConfig := &config.OperationConfig{
		PrivilegeName:  "test-privilege",
		RepositoryName: "test-repo",
		PackageManager: "npm",
		Action:         "create",
	}

	t.Run("Privilege already exists", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetPrivilege", "test-privilege").Return(&client.Privilege{Name: "test-privilege"}, nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreatePrivilege()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Create privilege success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetPrivilege", "test-privilege").Return(nil, errors.New("not found"))
		mockClient.On("CreatePrivilege", opConfig).Return(nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreatePrivilege()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Create privilege failure", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetPrivilege", "test-privilege").Return(nil, errors.New("not found"))
		mockClient.On("CreatePrivilege", opConfig).Return(errors.New("create error"))

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.CreatePrivilege()

		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestAddPrivilegeToRole(t *testing.T) {
	opConfig := &config.OperationConfig{
		RoleName:       "test-role",
		PrivilegeName:  "test-privilege",
		RepositoryName: "test-repo",
		Action:         "create",
	}

	t.Run("Role exists, privilege already added", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		role := &client.Role{
			Privileges: []string{"test-privilege"},
		}
		mockClient.On("GetRole", "test-role").Return(role, nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.AddPrivilegeToRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Role exists, add privilege success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		role := &client.Role{
			Privileges: []string{"other-privilege"},
		}
		mockClient.On("GetRole", "test-role").Return(role, nil)
		mockClient.On("UpdateRole", mock.MatchedBy(func(r *client.Role) bool {
			return len(r.Privileges) == 2 && r.Privileges[1] == "test-privilege"
		})).Return(nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.AddPrivilegeToRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Role does not exist, create role success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		// Simulate 404 Not Found
		httpErr := &client.HTTPError{StatusCode: 404}
		mockClient.On("GetRole", "test-role").Return(nil, httpErr)
		mockClient.On("CreateRole", opConfig).Return(nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.AddPrivilegeToRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestAddRoleToUser(t *testing.T) {
	opConfig := &config.OperationConfig{
		LdapUsername: "test-user",
		RoleName:     "test-role",
		ExtraRoles:   []string{"extra-role"},
		BaseRoles:    []string{"base-role"},
		Action:       "create",
	}

	t.Run("User not found", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetUser", "test-user").Return(nil, nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.AddRoleToUser()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user 'test-user' not found")
		mockClient.AssertExpectations(t)
	})

	t.Run("Add roles success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		user := &client.User{
			Roles: []string{"existing-role"},
		}
		mockClient.On("GetUser", "test-user").Return(user, nil)
		mockClient.On("UpdateUser", mock.MatchedBy(func(u *client.User) bool {
			// Check if all roles are present
			hasRole := false
			hasExtra := false
			hasBase := false
			for _, r := range u.Roles {
				if r == "test-role" {
					hasRole = true
				}
				if r == "extra-role" {
					hasExtra = true
				}
				if r == "base-role" {
					hasBase = true
				}
			}
			return hasRole && hasExtra && hasBase
		})).Return(nil)

		creator := NewNexusCreator(opConfig, mockClient)
		err := creator.AddRoleToUser()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}
