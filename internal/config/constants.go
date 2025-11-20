// Path: internal/config/constants.go
package config

import "time"

const (
	// Server configuration defaults
	DefaultReadTimeout     = 15 * time.Second
	DefaultWriteTimeout    = 15 * time.Second
	DefaultIdleTimeout     = 60 * time.Second
	DefaultShutdownTimeout = 5 * time.Second
)
