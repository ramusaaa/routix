package main

import (
	"log"
	"net/http"

	"github.com/ramusaaa/routix"
	"github.com/ramusaaa/routix/example/store"
)

func main() {
	// Create router
	router := routix.New()

	// Create store module
	storeModule := store.NewStoreModule()
	config := storeModule.Configure()

	// Build module
	module := routix.NewModuleBuilder(config.Path).
		WithMiddleware(config.Middleware...).
		WithController(config.Controllers[0]).
		WithService(config.Services[0]).
		Build()

	// Register module
	module.Register(router)

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
