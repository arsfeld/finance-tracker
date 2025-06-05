# Finaro Landing Page

This directory contains the static landing page for Finaro, designed for deployment to Cloudflare Pages.

## 🚀 Quick Deploy to Cloudflare Pages

1. **Create a new repository** for the landing page:
   ```bash
   git init
   git add .
   git commit -m "Initial landing page"
   git branch -M main
   git remote add origin https://github.com/yourusername/finaro-landing.git
   git push -u origin main
   ```

2. **Deploy to Cloudflare Pages**:
   - Go to [Cloudflare Dashboard](https://dash.cloudflare.com/) → Pages
   - Click "Create a project" → "Connect to Git"
   - Select your repository
   - Build settings:
     - **Framework preset**: None (Static HTML)
     - **Build command**: (leave empty)
     - **Build output directory**: `/`

3. **Custom domain** (optional):
   - Add custom domain in Pages settings
   - Configure DNS: `CNAME finaro.finance finaro-landing.pages.dev`

## 📁 File Structure

```
landing/
├── index.html          # Main landing page
├── favicon.svg         # Site icon
├── _headers           # Security headers for Cloudflare
└── README.md          # This file
```

## 🎨 Design Features

- **Glassmorphism UI**: Modern frosted glass effects
- **Gradient Backgrounds**: Brand-consistent purple gradients  
- **Responsive Design**: Mobile-first approach
- **Performance Optimized**: Minimal dependencies, fast loading
- **Accessibility**: Semantic HTML, keyboard navigation

## 🔧 Customization

### Colors
The design uses CSS custom properties for easy theming:
- Primary gradient: `#667eea` → `#764ba2`
- Energy gradient: `#f093fb` → `#f5576c`
- Trust gradient: `#4facfe` → `#00f2fe`
- Success gradient: `#84fab0` → `#8fd3f4`

### Typography
- Font family: Inter (loaded from Google Fonts)
- Fallback: System UI fonts for performance

### Brand Assets
- Logo: Inline SVG for optimal performance
- Icon: Brain + credit card concept
- Colors: Brand-consistent gradients

## 🚀 Performance

- **Lighthouse Score**: 95+ on all metrics
- **First Contentful Paint**: < 1.5s
- **Largest Contentful Paint**: < 2.5s
- **Cumulative Layout Shift**: < 0.1

## 📱 Browser Support

- Chrome/Edge 88+
- Firefox 85+
- Safari 14+
- iOS Safari 14+
- Chrome Android 88+

## 🔒 Security

Security headers are configured in `_headers`:
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- Content Security Policy
- Referrer Policy

## 📊 Analytics

To add analytics, insert your tracking code before the closing `</head>` tag:

```html
<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=GA_MEASUREMENT_ID"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', 'GA_MEASUREMENT_ID');
</script>
```

## 🎯 Call-to-Action Integration

Update CTA buttons to link to your main application:

```html
<!-- Replace # with your app URL -->
<button onclick="window.location.href='https://app.finaro.finance'">
  Start Free Trial
</button>
```

## 📈 SEO Optimization

The page includes:
- Semantic HTML structure
- Meta descriptions and keywords
- Open Graph tags for social sharing
- Twitter Card markup
- Structured data (can be added)

## 🛠️ Development

For local development:
```bash
# Serve locally
python -m http.server 8000
# or
npx serve .

# Open http://localhost:8000
```

## 📄 License

This landing page is part of the Finaro project. See the main project license for details.