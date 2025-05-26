package models

import (
	"fmt"
	"strings"
)

// Product represents a product in the catalog
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Stock       int     `json:"stock"`
	ImageURL    string  `json:"image_url,omitempty"`
}

// ProductCategory represents product categories
type ProductCategory string

const (
	CategoryElectronics ProductCategory = "electronics"
	CategoryClothing    ProductCategory = "clothing"
	CategoryBooks       ProductCategory = "books"
	CategoryHome        ProductCategory = "home"
	CategorySports      ProductCategory = "sports"
	CategoryBeauty      ProductCategory = "beauty"
	CategoryToys        ProductCategory = "toys"
	CategoryFood        ProductCategory = "food"
)

// IsAvailable checks if the product is in stock
func (p *Product) IsAvailable() bool {
	return p.Stock > 0
}

// CanFulfillQuantity checks if we have enough stock for the requested quantity
func (p *Product) CanFulfillQuantity(quantity int) bool {
	return p.Stock >= quantity
}

// FormatPrice returns a formatted price string
func (p *Product) FormatPrice() string {
	return fmt.Sprintf("$%.2f", p.Price)
}

// MatchesQuery checks if the product matches a search query
func (p *Product) MatchesQuery(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	// Check name, description, and category
	searchFields := []string{
		strings.ToLower(p.Name),
		strings.ToLower(p.Description),
		strings.ToLower(p.Category),
	}

	for _, field := range searchFields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

// String returns a string representation of the product
func (p *Product) String() string {
	return fmt.Sprintf("%s - %s (%s)", p.Name, p.FormatPrice(), p.Category)
}

// ProductSearchResult represents a search result with relevance score
type ProductSearchResult struct {
	Product   *Product `json:"product"`
	Relevance float64  `json:"relevance"`
}

// ProductFilter represents filters for product search
type ProductFilter struct {
	Category string  `json:"category,omitempty"`
	MinPrice float64 `json:"min_price,omitempty"`
	MaxPrice float64 `json:"max_price,omitempty"`
	InStock  bool    `json:"in_stock,omitempty"`
	Query    string  `json:"query,omitempty"`
	Limit    int     `json:"limit,omitempty"`
}

// ApplyFilter checks if a product matches the given filter
func (p *Product) ApplyFilter(filter ProductFilter) bool {
	// Category filter
	if filter.Category != "" && strings.ToLower(p.Category) != strings.ToLower(filter.Category) {
		return false
	}

	// Price range filter
	if filter.MinPrice > 0 && p.Price < filter.MinPrice {
		return false
	}
	if filter.MaxPrice > 0 && p.Price > filter.MaxPrice {
		return false
	}

	// Stock filter
	if filter.InStock && !p.IsAvailable() {
		return false
	}

	// Query filter
	if filter.Query != "" && !p.MatchesQuery(filter.Query) {
		return false
	}

	return true
}
