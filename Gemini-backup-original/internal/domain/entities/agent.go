package entities

import (
	"time"
)

// Agent represents a specialized AI agent with specific capabilities
type Agent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Capabilities []string          `json:"capabilities"`
	Config       map[string]any    `json:"config"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// NewAgent creates a new agent instance
func NewAgent(id, name, description string, capabilities []string) *Agent {
	now := time.Now()
	return &Agent{
		ID:           id,
		Name:         name,
		Description:  description,
		Capabilities: capabilities,
		Config:       make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdateConfig updates agent configuration
func (a *Agent) UpdateConfig(config map[string]interface{}) {
	a.Config = config
	a.UpdatedAt = time.Now()
}

// AddCapability adds a new capability to the agent
func (a *Agent) AddCapability(capability string) {
	for _, cap := range a.Capabilities {
		if cap == capability {
			return
		}
	}
	a.Capabilities = append(a.Capabilities, capability)
	a.UpdatedAt = time.Now()
}

// HasCapability checks if agent has a specific capability
func (a *Agent) HasCapability(capability string) bool {
	for _, cap := range a.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}