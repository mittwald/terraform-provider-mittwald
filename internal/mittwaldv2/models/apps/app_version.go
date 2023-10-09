package apps

type AppVersion struct {
	ID              string `json:"id"`
	ExternalVersion string `json:"externalVersion"`
	InternalVersion string `json:"internalVersion"`
}
