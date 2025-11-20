// internal/config/models.go
// Package config provides configuration loading, validation, and data models.
package config

// OperationConfig holds configuration for a single repository creation or deletion operation.
type OperationConfig struct {
	// Action is either "create" or "delete"
	Action string
	// LdapUsername is the LDAP user who will receive roles on the repository
	LdapUsername string
	// OrganizationID is the IQ Server organization identifier
	OrganizationID string
	// RemoteURL is the remote repository URL for proxy repositories
	RemoteURL string
	// ExtraRoles are additional roles to preserve or add during operations
	ExtraRoles []string
	// BaseRoles are the base roles to preserve during operations
	BaseRoles []string
	// RepositoryName is the generated or specified repository name
	RepositoryName string
	// PrivilegeName is the privilege name matching the repository
	PrivilegeName string
	// RoleName is the role name for privilege assignment
	RoleName string
	// PackageManager is the package manager type (e.g., "npm", "maven", "docker")
	PackageManager string
	// Shared indicates if the operation is for a shared resource
	Shared bool
	// AppID is the application identifier (if applicable)
	AppID string
}

// RepositoryRequest represents a single repository operation request from the API.
type RepositoryRequest struct {
	// OrganizationName is the IQ Server organization to associate with the repository
	OrganizationName string `binding:"required"`
	// LdapUsername is the LDAP user who will receive roles on the repository
	LdapUsername string `binding:"required"`
	// PackageManager specifies the repository format (e.g., npm, maven2, docker)
	// Note: Required for Create operations, and Delete operations where Shared=false.
	// Optional for Delete operations where Shared=true.
	PackageManager string
	// Shared indicates whether this repository is shared across applications
	Shared bool
	// AppID is the application identifier for non-shared repositories; must be empty for shared repositories
	AppID string
}

// FailedRequest represents a request that failed during processing along with the error reason.
type FailedRequest struct {
	// Request is the original repository request that failed
	Request RepositoryRequest
	// Reason is the error message describing why the request failed
	Reason string
}

type PackageManager struct {
	DefaultURL      string `validate:"required,url"`
	DefaultConfig   map[string]any
	PrivilegeFormat string
	APIEndpoint     *APIEndpoint `validate:"required"`
	// Future proofing for additional fields
	ExtraFields map[string]any `json:"-"`
}

type APIEndpoint struct {
	Path                 string `validate:"required"`
	FormatSpecificConfig map[string]any
}
