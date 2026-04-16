package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	IsBreaking  bool
}
type LintResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

func DefaultConfig() Config {
	return Config{
		Model:          "openai/gpt-oss-120b",
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

func ParseCommitMessage(raw string) (*CommitMessage, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty commit message")
	}
	subject := strings.TrimSpace(lines[0])
	body := ""
	if len(lines) > 1 {
		body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	msg := &CommitMessage{
		Subject: subject,
		Body:    body,
	}
	re := regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!)?:\s+(.+)`)
	matches := re.FindStringSubmatch(subject)
	if len(matches) > 0 {
		msg.Type = matches[1]
		if len(matches) > 2 {
			msg.Scope = matches[2]
		}
		if len(matches) > 3 && matches[3] == "!" {
			msg.IsBreaking = true
		}
		if len(matches) > 4 {
			msg.Description = matches[4]
		}
	}
	return msg, nil
}

func Lint(message string, config *Config) LintResult {
	result := LintResult{Valid: true, Errors: []string{}, Warnings: []string{}}
	if message == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Commit message cannot be empty")
		return result
	}
	parsed, err := ParseCommitMessage(message)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	if len(parsed.Subject) > config.MaxSubjectLen {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Subject length %d exceeds limit %d",
			len(parsed.Subject), config.MaxSubjectLen))
	}
	allowedTypes := []string{"feat", "fix", "docs", "style", "refactor",
		"test", "chore", "perf", "ci", "build", "revert", "ops "}
	if parsed.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Must follow format: <type>(scope): <description>")
	} else {
		validType := false
		for _, t := range allowedTypes {
			if t == parsed.Type {
				validType = true
				break
			}
		}
		if !validType {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid type '%s'. Allowed: %s", parsed.Type, strings.Join(allowedTypes, ", ")))
		}
	}
	if parsed.Body != "" {
		lines := strings.Split(parsed.Body, "\n")
		for i, line := range lines {
			if len(line) > config.MaxBodyLineLen {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Line %d in body exceeds %d chars", i+1, config.MaxBodyLineLen))
			}
		}
	}
	return result
}

func EnhanceWithAI(originalMessage string, config *Config) (string, error) {
	if config.GroqAPIKey == "" {
		return "", fmt.Errorf("Groq API key not configured. Run: commit-assistant --config-api-key YOUR_KEY")
	}

	prompt := fmt.Sprintf(`You are a git commit message expert. Improve this commit message following Conventional Commits format.

Original: "%s"

Rules:
- Format: <type>(scope): <description>
- Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
- Keep it concise (<72 chars for subject)
- Add body if needed for explanation
- Use imperative mood
- Don't add extra explanations or markdown

Return ONLY the improved commit message (no quotes, no extra text):`, originalMessage)

	requestBody := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a git commit message formatter. Output only the commit message."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  150,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("curl", "-X", "POST",
		"https://api.groq.com/openai/v1/chat/completions",
		"-H", "Content-Type: application/json",
		"-H", fmt.Sprintf("Authorization: Bearer %s", config.GroqAPIKey),
		"-d", string(jsonBody))

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("API call failed: %v\n%s", err, stderr.String())
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	enhanced := strings.TrimSpace(response.Choices[0].Message.Content)
	enhanced = strings.Trim(enhanced, "\"'")

	return enhanced, nil
}
func InstallGlobalHook() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	templateDir := filepath.Join(home, ".git-templates")
	hooksDir := filepath.Join(templateDir, "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	hookContent := fmt.Sprintf(`#!/bin/sh
# Commit Assistant - AI-powered commit message linter

COMMIT_MSG_FILE=$1

# Run the linter
%s --file "$COMMIT_MSG_FILE"

if [ $? -ne 0 ]; then
    echo ""
    echo "💡 Want AI to improve your message? Run: %s --improve \"your message\""
    echo "   Or set your Groq API key: %s --config-api-key YOUR_KEY"
    exit 1
fi

exit 0
`, binaryPath, binaryPath, binaryPath)

	hookPath := filepath.Join(hooksDir, "commit-msg")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "config", "--global", "init.templatedir", templateDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git template: %v", err)
	}

	// Reinitializehooks in existing repos (optional)
	fmt.Println("✅ Global hook installed!")
	fmt.Println("📌 To apply to existing repos, run 'git init' in each repository")

	return nil
}

func main() {
	var (
		filePath     = flag.String("file", "", "Commit message file to lint")
		message      = flag.String("message", "", "Commit message to lint directly")
		improve      = flag.String("improve", "", "Improve a commit message using AI")
		configAPIKey = flag.String("config-api-key", "", "Set your Groq API key")
		showConfig   = flag.Bool("show-config", false, "Show current configuration")
		install      = flag.Bool("install", false, "Install global git hook")
	)
	flag.Parse()

	if *configAPIKey != "" {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}
		config.GroqAPIKey = *configAPIKey
		if err := SaveConfig(config); err != nil {
			fmt.Printf("❌ Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ API key saved successfully!")
		return
	}

	if *showConfig {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n📋 Current Configuration:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("🔑 API Key: %s\n", maskAPIKey(config.GroqAPIKey))
		fmt.Printf("🤖 Model: %s\n", config.Model)
		fmt.Printf("📏 Max Subject Length: %d\n", config.MaxSubjectLen)
		fmt.Printf("📐 Max Body Line Length: %d\n", config.MaxBodyLineLen)
		fmt.Printf("⚠️  Strict Mode: %v\n", config.StrictMode)
		return
	}

	if *install {
		if err := InstallGlobalHook(); err != nil {
			fmt.Printf("❌ Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n🎉 Commit Assistant installed successfully!")
		fmt.Println("\nNext steps:")
		fmt.Println("1. Set your Groq API key:")
		fmt.Println("   commit-assistant --config-api-key YOUR_API_KEY")
		fmt.Println("2. Make a commit and watch it work!")
		return
	}

	if *improve != "" {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("🤖 Enhancing commit message with AI...")
		enhanced, err := EnhanceWithAI(*improve, config)
		if err != nil {
			fmt.Printf("❌ AI enhancement failed: %v\n", err)
			fmt.Println("\n💡 Tip: Make sure your Groq API key is set correctly")
			os.Exit(1)
		}

		fmt.Println("\n📝 Original message:")
		fmt.Printf("   %s\n", *improve)
		fmt.Println("\n✨ Improved message:")
		fmt.Printf("   %s\n", enhanced)
		fmt.Println("\n💡 Use this message? Copy it above or run:")
		fmt.Printf("   git commit -m \"%s\"\n", enhanced)
		return
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	var commitMessage string

	if *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Printf("❌ Error reading file: %v\n", err)
			os.Exit(1)
		}
		commitMessage = string(data)
	} else if *message != "" {
		commitMessage = *message
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			var sb strings.Builder
			for scanner.Scan() {
				sb.WriteString(scanner.Text())
				sb.WriteString("\n")
			}
			commitMessage = sb.String()
		} else {
			fmt.Println("📝 Enter commit message (Ctrl+D to finish):")
			scanner := bufio.NewScanner(os.Stdin)
			var sb strings.Builder
			for scanner.Scan() {
				sb.WriteString(scanner.Text())
				sb.WriteString("\n")
			}
			commitMessage = sb.String()
		}
	}

	commitMessage = strings.TrimSpace(commitMessage)
	if commitMessage == "" {
		fmt.Println("❌ No commit message provided")
		os.Exit(1)
	}

	result := Lint(commitMessage, config)

	parsed, _ := ParseCommitMessage(commitMessage)

	fmt.Println("\n📋 Commit Message Analysis:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if parsed.Type != "" {
		fmt.Printf("📝 Type: %s\n", parsed.Type)
		if parsed.Scope != "" {
			fmt.Printf("🎯 Scope: %s\n", parsed.Scope)
		}
		if parsed.IsBreaking {
			fmt.Println("⚠️  BREAKING CHANGE")
		}
		fmt.Printf("📄 Message: %s\n", parsed.Description)
	}

	fmt.Println("\n🔍 Validation Results:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(result.Errors) > 0 {
		fmt.Println("❌ Errors:")
		for _, err := range result.Errors {
			fmt.Printf("   • %s\n", err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("   • %s\n", warn)
		}
	}

	if result.Valid && len(result.Errors) == 0 {
		fmt.Println("✅ Commit message is valid!")
	}

	if !result.Valid && config.GroqAPIKey != "" {
		fmt.Println("\n🤖 AI Suggestion:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		improved, err := EnhanceWithAI(commitMessage, config)
		if err == nil {
			fmt.Printf("💡 Try this format:\n   %s\n", improved)
			fmt.Println("\n📌 To use this message:")
			fmt.Printf("   git commit -m \"%s\"\n", improved)
		}
	}

	if !result.Valid || (config.StrictMode && len(result.Warnings) > 0) {
		fmt.Println("\n❌ Commit rejected")
		os.Exit(1)
	}

	fmt.Println("\n🎉 Commit accepted!")
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "••••"
	}
	return key[:4] + "••••" + key[len(key)-4:]
}
