package apiext

type ErrNoDefaultStack struct {
	ProjectID string
}

func (e *ErrNoDefaultStack) Error() string {
	return "project " + e.ProjectID + " does not have a default stack"
}
