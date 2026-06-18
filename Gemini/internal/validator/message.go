package validator

import (
	"errors"
	"strconv"
)

const (
	maxMessageLength = 1000
)

var (
	ErrMessageEmpty    = errors.New("message cannot be empty")
	ErrMessageTooLong  = errors.New("message exceeds maximum length of " + strconv.Itoa(maxMessageLength) + " characters")
	ErrMessageInvalid  = errors.New("message contains invalid characters")
	ErrMessageTooShort = errors.New("message must be at least 1 character")
)

// ValidateMessage validates a message before processing
// Returns nil if valid, error describing the issue otherwise
func ValidateMessage(message string) error {
	if message == "" {
		return ErrMessageEmpty
	}

	if len(message) > maxMessageLength {
		return ErrMessageTooLong
	}

	for _, r := range message {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return ErrMessageInvalid
		}
	}

	return nil
}