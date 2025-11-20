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
