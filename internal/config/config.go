// internal/config/config.go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

var validate = validator.New()

// Config holds the application's configuration, loaded from .env and JSON files.
type Config struct {
	NexusURL         string `validate:"required,url"`
	NexusUsername    string `validate:"required"`
	NexusPassword    string `validate:"required"`
	BaseRoles        []string
	ExtraRoles       []string
	IQServerURL      string `validate:"required,url"`
	IQServerUsername string `validate:"required"`
	IQServerPassword string `validate:"required"`
	APIHost          string `validate:"required"`
	Port             int    `validate:"required,min=1,max=65535"`
	APIToken         string `validate:"required"`
	Orgs             map[string]string
	PackageManagers  map[string]PackageManager `validate:"required,dive"`
}

func parseRoles(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))
	for _, raw := range parts {
		if role := strings.TrimSpace(raw); role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}

// Load loads and validates the full application configuration.
func Load() (*Config, error) {
	// Load .env configuration
	v := viper.New()
	v.SetConfigFile("config/.env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetDefault("API_HOST", "127.0.0.1")
	v.SetDefault("PORT", 5000)

	if err := v.ReadInConfig(); err != nil {
		var cfgErr viper.ConfigFileNotFoundError
		if !errors.As(err, &cfgErr) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	appConfig := &Config{
		NexusURL:         v.GetString("NEXUS_URL"),
		NexusUsername:    v.GetString("NEXUS_USERNAME"),
		NexusPassword:    v.GetString("NEXUS_PASSWORD"),
		IQServerURL:      v.GetString("IQSERVER_URL"),
		IQServerUsername: v.GetString("IQSERVER_USERNAME"),
		IQServerPassword: v.GetString("IQSERVER_PASSWORD"),
		APIHost:          v.GetString("API_HOST"),
		Port:             v.GetInt("PORT"),
		APIToken:         v.GetString("API_TOKEN"),
	}

	extraRole := v.GetString("EXTRA_ROLE")
	appConfig.ExtraRoles = parseRoles(extraRole)

	// Parse Base Roles
	baseRoleStr := v.GetString("BASE_ROLE")
	appConfig.BaseRoles = parseRoles(baseRoleStr)

	// Validate: Manually check if at least one base role exists if it is required
	if len(appConfig.BaseRoles) == 0 {
		return nil, fmt.Errorf("BASE_ROLE cannot be empty")
	}

	// Load organizations.json
	file, err := os.Open("config/organizations.json")
	if err != nil {
		return nil, fmt.Errorf("open organizations.json: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&appConfig.Orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	// Load packageManager.json
	file, err = os.Open("config/packageManager.json")
	if err != nil {
		return nil, fmt.Errorf("open packageManager.json: %w", err)
	}
	defer file.Close()
	decoder = json.NewDecoder(file)
	if err := decoder.Decode(&appConfig.PackageManagers); err != nil {
		return nil, fmt.Errorf("failed to decode packageManager.json: %w", err)
	}

	// Validate everything together
	if err := validate.Struct(appConfig); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return appConfig, nil
}

// CreateOpConfig creates an OperationConfig from a validated repository request and action.
func (c Config) CreateOpConfig(r RepositoryRequest, action string) (*OperationConfig, error) {
	// Get Organization ID
	orgID, ok := c.Orgs[r.OrganizationName]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", r.OrganizationName)
	}

	// Get Package Manager remote URL
	manager, ok := c.PackageManagers[r.PackageManager]
	if !ok {
		return nil, fmt.Errorf("package manager '%s' not found", r.PackageManager)
	}
	remoteURL := manager.DefaultURL

	// Generate Repository Name
	suffix := r.AppID
	if r.Shared {
		suffix = "shared"
	}
	repoName := fmt.Sprintf("%s-release-%s", strings.ToLower(r.PackageManager), suffix)

	// Determine Role Name
	roleName := r.LdapUsername
	if r.Shared {
		roleName = "repositories.share"
	}

	return &OperationConfig{
		Action:         action,
		LdapUsername:   r.LdapUsername,
		OrganizationID: orgID,
		RemoteURL:      remoteURL,
		ExtraRoles:     c.ExtraRoles,
		BaseRoles:      c.BaseRoles,
		RepositoryName: repoName,
		PrivilegeName:  repoName,
		RoleName:       roleName,
		PackageManager: r.PackageManager,
	}, nil
}
