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
    echo "🔍 Trying GitHub API..."
    API_RESPONSE=$(curl -s --connect-timeout 5 --max-time 10 https://api.github.com/repos/ramusaaa/routix/releases/latest 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$API_RESPONSE" ]; then
        LATEST_VERSION=$(echo "$API_RESPONSE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -1)
        if [ -n "$LATEST_VERSION" ]; then
            echo "✅ Found version via API: $LATEST_VERSION"
        fi
    else
        echo "⚠️  GitHub API request failed"
    fi
fi

# Method 2: GitHub releases page
if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "" ]; then
    echo "🔍 Trying GitHub releases page..."
    RELEASES_PAGE=$(curl -s --connect-timeout 5 --max-time 10 https://github.com/ramusaaa/routix/releases/latest 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$RELEASES_PAGE" ]; then
        LATEST_VERSION=$(echo "$RELEASES_PAGE" | grep -o 'tag/v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 | sed 's/tag\///')
        if [ -n "$LATEST_VERSION" ]; then
            echo "✅ Found version via releases page: $LATEST_VERSION"
        fi
    else
        echo "⚠️  GitHub releases page request failed"
    fi
fi

# Fallback to known latest version
if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "" ]; then
    echo "⚠️  Could not fetch latest version, using v0.4.0"
    LATEST_VERSION="v0.4.0"
else
    echo "📋 Latest version: $LATEST_VERSION"
fi

# Install Routix
echo "📦 Downloading and installing routix..."
if [ "$LATEST_VERSION" = "v0.4.0" ]; then
    # If using fallback version, try @latest first
    echo "🔄 Trying @latest first..."
    if go install github.com/ramusaaa/routix/cmd/routix@latest 2>/dev/null; then
        echo "✅ Installed latest version from Go modules"
    else
        echo "⚠️  @latest failed, using fallback version"
        go install github.com/ramusaaa/routix/cmd/routix@$LATEST_VERSION
    fi
else
    go install github.com/ramusaaa/routix/cmd/routix@$LATEST_VERSION
fi

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
