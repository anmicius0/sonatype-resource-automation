package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
)

// nexusClient is an unexported concrete implementation of NexusClient.
type nexusClient struct {
	*HTTPClient
	supportedFormats map[string]config.PackageManager
}

// NewNexusClient creates a configured NexusClient implementation for the provided
// Nexus base URL and credentials. It accepts a map of supported package format
// configurations used when creating proxy repositories.
//
// The concrete returned type is unexported; callers work with the NexusClient
// interface.
func NewNexusClient(url, username, password string, supportedFormats map[string]config.PackageManager) NexusClient {
	return &nexusClient{
		HTTPClient:       NewHTTPClient(url, username, password),
		supportedFormats: supportedFormats,
	}
}

func (c *nexusClient) GetRepository(name string) (*Repository, error) {
	resp, err := c.DoReq("GET", fmt.Sprintf("/v1/repositories/%s", name), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get repository '%s': %w", name, err)
	}
	var repo Repository
	if err := json.Unmarshal(resp.Bytes(), &repo); err != nil {
		return nil, fmt.Errorf("get repository '%s': failed to unmarshal response: %w", name, err)
	}
	return &repo, nil
}

func (c *nexusClient) GetRepositories() ([]Repository, error) {
	resp, err := c.DoReq("GET", "/v1/repositories", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get repositories: %w", err)
	}
	var repos []Repository
	if err := json.Unmarshal(resp.Bytes(), &repos); err != nil {
		return nil, fmt.Errorf("get repositories: failed to unmarshal response: %w", err)
	}
	return repos, nil
}

func (c *nexusClient) CreateProxyRepository(config *config.OperationConfig) error {
	manager, ok := c.supportedFormats[strings.ToLower(config.PackageManager)]
	if !ok {
		return fmt.Errorf("create proxy repository '%s': unsupported package manager format '%s'", config.RepositoryName, config.PackageManager)
	}
	apiEndpointIface := manager.APIEndpoint
	path := manager.APIEndpoint.Path

	repoConfig := map[string]any{
		"name":   config.RepositoryName,
		"online": true,
		"storage": map[string]any{
			"blobStoreName":               "default",
			"strictContentTypeValidation": true,
		},
		"proxy": map[string]any{
			"remoteUrl":      config.RemoteURL,
			"contentMaxAge":  1440,
			"metadataMaxAge": 1440,
		},
		"negativeCache": map[string]any{
			"enabled":    true,
			"timeToLive": 1440,
		},
		"httpClient": map[string]any{
			"blocked":   false,
			"autoBlock": true,
		},
	}

	formatSpecific := apiEndpointIface.FormatSpecificConfig
	for k, v := range formatSpecific {
		repoConfig[k] = v
	}

	defaults := manager.DefaultConfig
	for k, v := range defaults {
		repoConfig[k] = v
	}

	_, err := c.DoReq("POST", path, repoConfig, nil)
	if err != nil {
		return fmt.Errorf("create proxy repository '%s' at endpoint '%s': %w", config.RepositoryName, path, err)
	}
	return nil
}

func (c *nexusClient) DeleteRepository(name string) error {
	resp, err := c.DoReq("DELETE", fmt.Sprintf("/v1/repositories/%s", name), nil, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode())
	}
	return nil
}

func (c *nexusClient) GetPrivilege(name string) (*Privilege, error) {
	resp, err := c.DoReq("GET", fmt.Sprintf("/v1/security/privileges/%s", name), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get privilege '%s': %w", name, err)
	}
	var priv Privilege
	if err := json.Unmarshal(resp.Bytes(), &priv); err != nil {
		return nil, fmt.Errorf("get privilege '%s': failed to unmarshal response: %w", name, err)
	}
	return &priv, nil
}

func (c *nexusClient) GetPrivileges() ([]Privilege, error) {
	resp, err := c.DoReq("GET", "/v1/security/privileges", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get privileges: %w", err)
	}
	var privs []Privilege
	if err := json.Unmarshal(resp.Bytes(), &privs); err != nil {
		return nil, fmt.Errorf("get privileges: failed to unmarshal response: %w", err)
	}
	return privs, nil
}

func (c *nexusClient) CreatePrivilege(config *config.OperationConfig) error {
	pmLower := strings.ToLower(config.PackageManager)
	privFormat := pmLower

	// We call it "maven" in the API but Nexus expects "maven2"
	if privFormat == "maven" {
		privFormat = "maven2"
	}

	privConfig := map[string]interface{}{
		"name":        config.PrivilegeName,
		"description": fmt.Sprintf("All permissions for repository '%s'", config.RepositoryName),
		"actions":     []string{"BROWSE", "READ", "EDIT", "ADD", "DELETE"},
		"format":      privFormat,
		"repository":  config.RepositoryName,
	}
	_, err := c.DoReq("POST", "/v1/security/privileges/repository-view", privConfig, nil)
	if err != nil {
		return fmt.Errorf("create privilege '%s' for repository '%s' (format='%s'): %w", config.PrivilegeName, config.RepositoryName, privFormat, err)
	}
	return nil
}

func (c *nexusClient) DeletePrivilege(name string) error {
	resp, err := c.DoReq("DELETE", fmt.Sprintf("/v1/security/privileges/%s", name), nil, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode())
	}
	return nil
}

func (c *nexusClient) GetRole(name string) (*Role, error) {
	resp, err := c.DoReq("GET", fmt.Sprintf("/v1/security/roles/%s", name), nil, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get role '%s': %w", name, err)
	}
	var role Role
	if err := json.Unmarshal(resp.Bytes(), &role); err != nil {
		return nil, fmt.Errorf("get role '%s': failed to unmarshal response: %w", name, err)
	}
	return &role, nil
}

func (c *nexusClient) CreateRole(config *config.OperationConfig) error {
	roleConfig := map[string]interface{}{
		"id":          config.RoleName,
		"name":        config.RoleName,
		"description": fmt.Sprintf("Role for %s", config.LdapUsername),
		"privileges":  []string{config.PrivilegeName},
		"roles":       []string{},
	}
	_, err := c.DoReq("POST", "/v1/security/roles", roleConfig, nil)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest {
			// 400 error likely means role already exists (duplicate), which is acceptable
			return nil
		}
		return fmt.Errorf("create role '%s' for user '%s': %w", config.RoleName, config.LdapUsername, err)
	}
	return nil
}

func (c *nexusClient) UpdateRole(role *Role) error {
	if role.ID == "" {
		return fmt.Errorf("update role: role id is empty")
	}
	_, err := c.DoReq("PUT", fmt.Sprintf("/v1/security/roles/%s", role.ID), role, nil)
	if err != nil {
		return fmt.Errorf("update role '%s': %w", role.ID, err)
	}
	return nil
}

func (c *nexusClient) DeleteRole(name string) error {
	resp, err := c.DoReq("DELETE", fmt.Sprintf("/v1/security/roles/%s", name), nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode())
	}
	return nil
}

func (c *nexusClient) GetUser(userID string) (*User, error) {
	params := map[string]string{"userId": userID}
	resp, err := c.DoReq("GET", "/v1/security/users", nil, params)
	if err != nil {
		return nil, fmt.Errorf("get user '%s': %w", userID, err)
	}
	var users []User
	if err := json.Unmarshal(resp.Bytes(), &users); err != nil {
		return nil, fmt.Errorf("get user '%s': failed to unmarshal response: %w", userID, err)
	}
	for _, u := range users {
		if u.UserID == userID {
			return &u, nil
		}
	}
	return nil, nil
}

func (c *nexusClient) UpdateUser(user *User) error {
	if user.UserID == "" {
		return fmt.Errorf("update user: userId is empty")
	}
	// always set these values
	user.EmailAddress = "useless@example.com"
	user.LastName = "useless"
	_, err := c.DoReq("PUT", fmt.Sprintf("/v1/security/users/%s", user.UserID), user, nil)
	if err != nil {
		return fmt.Errorf("update user '%s': %w", user.UserID, err)
	}
	return nil
}
