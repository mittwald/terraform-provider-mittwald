package database

type MySQLUser struct {
	ID             string `json:"id"`
	AccessLevel    string `json:"accessLevel"`
	DatabaseID     string `json:"databaseId"`
	Description    string `json:"description"`
	Disabled       bool   `json:"disabled"`
	ExternalAccess bool   `json:"externalAccess"`
	MainUser       bool   `json:"mainUser"`
	Name           string `json:"name"`
}
