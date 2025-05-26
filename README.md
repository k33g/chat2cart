# chat2cart

**chat2cart** is an AI-powered shopping assistant that lets users interact via chat to build, modify, and check out a shopping cart. Through a natural conversation, users can discover products, add or remove items, and complete or cancel their purchase—all from the chat interface.

## 🚀 Features

- 💬 **Natural Language Shopping**: Chat with AI to find and purchase products
- 🛒 **Smart Cart Management**: Add, remove, and modify items through conversation
- 🔍 **Product Discovery**: Search across multiple categories with AI assistance
- 🛍️ **Real-time Cart Updates**: See your cart update instantly as you chat
- 💳 **Seamless Checkout**: Complete purchases through the chat interface
- 📱 **Responsive Design**: Works perfectly on desktop and mobile devices

## 🏗️ Architecture

### Technology Stack
- **Backend**: Go with Gin web framework
- **AI Integration**: OpenAI GPT-4 with function calling for tool use
- **Frontend**: Vanilla JavaScript with modern CSS
- **Data Storage**: In-memory (perfect for demo/development)

### Project Structure
```
chat2cart/
├── Makefile                 # Build and development commands
├── main.go                  # Application entry point
├── config/
│   └── config.go           # Configuration management
├── internal/
│   ├── handlers/           # HTTP request handlers
│   ├── models/            # Data models and structures
│   ├── services/          # Business logic services
│   └── tools/             # AI tool definitions
└── web/
    ├── static/            # CSS and JavaScript files
    └── templates/         # HTML templates
```

## 🛠️ Setup Instructions

### Prerequisites
- Go 1.18 or higher
- OpenAI API key

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/ilopezluna/chat2cart.git
   cd chat2cart
   ```

2. **Install dependencies**
   ```bash
   make setup
   ```

3. **Set your OpenAI API key**
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```
   
   Or create a `.env` file:
   ```bash
   echo "OPENAI_API_KEY=your_api_key_here" > .env
   ```

4. **Run the application**
   ```bash
   make run
   ```

5. **Open your browser**
   Navigate to `http://localhost:8080`

## 🎯 Usage

### Getting Started
1. Open the chat interface in your browser
2. Start typing natural language requests like:
   - "I need a new laptop for work"
   - "Show me some running shoes under $200"
   - "Add an iPhone to my cart"
   - "What's in my cart?"
   - "Remove the headphones from my cart"
   - "I'm ready to checkout"

## 🛒 Available Products

The demo includes 50+ products across categories:
- **Electronics**: iPhones, laptops, headphones, cameras
- **Clothing**: Jeans, sneakers, jackets, accessories  
- **Books**: Fiction, non-fiction, business, self-help
- **Home**: Kitchen appliances, smart devices, furniture
- **Sports**: Fitness equipment, outdoor gear, athletic wear

## 🔧 Development

### Available Make Commands
```bash
make help      # Show available commands
make setup     # Install dependencies
make run       # Start development server
make dev       # Start with hot reload (requires air)
make build     # Build the application
make test      # Run tests
make clean     # Clean build artifacts
```

### Hot Reload Development
For faster development with automatic reloading:
```bash
make install-air  # Install air tool (one time)
make dev         # Start with hot reload
```

### Environment Variables
- `OPENAI_API_KEY`: Your OpenAI API key (required)
- `PORT`: Server port (default: 8080)
- `ENVIRONMENT`: Environment mode (default: development)

## 🏃‍♂️ Quick Start Example

```bash
# 1. Clone and setup
git clone https://github.com/ilopezluna/chat2cart.git
cd chat2cart
make setup

# 2. Set API key
export OPENAI_API_KEY=sk-your-key-here

# 3. Run
make run

# 4. Open browser to http://localhost:8080
# 5. Start chatting: "I need a new laptop"
```

## 🎨 AI-Powered Shopping Tools

The AI assistant uses OpenAI's function calling to execute shopping actions:

- **`search_products`**: Find products by name, description, or category
- **`add_to_cart`**: Add products with specified quantities
- **`remove_from_cart`**: Remove items from the shopping cart
- **`view_cart`**: Display current cart contents and totals
- **`update_quantity`**: Modify item quantities
- **`checkout`**: Process the purchase

## 🌐 API Endpoints

### Health Check
```bash
# Check application health
curl -X GET http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "chat2cart",
  "version": "1.0.0"
}
```

### Chat API

#### Send Chat Message
```bash
curl -X POST http://localhost:8080/api/v1/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I need a new laptop for work",
    "session_id": "session-123"
  }'
```

**Response:**
```json
{
  "message": "I'd be happy to help you find a laptop for work! Let me search our electronics collection for you...",
  "session_id": "session-123",
  "cart_summary": {
    "id": "session-123",
    "items": [],
    "item_count": 0,
    "subtotal": 0,
    "tax": 0,
    "total": 0,
    "is_empty": true
  },
  "timestamp": "2025-05-26T12:30:00Z"
}
```

#### Get Chat Session History
```bash
curl -X GET http://localhost:8080/api/v1/chat/session/session-123
```

**Response:**
```json
{
  "id": "session-123",
  "messages": [
    {
      "id": "msg-1",
      "role": "user",
      "content": "I need a new laptop for work",
      "timestamp": "2025-05-26T12:30:00Z",
      "session_id": "session-123"
    }
  ],
  "created_at": "2025-05-26T12:30:00Z",
  "updated_at": "2025-05-26T12:30:00Z",
  "cart_id": "session-123"
}
```

### Cart API

#### Get Cart Contents
```bash
curl -X GET http://localhost:8080/api/v1/cart/session-123
```

**Response:**
```json
{
  "id": "session-123",
  "items": [
    {
      "product": {
        "id": "e003",
        "name": "MacBook Air M3",
        "description": "Lightweight laptop with M3 chip",
        "price": 1299.99,
        "category": "electronics",
        "stock": 15
      },
      "quantity": 1,
      "added_at": "2025-05-26T12:30:00Z"
    }
  ],
  "item_count": 1,
  "subtotal": 1299.99,
  "tax": 110.50,
  "total": 1410.49,
  "is_empty": false,
  "updated_at": "2025-05-26T12:30:00Z"
}
```

#### Add Item to Cart
```bash
curl -X POST http://localhost:8080/api/v1/cart/session-123/add \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "e003",
    "quantity": 1
  }'
```

**Response:** *(Same as Get Cart Contents)*

#### Update Item Quantity
```bash
curl -X PUT http://localhost:8080/api/v1/cart/session-123/item/e003 \
  -H "Content-Type: application/json" \
  -d '{
    "quantity": 2
  }'
```

#### Remove Item from Cart
```bash
curl -X DELETE http://localhost:8080/api/v1/cart/session-123/item/e003
```

#### Checkout
```bash
curl -X POST http://localhost:8080/api/v1/cart/session-123/checkout
```

**Response:**
```json
{
  "order_id": "order-session-123-1716724200",
  "cart_summary": {
    "id": "session-123",
    "items": [
      {
        "product": {
          "id": "e003",
          "name": "MacBook Air M3",
          "description": "Lightweight laptop with M3 chip",
          "price": 1299.99,
          "category": "electronics",
          "stock": 15
        },
        "quantity": 1,
        "added_at": "2025-05-26T12:30:00Z"
      }
    ],
    "item_count": 1,
    "subtotal": 1299.99,
    "tax": 110.50,
    "total": 1410.49,
    "is_empty": false
  },
  "status": "completed",
  "message": "Order placed successfully! Your items will be shipped within 2-3 business days."
}
```

### Product API

#### Search Products
```bash
# Search by query and category
curl -X GET "http://localhost:8080/api/v1/products/search?q=laptop&category=electronics&limit=5"

# Search by category only
curl -X GET "http://localhost:8080/api/v1/products/search?category=books&limit=10"

# General search
curl -X GET "http://localhost:8080/api/v1/products/search?q=iPhone"

# Search with all parameters
curl -X GET "http://localhost:8080/api/v1/products/search?q=running&category=sports&limit=3"
```

**Response:**
```json
{
  "results": [
    {
      "product": {
        "id": "e003",
        "name": "MacBook Air M3",
        "description": "Lightweight laptop with M3 chip",
        "price": 1299.99,
        "category": "electronics",
        "stock": 15
      },
      "relevance": 10.0
    },
    {
      "product": {
        "id": "e008",
        "name": "Dell XPS 13",
        "description": "Ultrabook with Intel Core i7",
        "price": 1199.99,
        "category": "electronics",
        "stock": 12
      },
      "relevance": 5.0
    }
  ]
}
```

## 🔄 Complete Workflow Example

Here's a complete example showing how to use the API for a full shopping experience:

```bash
# 1. Start by searching for products
curl -X GET "http://localhost:8080/api/v1/products/search?q=laptop&category=electronics&limit=3"

# 2. Add a product to cart using AI chat
curl -X POST http://localhost:8080/api/v1/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Add the MacBook Air M3 to my cart",
    "session_id": "demo-session"
  }'

# 3. Check cart contents
curl -X GET http://localhost:8080/api/v1/cart/demo-session

# 4. Add another item via direct API
curl -X POST http://localhost:8080/api/v1/cart/demo-session/add \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "e005",
    "quantity": 1
  }'

# 5. Update quantity
curl -X PUT http://localhost:8080/api/v1/cart/demo-session/item/e005 \
  -H "Content-Type: application/json" \
  -d '{
    "quantity": 2
  }'

# 6. Ask AI about cart contents
curl -X POST http://localhost:8080/api/v1/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is in my cart?",
    "session_id": "demo-session"
  }'

# 7. Complete checkout via AI
curl -X POST http://localhost:8080/api/v1/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I am ready to checkout",
    "session_id": "demo-session"
  }'
```

## ⚠️ Error Responses

All endpoints return appropriate HTTP status codes and error messages:

```bash
# Example: Adding non-existent product
curl -X POST http://localhost:8080/api/v1/cart/session-123/add \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "invalid-id",
    "quantity": 1
  }'
```

**Error Response (400 Bad Request):**
```json
{
  "error": "product not found: product with ID invalid-id not found"
}
```

```bash
# Example: Missing OpenAI API key
curl -X POST http://localhost:8080/api/v1/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello",
    "session_id": "test"
  }'
```

**Error Response (500 Internal Server Error):**
```json
{
  "error": "Failed to process message"
}
```

## 🚀 Deployment

### Building for Production
```bash
make build
ENVIRONMENT=production ./bin/chat2cart
```

### Docker (Optional)
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o chat2cart main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/chat2cart .
COPY --from=builder /app/web ./web
CMD ["./chat2cart"]
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenAI for the powerful GPT-4 API and Go SDK
- Gin framework for the excellent HTTP router
- The Go community for amazing tools and libraries

## 📞 Support

If you encounter any issues or have questions:
1. Check the [Issues](https://github.com/ilopezluna/chat2cart/issues) page
2. Create a new issue with detailed information
3. Join our community discussions

---

**Happy Shopping with AI! 🛒✨**
