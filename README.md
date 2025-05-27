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
make build     # Build the application
make clean     # Clean build artifacts
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

