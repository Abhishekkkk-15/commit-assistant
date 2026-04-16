# 🚀 Commit Assistant

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![Groq AI](https://img.shields.io/badge/AI--Powered-Groq-orange?style=for-the-badge)](https://groq.com)

**Commit Assistant** is a powerful, AI-driven CLI tool designed to modernize your git workflow. It ensures your commit messages adhere to the **Conventional Commits** standard while providing intelligent, AI-powered suggestions whenever you're stuck or your message is rejected.

---

## ✨ Key Features

- **🛡️ Intelligent Linting**: Automatically validates commit messages against strict Conventional Commits standards.
- **🤖 AI Enhancement**: Seamless integration with **Groq AI** to suggest professional commit messages from brief descriptions.
- **🔗 Global Git Hook**: Install once and enjoy automated linting across all your local repositories.
- **⚙️ Deep Configuration**: Customize subject length, body line width, strict mode, and more.
- **💻 Cross-Platform**: Native installers for Windows (PowerShell) and Unix/macOS (Bash).

---

## 🚀 Quick Start

1. **Install**: Run the installer script for your OS (see [Installation](#-installation)).
2. **API Key**: Get your free Groq API key from the [Groq Console](https://console.groq.com/keys).
3. **Configure**: 
   ```bash
   commit-assistant --config-api-key YOUR_GROQ_API_KEY
   ```
4. **Commit**: Start committing!
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
go build -o commit-assistant main.go
```

---

## 📖 Usage

### CLI Commands

| Flag | Description |
| :--- | :--- |
| `--install` | Installs the global git hook to your `~/.git-templates`. |
| `--config-api-key <key>` | Securely saves your Groq API key to your local config. |
| `--improve "<msg>"` | Asks the AI to format a raw message into Conventional Commits. |
| `--show-config` | Displays your current settings (API key is partially masked). |
| `--message "<msg>"` | Manually lint a specific message string. |
| `--file <path>` | Lint a commit message from a file (used by git hooks). |
| `--install-repo <path>` | Install the hook in a specific repository path. |
| `--generate` | Generate a commit message from staged changes using AI. |

### Git Hook Integration
Once installed, the global hook triggers on every `git commit`. 
- **Valid Message**: The commit proceeds normally.
- **Invalid Message**: The commit is rejected, and an **AI Suggestion** is automatically displayed if your API key is configured.

---

## ⚙️ Configuration

Settings are stored in `~/.commit-assistant/config.json`.

| Setting | Default | Description |
| :--- | :--- | :--- |
| `model` | `openai/gpt-oss-120b` | The AI model used for suggestions. |
| `max_subject_length` | `120` | Maximum character count for the subject line. |
| `max_body_line_length` | `240` | Maximum character count per line in the body. |
| `strict_mode` | `false` | If true, warnings will also reject the commit. |

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


