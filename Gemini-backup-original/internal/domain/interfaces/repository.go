package interfaces

import (
	"time"

	"gemini-cli/internal/domain/entities"
)

// ConversationRepository defines the interface for conversation persistence
type ConversationRepository interface {
	// Save saves a conversation
	Save(conversationID string, history []entities.Message) error

	// Load loads a conversation
	Load(conversationID string) ([]entities.Message, error)

	// Delete deletes a conversation
	Delete(conversationID string) error

	// List returns all conversation IDs
	List() ([]string, error)

	// Cleanup removes old conversations
	Cleanup(olderThan time.Duration) error
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Get retrieves a value from cache
	Get(key string) (interface{}, bool)

	// Set sets a value in cache
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(key string) error

	// Clear removes all values from cache
	Clear() error

	// Exists checks if a key exists
	Exists(key string) bool

	// GetStats returns cache statistics
	GetStats() CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits       int64 `json:"hits"`
	Misses     int64 `json:"misses"`
	Size       int   `json:"size"`
	MaxSize    int   `json:"max_size"`
}