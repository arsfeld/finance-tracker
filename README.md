# Finance Tracker

A Go-based tool for tracking and analyzing financial transactions with AI-powered insights.

## Features

- Fetches financial transactions from SimpleFin API
- Analyzes transactions using OpenAI's LLM
- Supports multiple notification channels (ntfy, SMS, email)
- Caches account information to prevent duplicate notifications
- Configurable date ranges for analysis
- Beautiful console output with emojis and formatting

## Prerequisites

- Go 1.21 or later
- SimpleFin API access
- OpenAI API key
- (Optional) ntfy.sh account for notifications

## Configuration

Create a `.env` file in the project root with the following variables:

```env
SIMPLEFIN_BRIDGE_URL=your_simplefin_bridge_url
OPENAI_URL=your_openai_api_url
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=claude-3-5-sonnet-latest

# Optional notification settings
NTFY_TOPIC=your_ntfy_topic
TWILIO_ACCOUNT_SID=your_twilio_sid
TWILIO_AUTH_TOKEN=your_twilio_token
TWILIO_FROM_PHONE=your_twilio_from_phone
TWILIO_TO_PHONES=your_twilio_to_phones
MAILER_URL=your_mailer_url
MAILER_FROM=your_mailer_from
MAILER_TO=your_mailer_to
```

## Usage

```bash
# Basic usage (current month)
finance_tracker

# With options
finance_tracker --notifications ntfy --date-range last_month --verbose

# Custom date range
finance_tracker --start-date 2024-01-01 --end-date 2024-01-31

# Disable notifications
finance_tracker --disable-notifications

# Disable cache
finance_tracker --disable-cache
```

### Available Options

- `-n, --notifications`: Notification types to send (sms, email, ntfy)
- `--disable-notifications`: Disable all notifications
- `--disable-cache`: Disable caching
- `--verbose`: Enable verbose logging
- `--date-range`: Date range type (current_month, last_month, last_3_months, current_year, last_year, custom)
- `--start-date`: Start date for custom range (YYYY-MM-DD)
- `--end-date`: End date for custom range (YYYY-MM-DD)

## Building

```bash
go build
```

## License

MIT License
