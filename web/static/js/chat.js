class Chat2Cart {
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
        
        // Chat elements
        this.messages = document.getElementById('nonStreamingMessages');
        this.status = document.getElementById('nonStreamingStatus');
        
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

        this.messageInput.value = '';
        this.setLoading(true);

        // Add user message
        this.addMessage(message, 'user');

        // Update status
        this.updateStatus('processing', 'Processing...');

        try {
            await this.sendNonStreaming(message);
        } catch (error) {
            console.error('Error in chat:', error);
            this.showToast('Error processing message', 'error');
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
                    session_id: this.sessionId,
                    settings: {
                        model: modelSelect.value,
                        api_base_url: apiBaseUrl.value
                    }
                })
            });

            if (!response.ok) {
                throw new Error('Failed to send message');
            }

            const data = await response.json();
            
            // First add tool calls if any
            if (data.tool_calls && data.tool_calls.length > 0) {
                const toolCallsMessage = this.createToolCallsMessage(data.tool_calls);
                this.messages.appendChild(toolCallsMessage);
                this.scrollToBottom();
            }
            
            // Then add the AI response
            this.addMessage(data.message, 'assistant');
            
            this.updateStatus('complete', 'Complete');
            
            if (data.cart_summary) {
                this.updateCart(data.cart_summary);
            }
        } catch (error) {
            console.error('Error in chat:', error);
            this.addMessage('Sorry, I encountered an error. Please try again.', 'assistant');
            this.updateStatus('error', 'Error');
        }
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

    addMessage(content, role) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content';
        contentDiv.innerHTML = this.formatMessage(content);
        
        messageDiv.appendChild(contentDiv);
        this.messages.appendChild(messageDiv);
        
        this.scrollToBottom();
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
            let params = {};
            try {
                params = JSON.parse(toolCall.arguments || '{}');
            } catch (e) {
                params = { error: 'Could not parse arguments' };
            }
            paramsDiv.innerHTML = '<strong>Parameters:</strong><pre>' + JSON.stringify(params, null, 2) + '</pre>';
            contentDiv.appendChild(paramsDiv);
        }
        
        // Add result
        const resultDiv = document.createElement('div');
        resultDiv.className = 'tool-call-result';
        if (toolCall.success) {
            let resultContent = toolCall.result;
            if (typeof resultContent === 'object') {
                resultContent = JSON.stringify(resultContent, null, 2);
            }
            resultDiv.innerHTML = '<strong>Result:</strong><pre>' + resultContent + '</pre>';
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

    formatMessage(content) {
        // First extract and format think content if present
        const thinkMatch = content.match(/<think>([\s\S]*?)<\/think>/);
        let formattedContent = content;
        
        if (thinkMatch) {
            const thinkContent = thinkMatch[1].trim();
            const formattedThink = `<div class="think-content"><div class="think-header">🤔 Thinking Process:</div><div class="think-body">${thinkContent}</div></div>`;
            formattedContent = content.replace(/<think>[\s\S]*?<\/think>/, formattedThink);
        }
        
        // Then apply other formatting
        return formattedContent
            .replace(/\n/g, '<br>')
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>');
    }

    scrollToBottom() {
        this.messages.scrollTop = this.messages.scrollHeight;
    }

    updateStatus(statusClass, text) {
        this.status.className = `status ${statusClass}`;
        this.status.textContent = text;
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
                        <button class="quantity-btn" onclick="chat.updateQuantity('${item.product.id}', ${item.quantity - 1})">-</button>
                        <span class="quantity-display">${item.quantity}</span>
                        <button class="quantity-btn" onclick="chat.updateQuantity('${item.product.id}', ${item.quantity + 1})">+</button>
                    </div>
                </div>
                <button class="remove-btn" onclick="chat.removeItem('${item.product.id}')">Remove</button>
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
                this.addMessage(successMessage, 'assistant');
                
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
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    // Verify all required elements exist
    const requiredElements = [
        'messageInput',
        'sendButton',
        'nonStreamingMessages',
        'nonStreamingStatus',
        'cartItems',
        'cartCount',
        'cartSummary',
        'subtotal',
        'tax',
        'total',
        'checkoutBtn',
        'loadingIndicator',
        'toastContainer'
    ];

    const missingElements = requiredElements.filter(id => !document.getElementById(id));
    if (missingElements.length > 0) {
        console.error('Missing required elements:', missingElements);
        return;
    }

    window.chat = new Chat2Cart();
});

// Handle page visibility changes to reconnect if needed
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && window.chat) {
        window.chat.loadCart();
    }
});

// Handle online/offline events
window.addEventListener('online', () => {
    if (window.chat) {
        window.chat.showToast('Connection restored', 'success');
        window.chat.loadCart();
    }
});

window.addEventListener('offline', () => {
    if (window.chat) {
        window.chat.showToast('Connection lost. Some features may not work.', 'warning');
    }
});

// Settings configuration
let currentSettings = {
    model: 'gpt-4',
    apiBaseUrl: 'https://api.openai.com/v1'
};

// Available models for each provider
const providerModels = {
    'https://api.openai.com/v1': [
        { id: 'gpt-4', name: 'GPT-4' },
        { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo' }
    ],
    'http://localhost:11434/v1': [], // Will be populated dynamically
    'http://localhost:12434/engines/v1': []  // Will be populated dynamically
};

// DOM Elements for settings
const settingsBtn = document.getElementById('settingsBtn');
const settingsModal = document.getElementById('settingsModal');
const closeBtn = document.querySelector('.close-btn');
const modelSelect = document.getElementById('modelSelect');
const apiBaseUrl = document.getElementById('apiBaseUrl');
const saveSettings = document.getElementById('saveSettings');

// Load settings from localStorage
function loadSettings() {
    const savedSettings = localStorage.getItem('chat2cartSettings');
    if (savedSettings) {
        currentSettings = JSON.parse(savedSettings);
        apiBaseUrl.value = currentSettings.apiBaseUrl;
        updateModelSelector(currentSettings.apiBaseUrl);
        modelSelect.value = currentSettings.model;
    } else {
        apiBaseUrl.value = currentSettings.apiBaseUrl;
        updateModelSelector(currentSettings.apiBaseUrl);
        modelSelect.value = currentSettings.model;
    }
}

// Update model selector based on selected API provider
async function updateModelSelector(apiBaseUrl) {
    // Clear current options
    modelSelect.innerHTML = '';
    
    // Get models for the selected provider
    let models = providerModels[apiBaseUrl];
    
    // If it's Ollama or DMR, fetch available models
    if (apiBaseUrl === 'http://localhost:11434/v1' || apiBaseUrl === 'http://localhost:12434/engines/v1') {
        try {
            // Use different endpoints for Ollama and DMR
            const endpoint = apiBaseUrl === 'http://localhost:11434/v1' 
                ? 'http://localhost:11434/v1/models'
                : 'http://localhost:12434/engines/v1/models';
                
            const response = await fetch(endpoint);
            if (response.ok) {
                const data = await response.json();
                models = data.data.map(model => ({
                    id: model.id,
                    name: model.id // Use the model ID as the display name
                }));
                // Cache the models
                providerModels[apiBaseUrl] = models;
            }
        } catch (error) {
            console.error(`Error fetching models from ${apiBaseUrl}:`, error);
            window.chat.showToast(`Failed to fetch available models from ${apiBaseUrl === 'http://localhost:11434/v1' ? 'Ollama' : 'DMR'}`, 'error');
        }
    }
    
    // Add options to the selector
    models.forEach(model => {
        const option = document.createElement('option');
        option.value = model.id;
        option.textContent = model.name;
        modelSelect.appendChild(option);
    });
    
    // If the current model isn't in the list, select the first available one
    if (!models.some(model => model.id === currentSettings.model)) {
        currentSettings.model = models[0]?.id || '';
        modelSelect.value = currentSettings.model;
    }
}

// Save settings to localStorage
function saveSettingsToStorage() {
    currentSettings = {
        model: modelSelect.value,
        apiBaseUrl: apiBaseUrl.value
    };
    localStorage.setItem('chat2cartSettings', JSON.stringify(currentSettings));
    window.chat.showToast('Settings saved successfully!', 'success');
}

// Settings modal event listeners
settingsBtn.addEventListener('click', () => {
    settingsModal.style.display = 'block';
});

closeBtn.addEventListener('click', () => {
    settingsModal.style.display = 'none';
});

window.addEventListener('click', (event) => {
    if (event.target === settingsModal) {
        settingsModal.style.display = 'none';
    }
});

// Update model selector when API provider changes
apiBaseUrl.addEventListener('change', () => {
    updateModelSelector(apiBaseUrl.value);
});

saveSettings.addEventListener('click', () => {
    saveSettingsToStorage();
    settingsModal.style.display = 'none';
});

// Load settings when page loads
loadSettings();

// Modify the sendMessage function to include settings
async function sendMessage() {
    const messageInput = document.getElementById('messageInput');
    const message = messageInput.value.trim();
    
    if (!message) return;
    
    // Add user message to chat
    addMessageToChat('user', message);
    messageInput.value = '';
    
    // Show loading indicator
    document.getElementById('loadingIndicator').style.display = 'flex';
    
    try {
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                message: message,
                session_id: sessionId,
                settings: currentSettings
            }),
        });
        
        if (!response.ok) {
            throw new Error('Failed to send message');
        }
        
        const data = await response.json();
        
        // Add assistant message to chat
        addMessageToChat('assistant', data.message);
        
        // Update cart if there's a cart summary
        if (data.cart_summary) {
            updateCart(data.cart_summary);
        }
        
    } catch (error) {
        console.error('Error:', error);
        showToast('Error sending message. Please try again.');
    } finally {
        document.getElementById('loadingIndicator').style.display = 'none';
    }
}
