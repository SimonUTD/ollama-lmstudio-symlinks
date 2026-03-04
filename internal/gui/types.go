package gui

type apiError struct {
	Error string `json:"error"`
}

type statusResponse struct {
	Binary binaryStatus `json:"binary"`
	Server serverStatus `json:"server"`
}

type binaryStatus struct {
	Found         bool   `json:"found"`
	Path          string `json:"path,omitempty"`
	Source        string `json:"source,omitempty"`
	VersionOutput string `json:"versionOutput,omitempty"`
}

type serverStatus struct {
	Host      string `json:"host,omitempty"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
}

type scanRequest struct {
	Direction string `json:"direction"`
}

type scanResponse struct {
	Items []scanItem `json:"items"`
}

type scanItem struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	Status     string `json:"status"`
	Selectable bool   `json:"selectable"`
	Selected   bool   `json:"selected"`
	Message    string `json:"message,omitempty"`

	GGUFPath  string `json:"ggufPath,omitempty"`
	ModelName string `json:"modelName,omitempty"`
}

type applyRequest struct {
	Direction string     `json:"direction"`
	Selected  []string   `json:"selected"`
	Imports   []lmImport `json:"imports"`
}

type applyResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type lmImport struct {
	GGUFPath  string `json:"ggufPath"`
	ModelName string `json:"modelName"`
}
