package store

import (
	"github.com/ramusaaa/routix"
)

// StoreModule represents the store module
type StoreModule struct {
	*routix.Module
}

// NewStoreModule creates a new store module
func NewStoreModule() *StoreModule {
	return &StoreModule{
		Module: routix.NewModule("/store"),
	}
}

// Configure configures the store module
func (m *StoreModule) Configure() *routix.ModuleConfig {
	// Create services
	storeService := NewStoreService()

	// Create controllers
	storeController := NewStoreController(storeService)

	return &routix.ModuleConfig{
		Path: "/store",
		Controllers: []routix.Controller{
			storeController,
		},
		Services: []interface{}{
			storeService,
		},
		Middleware: []routix.Middleware{
			routix.Logger(),
		},
	}
}
