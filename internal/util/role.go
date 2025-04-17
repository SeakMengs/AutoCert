package util

import (
	"slices"

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
	},
	constant.ProjectRoleNone: {},
}

// checks if all permissions are granted by at least one of the roles.
func HasPermission(roles []constant.ProjectRole, permissions []constant.ProjectPermission) bool {
	for _, permission := range permissions {
		allowed := false
		for _, role := range roles {
			if slices.Contains(rolePermissions[role], permission) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}
