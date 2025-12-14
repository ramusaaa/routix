# Contributing to Routix

Thank you for your interest in contributing to Routix! We welcome all contributions, from bug fixes to new features and documentation improvements.

## Code of Conduct

We expect all contributors to adhere to our Code of Conduct. Please be professional and respectful to everyone in the community.

## How to Contribute

### Reporting Bugs

If you find a bug, please check the [GitHub Issues](https://github.com/ramusaaa/routix/issues) to see if it has already been reported. If not, open a new issue with a detailed description and reproduction steps.

### Feature Requests

We love new ideas! Please open an issue to discuss your feature request before submitting a Pull Request. This ensures that your work aligns with the project's goals.

### Pull Requests

1.  **Fork** the repository.
2.  Create a new branch: `git checkout -b feature/amazing-feature`.
3.  Commit your changes: `git commit -m 'Add amazing feature'`.
4.  Push to the branch: `git push origin feature/amazing-feature`.
5.  Open a **Pull Request**.

## Development Setup

**Prerequisites:** Go 1.20+

```bash
# 1. Clone the repo
git clone https://github.com/ramusaaa/routix.git
cd routix

# 2. Run tests
go test ./...

# 3. Format code
go fmt ./...
```

## Style Guide

*   Follow standard Go idioms (Effective Go).
*   Run `go fmt` before committing.
*   Keep functions small and testable.
