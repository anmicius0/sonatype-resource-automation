package service

import (
	"errors"
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteRepository(t *testing.T) {
	opConfig := &config.OperationConfig{
		RepositoryName: "test-repo",
		Action:         "delete",
	}

	t.Run("Delete success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("DeleteRepository", "test-repo").Return(nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.DeleteRepository()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete failure", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("DeleteRepository", "test-repo").Return(errors.New("delete error"))

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.DeleteRepository()

		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestDeletePrivilege(t *testing.T) {
	opConfig := &config.OperationConfig{
		PrivilegeName: "test-privilege",
		Action:        "delete",
	}

	t.Run("Delete success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("DeletePrivilege", "test-privilege").Return(nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.DeletePrivilege()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestCleanupRole(t *testing.T) {
	opConfig := &config.OperationConfig{
		RoleName: "test-role",
		Action:   "delete",
	}

	t.Run("Role not found", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetRole", "test-role").Return(nil, nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.CleanupRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Role empty, delete success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		role := &client.Role{
			Privileges: []string{},
		}
		mockClient.On("GetRole", "test-role").Return(role, nil)
		mockClient.On("DeleteRole", "test-role").Return(nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.CleanupRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Role has privileges, skip delete", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		role := &client.Role{
			Privileges: []string{"other-privilege"},
		}
		mockClient.On("GetRole", "test-role").Return(role, nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.CleanupRole()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestCleanupUserRoles(t *testing.T) {
	opConfig := &config.OperationConfig{
		LdapUsername: "test-user",
		RoleName:     "test-role",
		BaseRoles:    []string{"base-role"},
		ExtraRoles:   []string{},
		Action:       "delete",
	}

	t.Run("User not found", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		mockClient.On("GetUser", "test-user").Return(nil, nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.CleanupUserRoles()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Remove role success", func(t *testing.T) {
		mockClient := new(MockNexusClient)
		user := &client.User{
			Roles: []string{"test-role", "base-role"},
		}
		roleInfo := &client.Role{
			Privileges: []string{},
		}

		mockClient.On("GetUser", "test-user").Return(user, nil)
		mockClient.On("GetRole", "test-role").Return(roleInfo, nil)
		mockClient.On("UpdateUser", mock.Anything).Return(nil)

		cleaner := NewNexusCleaner(opConfig, mockClient)
		err := cleaner.CleanupUserRoles()

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestDeletionManager_Run_OffboardingMode(t *testing.T) {
	opConfig := &config.OperationConfig{
		Action:         "delete",
		Shared:         true,
		AppID:          "app-123",
		LdapUsername:   "offboard-user",
		RepositoryName: "npm-release-app-123",
		PrivilegeName:  "npm-release-app-123",
		RoleName:       "offboard-user",
		BaseRoles:      []string{"base-role"},
		ExtraRoles:     []string{"extra-role"},
	}

	mockClient := new(MockNexusClient)
	mockClient.On("GetUser", "offboard-user").Return(&client.User{Roles: []string{"some-role"}}, nil)
	mockClient.On("UpdateUser", mock.MatchedBy(func(u *client.User) bool {
		return u.Status == "disabled" && len(u.Roles) == 1 && u.Roles[0] == "base-role"
	})).Return(nil)
	mockClient.On("DeleteRole", "offboard-user").Return(nil)

	// Mock listing repositories
	mockClient.On("GetRepositories").Return([]client.Repository{
		{Name: "npm-release-app-123"},
		{Name: "maven-release-app-123"},
		{Name: "other-repo"},
	}, nil)

	// Mock listing privileges
	mockClient.On("GetPrivileges").Return([]client.Privilege{
		{Name: "npm-release-app-123"},
		{Name: "maven-release-app-123"},
		{Name: "other-priv"},
	}, nil)

	// Mock deleting discovered resources
	mockClient.On("DeleteRepository", "npm-release-app-123").Return(nil)
	mockClient.On("DeleteRepository", "maven-release-app-123").Return(nil)
	mockClient.On("DeletePrivilege", "npm-release-app-123").Return(nil)
	mockClient.On("DeletePrivilege", "maven-release-app-123").Return(nil)

	dm := NewDeletionManager(opConfig, mockClient)
	result, err := dm.Run()

	assert.NoError(t, err)
	assert.Equal(t, "offboarding", result["mode"])
	assert.Equal(t, "offboard-user", result["ldap_username"])
	mockClient.AssertExpectations(t)
}

func TestDeletionManager_Run_SharedRoleCleanup(t *testing.T) {
	opConfig := &config.OperationConfig{
		Action:         "delete",
		RoleName:       "repositories.share",
		Shared:         true,
		LdapUsername:   "shared-user",
		OrganizationID: "org-id",
		BaseRoles:      []string{"base-role"},
		ExtraRoles:     []string{"extra-role"},
	}

	mockClient := new(MockNexusClient)
	mockClient.On("GetUser", "shared-user").Return(&client.User{Roles: []string{"repositories.share", "extra-role"}}, nil)
	mockClient.On("GetRole", "repositories.share").Return(&client.Role{Privileges: []string{}}, nil)
	mockClient.On("UpdateUser", mock.MatchedBy(func(u *client.User) bool {
		return len(u.Roles) == 1 && u.Roles[0] == "base-role"
	})).Return(nil)

	dm := NewDeletionManager(opConfig, mockClient)
	result, err := dm.Run()

	assert.NoError(t, err)
	assert.Equal(t, "delete", result["action"])
	assert.Equal(t, "shared-user", result["ldap_username"])
	assert.Equal(t, "org-id", result["organization_id"])
	mockClient.AssertExpectations(t)
}

func TestDeletionManager_Run_FullCleanup(t *testing.T) {
	opConfig := &config.OperationConfig{
		Action:         "delete",
		RoleName:       "app-role",
		RepositoryName: "app-role-repo",
		PrivilegeName:  "app-role-repo",
		LdapUsername:   "app-user",
		OrganizationID: "org-b",
		BaseRoles:      []string{"base-role"},
		ExtraRoles:     []string{"extra-role"},
	}

	mockClient := new(MockNexusClient)
	mockClient.On("DeleteRepository", "app-role-repo").Return(nil)
	mockClient.On("DeletePrivilege", "app-role-repo").Return(nil)
	mockClient.On("GetRole", "app-role").Return(nil, nil).Twice()
	mockClient.On("GetUser", "app-user").Return(&client.User{Roles: []string{"app-role", "base-role"}}, nil)
	mockClient.On("UpdateUser", mock.MatchedBy(func(u *client.User) bool {
		return len(u.Roles) == 1 && u.Roles[0] == "base-role"
	})).Return(nil)

	dm := NewDeletionManager(opConfig, mockClient)
	result, err := dm.Run()

	assert.NoError(t, err)
	assert.Equal(t, "delete", result["action"])
	assert.Equal(t, "app-role-repo", result["repository_name"])
	assert.Equal(t, "app-user", result["ldap_username"])
	assert.Equal(t, "org-b", result["organization_id"])
	mockClient.AssertExpectations(t)
}
