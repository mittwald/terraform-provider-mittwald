package apps

type CreateAppInstallationRequest struct {
	AppVersionID string                          `json:"appVersionId"`
	Description  string                          `json:"description"`
	UpdatePolicy string                          `json:"updatePolicy"`
	UserInputs   []AppInstallationSavedUserInput `json:"userInputs"`
}

type CreateAppInstallationResponse struct {
	ID string `json:"id"`
}
