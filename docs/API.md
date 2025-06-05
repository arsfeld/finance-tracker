# API Documentation

## Overview

The Finaro API is a RESTful API that provides access to financial data and analytics. All endpoints require authentication except for auth endpoints.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

The API uses session-based authentication with secure HTTP-only cookies.

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}

Response 200:
{
  "user": {
    "id": "user_123",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "organizations": [
    {
      "id": "org_123",
      "name": "Personal Finance",
      "role": "owner"
    }
  ]
}
```

### Register

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe",
  "organization_name": "Personal Finance"
}

Response 201:
{
  "user": {
    "id": "user_123",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "organization": {
    "id": "org_123",
    "name": "Personal Finance"
  }
}
```

### Logout

```http
POST /auth/logout

Response 200:
{
  "message": "Logged out successfully"
}
```

## Common Headers

All authenticated requests must include the session cookie. Additionally:

```http
X-Organization-ID: org_123  # Required for all non-auth endpoints
Content-Type: application/json
Accept: application/json
```

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email is required",
    "field": "email",
    "request_id": "req_123"
  }
}
```

Common error codes:
- `UNAUTHORIZED` - Not authenticated
- `FORBIDDEN` - Not authorized for resource
- `NOT_FOUND` - Resource not found
- `VALIDATION_ERROR` - Invalid input
- `INTERNAL_ERROR` - Server error

## Endpoints

### Organizations

#### List Organizations

```http
GET /organizations

Response 200:
{
  "organizations": [
    {
      "id": "org_123",
      "name": "Personal Finance",
      "role": "owner",
      "member_count": 2,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Get Organization

```http
GET /organizations/:id

Response 200:
{
  "organization": {
    "id": "org_123",
    "name": "Personal Finance",
    "settings": {
      "billing_cycle_day": 15,
      "default_currency": "USD"
    },
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Update Organization

```http
PUT /organizations/:id
Content-Type: application/json

{
  "name": "Family Finance",
  "settings": {
    "billing_cycle_day": 1
  }
}

Response 200:
{
  "organization": {
    "id": "org_123",
    "name": "Family Finance",
    "settings": {
      "billing_cycle_day": 1,
      "default_currency": "USD"
    }
  }
}
```

### Organization Members

#### List Members

```http
GET /organizations/:id/members

Response 200:
{
  "members": [
    {
      "user_id": "user_123",
      "name": "John Doe",
      "email": "john@example.com",
      "role": "owner",
      "joined_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Invite Member

```http
POST /organizations/:id/members/invite
Content-Type: application/json

{
  "email": "jane@example.com",
  "role": "member"
}

Response 200:
{
  "invitation": {
    "id": "inv_123",
    "email": "jane@example.com",
    "role": "member",
    "expires_at": "2024-01-08T00:00:00Z"
  }
}
```

### Provider Connections

#### List Connections

```http
GET /connections

Response 200:
{
  "connections": [
    {
      "id": "conn_123",
      "provider_type": "simplefin",
      "name": "My Bank Connection",
      "last_sync": "2024-01-01T00:00:00Z",
      "sync_status": "success",
      "account_count": 3
    }
  ]
}
```

#### Create Connection

```http
POST /connections
Content-Type: application/json

{
  "provider_type": "simplefin",
  "name": "My Bank Connection",
  "credentials": {
    "bridge_url": "https://..."
  }
}

Response 201:
{
  "connection": {
    "id": "conn_123",
    "provider_type": "simplefin",
    "name": "My Bank Connection",
    "sync_status": "pending"
  }
}
```

#### Sync Connection

```http
POST /connections/:id/sync

Response 202:
{
  "message": "Sync initiated",
  "job_id": "job_123"
}
```

### Bank Accounts

#### List Accounts

```http
GET /accounts

Response 200:
{
  "accounts": [
    {
      "id": "acc_123",
      "name": "Checking Account",
      "institution": "Chase Bank",
      "account_type": "checking",
      "balance": 5000.00,
      "currency": "USD",
      "last_sync": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Get Account Transactions

```http
GET /accounts/:id/transactions?start_date=2024-01-01&end_date=2024-01-31&page=1&limit=50

Response 200:
{
  "transactions": [
    {
      "id": "tx_123",
      "amount": -50.00,
      "description": "Grocery Store",
      "merchant_name": "Whole Foods",
      "date": "2024-01-15",
      "category": {
        "id": 1,
        "name": "Groceries"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "pages": 3
  }
}
```

### Transactions

#### List All Transactions

```http
GET /transactions?start_date=2024-01-01&end_date=2024-01-31&category_id=1&search=grocery

Response 200:
{
  "transactions": [...],
  "summary": {
    "total_income": 5000.00,
    "total_expenses": -3000.00,
    "net": 2000.00,
    "transaction_count": 150
  },
  "pagination": {...}
}
```

#### Update Transaction

```http
PUT /transactions/:id
Content-Type: application/json

{
  "category_id": 5,
  "merchant_name": "Whole Foods Market"
}

Response 200:
{
  "transaction": {
    "id": "tx_123",
    "amount": -50.00,
    "merchant_name": "Whole Foods Market",
    "category": {
      "id": 5,
      "name": "Groceries"
    }
  }
}
```

#### Bulk Update

```http
POST /transactions/bulk-update
Content-Type: application/json

{
  "transaction_ids": ["tx_123", "tx_124", "tx_125"],
  "updates": {
    "category_id": 5
  }
}

Response 200:
{
  "updated_count": 3
}
```

### Categories

#### List Categories

```http
GET /categories

Response 200:
{
  "categories": [
    {
      "id": 1,
      "name": "Food & Dining",
      "color": "#FF6B6B",
      "icon": "utensils",
      "children": [
        {
          "id": 5,
          "name": "Groceries",
          "parent_id": 1
        }
      ]
    }
  ]
}
```

#### Create Category

```http
POST /categories
Content-Type: application/json

{
  "name": "Entertainment",
  "color": "#4ECDC4",
  "icon": "film",
  "rules": [
    {
      "field": "merchant_name",
      "operator": "contains",
      "value": "netflix"
    }
  ]
}

Response 201:
{
  "category": {
    "id": 10,
    "name": "Entertainment",
    "color": "#4ECDC4",
    "icon": "film"
  }
}
```

### Analytics

#### Financial Summary

```http
GET /analytics/summary?period=month&date=2024-01

Response 200:
{
  "summary": {
    "period": "2024-01",
    "income": 5000.00,
    "expenses": 3000.00,
    "net": 2000.00,
    "savings_rate": 0.40,
    "top_categories": [
      {
        "category": "Groceries",
        "amount": 800.00,
        "percentage": 0.267
      }
    ],
    "comparison": {
      "previous_period": "2023-12",
      "income_change": 0.10,
      "expense_change": -0.05
    }
  }
}
```

#### Spending by Category

```http
GET /analytics/spending?start_date=2024-01-01&end_date=2024-01-31&group_by=category

Response 200:
{
  "spending": [
    {
      "category": "Groceries",
      "amount": 800.00,
      "transaction_count": 12,
      "average_transaction": 66.67
    }
  ],
  "total": 3000.00
}
```

#### Trends

```http
GET /analytics/trends?period=monthly&months=12

Response 200:
{
  "trends": [
    {
      "period": "2024-01",
      "income": 5000.00,
      "expenses": 3000.00,
      "net": 2000.00,
      "categories": {
        "Groceries": 800.00,
        "Transport": 200.00
      }
    }
  ]
}
```

### Chat

#### Create Session

```http
POST /chat/sessions
Content-Type: application/json

{
  "title": "Budget Analysis"
}

Response 201:
{
  "session": {
    "id": "chat_123",
    "title": "Budget Analysis",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Send Message

```http
POST /chat/sessions/:id/messages
Content-Type: application/json

{
  "message": "How much did I spend on groceries last month?"
}

Response 200:
{
  "message": {
    "id": "msg_123",
    "role": "user",
    "content": "How much did I spend on groceries last month?"
  },
  "response": {
    "id": "msg_124",
    "role": "assistant",
    "content": "You spent $800 on groceries in January 2024...",
    "metadata": {
      "referenced_transactions": ["tx_123", "tx_124"],
      "date_range": {
        "start": "2024-01-01",
        "end": "2024-01-31"
      }
    }
  }
}
```

#### Stream Response (SSE)

```http
GET /chat/sessions/:id/stream
Accept: text/event-stream

Response:
event: message
data: {"chunk": "You spent "}

event: message
data: {"chunk": "$800 on groceries"}

event: done
data: {"metadata": {...}}
```

### Settings

#### Get Notification Settings

```http
GET /settings/notifications

Response 200:
{
  "settings": [
    {
      "channel": "email",
      "enabled": true,
      "settings": {
        "email": "user@example.com",
        "frequency": "daily"
      }
    }
  ]
}
```

#### Update Notification Settings

```http
PUT /settings/notifications
Content-Type: application/json

{
  "email": {
    "enabled": true,
    "settings": {
      "frequency": "weekly"
    }
  },
  "ntfy": {
    "enabled": false
  }
}

Response 200:
{
  "settings": [...]
}
```

## Rate Limiting

- 1000 requests per hour per user
- 100 requests per minute per user
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Webhooks

Organizations can configure webhooks for real-time events:

```json
{
  "event": "transactions.created",
  "organization_id": "org_123",
  "data": {
    "transaction_ids": ["tx_123", "tx_124"],
    "account_id": "acc_123"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

Events:
- `transactions.created`
- `transactions.updated`
- `sync.completed`
- `sync.failed`
- `analysis.completed`