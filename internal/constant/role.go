package constant

type ProjectRole int

const (
	ProjectRoleOwner ProjectRole = iota
	ProjectRoleSignatory
	ProjectRoleNone
)

type ProjectPermission string

const (
	// For builder
	AnnotateColumnAdd        ProjectPermission = "annotate:column:add"
	AnnotateColumnUpdate     ProjectPermission = "annotate:column:update"
	AnnotateColumnRemove     ProjectPermission = "annotate:column:remove"
	AnnotateSignatureAdd     ProjectPermission = "annotate:signature:add"
	AnnotateSignatureUpdate  ProjectPermission = "annotate:signature:update"
	AnnotateSignatureRemove  ProjectPermission = "annotate:signature:remove"
	AnnotateSignatureInvite  ProjectPermission = "annotate:signature:invite"
	AnnotateSignatureApprove ProjectPermission = "annotate:signature:approve"
	AnnotateSignatureReject  ProjectPermission = "annotate:signature:reject"
	SettingsUpdate           ProjectPermission = "settings:update"
	TableUpdate              ProjectPermission = "table:update"
)
