# Deployment Guide

This document provides instructions for deploying the Finaro application components to various cloud platforms.

## 🌐 Landing Page Deployment (Cloudflare Pages)

The landing page is a static HTML file optimized for Cloudflare Pages deployment.

### Prerequisites
- Cloudflare account
- Git repository containing the landing page

### Setup Instructions

1. **Prepare the Landing Page Repository**
   ```bash
   # Create a new repository for the landing page
   mkdir finaro-landing
   cd finaro-landing
   git init
   
   # Copy the landing page files
   cp /path/to/finance-tracker/landing/index.html .
   
   # Add favicon and assets
   mkdir assets
   # Add your favicon.svg, favicon.png, and og-image.png files
   
   git add .
   git commit -m "Initial landing page"
   git branch -M main
   git remote add origin https://github.com/yourusername/finaro-landing.git
   git push -u origin main
   ```

2. **Deploy to Cloudflare Pages**
   - Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
   - Navigate to **Pages** in the sidebar
   - Click **"Create a project"**
   - Choose **"Connect to Git"**
   - Select your landing page repository
   - Configure build settings:
     - **Framework preset**: None (Static HTML)
     - **Build command**: (leave empty)
     - **Build output directory**: `/`
     - **Root directory**: `/`

3. **Custom Domain Setup (Optional)**
   - In your Pages project, go to **Custom domains**
   - Click **"Set up a custom domain"**
   - Enter your domain (e.g., `finaro.finance`)
   - Follow DNS configuration instructions

4. **Performance Optimizations**
   - Enable **Auto Minify** for HTML, CSS, and JS
   - Enable **Brotli** compression
   - Set up **Cache Rules** for static assets

### Environment Variables
No environment variables are needed for the static landing page.

### Custom Headers (Optional)
Add these headers in your `_headers` file for security:

```
/*
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: camera=(), microphone=(), geolocation=()
```

---

## 🚀 Frontend UI Deployment (Cloudflare Pages)

The React/TypeScript frontend can be deployed to Cloudflare Pages with Vite build optimization.

### Prerequisites
- Cloudflare account
- Node.js 18+ and npm
- Git repository containing the frontend code

### Setup Instructions

1. **Prepare Build Configuration**
   
   Create a `wrangler.toml` file in the project root:
   ```toml
   name = "finaro-ui"
   compatibility_date = "2024-01-01"
   
   [env.production]
   vars = { NODE_ENV = "production" }
   
   [[env.production.routes]]
   pattern = "app.finaro.finance/*"
   ```

2. **Update Vite Configuration**
   
   Modify `vite.config.ts` for production builds:
   ```typescript
   import { defineConfig } from 'vite'
   import react from '@vitejs/plugin-react'
   import path from 'path'
   
   export default defineConfig({
     plugins: [react()],
     resolve: {
       alias: {
         '@': path.resolve(__dirname, './resources/js'),
       },
     },
     build: {
       outDir: 'dist',
       assetsDir: 'assets',
       sourcemap: false,
       rollupOptions: {
         output: {
           manualChunks: {
             vendor: ['react', 'react-dom'],
             router: ['@inertiajs/react'],
           },
         },
       },
     },
     base: '/',
   })
   ```

3. **Deploy to Cloudflare Pages**
   - Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
   - Navigate to **Pages** in the sidebar
   - Click **"Create a project"**
   - Choose **"Connect to Git"**
   - Select your frontend repository
   - Configure build settings:
     - **Framework preset**: Vite
     - **Build command**: `npm run build`
     - **Build output directory**: `dist`
     - **Root directory**: `/`
     - **Node.js version**: 18

4. **Environment Variables**
   Set these in Cloudflare Pages dashboard:
   ```
   VITE_API_URL=https://api.finaro.finance
   VITE_SUPABASE_URL=your_supabase_url
   VITE_SUPABASE_ANON_KEY=your_supabase_anon_key
   NODE_ENV=production
   ```

5. **Custom Domain Setup**
   - Add custom domain: `app.finaro.finance`
   - Configure DNS records as instructed

### Build Optimizations

1. **Preact Compatibility** (Optional size reduction):
   ```bash
   npm install --save-dev @preact/preset-vite
   ```
   
   Update `vite.config.ts`:
   ```typescript
   import preact from '@preact/preset-vite'
   
   export default defineConfig({
     plugins: [preact()],
     // ... rest of config
   })
   ```

2. **Bundle Analysis**:
   ```bash
   npm install --save-dev rollup-plugin-visualizer
   npm run build -- --analyze
   ```

---

## ☁️ Backend Deployment (Fly.io)

The Go backend service can be deployed to Fly.io with PostgreSQL database.

### Prerequisites
- Fly.io account and CLI installed
- Docker installed locally
- PostgreSQL database (Supabase or Fly PostgreSQL)

### Setup Instructions

1. **Install Fly CLI**
   ```bash
   # macOS
   brew install flyctl
   
   # Linux
   curl -L https://fly.io/install.sh | sh
   
   # Windows
   iwr https://fly.io/install.ps1 -useb | iex
   ```

2. **Initialize Fly Application**
   ```bash
   cd /path/to/finance-tracker
   fly auth login
   fly launch --no-deploy
   ```

3. **Create Dockerfile**
   
   Create `Dockerfile` in project root:
   ```dockerfile
   # Build stage
   FROM golang:1.21-alpine AS builder
   
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   
   COPY src/ ./src/
   RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./src
   
   # Production stage
   FROM alpine:latest
   
   RUN apk --no-cache add ca-certificates tzdata
   WORKDIR /root/
   
   COPY --from=builder /app/main .
   
   EXPOSE 8080
   CMD ["./main", "web"]
   ```

4. **Configure fly.toml**
   
   Update the generated `fly.toml`:
   ```toml
   app = "finaro-api"
   primary_region = "iad"
   
   [build]
   
   [env]
     PORT = "8080"
     GO_ENV = "production"
   
   [http_service]
     internal_port = 8080
     force_https = true
     auto_stop_machines = true
     auto_start_machines = true
     min_machines_running = 1
     processes = ["app"]
   
   [[http_service.checks]]
     interval = "10s"
     timeout = "2s"
     grace_period = "5s"
     method = "GET"
     path = "/health"
   
   [http_service.concurrency]
     type = "connections"
     hard_limit = 1000
     soft_limit = 1000
   
   [[vm]]
     memory = "1gb"
     cpu_kind = "shared"
     cpus = 1
   
   [processes]
     app = "./main web"
     worker = "./main worker"
   ```

5. **Set Environment Variables**
   ```bash
   # Database
   fly secrets set DATABASE_URL="postgresql://user:pass@host:5432/dbname"
   
   # Supabase
   fly secrets set SUPABASE_URL="your_supabase_url"
   fly secrets set SUPABASE_SERVICE_KEY="your_service_key"
   fly secrets set SUPABASE_JWT_SECRET="your_jwt_secret"
   
   # API Keys
   fly secrets set OPENROUTER_API_KEY="your_openrouter_key"
   fly secrets set SIMPLEFIN_TOKEN="your_simplefin_token"
   
   # Application
   fly secrets set APP_ENV="production"
   fly secrets set APP_SECRET="your_app_secret"
   fly secrets set CORS_ORIGINS="https://app.finaro.finance,https://finaro.finance"
   ```

6. **Deploy Application**
   ```bash
   fly deploy
   ```

7. **Set Up Custom Domain**
   ```bash
   fly certs create api.finaro.finance
   ```

### Database Setup

#### Option 1: Fly PostgreSQL
```bash
fly postgres create --name finaro-db --region iad
fly postgres attach --app finaro-api finaro-db
```

#### Option 2: External Supabase
Use the connection string from your Supabase project.

### Monitoring and Scaling

1. **View Logs**:
   ```bash
   fly logs
   ```

2. **Scale Application**:
   ```bash
   fly scale count 2  # Scale to 2 instances
   fly scale memory 2048  # Scale to 2GB RAM
   ```

3. **Health Checks**:
   The application exposes `/health` endpoint for monitoring.

### Worker Process Setup

For background job processing, add a worker process:

```toml
[processes]
  app = "./main web"
  worker = "./main worker"

[[vm]]
  processes = ["worker"]
  memory = "512mb"
  cpu_kind = "shared"
  cpus = 1
```

Deploy worker separately:
```bash
fly deploy --process-group worker
```

---

## 🔧 CI/CD Pipeline

### GitHub Actions for Automated Deployment

Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy-landing:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Deploy to Cloudflare Pages
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: finaro-landing
          directory: landing

  deploy-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run build
      - name: Deploy to Cloudflare Pages
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: finaro-ui
          directory: dist

  deploy-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

---

## 🌍 Domain Configuration

### DNS Setup

Configure these DNS records for your domain:

```
# Landing page
finaro.finance       CNAME   finaro-landing.pages.dev
www.finaro.finance   CNAME   finaro-landing.pages.dev

# Frontend app
app.finaro.finance   CNAME   finaro-ui.pages.dev

# Backend API
api.finaro.finance   CNAME   finaro-api.fly.dev
```

### SSL/TLS Configuration

Both Cloudflare Pages and Fly.io provide automatic SSL certificates:

- **Cloudflare**: Automatic with Universal SSL
- **Fly.io**: Automatic with Let's Encrypt via `fly certs create`

---

## 📊 Monitoring and Observability

### Cloudflare Analytics
- Enable **Web Analytics** for the landing page
- Set up **Real User Monitoring** for the frontend

### Fly.io Monitoring
- Use `fly logs` for application logs
- Set up **Sentry** or **LogRocket** for error tracking
- Configure **Grafana** dashboard for metrics

### Health Checks

Implement health check endpoints:

```go
// Backend health check
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "version": version.Version,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })
}
```

---

## 🔒 Security Considerations

### Environment Variables
- Never commit secrets to version control
- Use platform-specific secret management
- Rotate API keys regularly

### CORS Configuration
```go
c := cors.New(cors.Options{
    AllowedOrigins: []string{
        "https://finaro.finance",
        "https://app.finaro.finance",
    },
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders: []string{"*"},
    AllowCredentials: true,
})
```

### Content Security Policy
Add CSP headers to prevent XSS attacks:

```
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' cdn.tailwindcss.com; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src 'self' fonts.gstatic.com;
```

This deployment guide provides a comprehensive setup for all Finaro application components across multiple cloud platforms with proper security, monitoring, and CI/CD practices.