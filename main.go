package main

type Config struct {
	GroqAPIKey     string `json:"groq_api_key"`
	Model          string `json:"model"`
	MaxSubjectLen  int    `json:"max_subject_length"`
	MaxBodyLineLen int    `json:"max_body_line_length"`
	StrictMode     bool   `josn:"strict_mode"`
}

type CommitMessage struct {
	Subject     string
	Body        string
	Type        string
	Description string
	Scope       string
	IsBreaking  string
}
type LintResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

func DefaultConfig() Config {
	return Config{
		Model:          "mixtral-8x7b-32768",
		MaxSubjectLen:  72,
		MaxBodyLineLen: 80,
		StrictMode:     false,
	}
}
