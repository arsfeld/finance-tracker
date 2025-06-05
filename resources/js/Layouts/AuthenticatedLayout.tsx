import { Link, usePage } from '@inertiajs/react'
import { useState, useEffect, ReactNode } from 'react'
import { PageProps } from '@/types'

interface AuthenticatedLayoutProps {
  children: ReactNode
}

export default function AuthenticatedLayout({ children }: AuthenticatedLayoutProps) {
  const { user, organization, flash } = usePage<PageProps>().props
  const [showUserMenu, setShowUserMenu] = useState(false)
  const [showSuccessFlash, setShowSuccessFlash] = useState(!!flash?.success)
  const [showErrorFlash, setShowErrorFlash] = useState(!!flash?.error)

  useEffect(() => {
    if (flash?.success) {
      setShowSuccessFlash(true)
      const timer = setTimeout(() => {
        setShowSuccessFlash(false)
      }, 5000) // Auto-dismiss after 5 seconds
      return () => clearTimeout(timer)
    }
  }, [flash?.success])

  useEffect(() => {
    if (flash?.error) {
      setShowErrorFlash(true)
      const timer = setTimeout(() => {
        setShowErrorFlash(false)
      }, 8000) // Error messages stay longer
      return () => clearTimeout(timer)
    }
  }, [flash?.error])

  return (
    <div className="min-h-screen" style={{ background: 'var(--color-bg-primary)' }}>
      {/* Navigation */}
      <nav className="nav-glass sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              {/* Logo */}
              <div className="flex-shrink-0 flex items-center">
                <Link href="/dashboard" className="flex items-center space-x-2">
                  {/* Finaro Icon */}
                  <svg viewBox="0 0 40 40" xmlns="http://www.w3.org/2000/svg" width="32" height="32">
                    <defs>
                      <linearGradient id="iconGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" style={{ stopColor: '#667eea', stopOpacity: 1 }} />
                        <stop offset="100%" style={{ stopColor: '#764ba2', stopOpacity: 1 }} />
                      </linearGradient>
                    </defs>
                    
                    <g transform="translate(20, 20)">
                      <path d="M -10,-7.5 Q -12.5,-10 -7.5,-12.5 L -5,-12.5 L -2.5,-15 L 2.5,-15 L 5,-12.5 L 7.5,-12.5 
                               Q 12.5,-10 10,-7.5 L 10,-5 L 12.5,-2.5 L 12.5,2.5 L 10,5 L 10,7.5 
                               Q 12.5,10 7.5,10 L 5,10 L 2.5,12.5 L -2.5,12.5 L -5,10 L -7.5,10 
                               Q -12.5,7.5 -10,5 L -10,2.5 L -12.5,0 L -12.5,-5 L -10,-7.5 Z" 
                            fill="none" 
                            stroke="url(#iconGradient)" 
                            strokeWidth="1.5"/>
                      
                      <rect x="-6" y="-3.75" width="12" height="7.5" rx="1" ry="1" 
                            fill="rgba(102,126,234,0.1)" 
                            stroke="url(#iconGradient)" 
                            strokeWidth="0.75"/>
                      
                      <rect x="-4.5" y="-2.25" width="2.5" height="2" rx="0.25" ry="0.25" 
                            fill="url(#iconGradient)"/>
                    </g>
                  </svg>
                  <span className="text-xl font-extrabold gradient-text">
                    Finaro
                  </span>
                </Link>
              </div>

              {/* Navigation Links */}
              <div className="hidden space-x-8 sm:-my-px sm:ml-10 sm:flex">
                <Link
                  href="/dashboard"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-semibold text-purple-700 hover:text-purple-900 hover:border-purple-500 transition-all duration-300"
                >
                  Dashboard
                </Link>
                <Link
                  href="/transactions"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-semibold text-purple-700 hover:text-purple-900 hover:border-purple-500 transition-all duration-300"
                >
                  Transactions
                </Link>
                <Link
                  href="/accounts"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-semibold text-purple-700 hover:text-purple-900 hover:border-purple-500 transition-all duration-300"
                >
                  Accounts
                </Link>
                <Link
                  href="/analytics"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-semibold text-purple-700 hover:text-purple-900 hover:border-purple-500 transition-all duration-300"
                >
                  Analytics
                </Link>
                <Link
                  href="/settings"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-semibold text-purple-700 hover:text-purple-900 hover:border-purple-500 transition-all duration-300"
                >
                  Settings
                </Link>
              </div>
            </div>

            {/* User Menu */}
            <div className="hidden sm:ml-6 sm:flex sm:items-center">
              {organization && (
                <span className="mr-4 text-sm font-medium text-purple-700 glass-secondary px-3 py-1 rounded-full">
                  {organization.name}
                </span>
              )}
              
              <div className="ml-3 relative">
                <button
                  onClick={() => setShowUserMenu(!showUserMenu)}
                  className="flex text-sm rounded-full focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 focus:ring-offset-transparent"
                >
                  <span className="inline-flex items-center px-4 py-2 text-sm leading-4 font-semibold rounded-full text-purple-700 glass-secondary hover:bg-white/60 transition-all duration-300">
                    {user?.email}
                    <svg className="ml-2 -mr-0.5 h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </span>
                </button>

                {showUserMenu && (
                  <div className="origin-top-right absolute right-0 mt-2 w-48 rounded-2xl shadow-glass overflow-hidden glass">
                    <Link
                      href="/organizations"
                      className="block px-4 py-3 text-sm font-medium text-gray-700 hover:bg-white/50 transition-all duration-300"
                    >
                      Switch Organization
                    </Link>
                    <Link
                      href="/auth/logout"
                      method="post"
                      as="button"
                      className="block w-full text-left px-4 py-3 text-sm font-medium text-gray-700 hover:bg-white/50 transition-all duration-300"
                    >
                      Sign Out
                    </Link>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </nav>

      {/* Flash Messages */}
      <div className="fixed top-20 right-4 z-50 space-y-2">
        {flash?.success && showSuccessFlash && (
          <div 
            className="transform transition-all duration-300 ease-out"
            style={{
              animation: 'slideIn 0.3s ease-out',
              opacity: showSuccessFlash ? 1 : 0,
            }}
          >
            <div className="rounded-xl p-4 pr-12 shadow-lg relative max-w-md" style={{ background: 'var(--color-success-gradient)' }}>
              <p className="text-white font-medium">{flash.success}</p>
              <button
                onClick={() => setShowSuccessFlash(false)}
                className="absolute top-4 right-4 text-white/80 hover:text-white transition-colors"
                aria-label="Dismiss"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        )}
        {flash?.error && showErrorFlash && (
          <div 
            className="transform transition-all duration-300 ease-out"
            style={{
              animation: 'slideIn 0.3s ease-out',
              opacity: showErrorFlash ? 1 : 0,
            }}
          >
            <div className="rounded-xl p-4 pr-12 shadow-lg relative max-w-md" style={{ background: 'var(--color-danger-gradient)' }}>
              <p className="text-white font-medium">{flash.error}</p>
              <button
                onClick={() => setShowErrorFlash(false)}
                className="absolute top-4 right-4 text-white/80 hover:text-white transition-colors"
                aria-label="Dismiss"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        )}
      </div>

      <style jsx>{`
        @keyframes slideIn {
          from {
            transform: translateX(100%);
            opacity: 0;
          }
          to {
            transform: translateX(0);
            opacity: 1;
          }
        }
      `}</style>

      {/* Main Content */}
      <main className="p-2 sm:p-3">
        <div className="max-w-7xl mx-auto">
          {children}
        </div>
      </main>
    </div>
  )
}