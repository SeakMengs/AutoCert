package constant

type ProjectStatus int

const (
	ProjectStatusPreparing ProjectStatus = iota
	ProjectStatusProcessing
	ProjectStatusCompleted
)
