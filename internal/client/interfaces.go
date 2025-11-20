package client

import "github.com/anmicius0/sonatype-resource-automation/internal/config"

// NexusClient defines the operations we perform against a Nexus repository manager.
// Use the concrete NewNexusClient to obtain an implementation that satisfies this
// interface.
type NexusClient interface {
	GetRepository(name string) (*Repository, error)
	CreateProxyRepository(config *config.OperationConfig) error
	DeleteRepository(name string) error
	GetPrivilege(name string) (*Privilege, error)
	CreatePrivilege(config *config.OperationConfig) error
	DeletePrivilege(name string) error
	GetRole(name string) (*Role, error)
	CreateRole(config *config.OperationConfig) error
	UpdateRole(role *Role) error
	DeleteRole(name string) error
	GetUser(username string) (*User, error)
	UpdateUser(user *User) error
}

// IQClient defines the operations we perform against an IQ Server instance.
// Use NewIQServerClient to create a real implementation.
type IQClient interface {
	GetRoles() ([]IQRole, error)
	FindOwnerRoleID() (string, error)
	AddOwnerRoleToUser(opConfig *config.OperationConfig) error
	RemoveOwnerRoleFromUser(opConfig *config.OperationConfig) error
}
