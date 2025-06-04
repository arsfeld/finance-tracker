import { Link, usePage } from '@inertiajs/react'
import { useState, ReactNode } from 'react'
import { PageProps } from '@/types'

interface AuthenticatedLayoutProps {
  children: ReactNode
}

export default function AuthenticatedLayout({ children }: AuthenticatedLayoutProps) {
  const { user, organization, flash } = usePage<PageProps>().props
  const [showUserMenu, setShowUserMenu] = useState(false)

  return (
    <div className="min-h-screen bg-background">
      {/* Navigation */}
      <nav className="bg-card shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              {/* Logo */}
              <div className="flex-shrink-0 flex items-center">
                <Link href="/dashboard" className="text-xl font-bold text-primary hover:text-primary/90 transition-colors">
                  WalletMind
                </Link>
              </div>

              {/* Navigation Links */}
              <div className="hidden space-x-8 sm:-my-px sm:ml-10 sm:flex">
                <Link
                  href="/dashboard"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground transition-colors"
                >
                  Dashboard
                </Link>
                <Link
                  href="/transactions"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground transition-colors"
                >
                  Transactions
                </Link>
                <Link
                  href="/accounts"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground transition-colors"
                >
                  Accounts
                </Link>
                <Link
                  href="/analytics"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground transition-colors"
                >
                  Analytics
                </Link>
                <Link
                  href="/settings"
                  className="inline-flex items-center px-1 pt-1 border-b-2 border-transparent text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground transition-colors"
                >
                  Settings
                </Link>
              </div>
            </div>

            {/* User Menu */}
            <div className="hidden sm:ml-6 sm:flex sm:items-center">
              {organization && (
                <span className="mr-4 text-sm text-muted-foreground">
                  {organization.name}
                </span>
              )}
              
              <div className="ml-3 relative">
                <button
                  onClick={() => setShowUserMenu(!showUserMenu)}
                  className="flex text-sm rounded-full focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-ring"
                >
                  <span className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-muted-foreground bg-background hover:text-foreground focus:outline-none transition-colors duration-150">
                    {user?.email}
                    <svg className="ml-2 -mr-0.5 h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </span>
                </button>

                {showUserMenu && (
                  <div className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg py-1 bg-popover ring-1 ring-border">
                    <Link
                      href="/organizations"
                      className="block px-4 py-2 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
                    >
                      Switch Organization
                    </Link>
                    <Link
                      href="/auth/logout"
                      method="post"
                      as="button"
                      className="block w-full text-left px-4 py-2 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
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
      {flash?.success && (
        <div className="bg-success/10 border-l-4 border-success text-success p-4">
          <p>{flash.success}</p>
        </div>
      )}
      {flash?.error && (
        <div className="bg-destructive/10 border-l-4 border-destructive text-destructive p-4">
          <p>{flash.error}</p>
        </div>
      )}

      {/* Main Content */}
      <main>{children}</main>
    </div>
  )
}