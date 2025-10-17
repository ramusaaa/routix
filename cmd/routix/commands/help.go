package commands

import "fmt"

func ShowHelp() {
	fmt.Println(`
🚀 Routix CLI v0.2.0 - Go Web Framework

Usage:
  routix <command> [arguments]

Project Commands:
  new <name>              Create a new Routix project with interactive setup
  
Development Commands:
  serve                   Start development server with hot reload
  build                   Build application for production
  
Make Commands (Code Generation):
  make controller <name>  Create a new controller
  make model <name>       Create a new model  
  make middleware <name>  Create a new middleware
  make migration <name>   Create a new migration
  make seeder <name>      Create a new seeder
  make service <name>     Create a new service
  make request <name>     Create a new request validator
  make resource <name>    Create a new API resource
  make test <name>        Create a new test file
  make module <name>      Create a new module
  make job <name>         Create a new job

Database Commands:
  migrate                 Run database migrations
  migrate rollback        Rollback last migration
  migrate reset           Reset all migrations
  migrate fresh           Drop all tables and re-run migrations
  migrate status          Show migration status
  seed                    Run database seeders

Development Tools:
  route list              Show all registered routes
  route cache             Cache routes for better performance
  test                    Run all tests
  test unit               Run unit tests only
  test integration        Run integration tests only

Package Management:
  install <package>       Install a Routix package/plugin
  
Information Commands:
  version                 Show Routix version
  help                    Show this help message

Examples:
  routix new blog-api                    # Create new project
  routix make controller UserController  # Create controller
  routix make model User --migration     # Create model with migration
  routix make middleware Auth            # Create middleware
  routix serve --port=3000              # Start dev server on port 3000
  routix migrate                        # Run migrations
  routix test --coverage                # Run tests with coverage

Options:
  --resource              Create resource controller with CRUD methods
  --migration             Also create migration (for models)
  --model                 Also create model (for controllers)
  --port=<port>          Specify port for serve command
  --coverage             Generate coverage report for tests

For more information: https://github.com/ramusaaa/routix
`)
}

func ShowVersion() {
	fmt.Println("Routix CLI v0.2.0")
	fmt.Println("Go Web Framework")
}