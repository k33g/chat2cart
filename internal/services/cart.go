package services

import (
	"fmt"
	"sync"

	"chat2cart/internal/models"
)

// CartService manages shopping carts
type CartService struct {
	carts          map[string]*models.Cart
	productService *ProductService
	mutex          sync.RWMutex
}

// NewCartService creates a new cart service
func NewCartService(productService *ProductService) *CartService {
	return &CartService{
		carts:          make(map[string]*models.Cart),
		productService: productService,
	}
}

// GetCart returns a cart by ID, creating one if it doesn't exist
func (cs *CartService) GetCart(cartID string) *models.Cart {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cart, exists := cs.carts[cartID]
	if !exists {
		cart = models.NewCart(cartID)
		cs.carts[cartID] = cart
	}
	return cart
}

// AddToCart adds a product to the cart
func (cs *CartService) AddToCart(cartID, productID string, quantity int) (*models.CartSummary, error) {
	product, err := cs.productService.GetProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	err = cart.AddItem(product, quantity)
	if err != nil {
		return nil, err
	}

	return cart.GetSummary(), nil
}

// RemoveFromCart removes a product from the cart
func (cs *CartService) RemoveFromCart(cartID, productID string) (*models.CartSummary, error) {
	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	err := cart.RemoveItem(productID)
	if err != nil {
		return nil, err
	}

	return cart.GetSummary(), nil
}

// UpdateQuantity updates the quantity of a product in the cart
func (cs *CartService) UpdateQuantity(cartID, productID string, quantity int) (*models.CartSummary, error) {
	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	err := cart.UpdateQuantity(productID, quantity)
	if err != nil {
		return nil, err
	}

	return cart.GetSummary(), nil
}

// GetCartSummary returns a summary of the cart
func (cs *CartService) GetCartSummary(cartID string) *models.CartSummary {
	cart := cs.GetCart(cartID)
	return cart.GetSummary()
}

// ClearCart removes all items from the cart
func (cs *CartService) ClearCart(cartID string) *models.CartSummary {
	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cart.Clear()
	return cart.GetSummary()
}

// CheckoutCart simulates a checkout process
func (cs *CartService) CheckoutCart(cartID string) (*CheckoutResult, error) {
	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cart.IsEmpty() {
		return nil, fmt.Errorf("cannot checkout empty cart")
	}

	// Simulate checkout process
	result := &CheckoutResult{
		OrderID:     fmt.Sprintf("order-%s-%d", cartID, cart.UpdatedAt.Unix()),
		CartSummary: cart.GetSummary(),
		Status:      "completed",
		Message:     "Order placed successfully! Your items will be shipped within 2-3 business days.",
	}

	// Clear the cart after successful checkout
	cart.Clear()

	return result, nil
}

// AddProductByName adds a product to cart by searching for it by name
func (cs *CartService) AddProductByName(cartID, productName string, quantity int) (*models.CartSummary, error) {
	product, err := cs.productService.FindProductByName(productName)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	return cs.AddToCart(cartID, product.ID, quantity)
}

// CheckoutResult represents the result of a checkout operation
type CheckoutResult struct {
	OrderID     string              `json:"order_id"`
	CartSummary *models.CartSummary `json:"cart_summary"`
	Status      string              `json:"status"`
	Message     string              `json:"message"`
}

// CartOperation represents different cart operations for tool calling
type CartOperation string

const (
	OpAddToCart      CartOperation = "add_to_cart"
	OpRemoveFromCart CartOperation = "remove_from_cart"
	OpUpdateQuantity CartOperation = "update_quantity"
	OpViewCart       CartOperation = "view_cart"
	OpClearCart      CartOperation = "clear_cart"
	OpCheckout       CartOperation = "checkout"
)

// CartOperationResult represents the result of a cart operation
type CartOperationResult struct {
	Operation    CartOperation       `json:"operation"`
	Success      bool                `json:"success"`
	CartSummary  *models.CartSummary `json:"cart_summary,omitempty"`
	CheckoutInfo *CheckoutResult     `json:"checkout_info,omitempty"`
	Message      string              `json:"message"`
	Error        string              `json:"error,omitempty"`
}
