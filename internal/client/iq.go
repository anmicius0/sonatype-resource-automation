package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
)

// iqServerClient handles API interactions with Sonatype IQ Server.
// It is intentionally unexported so callers use the IQClient interface.
type iqServerClient struct {
	*HTTPClient
}

// NewIQServerClient creates a new IQServerClient instance.
func NewIQServerClient(url, username, password string) IQClient {
	return &iqServerClient{
		HTTPClient: NewHTTPClient(url, username, password),
	}
}

// GetRoles fetches all roles from IQ Server, returning empty on 404.
func (c *iqServerClient) GetRoles() ([]IQRole, error) {
	response, err := c.DoReq("GET", "/api/v2/roles", nil, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return []IQRole{}, nil
		}
		return nil, fmt.Errorf("get IQ Server roles: %w", err)
	}
	var rolesResponse struct {
		Roles []IQRole
	}
	if err := json.Unmarshal(response.Bytes(), &rolesResponse); err != nil {
		return nil, fmt.Errorf("get IQ Server roles: failed to unmarshal response: %w", err)
	}
	return rolesResponse.Roles, nil
}

// FindOwnerRoleID searches for the "Owner" role ID among fetched roles.
func (c *iqServerClient) FindOwnerRoleID() (string, error) {
	roles, err := c.GetRoles()
	if err != nil {
		return "", fmt.Errorf("find owner role: get roles failed: %w", err)
	}
	for _, role := range roles {
		if role.Name == "Owner" {
			if role.ID != "" {
				return role.ID, nil
			}
			utils.Logger.Warn("'Owner' role found but id is empty")
			return "", fmt.Errorf("find owner role: 'Owner' role exists but id is missing")
		}
	}
	utils.Logger.Warn("'Owner' role not found in IQ Server")
	return "", nil
}

// AddOwnerRoleToUser adds the Owner role to the user in the organization.
func (c *iqServerClient) AddOwnerRoleToUser(opConfig *config.OperationConfig) error {
	utils.Logger.Debug("AddOwnerRoleToUser called",
		zap.String("ldap_username", opConfig.LdapUsername),
		zap.String("organization_id", opConfig.OrganizationID))

	roleID, err := c.FindOwnerRoleID()
	if err != nil {
		return fmt.Errorf("add owner role to user '%s' in organization '%s': %w", opConfig.LdapUsername, opConfig.OrganizationID, err)
	}
	if roleID == "" {
		return fmt.Errorf("add owner role to user '%s' in organization '%s': owner role id not found", opConfig.LdapUsername, opConfig.OrganizationID)
	}
	endpoint := fmt.Sprintf("/api/v2/roleMemberships/organization/%s/role/%s/user/%s", opConfig.OrganizationID, roleID, opConfig.LdapUsername)
	_, err = c.DoReq("PUT", endpoint, nil, nil)
	if err != nil {
		utils.Logger.Error("Failed adding owner role to user",
			zap.String("ldap_username", opConfig.LdapUsername),
			zap.String("organization_id", opConfig.OrganizationID),
			zap.Error(err))
		return fmt.Errorf("add owner role to user '%s' in organization '%s': %w", opConfig.LdapUsername, opConfig.OrganizationID, err)
	}
	utils.Logger.Debug("Successfully requested add owner role",
		zap.String("ldap_username", opConfig.LdapUsername),
		zap.String("organization_id", opConfig.OrganizationID),
		zap.String("role_id", roleID))
	return nil
}

// RemoveOwnerRoleFromUser removes the Owner role from the user in the organization, ignoring 404.
func (c *iqServerClient) RemoveOwnerRoleFromUser(opConfig *config.OperationConfig) error {
	roleID, err := c.FindOwnerRoleID()
	if err != nil {
		return fmt.Errorf("remove owner role from user '%s' in organization '%s': %w", opConfig.LdapUsername, opConfig.OrganizationID, err)
	}
	if roleID == "" {
		return fmt.Errorf("remove owner role from user '%s' in organization '%s': owner role id not found", opConfig.LdapUsername, opConfig.OrganizationID)
	}
	endpoint := fmt.Sprintf("/api/v2/roleMemberships/organization/%s/role/%s/user/%s", opConfig.OrganizationID, roleID, opConfig.LdapUsername)
	response, err := c.DoReq("DELETE", endpoint, nil, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil // Membership not found; already removed
		}
		return fmt.Errorf("remove owner role from user '%s' in organization '%s': %w", opConfig.LdapUsername, opConfig.OrganizationID, err)
	}
	if response.StatusCode() != http.StatusOK && response.StatusCode() != http.StatusNoContent && response.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("remove owner role from user '%s' in organization '%s': unexpected status code %d", opConfig.LdapUsername, opConfig.OrganizationID, response.StatusCode())
	}
	return nil
}
