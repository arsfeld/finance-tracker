# GitHub Actions Workflows

This directory contains automated CI/CD workflows for the Finaro finance tracker application.

## 🚀 Workflows Overview

### 1. Landing Page Deployment (`deploy-landing.yml`)
**Triggers:** Push to `main`/`product`, PR to `main`, changes to `landing/` directory
- ✅ HTML optimization and minification
- 🔍 HTML validation and accessibility testing
- 🚀 Deploy to Cloudflare Pages
- 📊 Lighthouse performance auditing
- 🔒 Security scanning with Trivy

### 2. Frontend UI Deployment (`deploy-frontend.yml`)
**Triggers:** Push to `main`/`product`, PR to `main`, changes to frontend files
- 🧪 TypeScript type checking and testing
- 🎨 ESLint and Prettier formatting
- 🏗️ Vite build optimization
- 🚀 Deploy to Cloudflare Pages (staging + production)
- 📦 Bundle size analysis
- 🔒 Security scanning and vulnerability checks

### 3. Backend API Deployment (`deploy-backend.yml`)
**Triggers:** Push to `main`/`product`, PR to `main`, changes to `src/` directory
- 🧪 Go testing with race detection
- 🔍 Go vet and golangci-lint
- 🐳 Docker image building
- 🚀 Deploy to Fly.io (staging + production)
- 👷 Worker process deployment
- 🔄 Automatic rollback on failure
- 📈 Performance monitoring and load testing

### 4. Database Migrations (`database-migrations.yml`)
**Triggers:** Push to `main`/`product`, changes to `supabase/migrations/`, manual dispatch
- ✅ Migration file validation
- 🧪 Local migration testing
- 🗄️ Staging and production deployment
- 💾 Automatic backup creation
- 🔄 Emergency rollback procedures
- 📚 Documentation generation

### 5. Testing and Linting (`test-and-lint.yml`)
**Triggers:** Push to any branch, PR to `main`/`product`
- 🎯 Frontend: TypeScript, ESLint, Prettier, Vitest
- 🎯 Backend: Go vet, golangci-lint, testing
- 🔒 Security: Trivy, Gosec, npm audit, secret scanning
- 📊 Code quality: SonarCloud analysis
- 📦 Dependency vulnerability checking
- 🏎️ Performance benchmarking
- 📖 Documentation link checking and spell check

## 🔧 Required Secrets

Configure these secrets in your GitHub repository settings:

### Cloudflare Pages
```
CLOUDFLARE_API_TOKEN=your_cloudflare_api_token
CLOUDFLARE_ACCOUNT_ID=your_cloudflare_account_id
CLOUDFLARE_ZONE_ID=your_cloudflare_zone_id
```

### Fly.io Deployment
```
FLY_API_TOKEN=your_fly_api_token
```

### Supabase
```
SUPABASE_ACCESS_TOKEN=your_supabase_access_token
SUPABASE_PROJECT_REF=your_production_project_ref
SUPABASE_PROJECT_REF_STAGING=your_staging_project_ref
```

### Database URLs
```
DATABASE_URL=postgresql://user:pass@host:5432/production_db
DATABASE_URL_STAGING=postgresql://user:pass@host:5432/staging_db
```

### Environment Variables (Frontend)
```
VITE_SUPABASE_URL=your_supabase_url
VITE_SUPABASE_ANON_KEY=your_supabase_anon_key
VITE_SUPABASE_URL_STAGING=your_staging_supabase_url
VITE_SUPABASE_ANON_KEY_STAGING=your_staging_supabase_anon_key
```

### Code Quality & Security
```
SONAR_TOKEN=your_sonarcloud_token
CODECOV_TOKEN=your_codecov_token
```

### Notifications (Optional)
```
SLACK_WEBHOOK_URL=your_slack_webhook_url
```

## 🌍 Environments

The workflows use GitHub Environments for deployment protection:

### Staging Environment
- **Protection Rules:** None (automatic deployment)
- **Used by:** `product` branch, pull requests
- **Purpose:** Testing and validation

### Production Environment
- **Protection Rules:** 
  - Required reviewers (recommended)
  - Branch protection (main only)
  - Deployment timeout (30 minutes)
- **Used by:** `main` branch only
- **Purpose:** Live production deployment

## 📊 Workflow Dependencies

```mermaid
graph TD
    A[test-and-lint] --> B[deploy-frontend]
    A --> C[deploy-backend]
    A --> D[database-migrations]
    B --> E[deploy-landing]
    C --> F[performance-monitoring]
    D --> G[migration-docs]
```

## 🔄 Deployment Flow

### Development Flow
1. **Feature Branch** → Runs `test-and-lint`
2. **PR to Product** → Runs all workflows with staging deployment
3. **Merge to Product** → Deploys to staging environment
4. **PR to Main** → Final validation
5. **Merge to Main** → Production deployment

### Hotfix Flow
1. **Hotfix Branch from Main** → Emergency testing
2. **Direct Merge to Main** → Immediate production deployment
3. **Automatic Rollback** → On deployment failure

## 🚨 Monitoring and Alerts

### Health Checks
- **Frontend:** Lighthouse audits, accessibility testing
- **Backend:** Load testing with k6, health endpoints
- **Database:** Migration validation, backup verification

### Alert Channels
- **Slack:** Critical alerts and deployment notifications
- **GitHub:** Security scan results, code quality reports
- **Email:** Failure notifications (via GitHub)

## 🛠️ Local Development

### Running Tests Locally
```bash
# Frontend tests
npm run test
npm run lint
npm run type-check

# Backend tests
go test ./src/...
golangci-lint run ./src/...

# Security scans
trivy fs .
gosec ./src/...
```

### Local Deployment Testing
```bash
# Test frontend build
npm run build
npm run preview

# Test backend build
docker build -t finaro-api .
docker run -p 8080:8080 finaro-api

# Test database migrations
supabase db reset
supabase db push
```

## 📝 Configuration Files

### Frontend
- `.eslintrc.json` - ESLint configuration
- `.prettierrc` - Prettier formatting rules
- `vitest.config.ts` - Testing configuration
- `.lighthouserc.json` - Lighthouse audit settings

### Backend
- `.golangci.yml` - Go linting configuration
- `fly.toml` - Fly.io deployment configuration
- `Dockerfile` - Container build instructions

### Security
- `.audit-ci.json` - npm audit configuration
- `cspell.json` - Spell checking configuration
- `.markdown-link-check.json` - Link validation

### Code Quality
- `sonar-project.properties` - SonarCloud analysis
- `codecov.yml` - Code coverage reporting

## 🔄 Troubleshooting

### Common Issues

1. **Deployment Failures**
   - Check secrets configuration
   - Verify environment variables
   - Review deployment logs

2. **Test Failures**
   - Run tests locally first
   - Check for flaky tests
   - Verify test data and mocks

3. **Security Scan Failures**
   - Update dependencies
   - Address vulnerability findings
   - Review security policies

4. **Performance Issues**
   - Check bundle sizes
   - Review Lighthouse reports
   - Monitor response times

### Manual Interventions

```bash
# Force deployment
gh workflow run deploy-backend.yml --ref main

# Manual migration
gh workflow run database-migrations.yml -f environment=production -f direction=up

# Emergency rollback
gh workflow run deploy-backend.yml # Automatic rollback on failure
```

## 📚 Best Practices

1. **Branch Protection**
   - Require PR reviews
   - Require status checks
   - Restrict direct pushes to main

2. **Secret Management**
   - Use environment-specific secrets
   - Rotate secrets regularly
   - Never log secrets in workflows

3. **Testing Strategy**
   - Test locally before pushing
   - Use staging environment for integration tests
   - Monitor test coverage metrics

4. **Deployment Safety**
   - Always backup before migrations
   - Use blue-green deployments
   - Implement automatic rollbacks

This comprehensive CI/CD setup ensures reliable, secure, and automated deployments for the Finaro finance tracker application.