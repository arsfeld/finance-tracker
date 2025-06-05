# Inertia.js Setup Guide

This guide explains how to set up and work with Inertia.js in the Finaro application.

## Overview

Finaro uses Inertia.js to create a modern single-page application (SPA) experience while maintaining server-side routing. The stack includes:

- **Backend**: Go with Gonertia adapter
- **Frontend**: React with Vite
- **Styling**: Tailwind CSS

## Architecture

### Backend (Go + Gonertia)

The Go backend uses [Gonertia](https://github.com/romsar/gonertia) to handle Inertia.js protocol:

```go
// Initialize Inertia
inertia, err := config.NewInertiaConfig(isDevelopment)

// Render a page
inertia.Render(w, r, "Dashboard/Index", gonertia.Props{
    "user": user,
    "data": data,
})
```

### Frontend (React + Vite)

The frontend uses React components as pages:

```
resources/
├── js/
│   ├── app.jsx           # Inertia app initialization
│   ├── Pages/           # Page components
│   ├── Components/      # Shared components
│   └── Layouts/         # Layout components
└── css/
    └── app.css          # Tailwind CSS
```

## Development Setup

### 1. Install Node.js dependencies

```bash
npm install
```

### 2. Start the frontend dev server

```bash
npm run dev
```

This starts Vite on http://localhost:5173 with hot module replacement.

### 3. Start the Go backend

In development mode:
```bash
just web-dev
# or
go run src/main.go web --environment=development
```

### 4. Access the application

Visit http://localhost:8080 - the Go server will proxy frontend assets from Vite in development.

## Production Build

### 1. Build frontend assets

```bash
npm run build
```

This creates optimized assets in `src/web/static/build/`.

### 2. Run in production mode

```bash
just web-prod
# or
go run src/main.go web --environment=production
```

## Creating Pages

### 1. Create a React component

```jsx
// resources/js/Pages/MyPage/Index.jsx
import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'

export default function MyPage({ data }) {
  return (
    <AuthenticatedLayout>
      <Head title="My Page" />
      <div>
        <h1>My Page</h1>
        <p>{data.message}</p>
      </div>
    </AuthenticatedLayout>
  )
}
```

### 2. Create a Go handler

```go
func (h *PageHandlers) MyPage(w http.ResponseWriter, r *http.Request) {
    h.inertia.Render(w, r, "MyPage/Index", gonertia.Props{
        "data": map[string]interface{}{
            "message": "Hello from Inertia!",
        },
    })
}
```

### 3. Add route

```go
r.Get("/my-page", pageHandlers.MyPage)
```

## Forms and Data

### Frontend form submission

```jsx
import { useForm } from '@inertiajs/react'

export default function MyForm() {
  const { data, setData, post, processing, errors } = useForm({
    name: '',
    email: '',
  })

  const submit = (e) => {
    e.preventDefault()
    post('/my-endpoint')
  }

  return (
    <form onSubmit={submit}>
      <input
        type="text"
        value={data.name}
        onChange={e => setData('name', e.target.value)}
      />
      {errors.name && <span>{errors.name}</span>}
      
      <button type="submit" disabled={processing}>
        Submit
      </button>
    </form>
  )
}
```

### Backend validation

```go
func (h *Handlers) HandleForm(w http.ResponseWriter, r *http.Request) {
    // Validation errors
    errors := make(gonertia.ValidationErrors)
    if name == "" {
        errors["name"] = []string{"Name is required"}
    }

    if len(errors) > 0 {
        h.inertia.Share("errors", errors)
        h.inertia.Back(w, r, http.StatusUnprocessableEntity)
        return
    }

    // Success
    h.inertia.Share("flash", map[string]string{
        "success": "Form submitted successfully",
    })
    h.inertia.Location(w, r, "/success", http.StatusSeeOther)
}
```

## Shared Data

### Backend

```go
// Share data for all requests
h.inertia.Share("auth", map[string]interface{}{
    "user": user,
    "organization": org,
})

// Flash messages
h.inertia.Share("flash", map[string]string{
    "success": "Operation successful",
})
```

### Frontend

```jsx
import { usePage } from '@inertiajs/react'

export default function Component() {
  const { auth, flash } = usePage().props
  
  return (
    <div>
      {flash.success && <div className="alert">{flash.success}</div>}
      <p>Welcome, {auth.user.email}</p>
    </div>
  )
}
```

## Authentication

The application uses cookies for authentication:

1. Login sets an `auth_token` cookie
2. Auth middleware validates the token
3. User data is shared via Inertia props

## Directory Structure

```
finance-tracker/
├── src/
│   ├── web/
│   │   ├── handlers/         # Inertia page & API handlers
│   │   ├── templates/        # HTML template for Inertia
│   │   └── static/build/     # Production build output
│   └── internal/
│       └── config/
│           └── inertia.go    # Inertia configuration
├── resources/
│   ├── js/
│   │   ├── app.jsx          # Inertia initialization
│   │   ├── Pages/           # Page components
│   │   ├── Components/      # Reusable components
│   │   └── Layouts/         # Layout components
│   └── css/
│       └── app.css          # Tailwind styles
├── package.json             # Node dependencies
├── vite.config.js          # Vite configuration
└── tailwind.config.js      # Tailwind configuration
```

## Common Patterns

### Loading States

```jsx
import { router } from '@inertiajs/react'

// Show loading during navigation
router.on('start', () => NProgress.start())
router.on('finish', () => NProgress.done())
```

### Preserving Scroll Position

```jsx
import { Link } from '@inertiajs/react'

<Link href="/page" preserveScroll>
  Keep scroll position
</Link>
```

### Partial Reloads

```jsx
import { router } from '@inertiajs/react'

// Only reload specific props
router.reload({ only: ['transactions'] })
```

## Troubleshooting

### Assets not loading in development

- Ensure Vite is running on port 5173
- Check that `IsDevelopment` is set correctly
- Verify the app template paths are correct

### Page not found errors

- Ensure the page component exists in `resources/js/Pages/`
- Check that the component name matches the render call
- Verify the page is exported as default

### Form validation not working

- Return status 422 for validation errors
- Use `h.inertia.Back()` to preserve form state
- Ensure errors are shared before calling Back()

## Resources

- [Inertia.js Documentation](https://inertiajs.com)
- [Gonertia Documentation](https://github.com/romsar/gonertia)
- [React Documentation](https://react.dev)
- [Vite Documentation](https://vitejs.dev)