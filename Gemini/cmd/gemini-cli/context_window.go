package main

import "gemini-cli/internal/models"

type ContextWindow struct {
	History       []models.GeminiContent
	MemorySummary string
}

func NewContextWindow() *ContextWindow {
	return &ContextWindow{
		History:       []models.GeminiContent{},
		MemorySummary: "",
	}
}

func (cw *ContextWindow) AddMessage(msg models.GeminiContent) {
	cw.History = append(cw.History, msg)
}

func (cw *ContextWindow) Reset() {
	cw.History = []models.GeminiContent{}
	cw.MemorySummary = ""
}
