package database

type MySQLDatabaseCharacterSettings struct {
	Collation    string `json:"collation"`
	CharacterSet string `json:"characterSet"`
}

type MySQLDatabase struct {
	ID                string                         `json:"id"`
	CharacterSettings MySQLDatabaseCharacterSettings `json:"characterSettings"`
	Description       string                         `json:"description"`
	Hostname          string                         `json:"hostname"`
	Name              string                         `json:"name"`
	Version           string                         `json:"version"`
	ProjectID         string                         `json:"projectId"`
}
