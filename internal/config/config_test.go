package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateOpConfig(t *testing.T) {
	// Setup Config
	cfg := Config{
		Orgs: map[string]string{
			"org1": "org-id-1",
		},
		PackageManagers: map[string]PackageManager{
			"npm": {DefaultURL: "https://registry.npmjs.org"},
		},
		ExtraRoles: []string{"extra-role"},
		BaseRoles:  []string{"base-role"},
	}

	tests := []struct {
		name        string
		req         RepositoryRequest
		action      string
		expected    *OperationConfig
		expectError bool
	}{
		{
			name: "Valid Request",
			req: RepositoryRequest{
				OrganizationName: "org1",
				PackageManager:   "npm",
				AppID:            "app1",
				LdapUsername:     "user1",
				Shared:           false,
			},
			action: "create",
			expected: &OperationConfig{
				Action:         "create",
				LdapUsername:   "user1",
				OrganizationID: "org-id-1",
				RemoteURL:      "https://registry.npmjs.org",
				ExtraRoles:     []string{"extra-role"},
				BaseRoles:      []string{"base-role"},
				RepositoryName: "npm-release-app1",
				PrivilegeName:  "npm-release-app1",
				RoleName:       "user1",
				PackageManager: "npm",
			},
			expectError: false,
		},
		{
			name: "Shared Request",
			req: RepositoryRequest{
				OrganizationName: "org1",
				PackageManager:   "npm",
				AppID:            "app1",
				LdapUsername:     "user1",
				Shared:           true,
			},
			action: "create",
			expected: &OperationConfig{
				Action:         "create",
				LdapUsername:   "user1",
				OrganizationID: "org-id-1",
				RemoteURL:      "https://registry.npmjs.org",
				ExtraRoles:     []string{"extra-role"},
				BaseRoles:      []string{"base-role"},
				RepositoryName: "npm-release-shared",
				PrivilegeName:  "npm-release-shared",
				RoleName:       "repositories.share",
				PackageManager: "npm",
			},
			expectError: false,
		},
		{
			name: "Invalid Organization",
			req: RepositoryRequest{
				OrganizationName: "invalid-org",
				PackageManager:   "npm",
			},
			action:      "create",
			expected:    nil,
			expectError: true,
		},
		{
			name: "Invalid Package Manager",
			req: RepositoryRequest{
				OrganizationName: "org1",
				PackageManager:   "invalid-pm",
			},
			action:      "create",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opConfig, err := cfg.CreateOpConfig(tt.req, tt.action)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, opConfig)
			}
		})
	}
}
