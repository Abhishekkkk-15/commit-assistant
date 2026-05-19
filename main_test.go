package main

import (
	"testing"
)

func TestIsVersionCommit(t *testing.T) {
	tests := []struct {
		message  string
		expected bool
	}{
		{"v1.0.0", true},
		{"v12.34.56", true},
		{"v0.1.0-alpha.1", true},
		{"v2.0.0-rc+build.123", true},
		{"v1", true},
		{"v1.2", true},
		{"1.0.0", false},
		{"v1.0.0-", false},
		{"feat: v1.0.0", false},
		{"validate release", false},
		{"version 1.0.0", false},
	}

	for _, tc := range tests {
		result := IsVersionCommit(tc.message)
		if result != tc.expected {
			t.Errorf("IsVersionCommit(%q) = %v; want %v", tc.message, result, tc.expected)
		}
	}
}

func TestLint(t *testing.T) {
	config := DefaultConfig()

	// Standard Conventional Commit should pass
	res1 := Lint("feat: add some feature", &config)
	if !res1.Valid {
		t.Errorf("Expected 'feat: add some feature' to be valid, got errors: %v", res1.Errors)
	}

	// Invalid conventional commit should fail
	res2 := Lint("invalid commit message", &config)
	if res2.Valid {
		t.Errorf("Expected 'invalid commit message' to be invalid")
	}

	// Version commit (v*) should pass even if it doesn't match Conventional Commit pattern
	res3 := Lint("v1.2.3", &config)
	if !res3.Valid {
		t.Errorf("Expected 'v1.2.3' version commit to be valid, got errors: %v", res3.Errors)
	}

	res4 := Lint("v0.1.0-alpha.1", &config)
	if !res4.Valid {
		t.Errorf("Expected 'v0.1.0-alpha.1' version commit to be valid, got errors: %v", res4.Errors)
	}
}
