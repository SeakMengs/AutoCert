package constant

type ProjectStatus int

const (
	ProjectStatusPreparing ProjectStatus = iota
	ProjectStatusProcessing
	ProjectStatusCompleted
)

type SignatoryStatus int

const (
	SignatoryStatusNotInvited SignatoryStatus = iota
	SignatoryStatusInvited
	SignatoryStatusSigned
)
