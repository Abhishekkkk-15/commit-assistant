package main

import (
	"testing"
)

func TestParseCommitMessage(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
		wantDesc string
		wantErr  bool
	}{
		{"feat: add something", "feat", "add something", false},
		{"fix(ui): layout bug", "fix", "layout bug", false},
		{"feat!: breaking change", "feat", "breaking change", false},
		{"invalid message", "", "", false},
		{"", "", "", true},
	}

	for _, tt := range tests {
		msg, err := ParseCommitMessage(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCommitMessage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err == nil {
			if msg.Type != tt.wantType {
				t.Errorf("ParseCommitMessage(%q) Type = %q, want %q", tt.input, msg.Type, tt.wantType)
			}
			if msg.Description != tt.wantDesc {
				t.Errorf("ParseCommitMessage(%q) Description = %q, want %q", tt.input, msg.Description, tt.wantDesc)
			}
		}
	}
}

func TestLint(t *testing.T) {
	config := DefaultConfig()
	tests := []struct {
		message string
		isValid bool
	}{
		{"feat: valid message", true},
		{"fix(scope): valid with scope", true},
		{"invalid: message", false},
		{"feat: " + string(make([]byte, 100)), false}, // Subject too long (limit is 72 now)
	}

	for _, tt := range tests {
		result := Lint(tt.message, &config)
		if result.Valid != tt.isValid {
			t.Errorf("Lint(%q) Valid = %v, want %v. Errors: %v", tt.message, result.Valid, tt.isValid, result.Errors)
		}
	}
}

func TestFormatCommitMessage(t *testing.T) {
	input := "  feat: description  \n\nbody line 1  \n  body line 2  "
	want := "feat: description\n\nbody line 1\nbody line 2"
	got := FormatCommitMessage(input)
	if got != want {
		t.Errorf("FormatCommitMessage() = %q, want %q", got, want)
	}
}
