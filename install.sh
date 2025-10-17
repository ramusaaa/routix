#!/bin/bash

# Routix Installation Script
# This script installs Routix CLI and configures PATH automatically

set -e

echo "🚀 Installing Routix CLI..."

# Install Routix
echo "📦 Downloading and installing routix..."
go install github.com/ramusaaa/routix/cmd/routix@latest

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