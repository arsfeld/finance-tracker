# Finaro Design System

## 🎨 Design Tokens

### Colors

#### Primary Colors
```css
/* Intelligence Gradient */
--color-primary-start: #667eea;
--color-primary-end: #764ba2;
--color-primary-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

/* Energy Gradient */
--color-secondary-start: #f093fb;
--color-secondary-end: #f5576c;
--color-secondary-gradient: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);

/* Trust Gradient */
--color-accent-start: #4facfe;
--color-accent-end: #00f2fe;
--color-accent-gradient: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
```

#### State Colors
```css
/* Success */
--color-success-start: #84fab0;
--color-success-end: #8fd3f4;
--color-success-gradient: linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%);

/* Warning */
--color-warning-start: #ffecd2;
--color-warning-end: #fcb69f;
--color-warning-gradient: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%);

/* Danger */
--color-danger-start: #ff9a9e;
--color-danger-end: #fecfef;
--color-danger-gradient: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
```

#### Neutral Colors
```css
/* Text Colors */
--color-text-primary: rgba(26, 32, 44, 0.9);
--color-text-secondary: rgba(26, 32, 44, 0.7);
--color-text-tertiary: rgba(26, 32, 44, 0.5);
--color-text-inverse: rgba(255, 255, 255, 0.9);

/* Background Colors */
--color-bg-primary: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
--color-bg-glass-primary: rgba(255, 255, 255, 0.35);
--color-bg-glass-secondary: rgba(255, 255, 255, 0.25);
--color-bg-glass-tertiary: rgba(255, 255, 255, 0.15);

/* Border Colors */
--color-border-glass: rgba(255, 255, 255, 0.4);
--color-border-glass-light: rgba(255, 255, 255, 0.2);
```

### Typography

#### Font Family
```css
--font-family-primary: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
```

#### Font Weights
```css
--font-weight-regular: 400;
--font-weight-semibold: 600;
--font-weight-bold: 700;
--font-weight-extrabold: 800;
```

#### Font Sizes
```css
--font-size-xs: 0.75rem;    /* 12px */
--font-size-sm: 0.875rem;   /* 14px */
--font-size-base: 1rem;     /* 16px */
--font-size-lg: 1.125rem;   /* 18px */
--font-size-xl: 1.25rem;    /* 20px */
--font-size-2xl: 1.5rem;    /* 24px */
--font-size-3xl: 1.875rem;  /* 30px */
--font-size-4xl: 2.25rem;   /* 36px */
--font-size-5xl: 3rem;      /* 48px */
```

#### Line Heights
```css
--line-height-tight: 1.2;
--line-height-snug: 1.3;
--line-height-normal: 1.4;
--line-height-relaxed: 1.5;
--line-height-loose: 1.6;
```

### Spacing

#### Scale
```css
--spacing-xs: 0.25rem;   /* 4px */
--spacing-sm: 0.5rem;    /* 8px */
--spacing-md: 0.75rem;   /* 12px */
--spacing-lg: 1rem;      /* 16px */
--spacing-xl: 1.5rem;    /* 24px */
--spacing-2xl: 2rem;     /* 32px */
--spacing-3xl: 3rem;     /* 48px */
--spacing-4xl: 4rem;     /* 64px */
--spacing-5xl: 6rem;     /* 96px */
```

### Border Radius
```css
--radius-sm: 8px;
--radius-md: 12px;
--radius-lg: 16px;
--radius-xl: 20px;
--radius-2xl: 24px;
--radius-3xl: 32px;
```

### Shadows

#### Drop Shadows
```css
--shadow-sm: 0 4px 8px rgba(0, 0, 0, 0.1);
--shadow-md: 0 8px 16px rgba(0, 0, 0, 0.1);
--shadow-lg: 0 12px 24px rgba(0, 0, 0, 0.1);
--shadow-xl: 0 20px 40px rgba(0, 0, 0, 0.1);

/* Brand Specific Shadows */
--shadow-primary: 0 8px 16px rgba(102, 126, 234, 0.3);
--shadow-secondary: 0 8px 16px rgba(240, 147, 251, 0.3);
--shadow-glass: 0 20px 40px rgba(0, 0, 0, 0.1), 0 8px 32px rgba(0, 0, 0, 0.08);
```

### Effects

#### Glassmorphism
```css
--glass-blur: blur(20px);
--glass-blur-light: blur(10px);
--glass-border: 1px solid rgba(255, 255, 255, 0.4);
--glass-border-light: 1px solid rgba(255, 255, 255, 0.2);
```

---

## 🧩 Components

### Buttons

#### Primary Button
```css
.btn-primary {
  background: var(--color-primary-gradient);
  color: var(--color-text-inverse);
  padding: 12px 24px;
  border-radius: var(--radius-md);
  border: none;
  font-weight: var(--font-weight-semibold);
  font-size: var(--font-size-base);
  box-shadow: var(--shadow-primary);
  transition: all 0.3s ease;
  cursor: pointer;
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(102, 126, 234, 0.4);
}
```

#### Secondary Button
```css
.btn-secondary {
  background: var(--color-secondary-gradient);
  color: var(--color-text-inverse);
  padding: 12px 24px;
  border-radius: var(--radius-md);
  border: none;
  font-weight: var(--font-weight-semibold);
  font-size: var(--font-size-base);
  box-shadow: var(--shadow-secondary);
  transition: all 0.3s ease;
  cursor: pointer;
}
```

#### Outline Button
```css
.btn-outline {
  background: var(--color-bg-glass-secondary);
  color: var(--color-text-primary);
  padding: 12px 24px;
  border-radius: var(--radius-md);
  border: 2px solid rgba(255, 255, 255, 0.5);
  font-weight: var(--font-weight-semibold);
  font-size: var(--font-size-base);
  backdrop-filter: var(--glass-blur-light);
  transition: all 0.3s ease;
  cursor: pointer;
}
```

### Cards

#### Glass Card
```css
.card-glass {
  background: var(--color-bg-glass-primary);
  backdrop-filter: var(--glass-blur);
  border-radius: var(--radius-2xl);
  border: var(--glass-border);
  box-shadow: var(--shadow-glass);
  padding: var(--spacing-2xl);
  position: relative;
  overflow: hidden;
}

.card-glass::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.6), transparent);
}
```

#### Financial Card
```css
.card-financial {
  background: var(--color-bg-glass-primary);
  backdrop-filter: var(--glass-blur);
  border-radius: var(--radius-xl);
  padding: var(--spacing-2xl);
  border: var(--glass-border);
  box-shadow: var(--shadow-lg);
}

.card-financial-title {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  margin-bottom: var(--spacing-sm);
}

.card-financial-value {
  font-size: var(--font-size-4xl);
  font-weight: var(--font-weight-bold);
  background: var(--color-primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
```

### Form Elements

#### Input Field
```css
.input {
  width: 100%;
  padding: 16px 20px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.4);
  background: var(--color-bg-glass-secondary);
  backdrop-filter: var(--glass-blur-light);
  color: var(--color-text-primary);
  font-size: var(--font-size-base);
  font-family: var(--font-family-primary);
  transition: all 0.3s ease;
}

.input::placeholder {
  color: var(--color-text-tertiary);
}

.input:focus {
  outline: none;
  border-color: rgba(102, 126, 234, 0.6);
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}
```

#### Select Field
```css
.select {
  width: 100%;
  padding: 16px 20px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.4);
  background: var(--color-bg-glass-secondary);
  backdrop-filter: var(--glass-blur-light);
  color: var(--color-text-primary);
  font-size: var(--font-size-base);
  font-family: var(--font-family-primary);
  cursor: pointer;
}
```

### Navigation

#### Primary Navigation
```css
.nav-primary {
  background: var(--color-bg-glass-primary);
  backdrop-filter: var(--glass-blur);
  border-bottom: var(--glass-border-light);
  padding: var(--spacing-lg) 0;
}

.nav-item {
  color: var(--color-text-secondary);
  text-decoration: none;
  padding: var(--spacing-sm) var(--spacing-lg);
  border-radius: var(--radius-md);
  transition: all 0.3s ease;
  font-weight: var(--font-weight-semibold);
}

.nav-item:hover,
.nav-item.active {
  color: var(--color-text-primary);
  background: var(--color-bg-glass-secondary);
}
```

### Data Visualization

#### Chart Container
```css
.chart-container {
  background: var(--color-bg-glass-primary);
  backdrop-filter: var(--glass-blur);
  border-radius: var(--radius-xl);
  padding: var(--spacing-2xl);
  border: var(--glass-border);
  box-shadow: var(--shadow-lg);
}

.chart-title {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  margin-bottom: var(--spacing-lg);
}
```

#### Progress Bar
```css
.progress-bar {
  width: 100%;
  height: 8px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--color-primary-gradient);
  border-radius: var(--radius-sm);
  transition: width 0.5s ease;
}
```

### Notifications

#### Success Notification
```css
.notification-success {
  background: var(--color-success-gradient);
  color: var(--color-text-inverse);
  padding: var(--spacing-lg) var(--spacing-xl);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  margin-bottom: var(--spacing-lg);
}
```

#### Warning Notification
```css
.notification-warning {
  background: var(--color-warning-gradient);
  color: var(--color-text-primary);
  padding: var(--spacing-lg) var(--spacing-xl);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  margin-bottom: var(--spacing-lg);
}
```

#### Error Notification
```css
.notification-error {
  background: var(--color-danger-gradient);
  color: var(--color-text-inverse);
  padding: var(--spacing-lg) var(--spacing-xl);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  margin-bottom: var(--spacing-lg);
}
```

---

## 📐 Layout System

### Grid System

#### Container
```css
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--spacing-2xl);
}

@media (max-width: 768px) {
  .container {
    padding: 0 var(--spacing-lg);
  }
}
```

#### Grid
```css
.grid {
  display: grid;
  gap: var(--spacing-xl);
}

.grid-2 { grid-template-columns: repeat(2, 1fr); }
.grid-3 { grid-template-columns: repeat(3, 1fr); }
.grid-4 { grid-template-columns: repeat(4, 1fr); }

@media (max-width: 768px) {
  .grid-2,
  .grid-3,
  .grid-4 {
    grid-template-columns: 1fr;
  }
}
```

### Flexbox Utilities

```css
.flex { display: flex; }
.flex-col { flex-direction: column; }
.flex-wrap { flex-wrap: wrap; }

.items-center { align-items: center; }
.items-start { align-items: flex-start; }
.items-end { align-items: flex-end; }

.justify-center { justify-content: center; }
.justify-between { justify-content: space-between; }
.justify-start { justify-content: flex-start; }
.justify-end { justify-content: flex-end; }

.gap-sm { gap: var(--spacing-sm); }
.gap-md { gap: var(--spacing-md); }
.gap-lg { gap: var(--spacing-lg); }
.gap-xl { gap: var(--spacing-xl); }
```

---

## 📱 Responsive Design

### Breakpoints
```css
/* Mobile First Approach */
:root {
  --bp-sm: 480px;   /* Small mobile */
  --bp-md: 768px;   /* Tablet */
  --bp-lg: 1024px;  /* Small desktop */
  --bp-xl: 1280px;  /* Large desktop */
  --bp-2xl: 1536px; /* Extra large */
}
```

### Media Query Mixins
```css
@media (min-width: 480px) { /* sm */ }
@media (min-width: 768px) { /* md */ }
@media (min-width: 1024px) { /* lg */ }
@media (min-width: 1280px) { /* xl */ }
@media (min-width: 1536px) { /* 2xl */ }
```

### Responsive Typography
```css
/* Fluid typography for better mobile experience */
.heading-responsive {
  font-size: clamp(1.5rem, 4vw, 3rem);
}

.body-responsive {
  font-size: clamp(0.875rem, 2vw, 1rem);
}
```

---

## ♿ Accessibility

### Focus States
```css
.focusable:focus {
  outline: 2px solid rgba(102, 126, 234, 0.6);
  outline-offset: 2px;
  box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.1);
}
```

### Color Contrast
- Ensure minimum 4.5:1 contrast ratio for normal text
- Ensure minimum 3:1 contrast ratio for large text
- Test all gradient combinations for accessibility

### Screen Reader Support
```css
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
```

---

## 🚀 Performance

### Critical CSS
Load core design system variables and glassmorphism styles inline for faster initial render.

### Optimization Guidelines
- Use `will-change` property sparingly for animations
- Implement backdrop-filter carefully for performance
- Optimize gradient usage for mobile devices
- Use CSS custom properties for theming flexibility

### Animation Performance
```css
/* Use transform and opacity for smooth animations */
.smooth-animation {
  transition: transform 0.3s ease, opacity 0.3s ease;
  will-change: transform, opacity;
}

/* Remove will-change after animation completes */
.animation-complete {
  will-change: auto;
}
```

---

*This design system provides a comprehensive foundation for building the Finaro interface with consistent, accessible, and performant components that embody the brand's sophisticated AI-powered financial intelligence.*