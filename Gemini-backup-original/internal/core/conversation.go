package core

type Conversation struct {
	ID            string
	ContextWindow *ContextWindow
}

func NewConversation(id string) *Conversation {
	return &Conversation{
		ID:            id,
		ContextWindow: NewContextWindow(),
	}
}

func (c *Conversation) Reset() {
	c.ContextWindow.Reset()
}
