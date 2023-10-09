package apps

type AppInstallationVersions struct {
	Desired string `json:"desired"`
	Current string `json:"current,omitempty"`
}

type AppInstallationLinkedDatabase struct {
	DatabaseID      string            `json:"databaseId"`
	DatabaseUserIDs map[string]string `json:"databaseUserIds"`
	Kind            string            `json:"kind"`
	Purpose         string            `json:"purpose"`
}

type AppInstallationSystemSoftware struct {
	SystemSoftwareID      string                  `json:"systemSoftwareId"`
	SystemSoftwareVersion AppInstallationVersions `json:"systemSoftwareVersion"`
	UpdatePolicy          string                  `json:"updatePolicy"`
}

type AppInstallationSavedUserInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AppInstallation struct {
	ID                 string                          `json:"id"`
	AppID              string                          `json:"appId"`
	AppVersion         AppInstallationVersions         `json:"appVersion"`
	CustomDocumentRoot string                          `json:"customDocumentRoot,omitempty"`
	Description        string                          `json:"description"`
	Disabled           bool                            `json:"disabled"`
	InstallationPath   string                          `json:"installationPath"`
	LinkedDatabases    []AppInstallationLinkedDatabase `json:"linkedDatabases,omitempty"`
	ProjectID          string                          `json:"projectId"`
	SystemSoftware     []AppInstallationSystemSoftware `json:"systemSoftware"`
	UpdatePolicy       string                          `json:"updatePolicy"`
	UserInputs         []AppInstallationSavedUserInput `json:"userInputs"`
}
