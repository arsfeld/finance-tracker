import { Head, Link, router } from '@inertiajs/react'
import { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { SyncButton, JobProgress } from '@/components/SyncButton'

interface Connection {
  id: string
  name: string
  provider_type: string
  last_sync?: string
  sync_status: string
  error_message?: string
  created_at: string
}

interface ConnectionAccount {
  id: string
  connection_id: string
  provider_account_id: string
  name: string
  institution?: string
  account_type?: string
  balance?: number
  currency: string
  is_active: boolean
  last_sync?: string
}

export default function Connections() {
  const [connections, setConnections] = useState<Connection[]>([])
  const [accounts, setAccounts] = useState<{ [key: string]: ConnectionAccount[] }>({})
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    provider_type: 'simplefin',
    setup_token: ''
  })
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchConnections()
  }, [])

  const fetchConnections = async () => {
    try {
      const response = await fetch('/api/v1/connections')
      if (response.ok) {
        const data = await response.json()
        setConnections(data)
        
        // Fetch accounts for each connection
        for (const connection of data) {
          fetchConnectionAccounts(connection.id)
        }
      }
    } catch (error) {
      console.error('Failed to fetch connections:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchConnectionAccounts = async (connectionId: string) => {
    try {
      const response = await fetch(`/api/v1/connections/${connectionId}/accounts`)
      if (response.ok) {
        const data = await response.json()
        setAccounts(prev => ({ ...prev, [connectionId]: data }))
      }
    } catch (error) {
      console.error(`Failed to fetch accounts for connection ${connectionId}:`, error)
    }
  }

  const handleAddConnection = async (e: React.FormEvent) => {
    e.preventDefault()
    
    // Client-side validation
    if (!formData.name.trim()) {
      alert('Connection name is required')
      return
    }
    
    if (!formData.setup_token.trim()) {
      alert('SimpleFin Setup Token is required')
      return
    }
    
    setSubmitting(true)

    try {
      const response = await fetch('/api/v1/connections', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({
          name: formData.name,
          provider_type: formData.provider_type,
          setup_token: formData.setup_token
        })
      })

      if (response.ok) {
        setShowAddForm(false)
        setFormData({ name: '', provider_type: 'simplefin', setup_token: '' })
        fetchConnections()
        alert('Connection added successfully!')
      } else {
        const error = await response.text()
        const errorMessage = error || 'Unknown error occurred'
        alert(`Failed to add connection: ${errorMessage}`)
      }
    } catch (error) {
      console.error('Failed to add connection:', error)
      alert('Failed to add connection. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteConnection = async (connectionId: string) => {
    if (!confirm('Are you sure you want to delete this connection? This will also remove all associated account data.')) {
      return
    }

    try {
      const response = await fetch(`/api/v1/connections/${connectionId}`, {
        method: 'DELETE',
        headers: {
          'X-Requested-With': 'XMLHttpRequest',
        }
      })

      if (response.ok) {
        fetchConnections()
      } else {
        alert('Failed to delete connection')
      }
    } catch (error) {
      console.error('Failed to delete connection:', error)
      alert('Failed to delete connection. Please try again.')
    }
  }

  const handleTestConnection = async (connectionId: string) => {
    try {
      const response = await fetch(`/api/v1/connections/${connectionId}/test`, {
        method: 'POST',
        headers: {
          'X-Requested-With': 'XMLHttpRequest',
        }
      })

      const result = await response.json()
      if (result.success) {
        alert('Connection test successful!')
      } else {
        alert(`Connection test failed: ${result.error_message || 'Unknown error'}`)
      }
    } catch (error) {
      console.error('Failed to test connection:', error)
      alert('Failed to test connection. Please try again.')
    }
  }

  const handleToggleAccount = async (accountId: string, isActive: boolean) => {
    try {
      const response = await fetch(`/api/v1/bank-accounts/${accountId}/status`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ is_active: !isActive })
      })

      if (response.ok) {
        // Update local state
        setAccounts(prev => {
          const updated = { ...prev }
          Object.keys(updated).forEach(connectionId => {
            updated[connectionId] = updated[connectionId].map(account =>
              account.id === accountId ? { ...account, is_active: !isActive } : account
            )
          })
          return updated
        })
      } else {
        alert('Failed to update account status')
      }
    } catch (error) {
      console.error('Failed to toggle account:', error)
      alert('Failed to update account. Please try again.')
    }
  }

  const getStatusBadge = (status: string) => {
    const baseClasses = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
    switch (status) {
      case 'connected':
        return `${baseClasses} bg-green-100 text-green-800`
      case 'error':
        return `${baseClasses} bg-red-100 text-red-800`
      default:
        return `${baseClasses} bg-gray-100 text-gray-800`
    }
  }

  if (loading) {
    return (
      <AuthenticatedLayout>
        <Head title="Account Connections" />
        <div className="p-6">
          <div className="max-w-4xl mx-auto">
            <div className="flex justify-center items-center h-64">
              <div className="text-muted-foreground">Loading connections...</div>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <Head title="Account Connections" />
      
      <div className="p-6 space-y-6">
        <div className="max-w-4xl mx-auto">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h1 className="text-3xl font-bold text-foreground">Account Connections</h1>
              <p className="text-muted-foreground mt-1">
                Manage your financial data sources and bank account connections
              </p>
            </div>
            <Button onClick={() => setShowAddForm(true)}>
              Add Connection
            </Button>
          </div>

          {/* Add Connection Form */}
          {showAddForm && (
            <Card className="mb-6">
              <CardHeader>
                <CardTitle>Add SimpleFin Connection</CardTitle>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleAddConnection} className="space-y-4">
                  <div>
                    <Label htmlFor="name">Connection Name</Label>
                    <Input
                      id="name"
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g., My Bank Account"
                      required
                    />
                  </div>
                  <div>
                    <Label htmlFor="setup_token">SimpleFin Setup Token</Label>
                    <Input
                      id="setup_token"
                      type="text"
                      value={formData.setup_token}
                      onChange={(e) => setFormData({ ...formData, setup_token: e.target.value })}
                      placeholder="aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS9ERU1P"
                      required
                    />
                    <div className="text-sm text-muted-foreground mt-1 space-y-1">
                      <p>Get your SimpleFin Setup Token from:</p>
                      <a 
                        href="https://beta-bridge.simplefin.org/simplefin/create" 
                        target="_blank" 
                        rel="noopener noreferrer"
                        className="text-blue-600 hover:text-blue-800 underline"
                      >
                        SimpleFin Bridge Token Generator
                      </a>
                      <p>This token will be exchanged for secure access credentials.</p>
                    </div>
                  </div>
                  <div className="flex space-x-2">
                    <Button type="submit" disabled={submitting}>
                      {submitting ? 'Adding...' : 'Add Connection'}
                    </Button>
                    <Button type="button" variant="outline" onClick={() => setShowAddForm(false)}>
                      Cancel
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          )}

          {/* Connections List */}
          {connections.length === 0 ? (
            <Card>
              <CardContent className="text-center py-8">
                <p className="text-muted-foreground mb-4">
                  No connections configured yet.
                </p>
                <Button onClick={() => setShowAddForm(true)}>
                  Add Your First Connection
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {connections.map((connection) => (
                <Card key={connection.id}>
                  <CardHeader>
                    <div className="flex justify-between items-start">
                      <div>
                        <CardTitle className="flex items-center space-x-2">
                          <span>{connection.name}</span>
                          <span className={getStatusBadge(connection.sync_status)}>
                            {connection.sync_status}
                          </span>
                        </CardTitle>
                        <p className="text-sm text-muted-foreground">
                          {connection.provider_type} • Added {new Date(connection.created_at).toLocaleDateString()}
                        </p>
                        {connection.error_message && (
                          <p className="text-sm text-red-600 mt-1">
                            Error: {connection.error_message}
                          </p>
                        )}
                      </div>
                      <div className="flex flex-col space-y-2">
                        <div className="flex space-x-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleTestConnection(connection.id)}
                          >
                            Test
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDeleteConnection(connection.id)}
                          >
                            Delete
                          </Button>
                        </div>
                        <SyncButton
                          connectionId={connection.id}
                          connectionName={connection.name}
                          onSyncStart={(job) => {
                            console.log('Sync job started:', job);
                            // You could update local state to show the job progress
                          }}
                        />
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div>
                      <h4 className="font-medium mb-3">Bank Accounts</h4>
                      {accounts[connection.id]?.length ? (
                        <div className="space-y-2">
                          {accounts[connection.id].map((account) => (
                            <div
                              key={account.id}
                              className="flex items-center justify-between p-3 border rounded-lg"
                            >
                              <div className="flex-1">
                                <div className="flex items-center space-x-2">
                                  <span className="font-medium">{account.name}</span>
                                  {account.institution && (
                                    <span className="text-sm text-muted-foreground">
                                      at {account.institution}
                                    </span>
                                  )}
                                </div>
                                <div className="text-sm text-muted-foreground">
                                  {account.account_type && `${account.account_type} • `}
                                  {account.balance !== null ? `${account.currency} ${account.balance?.toFixed(2)}` : 'Balance unavailable'}
                                </div>
                              </div>
                              <div className="flex items-center space-x-2">
                                <span className={`text-sm ${account.is_active ? 'text-green-600' : 'text-gray-500'}`}>
                                  {account.is_active ? 'Active' : 'Inactive'}
                                </span>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleToggleAccount(account.id, account.is_active)}
                                >
                                  {account.is_active ? 'Disable' : 'Enable'}
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-sm text-muted-foreground">
                          No accounts found for this connection
                        </p>
                      )}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}