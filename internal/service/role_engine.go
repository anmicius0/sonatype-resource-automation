// internal/service/role_engine.go
package service

import (
	"slices"
)

// RoleDecisionEngine encapsulates the logic for determining final user roles.
type RoleDecisionEngine struct {
	baseRoles    []string
	extraRoles   []string
	afterRemoval []string
}

// NewRoleDecisionEngine creates a new role decision engine.
func NewRoleDecisionEngine(baseRoles []string, extraRoles []string) *RoleDecisionEngine {
	// Filter empty extra roles
	filteredExtra := make([]string, 0, len(extraRoles))
	for _, r := range extraRoles {
		if r != "" {
			filteredExtra = append(filteredExtra, r)
		}
	}

	// Filter empty base roles
	filteredBase := make([]string, 0, len(baseRoles))
	for _, r := range baseRoles {
		if r != "" {
			filteredBase = append(filteredBase, r)
		}
	}

	return &RoleDecisionEngine{
		baseRoles:  filteredBase,
		extraRoles: filteredExtra,
	}
}

// SetAfterRemovalRoles sets the list of roles after the target role is removed.
func (rde *RoleDecisionEngine) SetAfterRemovalRoles(roles []string) {
	rde.afterRemoval = roles
}

// DecideFinalRoles determines the final list of roles.
// Logic:
// 1. Base Roles are ALWAYS included.
// 2. Extra Roles are kept ONLY if the user has "Other Roles" (active project roles).
// 3. All other roles (project roles) are kept.
func (rde *RoleDecisionEngine) DecideFinalRoles() []string {
	keepExtra := rde.HasOtherRoles()

	// Use a map to prevent duplicates
	finalSet := make(map[string]struct{})
	finalRoles := make([]string, 0)

	// 1. Always add Base Roles
	for _, br := range rde.baseRoles {
		if _, exists := finalSet[br]; !exists {
			finalSet[br] = struct{}{}
			finalRoles = append(finalRoles, br)
		}
	}

	// 2. Process remaining roles
	for _, r := range rde.afterRemoval {
		// Skip if it's already added (e.g. it was a base role)
		if _, exists := finalSet[r]; exists {
			continue
		}

		// If it is an Extra Role, check if we should keep it
		if slices.Contains(rde.extraRoles, r) {
			if keepExtra {
				finalSet[r] = struct{}{}
				finalRoles = append(finalRoles, r)
			}
			continue
		}

		// It is a normal/project role, keep it
		finalSet[r] = struct{}{}
		finalRoles = append(finalRoles, r)
	}

	return finalRoles
}

// HasOtherRoles checks if there are roles other than BaseRoles, repositories.share, or ExtraRoles.
func (rde *RoleDecisionEngine) HasOtherRoles() bool {
	for _, r := range rde.afterRemoval {
		// Ignore if it is a Base Role
		if slices.Contains(rde.baseRoles, r) {
			continue
		}
		// Ignore if it is Shared role
		if r == "repositories.share" {
			continue
		}
		// Ignore if it is an Extra Role
		if slices.Contains(rde.extraRoles, r) {
			continue
		}
		// If we are here, it's a specific project role
		return true
	}
	return false
}

// GetRemovedExtraRoles returns the extra roles that were removed from the final list.
func (rde *RoleDecisionEngine) GetRemovedExtraRoles() []string {
	// (Implementation remains roughly the same logic, comparing extraRoles vs final result)
	removed := make([]string, 0)
	finalRoles := rde.DecideFinalRoles()
	for _, r := range rde.extraRoles {
		if !slices.Contains(finalRoles, r) {
			removed = append(removed, r)
		}
	}
	return removed
}
