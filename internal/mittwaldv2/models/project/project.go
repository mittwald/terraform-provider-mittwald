package project

type Project struct {
	ID          string            `json:"id"`
	ShortID     string            `json:"shortId"`
	CustomerID  string            `json:"customerId"`
	Description string            `json:"description"`
	ServerID    string            `json:"serverId,omitempty"`
	Directories map[string]string `json:"directories"`
	Spec        map[string]string `json:"spec"`
}
