import { createRoot } from 'react-dom/client'
import { createInertiaApp } from '@inertiajs/react'
import '../css/app.css'

const appName = 'Finaro'

createInertiaApp({
  title: (title: string) => `${title} - ${appName}`,
  resolve: async (name: string) => {
    const pages = import.meta.glob('./Pages/**/*.tsx', { eager: false })
    const pageImport = pages[`./Pages/${name}.tsx`]
    
    if (!pageImport) {
      console.error(`Page component not found: ./Pages/${name}.tsx`)
      console.log('Available pages:', Object.keys(pages))
      throw new Error(`Page component not found: ${name}`)
    }
    
    const page = await pageImport()
    return (page as any).default || page
  },
  setup({ el, App, props }) {
    if (!el) {
      throw new Error('App element not found')
    }
    
    const root = createRoot(el)
    root.render(<App {...props} />)
  },
  progress: {
    color: '#0ea5e9',
  },
})