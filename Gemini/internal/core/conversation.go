package core

type Conversation struct {
	ID            string
	ContextWindow *ContextWindow
	UserInput     string
	HandoffPlan   *RoutePlan
}

func NewConversation(id string) *Conversation {
	return &Conversation{
		ID:            id,
		ContextWindow: NewContextWindow(),
	}
}

func (c *Conversation) Reset() {
	c.UserInput = ""
	c.HandoffPlan = nil
	c.ContextWindow.Reset()
}
