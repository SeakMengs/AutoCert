package util

import (
	"os"
	"slices"
	"strings"

	"github.com/SeakMengs/AutoCert/internal/constant"
)

var rolePermissions = map[constant.ProjectRole][]constant.ProjectPermission{
	constant.ProjectRoleOwner: {
		constant.AnnotateColumnAdd,
		constant.AnnotateColumnUpdate,
		constant.AnnotateColumnRemove,
		constant.AnnotateSignatureAdd,
		constant.AnnotateSignatureUpdate,
		constant.AnnotateSignatureRemove,
		constant.AnnotateSignatureInvite,
		constant.SettingsUpdate,
		constant.TableUpdate,
	},
	constant.ProjectRoleSignatory: {
		constant.AnnotateSignatureApprove,
		constant.AnnotateSignatureReject,
	},
	constant.ProjectRoleNone: {},
}

func HasFullAccess(email string) bool {
	domain := os.Getenv("FULL_ACCESS_EMAIL_DOMAIN")
	if domain == "" {
		return true // unrestricted
	}
	return strings.HasSuffix(email, domain)
}

func GetAllowedRoles(email string) []constant.ProjectRole {
	if HasFullAccess(email) {
		return []constant.ProjectRole{
			constant.ProjectRoleOwner,
			constant.ProjectRoleSignatory,
		}
	}
	return []constant.ProjectRole{
		constant.ProjectRoleSignatory,
	}
}

// checks if all permissions are granted by at least one of the roles.
func HasPermission(email string, roles []constant.ProjectRole, permissions []constant.ProjectPermission) bool {
	if len(roles) == 0 || len(permissions) == 0 {
		return false
	}

	filteredRoles := FilterRolesByEmailAccess(email, roles)
	if len(filteredRoles) == 0 {
		return false
	}

	for _, permission := range permissions {
		hasPermission := false
		for _, role := range filteredRoles {
			if slices.Contains(rolePermissions[role], permission) {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return false
		}
	}
	return true
}

func HasRole(email string, roles []constant.ProjectRole, requiredRoles []constant.ProjectRole) bool {
	filteredRoles := FilterRolesByEmailAccess(email, roles)
	if len(filteredRoles) == 0 {
		return false
	}

	for _, role := range requiredRoles {
		if slices.Contains(filteredRoles, role) {
			return true
		}
	}
	return false
}

// FilterRolesByEmailAccess filters out roles not permitted for the given email domain.
//
// Example:
//
//	os.Setenv("FULL_ACCESS_EMAIL_DOMAIN", "@paragoniu.edu.kh")
//
//	email := "user@gmail.com"
//	roles := []constant.ProjectRole{ProjectRoleOwner, ProjectRoleSignatory}
//
//	result := FilterRolesByEmailAccess(email, roles)
//	result: []constant.ProjectRole{ProjectRoleSignatory}
//
//	email2 := "admin@paragoniu.edu.kh"
//	result2 := FilterRolesByEmailAccess(email2, roles)
//	result2: []constant.ProjectRole{ProjectRoleOwner, ProjectRoleSignatory}
func FilterRolesByEmailAccess(email string, roles []constant.ProjectRole) []constant.ProjectRole {
	allowed := GetAllowedRoles(email)

	allowedSet := make(map[constant.ProjectRole]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	filtered := make([]constant.ProjectRole, 0, len(roles))
	for _, r := range roles {
		if _, ok := allowedSet[r]; ok {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// Return restricted bool and full access of accepted email domain if any.
func IsRestrictedByEmailDomain(email string, roles []constant.ProjectRole) (bool, string) {
	if HasFullAccess(email) {
		return false, ""
	}

	if len(roles) > 0 && len(FilterRolesByEmailAccess(email, roles)) == 0 {
		return true, os.Getenv("FULL_ACCESS_EMAIL_DOMAIN")
	}
	return false, ""
}
