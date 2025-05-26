class DualChat2Cart {
    constructor() {
        this.sessionId = this.generateSessionId();
        this.isLoading = false;
        
        this.initializeElements();
        this.bindEvents();
        this.loadCart();
    }

    initializeElements() {
        // Input elements
        this.messageInput = document.getElementById('messageInput');
        this.sendButton = document.getElementById('sendButton');
        
        // Non-streaming elements
        this.nonStreamingMessages = document.getElementById('nonStreamingMessages');
        this.nonStreamingStatus = document.getElementById('nonStreamingStatus');
        
        // Streaming elements
        this.streamingMessages = document.getElementById('streamingMessages');
        this.streamingStatus = document.getElementById('streamingStatus');
        
        // Cart elements
        this.cartItems = document.getElementById('cartItems');
        this.cartCount = document.getElementById('cartCount');
        this.cartSummary = document.getElementById('cartSummary');
        this.subtotal = document.getElementById('subtotal');
        this.tax = document.getElementById('tax');
        this.total = document.getElementById('total');
        this.checkoutBtn = document.getElementById('checkoutBtn');
        
        // Other elements
        this.loadingIndicator = document.getElementById('loadingIndicator');
        this.toastContainer = document.getElementById('toastContainer');
    }

    bindEvents() {
        this.sendButton.addEventListener('click', () => this.sendToBoth());
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendToBoth();
            }
        });
        this.checkoutBtn.addEventListener('click', () => this.checkout());
    }

    generateSessionId() {
        return 'session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    }

    async sendToBoth() {
        const message = this.messageInput.value.trim();
        if (!message || this.isLoading) return;

        this.messageInput.value = '';
        this.setLoading(true);

        // Add user message to both chats
        this.addMessage(message, 'user', this.nonStreamingMessages);
        this.addMessage(message, 'user', this.streamingMessages);

        // Update status
        this.updateStatus('nonStreamingStatus', 'processing', 'Processing...');
        this.updateStatus('streamingStatus', 'processing', 'Streaming...');

        // Start both requests simultaneously
        const promises = [
            this.sendNonStreaming(message),
            this.sendStreaming(message)
        ];

        try {
            await Promise.all(promises);
        } catch (error) {
            console.error('Error in dual chat:', error);
            this.showToast('Error processing messages', 'error');
        } finally {
            this.setLoading(false);
        }
    }

    async sendNonStreaming(message) {
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
            this.addMessage(data.message, 'assistant', this.nonStreamingMessages);
            this.updateStatus('nonStreamingStatus', 'complete', 'Complete');
            
            if (data.cart_summary) {
                this.updateCart(data.cart_summary);
            }
        } catch (error) {
            console.error('Error in non-streaming:', error);
            this.addMessage('Sorry, I encountered an error. Please try again.', 'assistant', this.nonStreamingMessages);
            this.updateStatus('nonStreamingStatus', 'error', 'Error');
        }
    }

    async sendStreaming(message) {
        try {
            const response = await fetch('/api/v1/chat/message-stream', {
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
                throw new Error('Failed to start streaming');
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            
            // Create streaming message element
            const messageDiv = this.createStreamingMessage();
            this.streamingMessages.appendChild(messageDiv);
            const contentDiv = messageDiv.querySelector('.message-content');
            
            let buffer = '';
            let firstTokenReceived = false;

            while (true) {
                const { done, value } = await reader.read();
                
                if (done) break;
                
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop(); // Keep incomplete line in buffer
                
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6);
                        
                        if (data === '[DONE]') {
                            this.updateStatus('streamingStatus', 'complete', 'Complete');
                            contentDiv.classList.remove('streaming-text');
                            return;
                        }
                        
                        if (data.startsWith('[ERROR]')) {
                            this.updateStatus('streamingStatus', 'error', 'Error');
                            contentDiv.innerHTML += '<br><span style="color: red;">' + data + '</span>';
                            return;
                        }
                        
                        if (data.startsWith('[CART_UPDATE]')) {
                            // Handle cart update
                            continue;
                        }
                        
                        if (data.startsWith('[')) {
                            // Skip control messages
                            continue;
                        }
                        
                        // Add content to streaming message
                        if (data.trim()) {
                            contentDiv.innerHTML += data;
                            this.scrollToBottom(this.streamingMessages);
                        }
                    }
                }
            }
        } catch (error) {
            console.error('Error in streaming:', error);
            this.addMessage('Sorry, I encountered an error with streaming. Please try again.', 'assistant', this.streamingMessages);
            this.updateStatus('streamingStatus', 'error', 'Error');
        }
    }

    createStreamingMessage() {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message assistant';
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content streaming-text';
        
        messageDiv.appendChild(contentDiv);
        return messageDiv;
    }

    addMessage(content, role, container) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content';
        contentDiv.innerHTML = this.formatMessage(content);
        
        messageDiv.appendChild(contentDiv);
        container.appendChild(messageDiv);
        
        this.scrollToBottom(container);
    }

    formatMessage(content) {
        return content
            .replace(/\n/g, '<br>')
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>');
    }

    scrollToBottom(container) {
        container.scrollTop = container.scrollHeight;
    }

    updateStatus(elementId, statusClass, text) {
        const element = document.getElementById(elementId);
        element.className = `status ${statusClass}`;
        element.textContent = text;
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
                        <button class="quantity-btn" onclick="dualChat.updateQuantity('${item.product.id}', ${item.quantity - 1})">-</button>
                        <span class="quantity-display">${item.quantity}</span>
                        <button class="quantity-btn" onclick="dualChat.updateQuantity('${item.product.id}', ${item.quantity + 1})">+</button>
                    </div>
                </div>
                <button class="remove-btn" onclick="dualChat.removeItem('${item.product.id}')">Remove</button>
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
                
                const successMessage = `🎉 Congratulations! Your order has been placed successfully!\n\nOrder ID: ${result.order_id}\n${result.message}`;
                this.addMessage(successMessage, 'assistant', this.nonStreamingMessages);
                this.addMessage(successMessage, 'assistant', this.streamingMessages);
                
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
            this.sendButton.innerHTML = '<span>Processing...</span>';
        } else {
            this.sendButton.innerHTML = '<span>Send to Both</span>';
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
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.dualChat = new DualChat2Cart();
});

// Handle page visibility changes to reconnect if needed
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && window.dualChat) {
        window.dualChat.loadCart();
    }
});

// Handle online/offline events
window.addEventListener('online', () => {
    if (window.dualChat) {
        window.dualChat.showToast('Connection restored', 'success');
        window.dualChat.loadCart();
    }
});

window.addEventListener('offline', () => {
    if (window.dualChat) {
        window.dualChat.showToast('Connection lost. Some features may not work.', 'warning');
    }
});
