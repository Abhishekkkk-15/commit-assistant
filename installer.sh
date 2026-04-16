#!/bin/bash

set -e

echo "🚀 Installing Commit Assistant with AI Enhancement"
echo "=================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS
OS="$(uname -s)"
echo -e "${BLUE}📌 Detected OS: $OS${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go first.${NC}"
    echo "   Download from: https://golang.org/dl/"
    exit 1
fi

# Check if curl is installed
if ! command -v curl &> /dev/null; then
    echo -e "${RED}❌ curl is not installed. Please install curl first.${NC}"
    exit 1
fi

# Set binary name based on OS
if [[ "$OS" == "MINGW"* ]] || [[ "$OS" == "MSYS"* ]] || [[ "$OS" == "CYGWIN"* ]]; then
    BINARY_NAME="commit-assistant.exe"
else
    BINARY_NAME="commit-assistant"
fi

# Build the binary
echo -e "${BLUE}📦 Building commit-assistant...${NC}"
go build -o "$BINARY_NAME" main.go

# Determine installation path based on OS
if [[ "$OS" == "MINGW"* ]] || [[ "$OS" == "MSYS"* ]] || [[ "$OS" == "CYGWIN"* ]]; then
    # Windows with Git Bash
    INSTALL_DIR="/usr/bin"
    echo -e "${BLUE}📁 Installing to $INSTALL_DIR (Windows Git Bash)...${NC}"
elif [[ "$OS" == "Linux" ]] || [[ "$OS" == "Darwin" ]]; then
    # Linux or MacOS
    INSTALL_DIR="/usr/local/bin"
    echo -e "${BLUE}📁 Installing to $INSTALL_DIR...${NC}"
else
    # Fallback to user's local bin
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
    echo -e "${YELLOW}📁 Installing to $INSTALL_DIR...${NC}"
fi

# Move binary to install directory
if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/" 2>/dev/null || sudo mv "$BINARY_NAME" "$INSTALL_DIR/" 2>/dev/null
    else
        echo -e "${YELLOW}⚠️  Need permission to install to $INSTALL_DIR${NC}"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/" 2>/dev/null || {
            echo -e "${YELLOW}📁 Installing to user directory instead...${NC}"
            INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$INSTALL_DIR"
            mv "$BINARY_NAME" "$INSTALL_DIR/"
        
        # Add to PATH if not already there
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bash_profile"
            echo -e "${YELLOW}📌 Added $INSTALL_DIR to PATH${NC}"
        fi
    }
fi

# Verify installation
if command -v "$BINARY_NAME" &> /dev/null; then
    echo -e "${GREEN}✅ Binary installed successfully${NC}"
else
    echo -e "${YELLOW}⚠️  Binary installed but not in PATH. Using full path...${NC}"
    # Use full path for remaining commands
    COMMIT_ASSISTANT="$INSTALL_DIR/$BINARY_NAME"
fi

# Install global git hook
echo -e "${BLUE}🔧 Installing global git hook...${NC}"

# For Windows, we need to use a different approach for git hooks
if [[ "$OS" == "MINGW"* ]] || [[ "$OS" == "MSYS"* ]]; then
    # Windows: Use %USERPROFILE% for home directory
    HOME_DIR="$HOME"
    TEMPLATE_DIR="$HOME_DIR/.git-templates"
    HOOKS_DIR="$TEMPLATE_DIR/hooks"
    
    mkdir -p "$HOOKS_DIR"
    
    # Get absolute path of the binary
    BINARY_PATH=$(which "$BINARY_NAME" 2>/dev/null || echo "$INSTALL_DIR/$BINARY_NAME")
    BINARY_PATH=$(cd "$(dirname "$BINARY_PATH")" && pwd)/$(basename "$BINARY_PATH")
    
    # Create hook script for Windows/Git Bash
    cat > "$HOOKS_DIR/commit-msg" << EOF
#!/bin/sh
# Commit Assistant - AI-powered commit message linter

COMMIT_MSG_FILE=\$1

# Run the linter (use winpty for Windows compatibility)
"$BINARY_PATH" --file "\$COMMIT_MSG_FILE"

if [ \$? -ne 0 ]; then
    echo ""
    echo "💡 Want AI to improve your message? Run: commit-assistant --improve \"your message\""
    echo "   Or set your Groq API key: commit-assistant --config-api-key YOUR_KEY"
    exit 1
fi

exit 0
EOF
    
    chmod +x "$HOOKS_DIR/commit-msg"
    
    # Configure git to use this template
    git config --global init.templatedir "$TEMPLATE_DIR"
    
    echo -e "${GREEN}✅ Global hook installed for Windows Git!${NC}"
    echo -e "${YELLOW}📌 Note: For existing repos, run 'git init' inside each repo to activate the hook${NC}"
else
    # Linux/Mac - use original method
    /usr/local/bin/commit-assistant --install 2>/dev/null || "$INSTALL_DIR/commit-assistant" --install
fi

# Create config directory
echo -e "${BLUE}⚙️  Initializing configuration...${NC}"
"$INSTALL_DIR/$BINARY_NAME" --show-config &> /dev/null || true

echo ""
echo -e "${GREEN}✅ Installation complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Next Steps:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "1️⃣  Get your Groq API key:"
echo -e "   ${BLUE}https://console.groq.com/keys${NC}"
echo ""
echo "2️⃣  Configure your API key:"
echo -e "   ${GREEN}commit-assistant --config-api-key YOUR_API_KEY${NC}"
echo ""
echo "3️⃣  Test the linter:"
echo -e "   ${GREEN}git commit -m \"bad message\" --allow-empty${NC}"
echo ""
echo "4️⃣  Try AI enhancement:"
echo -e "   ${GREEN}commit-assistant --improve \"fixed bug\"${NC}"
echo ""
echo "5️⃣  View your config:"
echo -e "   ${GREEN}commit-assistant --show-config${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}💡 Pro tip:${NC} Restart your terminal or run 'source ~/.bashrc'"
echo "   to ensure the command is available in PATH"
echo ""