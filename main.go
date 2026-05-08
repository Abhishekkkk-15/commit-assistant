package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const Version = "2.0.0"

type Config struct {
	APIKey         string   `json:"api_key"`
	BaseURL        string   `json:"base_url"`
	Model          string   `json:"model"`
	MaxSubjectLen  int      `json:"max_subject_length"`
	MaxBodyLineLen int      `json:"max_body_line_length"`
	StrictMode     bool     `json:"strict_mode"`
	AllowedTypes   []string `json:"allowed_types"`
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
		BaseURL:        "https://api.groq.com/openai/v1",
		Model:          "llama-3.3-70b-versatile",
		MaxSubjectLen:  72,
		MaxBodyLineLen: 72,
		StrictMode:     false,
		AllowedTypes: []string{
			"feat", "fix", "docs", "style", "refactor",
			"perf", "test", "chore", "build", "ci",
			"revert", "style", "ops",
		},
	}
}

func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	var config Config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config = DefaultConfig()
		} else {
			return nil, err
		}
	} else {
		err = json.Unmarshal(data, &config)
		if err != nil {
			return nil, err
		}
	}

	// Environment variable overrides
	if envKey := os.Getenv("GROQ_API_KEY"); envKey != "" {
		config.APIKey = envKey
	}
	if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		config.APIKey = envKey
	}
	if envBase := os.Getenv("AI_BASE_URL"); envBase != "" {
		config.BaseURL = envBase
	}
	if envModel := os.Getenv("AI_MODEL"); envModel != "" {
		config.Model = envModel
	}

	// Project-specific config override
	localConfigPath := ".commit-assistant.json"
	if _, err := os.Stat(localConfigPath); err == nil {
		localData, err := os.ReadFile(localConfigPath)
		if err == nil {
			var localConfig Config
			if err := json.Unmarshal(localData, &localConfig); err == nil {
				// Selective merge (only non-empty fields)
				if localConfig.APIKey != "" {
					config.APIKey = localConfig.APIKey
				}
				if localConfig.BaseURL != "" {
					config.BaseURL = localConfig.BaseURL
				}
				if localConfig.Model != "" {
					config.Model = localConfig.Model
				}
				if localConfig.MaxSubjectLen != 0 {
					config.MaxSubjectLen = localConfig.MaxSubjectLen
				}
				if localConfig.MaxBodyLineLen != 0 {
					config.MaxBodyLineLen = localConfig.MaxBodyLineLen
				}
				if len(localConfig.AllowedTypes) > 0 {
					config.AllowedTypes = localConfig.AllowedTypes
				}
				config.StrictMode = localConfig.StrictMode
			}
		}
	}

	// Ensure AllowedTypes is never empty
	if len(config.AllowedTypes) == 0 {
		config.AllowedTypes = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert", "ops"}
	}

	return &config, nil
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
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty commit message")
	}
	lines := strings.Split(trimmed, "\n")
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
func FormatCommitMessage(raw string) string {
	// Replace literal \n with actual newlines
	raw = strings.ReplaceAll(raw, "\\n", "\n")
	raw = strings.ReplaceAll(raw, "\\r\\n", "\n")
	raw = strings.ReplaceAll(raw, "\\r", "\n")

	// Handle multiple spaces
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
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
	if parsed.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Must follow format: <type>(scope): <description>")
	} else {
		validType := false
		for _, t := range config.AllowedTypes {
			if t == parsed.Type {
				validType = true
				break
			}
		}
		if !validType {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid type '%s'. Allowed: %s", parsed.Type, strings.Join(config.AllowedTypes, ", ")))
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

func CallAI(prompt string, systemPrompt string, config *Config, maxTokens int) (string, error) {
	if config.APIKey == "" {
		return "", fmt.Errorf("API key not configured. Run: commit-assistant --config-api-key YOUR_KEY")
	}

	requestBody := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  maxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response choices from AI")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func EnhanceWithAI(originalMessage string, config *Config) (string, error) {
	prompt := fmt.Sprintf(`You are a git commit message expert. Improve this commit message following Conventional Commits format.

Original: "%s"

Rules:
- Format: <type>(scope): <description>
- Types: %s
- Keep it concise (<%d chars for subject)
- Add body if needed for explanation
- Use imperative mood
- Don't add extra explanations or markdown

Return ONLY the improved commit message (no quotes, no extra text):`,
		originalMessage, strings.Join(config.AllowedTypes, ", "), config.MaxSubjectLen)

	systemPrompt := "You are a git commit message formatter. Output only the commit message."
	enhanced, err := CallAI(prompt, systemPrompt, config, 150)
	if err != nil {
		return "", err
	}
	return strings.Trim(enhanced, "\"'"), nil
}

func GetGitContext() (string, string, string) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, _ := branchCmd.Output()
	branch := strings.TrimSpace(string(branchOut))

	filesCmd := exec.Command("git", "diff", "--cached", "--name-only")
	filesOut, _ := filesCmd.Output()
	files := strings.TrimSpace(string(filesOut))

	historyCmd := exec.Command("git", "log", "-n", "5", "--pretty=format:%s")
	historyOut, _ := historyCmd.Output()
	history := strings.TrimSpace(string(historyOut))

	return branch, files, history
}

func ExtractTicket(branch string) string {
	re := regexp.MustCompile(`([A-Z]+-\d+)`)
	match := re.FindString(branch)
	return match
}

func Spinner(message string, done chan bool) {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Printf("\r\033[K") // Clear line
			return
		default:
			fmt.Printf("\r\033[36m%s\033[0m %s", chars[i], message)
			i = (i + 1) % len(chars)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func GenerateFromDiff(config *Config, count int, hint string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get git diff: %v", err)
	}

	diff := strings.TrimSpace(out.String())
	if diff == "" {
		return nil, fmt.Errorf("no changes staged. Run 'git add' to stage changes first")
	}

	if len(diff) > 10000 {
		diff = diff[:10000] + "\n... (diff truncated)"
	}

	branch, files, history := GetGitContext()
	ticket := ExtractTicket(branch)

	hintText := ""
	if hint != "" {
		hintText = fmt.Sprintf("\nUser Hint: %s\n", hint)
	}

	prompt := fmt.Sprintf(`Analyze the following git diff and generate %d professional commit message(s) following Conventional Commits format.

Branch: %s
Ticket: %s
Modified Files:
%s

Recent Commit History (for style matching):
%s

Special Instructions:
- If binary files or images are modified, describe their likely purpose based on file names and locations.
- Focus on the *intent* of the changes.
%s
Diff:
"%s"

Rules:
- Format: <type>(scope): <description>
- Types: %s
- Keep the subject line concise
- Add a body if the changes are complex
- Use imperative mood
- Don't add extra explanations or markdown
- If generating multiple options, you MUST separate each complete option with the literal string: @@@
- Each option should start with a Conventional Commits type.

Return ONLY the commit message(s):`, count, branch, ticket, files, history, hintText, diff, strings.Join(config.AllowedTypes, ", "))

	systemPrompt := "You are a git commit message generator. Output only the commit message(s) based on the provided diff."

	done := make(chan bool)
	go Spinner("AI is analyzing your changes...", done)

	raw, err := CallAI(prompt, systemPrompt, config, 500)
	done <- true

	if err != nil {
		return nil, err
	}

	var options []string

	// Try multiple separators for robustness
	raw = strings.ReplaceAll(raw, "===SEP===", "@@@")
	parts := strings.Split(raw, "@@@")

	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"'")
		// Remove AI numbering if present (e.g., "1. ", "[1] ")
		p = regexp.MustCompile(`^\[?\d+\]?\.?\s*`).ReplaceAllString(p, "")
		if p != "" {
			options = append(options, p)
		}
	}

	// Fallback: If still only one option but it looks like multiple lines
	if len(options) == 1 && count > 1 {
		// Look for common patterns like multiple "feat:" or "fix:" at start of lines
		typePattern := fmt.Sprintf(`(?m)^(%s)(\(.*\))?!?:\s`, strings.Join(config.AllowedTypes, "|"))
		re := regexp.MustCompile(typePattern)
		indices := re.FindAllStringIndex(options[0], -1)
		if len(indices) > 1 {
			var newOptions []string
			for i := 0; i < len(indices); i++ {
				start := indices[i][0]
				end := len(options[0])
				if i+1 < len(indices) {
					end = indices[i+1][0]
				}
				newOptions = append(newOptions, strings.TrimSpace(options[0][start:end]))
			}
			options = newOptions
		}
	}

	return options, nil
}

func ReviewDiff(config *Config) (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get git diff: %v", err)
	}

	diff := strings.TrimSpace(out.String())
	if diff == "" {
		return "", fmt.Errorf("no changes staged for review")
	}

	prompt := fmt.Sprintf(`Review the following git diff for security vulnerabilities, code smells, and optimization opportunities.
Provide a concise list of suggestions (max 5) or say "LGTM" if the code looks excellent.

Diff:
"%s"

Format:
- [TYPE] Suggestion description`, diff)

	systemPrompt := "You are a senior software engineer and security auditor. Provide a concise code review."

	done := make(chan bool)
	go Spinner("AI is reviewing your code...", done)
	review, err := CallAI(prompt, systemPrompt, config, 500)
	done <- true

	return review, err
}

func GenerateCompletion(shell string) string {
	switch shell {
	case "bash":
		return `_commit_assistant_completions() {
    local cur opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts="--file --message --improve --config-api-key --config-model --config-base-url --show-config --install --generate --commit --hint --review --version --completion"
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _commit_assistant_completions commit-assistant`
	case "zsh":
		return `#compdef commit-assistant
_commit_assistant() {
    _arguments \
        '--file[Commit message file to lint]' \
        '--message[Commit message to lint directly]' \
        '--improve[Improve a commit message using AI]' \
        '--config-api-key[Set your AI API key]' \
        '--config-model[Set the AI model to use]' \
        '--config-base-url[Set the AI base URL]' \
        '--show-config[Show current configuration]' \
        '--install[Install global git hook]' \
        '--generate[Generate commit message from staged changes]' \
        '--commit[Automatically commit after generating]' \
        '--hint[Provide a hint to the AI generator]' \
        '--review[Perform a micro code review]' \
        '--version[Show version]' \
        '--completion[Generate completion script]'
}
_commit_assistant "$@"`
	default:
		return "Shell not supported for completion"
	}
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
"%s" --file "$COMMIT_MSG_FILE"

if [ $? -ne 0 ]; then
    echo ""
    echo "💡 Want AI to improve your message? Run: commit-assistant --improve \"your message\""
    echo "   Or set your Groq API key: commit-assistant --config-api-key YOUR_KEY"
    exit 1
fi

exit 0
`, binaryPath)

	hookPath := filepath.Join(hooksDir, "commit-msg")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "config", "--global", "init.templatedir", templateDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git template: %v", err)
	}

	// Reinitializehooks in existing repos (optional)
	fmt.Println("[DONE] Global hook installed successfully")
	return nil
}

func InstallHookInRepo(repoPath string) error {
	// Get absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	// Check if it's a git repo
	gitPath := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	hooksDir := filepath.Join(gitPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Get current binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Create hook content
	hookContent := fmt.Sprintf(`#!/bin/sh
# Commit Assistant - AI-powered commit message linter

COMMIT_MSG_FILE=$1

# Run the linter
"%s" --file "$COMMIT_MSG_FILE"

if [ $? -ne 0 ]; then
    echo ""
    echo "💡 Want AI to improve your message? Run: commit-assistant --improve \"your message\""
    exit 1
fi

exit 0
`, binaryPath)

	hookPath := filepath.Join(hooksDir, "commit-msg")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return err
	}

	fmt.Printf("[DONE] Hook installed in: %s\n", absPath)
	return nil
}

func main() {
	var (
		filePath      = flag.String("file", "", "Commit message file to lint")
		message       = flag.String("message", "", "Commit message to lint directly")
		improve       = flag.String("improve", "", "Improve a commit message using AI")
		configAPIKey  = flag.String("config-api-key", "", "Set your AI API key")
		configModel   = flag.String("config-model", "", "Set the AI model to use")
		configBaseURL = flag.String("config-base-url", "", "Set the AI base URL")
		showConfig    = flag.Bool("show-config", false, "Show current configuration")
		install       = flag.Bool("install", false, "Install global git hook")
		installRepo   = flag.String("install-repo", "", "Install hook in specific repository")
		generate      = flag.Bool("generate", false, "Generate commit message from staged changes")
		doCommit      = flag.Bool("commit", false, "Automatically commit after generating (use with --generate)")
		hint          = flag.String("hint", "", "Provide a hint to the AI generator (use with --generate)")
		review        = flag.Bool("review", false, "Perform a micro code review before committing")
		showVersion   = flag.Bool("version", false, "Show version information")
		completion    = flag.String("completion", "", "Generate shell completion script (bash/zsh)")
		noTUI         = flag.Bool("no-tui", false, "Disable the interactive TUI")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Commit Assistant \033[36mv%s\033[0m\n", Version)
		return
	}

	if *completion != "" {
		fmt.Println(GenerateCompletion(*completion))
		return
	}

	if *configAPIKey != "" || *configModel != "" || *configBaseURL != "" {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error loading config: %v\n", err)
			os.Exit(1)
		}
		if *configAPIKey != "" {
			config.APIKey = *configAPIKey
		}
		if *configModel != "" {
			config.Model = *configModel
		}
		if *configBaseURL != "" {
			config.BaseURL = *configBaseURL
		}
		if err := SaveConfig(config); err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\033[32m[DONE]\033[0m Configuration updated successfully")
		return
	}

	if *showConfig {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error loading config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n\033[1;34m--- CURRENT CONFIGURATION ---\033[0m")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("API Key:               %s\n", maskAPIKey(config.APIKey))
		fmt.Printf("Base URL:              %s\n", config.BaseURL)
		fmt.Printf("Model:                 %s\n", config.Model)
		fmt.Printf("Max Subject Length:    %d\n", config.MaxSubjectLen)
		fmt.Printf("Max Body Line Length:  %d\n", config.MaxBodyLineLen)
		fmt.Printf("Strict Mode:           %v\n", config.StrictMode)
		fmt.Printf("Allowed Types:         %s\n", strings.Join(config.AllowedTypes, ", "))
		return
	}

	if *install {
		if err := InstallGlobalHook(); err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n\033[32m[DONE]\033[0m Commit Assistant installed successfully")
		fmt.Println("\n\033[1mNEXT STEPS:\033[0m")
		fmt.Println("1. Set your API key: \033[36mcommit-assistant --config-api-key YOUR_KEY\033[0m")
		fmt.Println("2. (Optional) Set model: \033[36mcommit-assistant --config-model llama-3.3-70b-versatile\033[0m")
		fmt.Println("3. Make a commit and watch it work")
		return
	}

	if *installRepo != "" {
		if err := InstallHookInRepo(*installRepo); err != nil {
			fmt.Printf("[ERR] Installation failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *improve != "" {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\033[36m[AI]\033[0m Enhancing commit message...")
		enhanced, err := EnhanceWithAI(*improve, config)
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m AI enhancement failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n\033[1mORIGINAL MESSAGE:\033[0m")
		fmt.Printf("   %s\n", *improve)
		fmt.Println("\n\033[1;32mIMPROVED MESSAGE:\033[0m")
		fmt.Printf("   %s\n", enhanced)
		fmt.Println("\n\033[1mTIP:\033[0m Use this message? Copy it above or run:")
		fmt.Printf("   git commit -m \"%s\"\n", enhanced)
		return
	}

	if *generate {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error loading config: %v\n", err)
			os.Exit(1)
		}

		options, err := GenerateFromDiff(config, 3, *hint)
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Generation failed: %v\n", err)
			os.Exit(1)
		}

		var selected string
		if *noTUI {
			fmt.Println("\n\033[1;34m--- AI GENERATED OPTIONS ---\033[0m")
			for i, opt := range options {
				// Only show first line in preview
				firstLine := strings.Split(opt, "\n")[0]
				fmt.Printf("\033[36m[%d]\033[0m %s\n", i+1, firstLine)
			}
			fmt.Println("--------------------------------------------------")
			fmt.Printf("Select an option (1-%d) or 'q' to quit: ", len(options))
			var input string
			fmt.Scanln(&input)
			if input == "q" {
				return
			}
			var choice int
			fmt.Sscanf(input, "%d", &choice)
			if choice < 1 || choice > len(options) {
				fmt.Println("\033[31mInvalid selection\033[0m")
				return
			}
			selected = options[choice-1]
		} else {
			var err error
			selected, err = RunTUI(options)
			if err != nil {
				fmt.Printf("\033[31m[ERR]\033[0m TUI failed: %v\n", err)
				os.Exit(1)
			}
			if selected == "" {
				return
			}
		}

		fmt.Printf("\n\033[32mSelected:\033[0m %s\n", selected)

		if *doCommit {
			confirm := "y"
			fmt.Print("Commit these changes? [Y/n]: ")
			fmt.Scanln(&confirm)
			if confirm == "" || strings.ToLower(confirm) == "y" {
				cmd := exec.Command("git", "commit", "-m", selected)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("\033[31m[ERR]\033[0m Commit failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("\033[32m[DONE]\033[0m Changes committed successfully")
			}
		} else {
			fmt.Println("\n\033[1mTIP:\033[0m To commit directly, use:")
			fmt.Printf("   git commit -m \"%s\"\n", selected)
		}
		return
	}

	if *review {
		config, err := LoadConfig()
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Error loading config: %v\n", err)
			os.Exit(1)
		}

		reviewText, err := ReviewDiff(config)
		if err != nil {
			fmt.Printf("\033[31m[ERR]\033[0m Review failed: %v\n", err)
		} else {
			fmt.Println("\n\033[1;34m--- MICRO CODE REVIEW ---\033[0m")
			fmt.Println(reviewText)
			fmt.Println("--------------------------------------------------")

			confirm := "y"
			fmt.Print("Proceed to commit? [Y/n]: ")
			fmt.Scanln(&confirm)
			if confirm != "" && strings.ToLower(confirm) != "y" {
				return
			}
		}
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("[ERR] Error loading config: %v\n", err)
		os.Exit(1)
	}

	var commitMessage string

	if *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Printf("[ERR] Error reading file: %v\n", err)
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

	commitMessage = FormatCommitMessage(commitMessage)
	commitMessage = strings.TrimSpace(commitMessage)
	if commitMessage == "" {
		fmt.Println("[ERR] No commit message provided")
		os.Exit(1)
	}

	result := Lint(commitMessage, config)

	parsed, _ := ParseCommitMessage(commitMessage)

	fmt.Println("\n--- COMMIT ANALYSIS ---")
	fmt.Println("--------------------------------------------------")

	if parsed.Type != "" {
		fmt.Printf("Type:        %s\n", parsed.Type)
		if parsed.Scope != "" {
			fmt.Printf("Scope:       %s\n", parsed.Scope)
		}
		if parsed.IsBreaking {
			fmt.Println("BREAKING:    YES")
		}
		fmt.Printf("Description: %s\n", parsed.Description)
	}

	fmt.Println("\n--- VALIDATION RESULTS ---")
	fmt.Println("--------------------------------------------------")

	if len(result.Errors) > 0 {
		fmt.Println("[FAIL] Errors:")
		for _, err := range result.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("[WARN] Warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("   - %s\n", warn)
		}
	}

	if result.Valid && len(result.Errors) == 0 {
		fmt.Println("\033[32m[PASS]\033[0m Commit message is valid")
	}

	if !result.Valid && config.APIKey != "" {
		fmt.Println("\n\033[1;36m[ AI ] Suggestion:\033[0m")
		fmt.Println("--------------------------------------------------")
		improved, err := EnhanceWithAI(commitMessage, config)
		if err == nil && improved != "" {
			fmt.Printf("\033[1mFormat:\033[0m %s\n", improved)

			fmt.Print("\nApply this suggestion? [y/N]: ")
			var apply string
			fmt.Scanln(&apply)
			if strings.ToLower(apply) == "y" {
				if *filePath != "" {
					err = os.WriteFile(*filePath, []byte(improved), 0644)
					if err != nil {
						fmt.Printf("\033[31m[FAIL]\033[0m Failed to update commit message: %v\n", err)
					} else {
						fmt.Println("\033[32m[DONE]\033[0m Suggestion applied. Please run the commit command again.")
						return
					}
				} else {
					fmt.Println("\033[1mTIP:\033[0m Use this message with:")
					fmt.Printf("   git commit -m \"%s\"\n", improved)
				}
			}
		} else if err != nil {
			fmt.Printf("\033[33m[WARN]\033[0m AI suggestion failed: %v\n", err)
		}
	}

	if !result.Valid || (config.StrictMode && len(result.Warnings) > 0) {
		fmt.Println("\n\033[31m❌ Commit rejected\033[0m")
		os.Exit(1)
	}

	fmt.Println("\n\033[32m🎉 Commit accepted!\033[0m")
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "••••"
	}
	return key[:4] + "••••" + key[len(key)-4:]
}
