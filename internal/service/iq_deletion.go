package service

import (
	"fmt"
	"slices"

	"github.com/anmicius0/sonatype-resource-automation/internal/client"
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

// IQServerCleaner handles revocation of Owner role in IQ Server organizations.
type IQServerCleaner struct {
	opConfig    *config.OperationConfig
	iqClient    client.IQClient
	nexusClient client.NexusClient
}

// NewIQServerCleaner creates a new IQServerCleaner instance.
func NewIQServerCleaner(opConfig *config.OperationConfig, iqClient client.IQClient, nexusClient client.NexusClient) *IQServerCleaner {
	return &IQServerCleaner{opConfig: opConfig, iqClient: iqClient, nexusClient: nexusClient}
}

// CleanupUserFromOrganization removes the Owner role from the user in the organization.
func (ic IQServerCleaner) CleanupUserFromOrganization() error {
	if ic.opConfig.Shared && ic.opConfig.AppID != "" {
		utils.WithComponent("iq_cleaner").Debug("Offboarding mode detected; including IQ Server owner cleanup",
			zap.String("username", ic.opConfig.LdapUsername),
			zap.String("app_id", ic.opConfig.AppID))
	}

	utils.WithComponent("iq_cleaner").Debug("Starting IQ Server user cleanup from organization",
		zap.String("action", ic.opConfig.Action),
		zap.String("username", ic.opConfig.LdapUsername),
		zap.String("organization_id", ic.opConfig.OrganizationID))
	if ic.opConfig.OrganizationID == "" {
		utils.WithComponent("iq_cleaner").Debug("No organization_id; skipping IQ Server cleanup",
			zap.String("username", ic.opConfig.LdapUsername))
		return nil
	}
	removeOwner, err := ic.shouldRemoveOwnerRole()
	if err != nil {
		return err
	}
	if !removeOwner {
		utils.WithComponent("iq_cleaner").Debug("Skipping IQ Server Owner role removal (conditions not met)",
			zap.String("username", ic.opConfig.LdapUsername))
		return nil
	}
	if err := ic.iqClient.RemoveOwnerRoleFromUser(ic.opConfig); err != nil {
		return fmt.Errorf("remove owner role: %w", err)
	}
	utils.WithComponent("iq_cleaner").Info("Successfully removed Owner role from user in IQ Server organization",
		zap.String("username", ic.opConfig.LdapUsername),
		zap.String("organization_id", ic.opConfig.OrganizationID))
	return nil
}

func (ic IQServerCleaner) shouldRemoveOwnerRole() (bool, error) {
	if ic.nexusClient == nil {
		return false, fmt.Errorf("evaluate owner role removal: nexus client not configured")
	}
	user, err := ic.nexusClient.GetUser(ic.opConfig.LdapUsername)
	if err != nil {
		return false, fmt.Errorf("evaluate owner role removal: get user '%s' failed: %w", ic.opConfig.LdapUsername, err)
	}
	if user == nil {
		utils.WithComponent("iq_cleaner").Debug("User not found while evaluating IQ Server owner removal",
			zap.String("username", ic.opConfig.LdapUsername))
		return false, nil
	}
	roles := user.Roles
	if ic.opConfig.RoleName != "" {
		for i, r := range roles {
			if r == ic.opConfig.RoleName {
				roles = append(roles[:i], roles[i+1:]...)
				break
			}
		}
	}
	roleEngine := NewRoleDecisionEngine(ic.opConfig.BaseRoles, ic.opConfig.ExtraRoles)
	roleEngine.SetAfterRemovalRoles(roles)
	hasOtherRoles := roleEngine.HasOtherRoles()
	shareRoleAssigned := slices.Contains(roles, "repositories.share")
	shareRoleEmpty := true
	if shareRoleAssigned {
		shareRole, err := ic.nexusClient.GetRole("repositories.share")
		if err != nil {
			return false, fmt.Errorf("evaluate owner role removal: get repositories.share role failed: %w", err)
		}
		if shareRole != nil {
			if len(shareRole.Privileges) > 0 {
				shareRoleEmpty = false
			}
		}
	}
	// Check if the remaining roles are EXACTLY the base roles (no more, no less)
	// or a subset of base roles (e.g., user has 1 of 2 base roles).
	// Logic: If every role the user has is found in BaseRoles, then they "only" have BaseRoles.
	onlyBaseRole := len(ic.opConfig.BaseRoles) > 0 && len(roles) > 0
	if onlyBaseRole {
		for _, r := range roles {
			if !slices.Contains(ic.opConfig.BaseRoles, r) {
				onlyBaseRole = false
				break
			}
		}
	}
	shouldRemove := !hasOtherRoles && shareRoleEmpty && onlyBaseRole
	utils.WithComponent("iq_cleaner").Debug("IQ Server owner removal decision",
		zap.String("username", ic.opConfig.LdapUsername),
		zap.Bool("has_other_roles", hasOtherRoles),
		zap.Bool("share_role_assigned", shareRoleAssigned),
		zap.Bool("share_role_empty", shareRoleEmpty),
		zap.Bool("only_base_role", onlyBaseRole),
		zap.Bool("should_remove_owner", shouldRemove),
		zap.Strings("roles", roles))
	return shouldRemove, nil
}

// DeletionManager orchestrates IQ Server cleanup.
type IQDeletionManager struct {
	opConfig *config.OperationConfig
	cleaner  *IQServerCleaner
}

// NewIQDeletionManager creates a configured IQ Server deletion manager.
func NewIQDeletionManager(opConfig *config.OperationConfig, iqClient client.IQClient, nexusClient client.NexusClient) *IQDeletionManager {
	return &IQDeletionManager{
		opConfig: opConfig,
		cleaner:  NewIQServerCleaner(opConfig, iqClient, nexusClient),
	}
}

// Run executes the cleanup workflow.
func (dm IQDeletionManager) Run() (map[string]interface{}, error) {
	if err := dm.cleaner.CleanupUserFromOrganization(); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"action":          dm.opConfig.Action,
		"ldap_username":   dm.opConfig.LdapUsername,
		"organization_id": dm.opConfig.OrganizationID,
	}, nil
}
