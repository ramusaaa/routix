package store

import (
	"github.com/ramusaaa/routix"
)

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
