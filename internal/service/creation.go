package service

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

// NexusCreator handles idempotent creation of Nexus resources like repositories, privileges, and roles.
type NexusCreator struct {
	opConfig *config.OperationConfig
	nexus    client.NexusClient
}

var roleModificationLock sync.Mutex

// NewNexusCreator creates a new NexusCreator instance.
func NewNexusCreator(opConfig *config.OperationConfig, nexus client.NexusClient) *NexusCreator {
	return &NexusCreator{opConfig, nexus}
}

// CreateRepository creates a proxy repository if it does not exist.
func (nc *NexusCreator) CreateRepository() error {
	utils.WithComponent("nexus_creator").Debug("CreateRepository called",
		zap.String("action", nc.opConfig.Action),
		zap.String("repository_name", nc.opConfig.RepositoryName))

	_, err := nc.nexus.GetRepository(nc.opConfig.RepositoryName)
	if err == nil {
		// Repository exists, idempotent skip
		utils.WithComponent("nexus_creator").Debug("Repository already exists, skipping creation",
			zap.String("repository_name", nc.opConfig.RepositoryName))
		return nil
	}
	if err := nc.nexus.CreateProxyRepository(nc.opConfig); err != nil {
		return fmt.Errorf("create proxy repository '%s' (package_manager='%s', remote_url='%s'): %w", nc.opConfig.RepositoryName, nc.opConfig.PackageManager, nc.opConfig.RemoteURL, err)
	}
	utils.WithComponent("nexus_creator").Info("Successfully created proxy repository",
		zap.String("repository_name", nc.opConfig.RepositoryName),
		zap.String("package_manager", nc.opConfig.PackageManager),
		zap.String("remote_url", nc.opConfig.RemoteURL))
	return nil
}

// CreatePrivilege creates a repository privilege if it does not exist.
func (nc *NexusCreator) CreatePrivilege() error {
	utils.WithComponent("nexus_creator").Debug("CreatePrivilege called",
		zap.String("action", nc.opConfig.Action),
		zap.String("privilege_name", nc.opConfig.PrivilegeName))

	_, err := nc.nexus.GetPrivilege(nc.opConfig.PrivilegeName)
	if err == nil {
		// Privilege exists, idempotent skip
		utils.WithComponent("nexus_creator").Warn("Privilege already exists, skipping creation",
			zap.String("privilege_name", nc.opConfig.PrivilegeName))
		return nil
	}
	if err := nc.nexus.CreatePrivilege(nc.opConfig); err != nil {
		return fmt.Errorf("create privilege '%s' for repository '%s': %w", nc.opConfig.PrivilegeName, nc.opConfig.RepositoryName, err)
	}
	utils.WithComponent("nexus_creator").Info("Successfully created repository privilege",
		zap.String("privilege_name", nc.opConfig.PrivilegeName),
		zap.String("repository_name", nc.opConfig.RepositoryName),
		zap.String("package_manager", nc.opConfig.PackageManager))
	return nil
}

// AddPrivilegeToRole adds the repository privilege to the role, creating the role if necessary.
func (nc *NexusCreator) AddPrivilegeToRole() error {
	roleModificationLock.Lock()
	defer roleModificationLock.Unlock()

	utils.WithComponent("nexus_creator").Debug("AddPrivilegeToRole called",
		zap.String("action", nc.opConfig.Action),
		zap.String("role_name", nc.opConfig.RoleName),
		zap.String("privilege_name", nc.opConfig.PrivilegeName))

	role, err := nc.nexus.GetRole(nc.opConfig.RoleName)
	if err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			// Role doesn't exist; continue to create
			role = nil
		} else {
			return fmt.Errorf("add privilege '%s' to role '%s': get role failed: %w", nc.opConfig.PrivilegeName, nc.opConfig.RoleName, err)
		}
	}
	if role != nil {
		privileges := role.Privileges
		if slices.Contains(privileges, nc.opConfig.PrivilegeName) {
			utils.WithComponent("nexus_creator").Debug("Privilege already in role, skipping addition",
				zap.String("role_name", nc.opConfig.RoleName),
				zap.String("privilege_name", nc.opConfig.PrivilegeName))
			return nil
		}
		privileges = append(privileges, nc.opConfig.PrivilegeName)
		role.Privileges = privileges
		if err := nc.nexus.UpdateRole(role); err != nil {
			return fmt.Errorf("add privilege to role '%s': update role failed: %w", nc.opConfig.RoleName, err)
		}
		utils.WithComponent("nexus_creator").Info("Successfully added privilege to existing role",
			zap.String("role_name", nc.opConfig.RoleName),
			zap.String("privilege_name", nc.opConfig.PrivilegeName),
			zap.String("repository_name", nc.opConfig.RepositoryName))
		return nil
	}
	// Role does not exist; create it with the privilege
	if err := nc.nexus.CreateRole(nc.opConfig); err != nil {
		return fmt.Errorf("add privilege '%s' to role '%s': create role failed: %w", nc.opConfig.PrivilegeName, nc.opConfig.RoleName, err)
	}
	utils.WithComponent("nexus_creator").Info("Successfully created role with privilege",
		zap.String("role_name", nc.opConfig.RoleName),
		zap.String("privilege_name", nc.opConfig.PrivilegeName),
		zap.String("repository_name", nc.opConfig.RepositoryName))
	return nil
}

// AddRoleToUser adds the role and extra roles to the user, deduplicating existing roles.
func (nc *NexusCreator) AddRoleToUser() error {
	utils.WithComponent("nexus_creator").Debug("AddRoleToUser called",
		zap.String("action", nc.opConfig.Action),
		zap.String("role_name", nc.opConfig.RoleName),
		zap.String("username", nc.opConfig.LdapUsername))

	user, err := nc.nexus.GetUser(nc.opConfig.LdapUsername)
	if err != nil {
		return fmt.Errorf("add role to user '%s': get user failed: %w", nc.opConfig.LdapUsername, err)
	}
	if user == nil {
		return fmt.Errorf("user '%s' not found", nc.opConfig.LdapUsername)
	}

	currentRoles := user.Roles

	// Add target role if not present
	if !slices.Contains(currentRoles, nc.opConfig.RoleName) {
		currentRoles = append(currentRoles, nc.opConfig.RoleName)
	}
	// Add extra roles if not present
	for _, extraRole := range nc.opConfig.ExtraRoles {
		if extraRole != "" && !slices.Contains(currentRoles, extraRole) {
			currentRoles = append(currentRoles, extraRole)
		}
	}
	// Always enforce Base Roles
	for _, baseRole := range nc.opConfig.BaseRoles {
		if baseRole != "" && !slices.Contains(currentRoles, baseRole) {
			currentRoles = append(currentRoles, baseRole)
		}
	}

	user.Roles = currentRoles
	if err := nc.nexus.UpdateUser(user); err != nil {
		return fmt.Errorf("add role to user '%s': update user failed: %w", nc.opConfig.LdapUsername, err)
	}
	utils.WithComponent("nexus_creator").Info("Successfully updated user roles",
		zap.String("username", nc.opConfig.LdapUsername),
		zap.String("role_name", nc.opConfig.RoleName),
		zap.Int("extra_roles_count", len(nc.opConfig.ExtraRoles)))
	return nil
}

// CreationManager orchestrates the full creation workflow for repositories and roles.
type CreationManager struct {
	opConfig     *config.OperationConfig
	nexusCreator *NexusCreator
}

// NewCreationManager creates a new CreationManager instance.
func NewCreationManager(opConfig *config.OperationConfig, nexusClient client.NexusClient) *CreationManager {
	return &CreationManager{
		opConfig:     opConfig,
		nexusCreator: NewNexusCreator(opConfig, nexusClient),
	}
}

// Run executes the creation workflow: repository, privilege, role, and user assignment.
func (cm *CreationManager) Run() (map[string]interface{}, error) {
	utils.Logger.Debug("CreationManager.Run invoked",
		zap.String("repository_name", cm.opConfig.RepositoryName),
		zap.String("action", cm.opConfig.Action),
		zap.String("ldap_username", cm.opConfig.LdapUsername))

	if err := cm.nexusCreator.CreateRepository(); err != nil {
		return nil, err
	}
	if err := cm.nexusCreator.CreatePrivilege(); err != nil {
		return nil, err
	}
	if err := cm.nexusCreator.AddPrivilegeToRole(); err != nil {
		return nil, err
	}
	if err := cm.nexusCreator.AddRoleToUser(); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"action":          cm.opConfig.Action,
		"repository_name": cm.opConfig.RepositoryName,
		"ldap_username":   cm.opConfig.LdapUsername,
		"organization_id": cm.opConfig.OrganizationID,
	}, nil
}
