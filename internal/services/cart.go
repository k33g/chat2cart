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
func (cs *CartService) AddToCart(cartID, productName string, quantity int) (*models.CartSummary, error) {
	product, err := cs.productService.FindProductByName(productName)
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
func (cs *CartService) RemoveFromCart(cartID, productName string) (*models.CartSummary, error) {
	product, err := cs.productService.FindProductByName(productName)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	err = cart.RemoveItem(product.ID)
	if err != nil {
		return nil, err
	}

	return cart.GetSummary(), nil
}

// UpdateQuantity updates the quantity of a product in the cart
func (cs *CartService) UpdateQuantity(cartID, productName string, quantity int) (*models.CartSummary, error) {
	product, err := cs.productService.FindProductByName(productName)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	cart := cs.GetCart(cartID)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	err = cart.UpdateQuantity(product.ID, quantity)
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

// CheckoutResult represents the result of a checkout operation
type CheckoutResult struct {
	OrderID     string              `json:"order_id"`
	CartSummary *models.CartSummary `json:"cart_summary"`
	Status      string              `json:"status"`
	Message     string              `json:"message"`
}
