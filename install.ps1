# install.ps1 - Run this in PowerShell as Administrator

Write-Host "🚀 Installing Commit Assistant with AI Enhancement" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
$goVersion = go version 2>$null
if (-not $goVersion) {
    Write-Host "❌ Go is not installed. Please install Go first." -ForegroundColor Red
    Write-Host "   Download from: https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}
Write-Host "✅ Go detected: $goVersion" -ForegroundColor Green

# Build the binary
Write-Host "📦 Building commit-assistant..." -ForegroundColor Blue
go build -o commit-assistant.exe main.go

if (-not (Test-Path "commit-assistant.exe")) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    exit 1
}

# Create installation directory
$installDir = "$env:USERPROFILE\bin"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Move binary
Write-Host "📁 Installing to $installDir..." -ForegroundColor Blue
Move-Item -Force "commit-assistant.exe" "$installDir\" -ErrorAction SilentlyContinue

# Add to PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Host "🔧 Adding to PATH..." -ForegroundColor Blue
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path += ";$installDir"
    Write-Host "✅ Added to PATH. Please restart your terminal after installation." -ForegroundColor Yellow
}

# Install global git hook for Windows
Write-Host "🔧 Installing global git hook..." -ForegroundColor Blue

$homeDir = $env:USERPROFILE
$templateDir = "$homeDir\.git-templates"
$hooksDir = "$templateDir\hooks"

if (-not (Test-Path $hooksDir)) {
    New-Item -ItemType Directory -Path $hooksDir -Force | Out-Null
}

$binaryPath = "$installDir\commit-assistant.exe"

$hookContent = @"
#!/bin/sh
# Commit Assistant - AI-powered commit message linter

COMMIT_MSG_FILE=`$1

# Convert Windows path to Git Bash path if needed
if echo `$COMMIT_MSG_FILE | grep -q '^[A-Za-z]:'; then
    COMMIT_MSG_FILE=`/`/`$COMMIT_MSG_FILE | sed 's/\\/\\\\/g' | sed 's/://' | sed 's/^\\\\/\\\\\\\\/'
fi

"$binaryPath" --file "`$COMMIT_MSG_FILE"

if [ `$? -ne 0 ]; then
    echo ""
    echo "💡 Want AI to improve your message? Run: commit-assistant.exe --improve `"your message`""
    echo "   Or set your Groq API key: commit-assistant.exe --config-api-key YOUR_KEY"
    exit 1
fi

exit 0
"@

$hookPath = "$hooksDir\commit-msg"
$hookContent | Out-File -FilePath $hookPath -Encoding UTF8 -Force

# Set git config
git config --global init.templatedir "$templateDir"

Write-Host "✅ Global hook installed!" -ForegroundColor Green
Write-Host "📌 Note: For existing repos, run 'git init' in each repo to activate" -ForegroundColor Yellow

# Initialize config
Write-Host "⚙️  Initializing configuration..." -ForegroundColor Blue
& "$installDir\commit-assistant.exe" --show-config 2>$null | Out-Null

Write-Host ""
Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "🚀 Next Steps:" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
Write-Host "1️⃣  Get your Groq API key:" -ForegroundColor White
Write-Host "   https://console.groq.com/keys" -ForegroundColor Blue
Write-Host ""
Write-Host "2️⃣  Configure your API key:" -ForegroundColor White
Write-Host "   commit-assistant.exe --config-api-key YOUR_API_KEY" -ForegroundColor Green
Write-Host ""
Write-Host "3️⃣  Test the linter:" -ForegroundColor White
Write-Host "   git commit -m `"bad message`" --allow-empty" -ForegroundColor Green
Write-Host ""
Write-Host "4️⃣  Try AI enhancement:" -ForegroundColor White
Write-Host "   commit-assistant.exe --improve `"fixed bug`"" -ForegroundColor Green
Write-Host ""
Write-Host "5️⃣  View your config:" -ForegroundColor White
Write-Host "   commit-assistant.exe --show-config" -ForegroundColor Green
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "💡 IMPORTANT: Restart your terminal or run:" -ForegroundColor Yellow
Write-Host "   `$env:Path = [System.Environment]::GetEnvironmentVariable(`"Path`",`"User`")" -ForegroundColor Yellow
Write-Host ""