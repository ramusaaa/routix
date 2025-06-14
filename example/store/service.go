package store

import (
	"fmt"
	"sync"
)

// Product represents a product in the store
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// StoreService handles store-related business logic
type StoreService struct {
	products map[string]*Product
	mu       sync.RWMutex
}

// NewStoreService creates a new store service
func NewStoreService() *StoreService {
	return &StoreService{
		products: make(map[string]*Product),
	}
}

// GetProducts returns all products
func (s *StoreService) GetProducts() ([]*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]*Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	return products, nil
}

// GetProduct returns a product by ID
func (s *StoreService) GetProduct(id string) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[id]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", id)
	}
	return product, nil
}

// CreateProduct creates a new product
func (s *StoreService) CreateProduct(product *Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[product.ID]; ok {
		return fmt.Errorf("product already exists: %s", product.ID)
	}

	s.products[product.ID] = product
	return nil
}

// UpdateProduct updates an existing product
func (s *StoreService) UpdateProduct(id string, product *Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[id]; !ok {
		return fmt.Errorf("product not found: %s", id)
	}

	product.ID = id
	s.products[id] = product
	return nil
}

// DeleteProduct deletes a product
func (s *StoreService) DeleteProduct(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[id]; !ok {
		return fmt.Errorf("product not found: %s", id)
	}

	delete(s.products, id)
	return nil
}
