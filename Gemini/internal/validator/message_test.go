package validator

import (
	"testing"
)


func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected error
	}{
		// Valid messages
		{"empty string", "", true, ErrMessageEmpty},
		{"single char", "a", false, nil},
		{"normal message", "hello world", false, nil},
		{"long message", "This is a very long message that should be valid as long as it doesn't exceed the maximum length limit", false, nil},
		{"with punctuation", "Hello, world!", false, nil},
		{"with numbers", "Message 123", false, nil},
		{"with newlines", "Line 1\nLine 2", false, nil},
		{"with emojis", "Hello 😊", false, nil},
		{"with unicode", "Chào thế giới!", false, nil},

		// Invalid messages
		{"exceeds max length", "This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters. This is a very long message that exceeds the maximum length limit of one thousand characters.", true, ErrMessageTooLong},
		{"contains control chars", "Message\x00with null", true, ErrMessageInvalid},
		{"contains invalid chars", "Message\x01\x02\x03", true, ErrMessageInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateMessage(%q) = nil; want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateMessage(%q) = %v; want nil", tt.input, err)
			}
			if tt.wantErr && err != nil && err != tt.expected {
				t.Errorf("ValidateMessage(%q) = %v; want %v", tt.input, err, tt.expected)
			}
		})
	}
}