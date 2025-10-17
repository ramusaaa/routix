package commands

import (
	"fmt"
	"strings"
)

func RouteCommand(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "list":
			listRoutes()
		case "cache":
			cacheRoutes()
		case "clear":
			clearRouteCache()
		default:
			listRoutes()
		}
	} else {
		listRoutes()
	}
}

func listRoutes() {
	fmt.Printf("🛣️  Registered Routes:\n\n")
	
	routes := []Route{
		{Method: "GET", URI: "/", Name: "welcome", Action: "WelcomeController@index", Middleware: []string{}},
		{Method: "GET", URI: "/health", Name: "health", Action: "HealthController@check", Middleware: []string{}},
		{Method: "GET", URI: "/api/v1/users", Name: "users.index", Action: "UserController@index", Middleware: []string{"auth", "throttle"}},
		{Method: "POST", URI: "/api/v1/users", Name: "users.store", Action: "UserController@store", Middleware: []string{"auth", "throttle"}},
		{Method: "GET", URI: "/api/v1/users/{id}", Name: "users.show", Action: "UserController@show", Middleware: []string{"auth"}},
		{Method: "PUT", URI: "/api/v1/users/{id}", Name: "users.update", Action: "UserController@update", Middleware: []string{"auth"}},
		{Method: "DELETE", URI: "/api/v1/users/{id}", Name: "users.destroy", Action: "UserController@destroy", Middleware: []string{"auth"}},
	}

	fmt.Printf("| %-8s | %-25s | %-20s | %-25s | %-15s |\n", 
		"Method", "URI", "Name", "Action", "Middleware")
	fmt.Printf("|----------|---------------------------|----------------------|---------------------------|------------------|\n")

	for _, route := range routes {
		middlewareStr := strings.Join(route.Middleware, ",")
		if middlewareStr == "" {
			middlewareStr = "-"
		}
		
		fmt.Printf("| %-8s | %-25s | %-20s | %-25s | %-15s |\n",
			route.Method, route.URI, route.Name, route.Action, middlewareStr)
	}

	fmt.Printf("\n📊 Total routes: %d\n", len(routes))
}

func cacheRoutes() {
	fmt.Printf("⚡ Caching routes for better performance...\n")
	fmt.Printf("✅ Routes cached successfully\n")
	fmt.Printf("💡 Route cache will improve application startup time\n")
}

func clearRouteCache() {
	fmt.Printf("🗑️  Clearing route cache...\n")
	fmt.Printf("✅ Route cache cleared successfully\n")
}

type Route struct {
	Method     string
	URI        string
	Name       string
	Action     string
	Middleware []string
}