package services

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"chat2cart/internal/models"
)

// ProductService manages the product catalog
type ProductService struct {
	products map[string]*models.Product
	mutex    sync.RWMutex
}

// NewProductService creates a new product service with mock data
func NewProductService() *ProductService {
	service := &ProductService{
		products: make(map[string]*models.Product),
	}
	service.loadMockData()
	return service
}

// GetProduct returns a product by ID
func (ps *ProductService) GetProduct(id string) (*models.Product, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	product, exists := ps.products[id]
	if !exists {
		return nil, fmt.Errorf("product with ID %s not found", id)
	}
	return product, nil
}

// SearchProducts searches for products based on a filter
func (ps *ProductService) SearchProducts(filter models.ProductFilter) ([]*models.ProductSearchResult, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var results []*models.ProductSearchResult

	for _, product := range ps.products {
		if product.ApplyFilter(filter) {
			relevance := ps.calculateRelevance(product, filter.Query)
			results = append(results, &models.ProductSearchResult{
				Product:   product,
				Relevance: relevance,
			})
		}
	}

	// Sort by relevance (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	// Apply limit if specified
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// GetAllProducts returns all products
func (ps *ProductService) GetAllProducts() []*models.Product {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	products := make([]*models.Product, 0, len(ps.products))
	for _, product := range ps.products {
		products = append(products, product)
	}
	return products
}

// GetProductsByCategory returns products in a specific category
func (ps *ProductService) GetProductsByCategory(category string) []*models.Product {
	filter := models.ProductFilter{
		Category: category,
	}
	results, _ := ps.SearchProducts(filter)

	products := make([]*models.Product, len(results))
	for i, result := range results {
		products[i] = result.Product
	}
	return products
}

// FindProductByName searches for a product by exact or partial name match
func (ps *ProductService) FindProductByName(name string) (*models.Product, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	name = strings.ToLower(strings.TrimSpace(name))

	// First try exact match
	for _, product := range ps.products {
		if strings.ToLower(product.Name) == name {
			return product, nil
		}
	}

	// Then try partial match
	for _, product := range ps.products {
		if strings.Contains(strings.ToLower(product.Name), name) {
			return product, nil
		}
	}

	return nil, fmt.Errorf("product with name '%s' not found", name)
}

// calculateRelevance calculates relevance score for search results
func (ps *ProductService) calculateRelevance(product *models.Product, query string) float64 {
	if query == "" {
		return 1.0
	}

	query = strings.ToLower(query)
	score := 0.0

	// Exact name match gets highest score
	if strings.ToLower(product.Name) == query {
		score += 10.0
	}

	// Name contains query
	if strings.Contains(strings.ToLower(product.Name), query) {
		score += 5.0
	}

	// Description contains query
	if strings.Contains(strings.ToLower(product.Description), query) {
		score += 2.0
	}

	// Category matches
	if strings.ToLower(product.Category) == query {
		score += 3.0
	}

	return score
}

// loadMockData loads mock product data
func (ps *ProductService) loadMockData() {
	products := []*models.Product{
		// Electronics
		{ID: "e001", Name: "iPhone 15 Pro", Description: "Latest Apple smartphone with A17 Pro chip", Price: 999.99, Category: "electronics", Stock: 25},
		{ID: "e002", Name: "Samsung Galaxy S24", Description: "Android flagship with AI features", Price: 899.99, Category: "electronics", Stock: 30},
		{ID: "e003", Name: "MacBook Air M3", Description: "Lightweight laptop with M3 chip", Price: 1299.99, Category: "electronics", Stock: 15},
		{ID: "e004", Name: "iPad Pro 12.9", Description: "Professional tablet with M2 chip", Price: 1099.99, Category: "electronics", Stock: 20},
		{ID: "e005", Name: "AirPods Pro", Description: "Wireless earbuds with noise cancellation", Price: 249.99, Category: "electronics", Stock: 50},
		{ID: "e006", Name: "Sony WH-1000XM5", Description: "Premium noise-canceling headphones", Price: 399.99, Category: "electronics", Stock: 35},
		{ID: "e007", Name: "Nintendo Switch OLED", Description: "Gaming console with OLED screen", Price: 349.99, Category: "electronics", Stock: 40},
		{ID: "e008", Name: "Dell XPS 13", Description: "Ultrabook with Intel Core i7", Price: 1199.99, Category: "electronics", Stock: 12},
		{ID: "e009", Name: "Apple Watch Series 9", Description: "Smartwatch with health monitoring", Price: 399.99, Category: "electronics", Stock: 45},
		{ID: "e010", Name: "Canon EOS R6", Description: "Mirrorless camera for professionals", Price: 2499.99, Category: "electronics", Stock: 8},

		// Clothing
		{ID: "c001", Name: "Levi's 501 Jeans", Description: "Classic straight-leg denim jeans", Price: 89.99, Category: "clothing", Stock: 60},
		{ID: "c002", Name: "Nike Air Force 1", Description: "Iconic white sneakers", Price: 110.00, Category: "clothing", Stock: 75},
		{ID: "c003", Name: "Patagonia Fleece Jacket", Description: "Warm outdoor fleece jacket", Price: 149.99, Category: "clothing", Stock: 25},
		{ID: "c004", Name: "Adidas Ultraboost 22", Description: "Running shoes with boost technology", Price: 180.00, Category: "clothing", Stock: 40},
		{ID: "c005", Name: "Uniqlo Heattech Shirt", Description: "Thermal base layer shirt", Price: 19.99, Category: "clothing", Stock: 100},
		{ID: "c006", Name: "Ray-Ban Aviators", Description: "Classic aviator sunglasses", Price: 154.99, Category: "clothing", Stock: 30},
		{ID: "c007", Name: "North Face Puffer Jacket", Description: "Insulated winter jacket", Price: 299.99, Category: "clothing", Stock: 20},
		{ID: "c008", Name: "Converse Chuck Taylor", Description: "Classic canvas sneakers", Price: 65.00, Category: "clothing", Stock: 55},
		{ID: "c009", Name: "Polo Ralph Lauren Shirt", Description: "Cotton polo shirt", Price: 89.99, Category: "clothing", Stock: 45},
		{ID: "c010", Name: "Lululemon Yoga Pants", Description: "High-performance athletic leggings", Price: 128.00, Category: "clothing", Stock: 35},

		// Books
		{ID: "b001", Name: "The Psychology of Money", Description: "Financial wisdom by Morgan Housel", Price: 16.99, Category: "books", Stock: 80},
		{ID: "b002", Name: "Atomic Habits", Description: "Build good habits by James Clear", Price: 18.99, Category: "books", Stock: 95},
		{ID: "b003", Name: "Dune", Description: "Classic sci-fi novel by Frank Herbert", Price: 14.99, Category: "books", Stock: 60},
		{ID: "b004", Name: "The Midnight Library", Description: "Fiction by Matt Haig", Price: 15.99, Category: "books", Stock: 70},
		{ID: "b005", Name: "Sapiens", Description: "History of humankind by Yuval Noah Harari", Price: 19.99, Category: "books", Stock: 55},
		{ID: "b006", Name: "The Lean Startup", Description: "Business methodology by Eric Ries", Price: 17.99, Category: "books", Stock: 40},
		{ID: "b007", Name: "1984", Description: "Dystopian novel by George Orwell", Price: 13.99, Category: "books", Stock: 85},
		{ID: "b008", Name: "The Alchemist", Description: "Philosophical novel by Paulo Coelho", Price: 14.99, Category: "books", Stock: 75},
		{ID: "b009", Name: "Clean Code", Description: "Programming best practices by Robert Martin", Price: 42.99, Category: "books", Stock: 30},
		{ID: "b010", Name: "The 7 Habits", Description: "Self-help by Stephen Covey", Price: 16.99, Category: "books", Stock: 50},

		// Home
		{ID: "h001", Name: "Dyson V15 Vacuum", Description: "Cordless vacuum with laser detection", Price: 749.99, Category: "home", Stock: 15},
		{ID: "h002", Name: "Instant Pot Duo", Description: "7-in-1 electric pressure cooker", Price: 99.99, Category: "home", Stock: 35},
		{ID: "h003", Name: "Philips Hue Starter Kit", Description: "Smart LED light bulbs", Price: 199.99, Category: "home", Stock: 25},
		{ID: "h004", Name: "Roomba i7+", Description: "Robot vacuum with auto-empty base", Price: 599.99, Category: "home", Stock: 12},
		{ID: "h005", Name: "Ninja Blender", Description: "High-performance blender", Price: 149.99, Category: "home", Stock: 40},
		{ID: "h006", Name: "Nest Thermostat", Description: "Smart programmable thermostat", Price: 249.99, Category: "home", Stock: 20},
		{ID: "h007", Name: "KitchenAid Stand Mixer", Description: "Professional stand mixer", Price: 379.99, Category: "home", Stock: 18},
		{ID: "h008", Name: "Casper Mattress Queen", Description: "Memory foam mattress", Price: 1095.00, Category: "home", Stock: 10},
		{ID: "h009", Name: "Ring Video Doorbell", Description: "Smart doorbell with camera", Price: 199.99, Category: "home", Stock: 30},
		{ID: "h010", Name: "Shark Steam Mop", Description: "Steam cleaning mop", Price: 79.99, Category: "home", Stock: 45},

		// Sports
		{ID: "s001", Name: "Peloton Bike+", Description: "Interactive exercise bike", Price: 2495.00, Category: "sports", Stock: 5},
		{ID: "s002", Name: "Wilson Tennis Racket", Description: "Professional tennis racket", Price: 199.99, Category: "sports", Stock: 25},
		{ID: "s003", Name: "Spalding Basketball", Description: "Official size basketball", Price: 29.99, Category: "sports", Stock: 60},
		{ID: "s004", Name: "Yoga Mat Premium", Description: "Non-slip exercise mat", Price: 49.99, Category: "sports", Stock: 80},
		{ID: "s005", Name: "Bowflex Dumbbells", Description: "Adjustable weight dumbbells", Price: 349.99, Category: "sports", Stock: 15},
		{ID: "s006", Name: "Fitbit Charge 5", Description: "Fitness tracker with GPS", Price: 179.99, Category: "sports", Stock: 40},
		{ID: "s007", Name: "Coleman Camping Tent", Description: "4-person waterproof tent", Price: 129.99, Category: "sports", Stock: 20},
		{ID: "s008", Name: "Hydro Flask Water Bottle", Description: "Insulated stainless steel bottle", Price: 44.99, Category: "sports", Stock: 70},
		{ID: "s009", Name: "TRX Suspension Trainer", Description: "Bodyweight training system", Price: 195.00, Category: "sports", Stock: 30},
		{ID: "s010", Name: "Garmin GPS Watch", Description: "Multi-sport GPS smartwatch", Price: 449.99, Category: "sports", Stock: 22},
	}

	for _, product := range products {
		ps.products[product.ID] = product
	}
}
