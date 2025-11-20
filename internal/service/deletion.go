package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	return nc.DeleteRepositoryByName(nc.opConfig.RepositoryName)
}

// DeleteRepositoryByName deletes a repository by its name.
func (nc *NexusCleaner) DeleteRepositoryByName(name string) error {
	utils.WithComponent("nexus_cleaner").Debug("Starting repository deletion",
		zap.String("action", nc.opConfig.Action),
		zap.String("repository_name", name),
		zap.String("username", nc.opConfig.LdapUsername))
	if err := nc.nexusClient.DeleteRepository(name); err != nil {
		return fmt.Errorf("delete repository '%s': %w", name, err)
	}
	utils.WithComponent("nexus_cleaner").Info("Successfully deleted proxy repository",
		zap.String("repository_name", name))
	return nil
}

// DeletePrivilege deletes the specified repository privilege.
func (nc *NexusCleaner) DeletePrivilege() error {
	return nc.DeletePrivilegeByName(nc.opConfig.PrivilegeName)
}

// DeletePrivilegeByName deletes a privilege by its name.
func (nc *NexusCleaner) DeletePrivilegeByName(name string) error {
	utils.WithComponent("nexus_cleaner").Debug("Starting privilege deletion",
		zap.String("action", nc.opConfig.Action),
		zap.String("privilege_name", name),
		zap.String("username", nc.opConfig.LdapUsername))
	if err := nc.nexusClient.DeletePrivilege(name); err != nil {
		return fmt.Errorf("delete privilege '%s': %w", name, err)
	}
	utils.WithComponent("nexus_cleaner").Info("Successfully deleted repository privilege",
		zap.String("privilege_name", name))
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

// ForceDeleteRole unconditionally deletes a role, ignoring 404 Not Found errors.
func (nc *NexusCleaner) ForceDeleteRole(roleName string) error {
	utils.WithComponent("nexus_cleaner").Debug("Force deleting role", zap.String("role_name", roleName))
	if err := nc.nexusClient.DeleteRole(roleName); err != nil {
		// If the role is not found (404), it is already deleted, so we treat it as success.
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			utils.WithComponent("nexus_cleaner").Debug("Role not found during force delete, ignoring",
				zap.String("role_name", roleName))
			return nil
		}
		return fmt.Errorf("force delete role '%s': %w", roleName, err)
	}
	return nil
}

// DisableUserAndResetRoles resets the user's roles to BaseRoles only and sets status to disabled.
func (nc *NexusCleaner) DisableUserAndResetRoles() error {
	utils.WithComponent("nexus_cleaner").Debug("Disabling user and resetting roles",
		zap.String("username", nc.opConfig.LdapUsername))

	user, err := nc.nexusClient.GetUser(nc.opConfig.LdapUsername)
	if err != nil {
		return fmt.Errorf("disable user '%s': get user failed: %w", nc.opConfig.LdapUsername, err)
	}
	if user == nil {
		return fmt.Errorf("user '%s' not found", nc.opConfig.LdapUsername)
	}

	// Set to BaseRoles only
	user.Roles = nc.opConfig.BaseRoles
	user.Status = "disabled"

	if err := nc.nexusClient.UpdateUser(user); err != nil {
		return fmt.Errorf("disable user '%s': update failed: %w", nc.opConfig.LdapUsername, err)
	}
	utils.WithComponent("nexus_cleaner").Info("User disabled and roles reset",
		zap.String("username", nc.opConfig.LdapUsername))
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
	// Special Offboarding Mode: Shared=true AND AppID is present (during delete)
	if dm.opConfig.Shared && dm.opConfig.AppID != "" {
		utils.WithComponent("deletion_manager").Info("Executing Offboarding Mode (Delete Shared+AppID)",
			zap.String("username", dm.opConfig.LdapUsername),
			zap.String("app_id", dm.opConfig.AppID))

		// Reset User: Keep only base roles, set status disabled
		if err := dm.nexusCleaner.DisableUserAndResetRoles(); err != nil {
			return nil, err
		}

		// Remove the Role named after the LDAP username
		if err := dm.nexusCleaner.ForceDeleteRole(dm.opConfig.LdapUsername); err != nil {
			// We log but continue, as the role might not exist
			utils.WithComponent("deletion_manager").Warn("Failed to delete user role during offboarding",
				zap.Error(err), zap.String("role", dm.opConfig.LdapUsername))
		}

		// Remove ALL repositories and privileges associated with this AppID.
		// We assume the naming convention *-release-[appID]
		// Fetch all repositories
		allRepos, err := dm.nexusClient.GetRepositories()
		if err != nil {
			return nil, fmt.Errorf("offboarding: failed to list repositories: %w", err)
		}

		suffix := fmt.Sprintf("-release-%s", dm.opConfig.AppID)

		// Filter and delete matching repositories
		for _, repo := range allRepos {
			if strings.HasSuffix(repo.Name, suffix) {
				if err := dm.nexusCleaner.DeleteRepositoryByName(repo.Name); err != nil {
					utils.WithComponent("deletion_manager").Warn("Failed to delete repository during offboarding",
						zap.String("repository", repo.Name), zap.Error(err))
				}
			}
		}

		// Fetch all privileges
		allPrivs, err := dm.nexusClient.GetPrivileges()
		if err != nil {
			return nil, fmt.Errorf("offboarding: failed to list privileges: %w", err)
		}

		// 4. Filter and delete matching privileges
		for _, priv := range allPrivs {
			if strings.HasSuffix(priv.Name, suffix) {
				if err := dm.nexusCleaner.DeletePrivilegeByName(priv.Name); err != nil {
					utils.WithComponent("deletion_manager").Warn("Failed to delete privilege during offboarding",
						zap.String("privilege", priv.Name), zap.Error(err))
				}
			}
		}

		return map[string]interface{}{
			"action":        dm.opConfig.Action,
			"mode":          "offboarding",
			"ldap_username": dm.opConfig.LdapUsername,
			"app_id":        dm.opConfig.AppID,
		}, nil
	}

	// Standard Deletion Logic
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
