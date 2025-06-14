package main

import (
	"log"
	"net/http"

	"github.com/ramusaaa/routix"
)

func main() {
	// Create router
	router := routix.New()

	// Create store service
	storeService := NewStoreService()

	// Create store controller
	storeController := NewStoreController(storeService)

	// Create store module
	storeModule := routix.NewModuleBuilder("/store").
		WithMiddleware(routix.Logger()).
		WithController(storeController).
		WithService(storeService).
		Build()

	// Register store module
	storeModule.Register(router)

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
