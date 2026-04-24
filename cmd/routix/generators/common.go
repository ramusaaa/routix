package generators

import "path/filepath"

func GenerateCommonFiles(projectName string, config ProjectConfig) {
	generateGitignore(projectName)
	generateReadme(projectName, config)
	generateMakefile(projectName, config)
	generateStorageKeepFiles(projectName)
	GenerateRoutes(projectName, config)
}

func generateGitignore(projectName string) {
	content := `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local
.env.development
.env.testing
.env.production

# Database
*.db
*.sqlite
*.sqlite3

# Logs
*.log
logs/
storage/logs/*
!storage/logs/.gitkeep

# Cache
storage/cache/*
!storage/cache/.gitkeep

# Temporary files
tmp/
temp/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Coverage reports
coverage.out
coverage.html

# Build artifacts
dist/
build/

# Node modules (if using frontend tools)
node_modules/

# Docker
.dockerignore

# Air (hot reload tool)
.air.toml
tmp/`

	writeFile(filepath.Join(projectName, ".gitignore"), content)
}

func generateStorageKeepFiles(projectName string) {
	keepContent := `# This file ensures the directory is tracked by Git
# You can safely delete this file once you have other files in this directory
`
	writeFile(filepath.Join(projectName, "storage", "logs", ".gitkeep"), keepContent)
	writeFile(filepath.Join(projectName, "storage", "cache", ".gitkeep"), keepContent)
}
