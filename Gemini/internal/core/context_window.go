package core

import "gemini-cli/internal/models/messaging"

type ContextWindow struct {
	History       []messaging.Message
	MemorySummary string
}

func NewContextWindow() *ContextWindow {
	return &ContextWindow{
		History:       []messaging.Message{},
		MemorySummary: "",
	}
}

func (cw *ContextWindow) AddMessage(msg messaging.Message) {
	cw.History = append(cw.History, msg)
}

func (cw *ContextWindow) Reset() {
	cw.History = []messaging.Message{}
	cw.MemorySummary = ""
}
