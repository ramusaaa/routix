#!/usr/bin/env bash
set -euo pipefail

# Routix CLI installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/ramusaaa/routix/main/install.sh | bash

VERSION="v0.4.0"
MODULE="github.com/ramusaaa/routix/cmd/routix"
BINARY="routix"

# ─── helpers ────────────────────────────────────────────────────────────────

print_step() { printf "\033[36m::\033[0m %s\n" "$1"; }
print_ok()   { printf "\033[32m ok\033[0m %s\n" "$1"; }
print_warn() { printf "\033[33mwarn\033[0m %s\n" "$1" >&2; }
print_err()  { printf "\033[31merr\033[0m %s\n" "$1" >&2; }

die() {
    print_err "$1"
    exit 1
}

# ─── os / arch detection ─────────────────────────────────────────────────────

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  OS_NAME="linux" ;;
    Darwin) OS_NAME="darwin" ;;
    *)      die "Unsupported operating system: $OS. Use install.ps1 on Windows." ;;
esac

case "$ARCH" in
    x86_64)  ARCH_NAME="amd64" ;;
    aarch64|arm64) ARCH_NAME="arm64" ;;
    armv7l)  ARCH_NAME="arm" ;;
    i386|i686) ARCH_NAME="386" ;;
    *) die "Unsupported architecture: $ARCH" ;;
esac

# ─── preflight checks ────────────────────────────────────────────────────────

print_step "Checking prerequisites..."

if ! command -v go &>/dev/null; then
    die "Go is not installed. Install it from https://go.dev/dl/ and re-run this script."
fi

GO_VERSION="$(go version | awk '{print $3}' | sed 's/go//')"
GO_MAJOR="$(echo "$GO_VERSION" | cut -d. -f1)"
GO_MINOR="$(echo "$GO_VERSION" | cut -d. -f2)"

if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]; }; then
    die "Go 1.21 or newer is required (found $GO_VERSION). Update at https://go.dev/dl/"
fi

print_ok "Go $GO_VERSION ($OS_NAME/$ARCH_NAME)"

# ─── resolve version ─────────────────────────────────────────────────────────

print_step "Resolving latest version..."

INSTALL_VERSION="$VERSION"

if RESOLVED=$(curl -sf --connect-timeout 8 \
    https://api.github.com/repos/ramusaaa/routix/releases/latest \
    | grep '"tag_name"' \
    | sed -E 's/.*"([^"]+)".*/\1/'); then
    [ -n "$RESOLVED" ] && INSTALL_VERSION="$RESOLVED"
fi

print_ok "Installing $INSTALL_VERSION"

# ─── install ─────────────────────────────────────────────────────────────────

print_step "Running go install..."

if ! go install "${MODULE}@${INSTALL_VERSION}" 2>&1; then
    print_warn "Tagged version failed, trying @latest..."
    go install "${MODULE}@latest" || die "Installation failed. Check your internet connection and try again."
fi

# ─── path setup ──────────────────────────────────────────────────────────────

GOBIN="$(go env GOPATH)/bin"
if [ -z "$GOBIN" ] || [ "$GOBIN" = "/bin" ]; then
    GOBIN="$HOME/go/bin"
fi

# Detect shell config file
detect_shell_rc() {
    local shell_name
    shell_name="$(basename "${SHELL:-bash}")"
    case "$shell_name" in
        zsh)  echo "$HOME/.zshrc" ;;
        bash)
            if [ -f "$HOME/.bash_profile" ] && [ "$OS_NAME" = "darwin" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        fish) echo "$HOME/.config/fish/config.fish" ;;
        *)    echo "$HOME/.profile" ;;
    esac
}

SHELL_RC="$(detect_shell_rc)"
PATH_LINE="export PATH=\"\$PATH:${GOBIN}\""
FISH_PATH_LINE="fish_add_path ${GOBIN}"

if [[ ":$PATH:" != *":${GOBIN}:"* ]]; then
    print_step "Adding ${GOBIN} to PATH in ${SHELL_RC}..."

    if [[ "$(basename "${SHELL:-bash}")" == "fish" ]]; then
        echo "$FISH_PATH_LINE" >> "$SHELL_RC"
    else
        {
            echo ""
            echo "# Routix / Go binaries"
            echo "$PATH_LINE"
        } >> "$SHELL_RC"
    fi

    export PATH="$PATH:${GOBIN}"
    print_ok "PATH updated"
else
    print_ok "${GOBIN} already in PATH"
fi

# ─── verify ──────────────────────────────────────────────────────────────────

print_step "Verifying installation..."

if ! command -v "$BINARY" &>/dev/null; then
    # Binary may be present but not yet in shell PATH — check directly
    if [ -x "${GOBIN}/${BINARY}" ]; then
        print_ok "Installed at ${GOBIN}/${BINARY}"
        print_warn "Open a new terminal (or run: source ${SHELL_RC}) for 'routix' to be available."
    else
        die "Installation failed. Binary not found at ${GOBIN}/${BINARY}"
    fi
else
    INSTALLED_VERSION="$("$BINARY" version 2>/dev/null || echo "unknown")"
    print_ok "${BINARY} ${INSTALLED_VERSION}"
fi

# ─── done ────────────────────────────────────────────────────────────────────

printf "\n"
printf "  \033[1mRoutix is ready.\033[0m\n\n"
printf "  Get started:\n\n"
printf "    routix new my-api\n"
printf "    cd my-api\n"
printf "    routix serve\n\n"
printf "  Run \033[1mroutix help\033[0m to see all commands.\n\n"

if [[ ":$PATH:" != *":${GOBIN}:"* ]]; then
    printf "  \033[33mNote:\033[0m Restart your terminal or run:\n"
    printf "    source %s\n\n" "$SHELL_RC"
fi
