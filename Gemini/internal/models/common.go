package models

// --- Common Shared Models ---
type Parameters struct {
	Name        string              `json:"-"`
	Description string              `json:"-"`
	Type        string              `json:"type"`
	Properties  map[string]Property `json:"properties"`
	Required    []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

