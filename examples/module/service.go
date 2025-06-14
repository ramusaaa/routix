package main

import (
	"fmt"

	"github.com/ramusaaa/routix"
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
}

// NewStoreService creates a new store service
func NewStoreService() *StoreService {
	return &StoreService{
		products: make(map[string]*Product),
	}
}

// StoreController handles store-related requests
type StoreController struct {
	storeService *StoreService
}

// NewStoreController creates a new store controller
func NewStoreController(storeService *StoreService) *StoreController {
	return &StoreController{
		storeService: storeService,
	}
}

// Register registers the controller's routes
func (c *StoreController) Register(r *routix.Router) {
	store := r.Group("/store")
	store.GET("/products", c.GetProducts)
	store.POST("/products", c.CreateProduct)
	store.GET("/products/:id", c.GetProduct)
	store.PUT("/products/:id", c.UpdateProduct)
	store.DELETE("/products/:id", c.DeleteProduct)
}

// GetProducts handles GET /store/products
func (c *StoreController) GetProducts(ctx *routix.Context) error {
	products, err := c.storeService.GetProducts()
	if err != nil {
		return ctx.Error(err, "Failed to get products")
	}
	return ctx.Success(products)
}

// CreateProduct handles POST /store/products
func (c *StoreController) CreateProduct(ctx *routix.Context) error {
	var product Product
	if err := ctx.ParseJSON(&product); err != nil {
		return ctx.Error(err, "Invalid product data")
	}

	if err := c.storeService.CreateProduct(&product); err != nil {
		return ctx.Error(err, "Failed to create product")
	}

	return ctx.Success(product)
}

// GetProduct handles GET /store/products/:id
func (c *StoreController) GetProduct(ctx *routix.Context) error {
	id := ctx.Params["id"]
	product, err := c.storeService.GetProduct(id)
	if err != nil {
		return ctx.Error(err, "Product not found")
	}
	return ctx.Success(product)
}

// UpdateProduct handles PUT /store/products/:id
func (c *StoreController) UpdateProduct(ctx *routix.Context) error {
	id := ctx.Params["id"]
	var product Product
	if err := ctx.ParseJSON(&product); err != nil {
		return ctx.Error(err, "Invalid product data")
	}

	if err := c.storeService.UpdateProduct(id, &product); err != nil {
		return ctx.Error(err, "Failed to update product")
	}

	return ctx.Success(product)
}

// DeleteProduct handles DELETE /store/products/:id
func (c *StoreController) DeleteProduct(ctx *routix.Context) error {
	id := ctx.Params["id"]
	if err := c.storeService.DeleteProduct(id); err != nil {
		return ctx.Error(err, "Failed to delete product")
	}
	return ctx.Success(nil)
}

// GetProducts returns all products
func (s *StoreService) GetProducts() ([]*Product, error) {
	products := make([]*Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	return products, nil
}

// GetProduct returns a product by ID
func (s *StoreService) GetProduct(id string) (*Product, error) {
	product, ok := s.products[id]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", id)
	}
	return product, nil
}

// CreateProduct creates a new product
func (s *StoreService) CreateProduct(product *Product) error {
	if _, ok := s.products[product.ID]; ok {
		return fmt.Errorf("product already exists: %s", product.ID)
	}
	s.products[product.ID] = product
	return nil
}

// UpdateProduct updates an existing product
func (s *StoreService) UpdateProduct(id string, product *Product) error {
	if _, ok := s.products[id]; !ok {
		return fmt.Errorf("product not found: %s", id)
	}
	product.ID = id
	s.products[id] = product
	return nil
}

// DeleteProduct deletes a product
func (s *StoreService) DeleteProduct(id string) error {
	if _, ok := s.products[id]; !ok {
		return fmt.Errorf("product not found: %s", id)
	}
	delete(s.products, id)
	return nil
}
