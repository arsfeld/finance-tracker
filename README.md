# Finance Tracker

<div align="center">
  <img src="https://raw.githubusercontent.com/arsfeld/finance-tracker/refs/heads/main/logo.jpg" alt="Logo" width="200" height="200">
</div>

A smart and friendly tool that helps you keep track of your finances using AI! This tool connects to your SimpleFin account, analyzes your transactions, and provides insights about your spending patterns.

## ✨ Features

- 🤖 AI-powered analysis of your spending habits
- 📊 Multiple date range options for analysis
- 📱 Multiple notification channels:
  - 📧 Email
  - 🔔 Ntfy
- 💾 Smart caching to prevent duplicate notifications
- 🔍 Detailed transaction analysis
- 🎯 Customizable date ranges

## 🚀 Getting Started

### Prerequisites

- Go 1.x or later
- A SimpleFin account
- (Optional) Ntfy account for push notifications
- (Optional) Logo image hosted at a publicly accessible URL for email templates

### 🔧 Configuration

Create a `.env` file in your project root with the following variables:

```env
# Required
SIMPLEFIN_BRIDGE_URL=your_simplefin_bridge_url
OPENROUTER_URL=your_openrouter_url
OPENROUTER_API_KEY=your_openrouter_api_key
OPENROUTER_MODEL=your_preferred_model

# Optional - for email notifications
MAILER_URL=your_mailer_url
MAILER_FROM=your_sender_email
MAILER_TO=recipient_email

# Optional - for Ntfy notifications
NTFY_TOPIC=your_ntfy_topic
```