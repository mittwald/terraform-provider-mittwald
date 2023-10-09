package database

type CreateMySQLDatabaseRequestDatabaseCharacterSettings struct {
	CharacterSet string `json:"characterSet"`
	Collation    string `json:"collation"`
}

type CreateMySQLDatabaseRequestDatabase struct {
	Description       string                                              `json:"description"`
	Version           string                                              `json:"version"`
	CharacterSettings CreateMySQLDatabaseRequestDatabaseCharacterSettings `json:"characterSettings,omitempty"`
}

type CreateMySQLDatabaseRequestUser struct {
	Password    string `json:"password"`
	AccessLevel string `json:"accessLevel"`
}

type CreateMySQLDatabaseRequest struct {
	Database CreateMySQLDatabaseRequestDatabase `json:"database"`
	User     CreateMySQLDatabaseRequestUser     `json:"user"`
}

type CreateMySQLDatabaseResponse struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
}
