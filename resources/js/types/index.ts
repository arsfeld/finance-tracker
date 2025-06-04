// Common types for the application
import { PageProps as InertiaPageProps } from '@inertiajs/core'

export interface User {
  id: string
  email: string
  email_verified?: boolean
  created_at?: string
  updated_at?: string
}

export interface Organization {
  id: string
  name: string
  settings?: Record<string, any>
  created_at?: string
  updated_at?: string
}

export interface PageProps extends InertiaPageProps {
  title?: string
  user?: User
  organization?: Organization
  flash?: {
    success?: string
    error?: string
  }
  errors?: Record<string, string[]>
  [key: string]: any
}

export interface AuthPageProps extends PageProps {
  // Additional props for auth pages
}

export interface DashboardPageProps extends PageProps {
  user: User
  organization?: Organization
}

export interface TransactionPageProps extends PageProps {
  user: User
  organization?: Organization
}

export interface AccountPageProps extends PageProps {
  user: User
  organization?: Organization
  accountId?: string
}

export interface AnalyticsPageProps extends PageProps {
  user: User
  organization?: Organization
}

export interface OrganizationPageProps extends PageProps {
  organizations?: Organization[]
  organizationId?: string
  members?: any[]
}