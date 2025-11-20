package client

// Repository represents a Nexus repository.
type Repository struct {
	Name       string         `json:"name"`
	Format     string         `json:"format"`
	Type       string         `json:"type"`
	Url        string         `json:"url"`
	Online     bool           `json:"online"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Privilege represents a Nexus privilege.
type Privilege struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Actions     []string `json:"actions"`
	Format      string   `json:"format,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Type        string   `json:"type,omitempty"`
}

// Role represents a Nexus role.
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Privileges  []string `json:"privileges"`
	Roles       []string `json:"roles"`
}

// User represents a Nexus user.
type User struct {
	UserID       string   `json:"userId"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	EmailAddress string   `json:"emailAddress"`
	Source       string   `json:"source"`
	Status       string   `json:"status"`
	Roles        []string `json:"roles"`
}

// IQRole represents a role in IQ Server.
type IQRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}
