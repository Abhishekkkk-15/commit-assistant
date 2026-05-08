# 🚀 Commit Assistant

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![Groq AI](https://img.shields.io/badge/AI--Powered-Groq-orange?style=for-the-badge)](https://groq.com)

**Commit Assistant** is a powerful, AI-driven CLI tool designed to modernize your git workflow. It ensures your commit messages adhere to the **Conventional Commits** standard while providing intelligent, AI-powered suggestions whenever you're stuck or your message is rejected.

---

- **🛡️ Intelligent Linting**: Automatically validates commit messages against strict Conventional Commits standards.
- **🤖 AI Multi-Provider**: Seamless integration with **Groq AI**, **Ollama**, **OpenAI**, or any OpenAI-compatible API.
- **✨ Enhanced Generation**: Analyze staged changes with contextual awareness (branch name, file list) and choose from multiple AI-generated options.
- **🔗 Global Git Hook**: Install once and enjoy automated linting across all your local repositories.
- **⚙️ Deep Configuration**: Customize subject length, body line width, strict mode, allowed types, AI model, and base URL.
- **💻 Cross-Platform**: Native installers for Windows (PowerShell) and Unix/macOS (Bash).

---

## 🚀 Quick Start

1. **Install**: Run the installer script for your OS (see [Installation](#-installation)).
2. **Configure**: 
   ```bash
   # Set your API Key (Groq, OpenAI, etc.)
   commit-assistant --config-api-key YOUR_API_KEY

   # (Optional) Set a custom model and base URL
   commit-assistant --config-model llama-3.3-70b-versatile
   commit-assistant --config-base-url https://api.groq.com/openai/v1
   ```
3. **Commit**: Start committing!
   ```bash
   git commit -m "feat: add ai powered linting"
   ```

---

## 🛠 Installation

### Windows (PowerShell)
Open PowerShell as Administrator and run:
```powershell
.\install.ps1
```

### Unix / macOS (Bash/Zsh)
Run the following in your terminal:
```bash
chmod +x installer.sh
./installer.sh
```

### Manual Build (Go)
If you have Go installed, you can build it from source:
```bash
go build -o commit-assistant .
```

---

## 📖 Usage

### CLI Commands

| Flag | Description |
| :--- | :--- |
| `--install` | Installs the global git hook to your `~/.git-templates`. |
| `--config-api-key <key>` | Securely saves your AI API key. |
| `--config-model <model>` | Sets the AI model to use (e.g., `gpt-4`). |
| `--config-base-url <url>` | Sets the API base URL (OpenAI compatible). |
| `--improve "<msg>"` | Asks the AI to format a raw message into Conventional Commits. |
| `--show-config` | Displays your current settings. |
| `--message "<msg>"` | Manually lint a specific message string. |
| `--file <path>` | Lint a commit message from a file (used by git hooks). |
| `--generate` | Generate commit message options from staged changes. |
| `--commit` | Use with `--generate` to commit immediately after selection. |
| `--hint "<hint>"` | Provide a hint to guide the AI generator (e.g., `--hint "refactor"`). |

### Git Hook Integration
Once installed, the global hook triggers on every `git commit`. 
- **Valid Message**: The commit proceeds normally.
- **Invalid Message**: The commit is rejected, and an **AI Suggestion** is automatically displayed if your API key is configured.

---

## ⚙️ Configuration

Settings are stored in `~/.commit-assistant/config.json`. You can also use environment variables like `AI_API_KEY`, `AI_BASE_URL`, and `AI_MODEL`.

| Setting | Default | Description |
| :--- | :--- | :--- |
| `api_key` | `""` | Your AI provider API key. |
| `base_url` | `https://api.groq.com/openai/v1` | OpenAI-compatible base URL. |
| `model` | `llama-3.3-70b-versatile` | The AI model used for suggestions. |
| `max_subject_length` | `72` | Maximum character count for the subject line. |
| `max_body_line_length` | `72` | Maximum character count per line in the body. |
| `strict_mode` | `false` | If true, warnings will also reject the commit. |
| `allowed_types` | `[...]` | List of allowed conventional commit types. |

---

## 📝 Conventional Commits Standard

Commit Assistant enforces the following types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

**Format**: `<type>(scope): <description>`

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---


