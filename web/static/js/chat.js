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
            
            // First add tool calls if any
            if (data.tool_calls && data.tool_calls.length > 0) {
                const toolCallsMessage = this.createToolCallsMessage(data.tool_calls);
                this.nonStreamingMessages.appendChild(toolCallsMessage);
                this.scrollToBottom(this.nonStreamingMessages);
            }
            
            // Then add the AI response
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
                throw new Error('Failed to send message');
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';
            let currentMessage = '';

            // Create initial message container
            let messageDiv = document.createElement('div');
            messageDiv.className = 'message assistant';
            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';
            messageDiv.appendChild(contentDiv);
            this.streamingMessages.appendChild(messageDiv);

            // Add streaming cursor
            const cursor = document.createElement('span');
            cursor.className = 'streaming-cursor';
            cursor.textContent = '▋';
            contentDiv.appendChild(cursor);

            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || ''; // Keep the last incomplete line in the buffer

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        console.log(line);
                        const data = line.slice(6);
                        
                        if (data === '[DONE]') {
                            // End of stream
                            cursor.remove();
                            continue;
                        }

                        if (data.startsWith('[ERROR]')) {
                            cursor.remove();
                            throw new Error(data.slice(8));
                        }

                        if (data.startsWith('[CART_UPDATE]')) {
                            try {
                                const cartData = JSON.parse(data.slice(13));
                                this.updateCart(cartData);
                            } catch (e) {
                                console.error('Error parsing cart update:', e);
                            }
                            continue;
                        }

                        // Regular message content
                        currentMessage += data;
                        // Update content before the cursor
                        contentDiv.innerHTML = this.formatMessage(currentMessage);
                        contentDiv.appendChild(cursor);
                        this.scrollToBottom(this.streamingMessages);
                    }
                }
            }

            this.updateStatus('streamingStatus', 'complete', 'Complete');
        } catch (error) {
            console.error('Error in streaming:', error);
            this.addMessage('Sorry, I encountered an error. Please try again.', 'assistant', this.streamingMessages);
            this.updateStatus('streamingStatus', 'error', 'Error');
        }
    }

    updateStreamingMessage(content) {
        // Find the last assistant message or create a new one
        let messageDiv = this.streamingMessages.querySelector('.message.assistant:last-child');
        if (!messageDiv) {
            messageDiv = document.createElement('div');
            messageDiv.className = 'message assistant';
            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';
            messageDiv.appendChild(contentDiv);
            this.streamingMessages.appendChild(messageDiv);
        }

        // Update the content
        const contentDiv = messageDiv.querySelector('.message-content');
        contentDiv.innerHTML = this.formatMessage(content);
        this.scrollToBottom(this.streamingMessages);
    }

    createToolCallsMessage(toolCalls) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message assistant';
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content';
        
        // Create a container for all tool calls
        const toolCallsContainer = document.createElement('div');
        toolCallsContainer.className = 'tool-calls-container';
        
        toolCalls.forEach(toolCall => {
            const toolCallDiv = this.createToolCallMessage(toolCall);
            toolCallsContainer.appendChild(toolCallDiv);
        });
        
        contentDiv.appendChild(toolCallsContainer);
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

    createToolCallMessage(toolCall) {
        const toolCallDiv = document.createElement('div');
        toolCallDiv.className = `tool-call-message ${toolCall.success ? 'success' : 'error'}`;
        
        // Create header (collapsed state)
        const headerDiv = document.createElement('div');
        headerDiv.className = 'tool-call-header';
        headerDiv.innerHTML = `
            <span class="tool-call-icon">🔧</span>
            <span class="tool-call-name">${toolCall.tool_name}</span>
            <span class="tool-call-status">${toolCall.success ? '✓' : '✗'}</span>
            <span class="tool-call-toggle">▼</span>
        `;
        
        // Create content (expanded state)
        const contentDiv = document.createElement('div');
        contentDiv.className = 'tool-call-content';
        
        // Add parameters if any
        if (toolCall.tool_name !== 'view_cart' && toolCall.tool_name !== 'checkout') {
            const paramsDiv = document.createElement('div');
            paramsDiv.className = 'tool-call-params';
            paramsDiv.innerHTML = '<strong>Parameters:</strong> ' + this.formatFunctionParams(toolCall.tool_name, toolCall.result);
            contentDiv.appendChild(paramsDiv);
        }
        
        // Add result
        const resultDiv = document.createElement('div');
        resultDiv.className = 'tool-call-result';
        if (toolCall.success) {
            resultDiv.innerHTML = '<strong>Result:</strong> ' + this.formatFunctionResult(toolCall);
        } else {
            resultDiv.innerHTML = '<strong>Error:</strong> ' + toolCall.error;
        }
        contentDiv.appendChild(resultDiv);
        
        // Add click handler for collapse/expand
        headerDiv.addEventListener('click', () => {
            contentDiv.style.display = contentDiv.style.display === 'none' ? 'block' : 'none';
            headerDiv.querySelector('.tool-call-toggle').textContent = 
                contentDiv.style.display === 'none' ? '▼' : '▲';
        });
        
        // Initially hide content
        contentDiv.style.display = 'none';
        
        toolCallDiv.appendChild(headerDiv);
        toolCallDiv.appendChild(contentDiv);
        
        return toolCallDiv;
    }

    addFunctionCallsToSection(toolCalls, containerId) {
        const container = document.getElementById(containerId);
        
        // Clear the "no tool calls" message if it exists
        const noToolCallsDiv = container.querySelector('.no-tool-calls');
        if (noToolCallsDiv) {
            noToolCallsDiv.remove();
        }
        
        toolCalls.forEach(toolCall => {
            const callDiv = document.createElement('div');
            callDiv.className = `tool-call-item ${toolCall.success ? 'success' : 'error'}`;
            
            // Function name
            const nameDiv = document.createElement('div');
            nameDiv.className = 'tool-call-name';
            nameDiv.textContent = toolCall.tool_name;
            callDiv.appendChild(nameDiv);
            
            // Parameters (if any)
            if (toolCall.tool_name !== 'view_cart' && toolCall.tool_name !== 'checkout') {
                const paramsDiv = document.createElement('div');
                paramsDiv.className = 'tool-call-params';
                paramsDiv.innerHTML = '<strong>Parameters:</strong> ' + this.formatFunctionParams(toolCall.tool_name, toolCall.result);
                callDiv.appendChild(paramsDiv);
            }
            
            // Result
            const resultDiv = document.createElement('div');
            resultDiv.className = 'tool-call-result';
            if (toolCall.success) {
                resultDiv.innerHTML = '<strong>Result:</strong> ' + this.formatFunctionResult(toolCall);
            } else {
                resultDiv.innerHTML = '<strong>Error:</strong> ' + toolCall.error;
            }
            callDiv.appendChild(resultDiv);
            
            container.appendChild(callDiv);
        });
        
        // Scroll to show the new tool calls
        container.scrollTop = container.scrollHeight;
    }

    addFunctionCalls(toolCalls, container) {
        const functionCallsDiv = document.createElement('div');
        functionCallsDiv.className = 'function-calls';
        
        const headerDiv = document.createElement('div');
        headerDiv.className = 'function-calls-header';
        headerDiv.innerHTML = '🔧 Function Calls';
        functionCallsDiv.appendChild(headerDiv);
        
        toolCalls.forEach(toolCall => {
            const callDiv = document.createElement('div');
            callDiv.className = `function-call ${toolCall.success ? 'success' : 'error'}`;
            
            // Function name
            const nameDiv = document.createElement('div');
            nameDiv.className = 'function-name';
            nameDiv.textContent = toolCall.tool_name;
            callDiv.appendChild(nameDiv);
            
            // Parameters (if any)
            if (toolCall.tool_name !== 'view_cart' && toolCall.tool_name !== 'checkout') {
                const paramsDiv = document.createElement('div');
                paramsDiv.className = 'function-params';
                paramsDiv.innerHTML = '<strong>Parameters:</strong> ' + this.formatFunctionParams(toolCall.tool_name, toolCall.result);
                callDiv.appendChild(paramsDiv);
            }
            
            // Result
            const resultDiv = document.createElement('div');
            resultDiv.className = 'function-result';
            if (toolCall.success) {
                resultDiv.innerHTML = '<strong>Result:</strong> ' + this.formatFunctionResult(toolCall);
            } else {
                resultDiv.innerHTML = '<strong>Error:</strong> ' + toolCall.error;
            }
            callDiv.appendChild(resultDiv);
            
            functionCallsDiv.appendChild(callDiv);
        });
        
        container.appendChild(functionCallsDiv);
        this.scrollToBottom(container);
    }

    formatFunctionParams(toolName, result) {
        switch (toolName) {
            case 'search_products':
                // Extract params from the search results context
                return 'query, category, limit';
            case 'add_to_cart':
                return 'product_id, quantity';
            case 'remove_from_cart':
                return 'product_id';
            case 'update_quantity':
                return 'product_id, quantity';
            default:
                return 'none';
        }
    }

    formatFunctionResult(toolCall) {
        if (!toolCall.success) {
            return toolCall.error;
        }
        
        switch (toolCall.tool_name) {
            case 'search_products':
                if (Array.isArray(toolCall.result)) {
                    return `Found ${toolCall.result.length} products`;
                }
                return 'Search completed';
            case 'add_to_cart':
                return 'Item added to cart';
            case 'remove_from_cart':
                return 'Item removed from cart';
            case 'update_quantity':
                return 'Quantity updated';
            case 'view_cart':
                return 'Cart viewed';
            case 'checkout':
                return 'Checkout processed';
            default:
                return 'Function executed';
        }
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
