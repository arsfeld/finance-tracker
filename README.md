# WalletMind 🧠💰

<div align="center">
  <img src="https://raw.githubusercontent.com/arsfeld/finance-tracker/refs/heads/main/logo.jpg" alt="WalletMind Logo" width="200" height="200">
  
  **Your AI-Powered Financial Intelligence Platform**
  
  [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
  [![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
  [![Supabase](https://img.shields.io/badge/Supabase-3.0-3ECF8E.svg)](https://supabase.com)
  
  [Features](#features) • [Getting Started](#getting-started) • [Documentation](#documentation) • [Contributing](#contributing)
</div>

---

## 🎯 What is WalletMind?

WalletMind transforms how you understand and manage your money by combining automatic bank synchronization with conversational AI. Simply ask questions about your finances in plain English and get intelligent, actionable insights.

### 🌟 Key Features

- **🤖 AI-Powered Chat**: Ask questions like "How much did I spend on dining last month?"
- **🏦 Multi-Bank Support**: Connect all your accounts in one place
- **👥 Collaborative**: Share financial insights with family or team members
- **📊 Smart Analytics**: Automatic categorization and spending trends
- **🔔 Intelligent Alerts**: Get notified about unusual transactions or spending patterns
- **🔒 Bank-Level Security**: Your data is encrypted and never shared

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or later
- A Supabase account (free tier available)
- A SimpleFin account for bank connections

### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/walletmind.git
   cd walletmind
   ```

2. **Set up Supabase**
   - Create a project at [supabase.com](https://supabase.com)
   - Run the migrations from `docs/SUPABASE_SETUP.md`

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

4. **Run the application**
   ```bash
   go run src/main.go web
   ```

5. **Visit http://localhost:8080**

## 📚 Documentation

- [Product Overview](docs/PRODUCT.md) - Vision, mission, and strategy
- [Features Specification](docs/FEATURES.md) - Detailed feature list and priorities
- [Product Roadmap](docs/ROADMAP.md) - Development timeline and milestones
- [Architecture](docs/ARCHITECTURE.md) - Technical architecture and design
- [API Documentation](docs/API.md) - REST API reference
- [Database Schema](docs/DATABASE.md) - Data model and relationships
- [Supabase Setup](docs/SUPABASE_SETUP.md) - Database setup instructions

## 🏗️ Architecture Overview

```
WalletMind
├── 🧠 AI Layer (OpenRouter)
│   ├── Natural language processing
│   ├── Financial insights generation
│   └── Anomaly detection
├── 🔄 Sync Layer (Go)
│   ├── Bank connections (SimpleFin)
│   ├── Transaction processing
│   └── Background workers
├── 💾 Data Layer (Supabase)
│   ├── PostgreSQL database
│   ├── Row Level Security
│   └── Real-time subscriptions
└── 🎨 UI Layer (Inertia.js + React)
    ├── Modern SPA experience
    ├── Server-side routing
    └── Real-time updates
```

## 🤝 Contributing

We love contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
golangci-lint run

# Start development server
just run
```

## 🗺️ Roadmap

### Currently Working On (Q1 2025)
- [x] Multi-tenant architecture
- [x] Supabase integration
- [ ] AI chat interface
- [ ] SimpleFin bank sync
- [ ] Basic analytics dashboard

### Coming Soon
- [ ] Mobile apps (iOS & Android)
- [ ] Investment tracking
- [ ] Tax optimization features
- [ ] API for developers

See our [detailed roadmap](docs/ROADMAP.md) for more information.

## 💬 Community

- [Discord](https://discord.gg/walletmind) - Join our community
- [Twitter](https://twitter.com/walletmind) - Follow for updates
- [Blog](https://blog.walletmind.io) - Financial insights and tips

## 🔒 Security

- End-to-end encryption for credentials
- Read-only bank access
- SOC2 compliance (in progress)
- Regular security audits

Found a security issue? Please email security@walletmind.io

## 📄 License

WalletMind is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [SimpleFin](https://simplefin.org) for bank connectivity
- [Supabase](https://supabase.com) for the backend platform
- [OpenRouter](https://openrouter.ai) for AI capabilities
- Our amazing beta testers and contributors

---

<div align="center">
  Made with ❤️ by the WalletMind team
  
  **[Website](https://walletmind.io) • [Documentation](https://docs.walletmind.io) • [Support](mailto:support@walletmind.io)**
</div>