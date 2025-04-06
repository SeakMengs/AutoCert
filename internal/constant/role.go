package constant

type ProjectRole int

const (
	ProjectRoleOwner ProjectRole = iota
	ProjectRoleSignatory
	ProjectRoleNone
)
