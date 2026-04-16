package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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

func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			defaultConfig := DefaultConfig()
			saveErr := SaveConfig(&defaultConfig)
			if saveErr != nil {
				return nil, saveErr
			}
			return &defaultConfig, nil
		}
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	return &config, err
}

func SaveConfig(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
	return nil
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".commit-assistant")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}
