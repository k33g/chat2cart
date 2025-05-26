package models

import (
	"fmt"
	"time"
)

// CartItem represents an item in the shopping cart
type CartItem struct {
	Product  *Product  `json:"product"`
	Quantity int       `json:"quantity"`
	AddedAt  time.Time `json:"added_at"`
}

// GetSubtotal calculates the subtotal for this cart item
func (ci *CartItem) GetSubtotal() float64 {
	return ci.Product.Price * float64(ci.Quantity)
}

// String returns a string representation of the cart item
func (ci *CartItem) String() string {
	return fmt.Sprintf("%s x%d - %s", ci.Product.Name, ci.Quantity, ci.FormatSubtotal())
}

// FormatSubtotal returns a formatted subtotal string
func (ci *CartItem) FormatSubtotal() string {
	return fmt.Sprintf("$%.2f", ci.GetSubtotal())
}

// Cart represents a shopping cart
type Cart struct {
	ID        string               `json:"id"`
	Items     map[string]*CartItem `json:"items"` // key is product ID
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// NewCart creates a new empty shopping cart
func NewCart(id string) *Cart {
	now := time.Now()
	return &Cart{
		ID:        id,
		Items:     make(map[string]*CartItem),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddItem adds a product to the cart or updates quantity if it already exists
func (c *Cart) AddItem(product *Product, quantity int) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if !product.CanFulfillQuantity(quantity) {
		return fmt.Errorf("insufficient stock: only %d available", product.Stock)
	}

	if existingItem, exists := c.Items[product.ID]; exists {
		newQuantity := existingItem.Quantity + quantity
		if !product.CanFulfillQuantity(newQuantity) {
			return fmt.Errorf("insufficient stock: only %d available, you already have %d in cart",
				product.Stock, existingItem.Quantity)
		}
		existingItem.Quantity = newQuantity
	} else {
		c.Items[product.ID] = &CartItem{
			Product:  product,
			Quantity: quantity,
			AddedAt:  time.Now(),
		}
	}

	c.UpdatedAt = time.Now()
	return nil
}

// RemoveItem removes a product from the cart completely
func (c *Cart) RemoveItem(productID string) error {
	if _, exists := c.Items[productID]; !exists {
		return fmt.Errorf("product not found in cart")
	}

	delete(c.Items, productID)
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateQuantity updates the quantity of a specific item in the cart
func (c *Cart) UpdateQuantity(productID string, quantity int) error {
	item, exists := c.Items[productID]
	if !exists {
		return fmt.Errorf("product not found in cart")
	}

	if quantity <= 0 {
		return c.RemoveItem(productID)
	}

	if !item.Product.CanFulfillQuantity(quantity) {
		return fmt.Errorf("insufficient stock: only %d available", item.Product.Stock)
	}

	item.Quantity = quantity
	c.UpdatedAt = time.Now()
	return nil
}

// GetItem returns a cart item by product ID
func (c *Cart) GetItem(productID string) (*CartItem, bool) {
	item, exists := c.Items[productID]
	return item, exists
}

// GetItemCount returns the total number of items in the cart
func (c *Cart) GetItemCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}

// GetSubtotal calculates the subtotal of all items in the cart
func (c *Cart) GetSubtotal() float64 {
	subtotal := 0.0
	for _, item := range c.Items {
		subtotal += item.GetSubtotal()
	}
	return subtotal
}

// GetTax calculates tax (8.5% for this example)
func (c *Cart) GetTax() float64 {
	return c.GetSubtotal() * 0.085
}

// GetTotal calculates the total including tax
func (c *Cart) GetTotal() float64 {
	return c.GetSubtotal() + c.GetTax()
}

// IsEmpty returns true if the cart has no items
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

// Clear removes all items from the cart
func (c *Cart) Clear() {
	c.Items = make(map[string]*CartItem)
	c.UpdatedAt = time.Now()
}

// GetItemsList returns a slice of all cart items
func (c *Cart) GetItemsList() []*CartItem {
	items := make([]*CartItem, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, item)
	}
	return items
}

// CartSummary represents a summary of the cart for API responses
type CartSummary struct {
	ID        string      `json:"id"`
	Items     []*CartItem `json:"items"`
	ItemCount int         `json:"item_count"`
	Subtotal  float64     `json:"subtotal"`
	Tax       float64     `json:"tax"`
	Total     float64     `json:"total"`
	IsEmpty   bool        `json:"is_empty"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// GetSummary returns a summary of the cart
func (c *Cart) GetSummary() *CartSummary {
	return &CartSummary{
		ID:        c.ID,
		Items:     c.GetItemsList(),
		ItemCount: c.GetItemCount(),
		Subtotal:  c.GetSubtotal(),
		Tax:       c.GetTax(),
		Total:     c.GetTotal(),
		IsEmpty:   c.IsEmpty(),
		UpdatedAt: c.UpdatedAt,
	}
}

// FormatSummary returns a formatted string summary of the cart
func (c *Cart) FormatSummary() string {
	if c.IsEmpty() {
		return "Your cart is empty."
	}

	summary := fmt.Sprintf("Cart Summary (%d items):\n", c.GetItemCount())
	for _, item := range c.GetItemsList() {
		summary += fmt.Sprintf("- %s\n", item.String())
	}
	summary += fmt.Sprintf("\nSubtotal: $%.2f\n", c.GetSubtotal())
	summary += fmt.Sprintf("Tax: $%.2f\n", c.GetTax())
	summary += fmt.Sprintf("Total: $%.2f", c.GetTotal())

	return summary
}
