#!/bin/bash

# Routix Installation Script
# This script installs Routix CLI and configures PATH automatically

set -e

echo "🚀 Installing Routix CLI..."

# Get latest version from GitHub API
echo "🔍 Checking for latest version..."

# Try multiple methods to get latest version
LATEST_VERSION=""

# Method 1: GitHub API
if [ -z "$LATEST_VERSION" ]; then
    LATEST_VERSION=$(curl -s --connect-timeout 5 --max-time 10 https://api.github.com/repos/ramusaaa/routix/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -1)
fi

# Method 2: GitHub releases page
if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "" ]; then
    LATEST_VERSION=$(curl -s --connect-timeout 5 --max-time 10 https://github.com/ramusaaa/routix/releases/latest 2>/dev/null | grep -o 'tag/v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 | sed 's/tag\///')
fi

# Fallback to known latest version
if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "" ]; then
    echo "⚠️  Could not fetch latest version, using v0.3.7"
    LATEST_VERSION="v0.3.7"
else
    echo "📋 Latest version: $LATEST_VERSION"
fi

# Install Routix
echo "📦 Downloading and installing routix..."
go install github.com/ramusaaa/routix/cmd/routix@$LATEST_VERSION

# Detect shell
SHELL_NAME=$(basename "$SHELL")
case $SHELL_NAME in
    "zsh")
        SHELL_RC="$HOME/.zshrc"
        ;;
    "bash")
        SHELL_RC="$HOME/.bashrc"
        ;;
    *)
        SHELL_RC="$HOME/.profile"
        ;;
esac

# Check if Go bin is already in PATH
if [[ ":$PATH:" != *":$HOME/go/bin:"* ]]; then
    echo "🔧 Adding Go bin directory to PATH in $SHELL_RC..."
    echo 'export PATH="$HOME/go/bin:$PATH"' >> "$SHELL_RC"
    export PATH="$HOME/go/bin:$PATH"
    echo "✅ PATH updated successfully!"
else
    echo "✅ Go bin directory already in PATH"
fi

# Verify installation
if command -v routix &> /dev/null; then
    echo "🎉 Routix installed successfully!"
    echo "📋 Version: $(routix --version)"
    echo ""
    echo "🚀 Quick start:"
    echo "   routix new my-awesome-api"
    echo "   cd my-awesome-api"
    echo "   routix serve"
    echo ""
    echo "📚 For more commands, run: routix help"
else
    echo "❌ Installation failed. Please restart your terminal and try again."
    echo "   Or manually add $HOME/go/bin to your PATH"
    exit 1
fi

echo ""
echo "🔄 Please restart your terminal or run: source $SHELL_RC"
