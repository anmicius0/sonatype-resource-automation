package service

import (
	"fmt"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

// NexusCleaner handles cleanup of Nexus resources like repositories, privileges, and roles.
type NexusCleaner struct {
	opConfig    *config.OperationConfig
	nexusClient client.NexusClient
}

// NewNexusCleaner creates a new NexusCleaner instance.
func NewNexusCleaner(opConfig *config.OperationConfig, nexusClient client.NexusClient) *NexusCleaner {
	return &NexusCleaner{opConfig: opConfig, nexusClient: nexusClient}
}

// DeleteRepository deletes the specified proxy repository.
func (nc *NexusCleaner) DeleteRepository() error {
	utils.WithComponent("nexus_cleaner").Debug("Starting repository deletion",
		zap.String("action", nc.opConfig.Action),
		zap.String("repository_name", nc.opConfig.RepositoryName),
		zap.String("username", nc.opConfig.LdapUsername))
	if err := nc.nexusClient.DeleteRepository(nc.opConfig.RepositoryName); err != nil {
		return fmt.Errorf("delete repository '%s' (package_manager='%s'): %w", nc.opConfig.RepositoryName, nc.opConfig.PackageManager, err)
	}
	utils.WithComponent("nexus_cleaner").Info("Successfully deleted proxy repository",
		zap.String("repository_name", nc.opConfig.RepositoryName),
		zap.String("package_manager", nc.opConfig.PackageManager),
		zap.String("remote_url", nc.opConfig.RemoteURL))
	return nil
}

// DeletePrivilege deletes the specified repository privilege.
func (nc *NexusCleaner) DeletePrivilege() error {
	utils.WithComponent("nexus_cleaner").Debug("Starting privilege deletion",
		zap.String("action", nc.opConfig.Action),
		zap.String("privilege_name", nc.opConfig.PrivilegeName),
		zap.String("username", nc.opConfig.LdapUsername))
	if err := nc.nexusClient.DeletePrivilege(nc.opConfig.PrivilegeName); err != nil {
		return fmt.Errorf("delete privilege '%s' for repository '%s': %w", nc.opConfig.PrivilegeName, nc.opConfig.RepositoryName, err)
	}
	utils.WithComponent("nexus_cleaner").Info("Successfully deleted repository privilege",
		zap.String("privilege_name", nc.opConfig.PrivilegeName),
		zap.String("repository_name", nc.opConfig.RepositoryName),
		zap.String("package_manager", nc.opConfig.PackageManager))
	return nil
}

// CleanupRole deletes the role if it has no privileges; otherwise skips.
func (nc *NexusCleaner) CleanupRole() error {
	utils.WithComponent("nexus_cleaner").Debug("Starting role cleanup",
		zap.String("action", nc.opConfig.Action),
		zap.String("role_name", nc.opConfig.RoleName),
		zap.String("username", nc.opConfig.LdapUsername))

	role, err := nc.nexusClient.GetRole(nc.opConfig.RoleName)
	if err != nil {
		return fmt.Errorf("cleanup role '%s': get role failed: %w", nc.opConfig.RoleName, err)
	}
	if role == nil {
		// Role not found; nothing to clean
		utils.WithComponent("nexus_cleaner").Debug("Role not found, nothing to cleanup",
			zap.String("role_name", nc.opConfig.RoleName))
		return nil
	}
	privileges := role.Privileges
	if len(privileges) == 0 {
		// Empty role; safe to delete
		if err := nc.nexusClient.DeleteRole(nc.opConfig.RoleName); err != nil {
			return fmt.Errorf("cleanup role '%s': delete empty role failed: %w", nc.opConfig.RoleName, err)
		}
		utils.WithComponent("nexus_cleaner").Info("Successfully deleted empty role",
			zap.String("role_name", nc.opConfig.RoleName),
			zap.String("privilege_name", nc.opConfig.PrivilegeName))
	} else {
		// Role has privileges; skip deletion to avoid breaking access
		utils.WithComponent("nexus_cleaner").Debug("Role has privileges, skipping deletion",
			zap.String("role_name", nc.opConfig.RoleName),
			zap.Int("privilege_count", len(privileges)))
	}
	return nil
}

// CleanupUserRoles removes the target role from the user, applying the new logic based on remaining role combinations.
func (nc *NexusCleaner) CleanupUserRoles() error {
	utils.WithComponent("nexus_cleaner").Debug("Starting user roles cleanup",
		zap.String("action", nc.opConfig.Action),
		zap.String("username", nc.opConfig.LdapUsername),
		zap.String("rolename", nc.opConfig.RoleName))

	user, err := nc.nexusClient.GetUser(nc.opConfig.LdapUsername)
	if err != nil {
		return fmt.Errorf("cleanup user roles for '%s': get user failed: %w", nc.opConfig.LdapUsername, err)
	}
	if user == nil {
		utils.WithComponent("nexus_cleaner").Warn("User not found, skipping role cleanup",
			zap.String("username", nc.opConfig.LdapUsername))
		return nil
	}

	roles := user.Roles

	// Remove target role only if the role itself is empty.
	// If the role still contains privileges (something still inside the role),
	// do not remove it from the user's roles because it's still providing access.
	if nc.opConfig.RoleName != "" {
		roleInfo, err := nc.nexusClient.GetRole(nc.opConfig.RoleName)
		if err != nil {
			return fmt.Errorf("cleanup user roles for '%s': get role '%s' failed: %w", nc.opConfig.LdapUsername, nc.opConfig.RoleName, err)
		}
		// If role not found or role has no privileges, it's safe to remove from user.
		canRemove := true
		if roleInfo != nil {
			if len(roleInfo.Privileges) > 0 {
				// Role still has privileges -> do not remove it from the user
				canRemove = false
			}
		}
		if canRemove {
			// Remove the role from the slice
			for i, r := range roles {
				if r == nc.opConfig.RoleName {
					roles = append(roles[:i], roles[i+1:]...)
					break
				}
			}
		} else {
			utils.WithComponent("nexus_cleaner").Debug("Role still contains privileges; keeping role on user",
				zap.String("username", nc.opConfig.LdapUsername),
				zap.String("role_name", nc.opConfig.RoleName))
		}
	}

	// Use RoleDecisionEngine to determine final roles
	roleEngine := NewRoleDecisionEngine(nc.opConfig.BaseRoles, nc.opConfig.ExtraRoles)
	roleEngine.SetAfterRemovalRoles(roles)
	finalRoles := roleEngine.DecideFinalRoles()

	// Log the decision
	if roleEngine.HasOtherRoles() {
		utils.WithComponent("nexus_cleaner").Debug("Other roles present, keeping all remaining roles",
			zap.String("username", nc.opConfig.LdapUsername))
	} else {
		removedExtra := roleEngine.GetRemovedExtraRoles()
		utils.WithComponent("nexus_cleaner").Info("No other roles, removed extra roles",
			zap.String("username", nc.opConfig.LdapUsername),
			zap.Strings("removed_extra_roles", removedExtra))
	}

	user.Roles = finalRoles
	if err := nc.nexusClient.UpdateUser(user); err != nil {
		return fmt.Errorf("cleanup user roles for '%s': update user failed: %w", nc.opConfig.LdapUsername, err)
	}

	utils.WithComponent("nexus_cleaner").Info("Successfully updated user roles after cleanup",
		zap.String("username", nc.opConfig.LdapUsername),
		zap.String("removedrole", nc.opConfig.RoleName))

	return nil
}

// DeletionManager orchestrates the full deletion workflow for repositories and roles.
type DeletionManager struct {
	opConfig     *config.OperationConfig
	nexusClient  client.NexusClient
	nexusCleaner *NexusCleaner
}

// NewDeletionManager creates a new DeletionManager instance.
func NewDeletionManager(opConfig *config.OperationConfig, nexusClient client.NexusClient) *DeletionManager {
	return &DeletionManager{
		opConfig:     opConfig,
		nexusClient:  nexusClient,
		nexusCleaner: NewNexusCleaner(opConfig, nexusClient),
	}
}

// Run executes the deletion workflow: conditional on shared role or full cleanup.
func (dm *DeletionManager) Run() (map[string]interface{}, error) {
	if dm.opConfig.RoleName == "repositories.share" {
		// Shared role: only cleanup user roles
		if err := dm.nexusCleaner.CleanupUserRoles(); err != nil {
			return nil, err
		}
	} else {
		// Full cleanup: repo, privilege, role, user
		if err := dm.nexusCleaner.DeleteRepository(); err != nil {
			return nil, err
		}
		if err := dm.nexusCleaner.DeletePrivilege(); err != nil {
			return nil, err
		}
		if err := dm.nexusCleaner.CleanupRole(); err != nil {
			return nil, err
		}
		if err := dm.nexusCleaner.CleanupUserRoles(); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"action":          dm.opConfig.Action,
		"repository_name": dm.opConfig.RepositoryName,
		"ldap_username":   dm.opConfig.LdapUsername,
		"organization_id": dm.opConfig.OrganizationID,
	}, nil
}
