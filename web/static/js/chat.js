class Chat2Cart {
    constructor() {
        this.sessionId = this.generateSessionId();
        this.isLoading = false;
        
        this.initializeElements();
        this.bindEvents();
        this.loadCart();
    }

    initializeElements() {
        this.messageInput = document.getElementById('messageInput');
        this.sendButton = document.getElementById('sendButton');
        this.chatMessages = document.getElementById('chatMessages');
        this.cartItems = document.getElementById('cartItems');
        this.cartCount = document.getElementById('cartCount');
        this.cartSummary = document.getElementById('cartSummary');
        this.subtotal = document.getElementById('subtotal');
        this.tax = document.getElementById('tax');
        this.total = document.getElementById('total');
        this.checkoutBtn = document.getElementById('checkoutBtn');
        this.loadingIndicator = document.getElementById('loadingIndicator');
        this.toastContainer = document.getElementById('toastContainer');
    }

    bindEvents() {
        this.sendButton.addEventListener('click', () => this.sendMessage());
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
        this.checkoutBtn.addEventListener('click', () => this.checkout());
    }

    generateSessionId() {
        return 'session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    }

    async sendMessage() {
        const message = this.messageInput.value.trim();
        if (!message || this.isLoading) return;

        this.addMessage(message, 'user');
        this.messageInput.value = '';
        this.setLoading(true);

        try {
            const response = await fetch('/api/v1/chat/message', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    message: message,
                    session_id: this.sessionId
                })
            });

            if (!response.ok) {
                throw new Error('Failed to send message');
            }

            const data = await response.json();
            this.addMessage(data.message, 'assistant');
            
            if (data.cart_summary) {
                this.updateCart(data.cart_summary);
            }
        } catch (error) {
            console.error('Error sending message:', error);
            this.addMessage('Sorry, I encountered an error. Please try again.', 'assistant');
            this.showToast('Failed to send message', 'error');
        } finally {
            this.setLoading(false);
        }
    }

    addMessage(content, role) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content';
        contentDiv.innerHTML = this.formatMessage(content);
        
        messageDiv.appendChild(contentDiv);
        this.chatMessages.appendChild(messageDiv);
        
        // Scroll to bottom
        this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
    }

    formatMessage(content) {
        // Simple formatting - convert newlines to <br> and preserve basic structure
        return content
            .replace(/\n/g, '<br>')
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>');
    }

    async loadCart() {
        try {
            const response = await fetch(`/api/v1/cart/${this.sessionId}`);
            if (response.ok) {
                const cartData = await response.json();
                this.updateCart(cartData);
            }
        } catch (error) {
            console.error('Error loading cart:', error);
        }
    }

    updateCart(cartData) {
        this.updateCartCount(cartData.item_count);
        this.updateCartItems(cartData.items);
        this.updateCartSummary(cartData);
    }

    updateCartCount(count) {
        this.cartCount.textContent = `${count} item${count !== 1 ? 's' : ''}`;
    }

    updateCartItems(items) {
        if (!items || items.length === 0) {
            this.cartItems.innerHTML = `
                <div class="empty-cart">
                    <p>Your cart is empty</p>
                    <p class="empty-cart-subtitle">Start chatting to add items!</p>
                </div>
            `;
            return;
        }

        this.cartItems.innerHTML = items.map(item => `
            <div class="cart-item" data-product-id="${item.product.id}">
                <div class="item-info">
                    <div class="item-name">${item.product.name}</div>
                    <div class="item-price">$${item.product.price.toFixed(2)} each</div>
                    <div class="item-quantity">
                        <button class="quantity-btn" onclick="chat2cart.updateQuantity('${item.product.id}', ${item.quantity - 1})">-</button>
                        <span class="quantity-display">${item.quantity}</span>
                        <button class="quantity-btn" onclick="chat2cart.updateQuantity('${item.product.id}', ${item.quantity + 1})">+</button>
                    </div>
                </div>
                <button class="remove-btn" onclick="chat2cart.removeItem('${item.product.id}')">Remove</button>
            </div>
        `).join('');
    }

    updateCartSummary(cartData) {
        if (cartData.is_empty) {
            this.cartSummary.style.display = 'none';
            return;
        }

        this.cartSummary.style.display = 'block';
        this.subtotal.textContent = `$${cartData.subtotal.toFixed(2)}`;
        this.tax.textContent = `$${cartData.tax.toFixed(2)}`;
        this.total.textContent = `$${cartData.total.toFixed(2)}`;
    }

    async updateQuantity(productId, newQuantity) {
        if (newQuantity <= 0) {
            return this.removeItem(productId);
        }

        try {
            const response = await fetch(`/api/v1/cart/${this.sessionId}/item/${productId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    quantity: newQuantity
                })
            });

            if (response.ok) {
                const cartData = await response.json();
                this.updateCart(cartData);
                this.showToast('Quantity updated', 'success');
            } else {
                throw new Error('Failed to update quantity');
            }
        } catch (error) {
            console.error('Error updating quantity:', error);
            this.showToast('Failed to update quantity', 'error');
        }
    }

    async removeItem(productId) {
        try {
            const response = await fetch(`/api/v1/cart/${this.sessionId}/item/${productId}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                const cartData = await response.json();
                this.updateCart(cartData);
                this.showToast('Item removed from cart', 'success');
            } else {
                throw new Error('Failed to remove item');
            }
        } catch (error) {
            console.error('Error removing item:', error);
            this.showToast('Failed to remove item', 'error');
        }
    }

    async checkout() {
        if (this.isLoading) return;

        this.setLoading(true);

        try {
            const response = await fetch(`/api/v1/cart/${this.sessionId}/checkout`, {
                method: 'POST'
            });

            if (response.ok) {
                const result = await response.json();
                this.showToast('Order placed successfully!', 'success');
                this.addMessage(`🎉 Congratulations! Your order has been placed successfully!\n\nOrder ID: ${result.order_id}\n${result.message}`, 'assistant');
                
                // Clear cart after successful checkout
                this.updateCart({
                    items: [],
                    item_count: 0,
                    subtotal: 0,
                    tax: 0,
                    total: 0,
                    is_empty: true
                });
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Checkout failed');
            }
        } catch (error) {
            console.error('Error during checkout:', error);
            this.showToast(error.message || 'Checkout failed', 'error');
        } finally {
            this.setLoading(false);
        }
    }

    setLoading(loading) {
        this.isLoading = loading;
        this.sendButton.disabled = loading;
        this.loadingIndicator.style.display = loading ? 'flex' : 'none';
        
        if (loading) {
            this.sendButton.innerHTML = '<span>Sending...</span>';
        } else {
            this.sendButton.innerHTML = '<span>Send</span>';
        }
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        
        this.toastContainer.appendChild(toast);
        
        // Auto remove after 3 seconds
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 3000);
    }

    // Utility method to format currency
    formatCurrency(amount) {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD'
        }).format(amount);
    }

    // Method to handle API errors
    handleApiError(error, defaultMessage) {
        console.error('API Error:', error);
        let message = defaultMessage;
        
        if (error.response) {
            // Server responded with error status
            message = error.response.data?.error || defaultMessage;
        } else if (error.request) {
            // Network error
            message = 'Network error. Please check your connection.';
        }
        
        this.showToast(message, 'error');
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.chat2cart = new Chat2Cart();
});

// Handle page visibility changes to reconnect if needed
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && window.chat2cart) {
        // Reload cart when page becomes visible again
        window.chat2cart.loadCart();
    }
});

// Handle online/offline events
window.addEventListener('online', () => {
    if (window.chat2cart) {
        window.chat2cart.showToast('Connection restored', 'success');
        window.chat2cart.loadCart();
    }
});

window.addEventListener('offline', () => {
    if (window.chat2cart) {
        window.chat2cart.showToast('Connection lost. Some features may not work.', 'warning');
    }
});
