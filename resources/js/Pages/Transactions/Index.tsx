import { Head } from '@inertiajs/react'
import { useState, useMemo } from 'react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import TransactionDetail from '@/components/TransactionDetail'
import { 
  Search, 
  Filter, 
  Download, 
  Calendar,
  CreditCard,
  ArrowUpDown,
  MoreHorizontal
} from 'lucide-react'

// Mock transaction data - replace with real data from props
const mockTransactions = [
  {
    id: '1',
    date: '2024-01-15',
    description: 'Starbucks Coffee',
    merchant: 'Starbucks',
    amount: -4.85,
    category: 'Food & Dining',
    account: 'Chase Checking',
    pending: false,
    tags: ['coffee', 'breakfast']
  },
  {
    id: '2', 
    date: '2024-01-15',
    description: 'Grocery Store Purchase',
    merchant: 'Whole Foods',
    amount: -87.32,
    category: 'Groceries',
    account: 'Chase Checking',
    pending: false,
    tags: ['groceries', 'weekly-shop']
  },
  {
    id: '3',
    date: '2024-01-14',
    description: 'Salary Deposit',
    merchant: 'Employer Inc',
    amount: 2500.00,
    category: 'Income',
    account: 'Chase Checking', 
    pending: false,
    tags: ['salary', 'income']
  },
  {
    id: '4',
    date: '2024-01-14',
    description: 'Gas Station',
    merchant: 'Shell',
    amount: -45.20,
    category: 'Transportation',
    account: 'Chase Credit Card',
    pending: true,
    tags: ['gas', 'commute']
  },
  {
    id: '5',
    date: '2024-01-13',
    description: 'Netflix Subscription',
    merchant: 'Netflix',
    amount: -15.99,
    category: 'Entertainment',
    account: 'Chase Credit Card',
    pending: false,
    tags: ['subscription', 'streaming']
  }
]

const categories = ['All', 'Food & Dining', 'Groceries', 'Transportation', 'Entertainment', 'Income', 'Shopping', 'Bills']
const accounts = ['All Accounts', 'Chase Checking', 'Chase Credit Card', 'Savings Account']

export default function Transactions() {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('All')
  const [selectedAccount, setSelectedAccount] = useState('All Accounts')
  const [showFilters, setShowFilters] = useState(false)
  const [sortBy, setSortBy] = useState('date')
  const [sortOrder, setSortOrder] = useState('desc')
  const [selectedTransaction, setSelectedTransaction] = useState(null)

  const filteredTransactions = useMemo(() => {
    let filtered = mockTransactions.filter(transaction => {
      const matchesSearch = transaction.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           transaction.merchant.toLowerCase().includes(searchTerm.toLowerCase())
      const matchesCategory = selectedCategory === 'All' || transaction.category === selectedCategory
      const matchesAccount = selectedAccount === 'All Accounts' || transaction.account === selectedAccount
      
      return matchesSearch && matchesCategory && matchesAccount
    })

    // Sort transactions
    filtered.sort((a, b) => {
      let aVal = a[sortBy]
      let bVal = b[sortBy]
      
      if (sortBy === 'amount') {
        aVal = Math.abs(aVal)
        bVal = Math.abs(bVal)
      }
      
      if (sortOrder === 'asc') {
        return aVal > bVal ? 1 : -1
      } else {
        return aVal < bVal ? 1 : -1
      }
    })

    return filtered
  }, [searchTerm, selectedCategory, selectedAccount, sortBy, sortOrder])

  const totalSpent = useMemo(() => {
    return filteredTransactions
      .filter(t => t.amount < 0)
      .reduce((sum, t) => sum + Math.abs(t.amount), 0)
  }, [filteredTransactions])

  const totalIncome = useMemo(() => {
    return filteredTransactions
      .filter(t => t.amount > 0)
      .reduce((sum, t) => sum + t.amount, 0)
  }, [filteredTransactions])

  const formatCurrency = (amount) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(Math.abs(amount))
  }

  const formatDate = (dateStr) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    })
  }

  const getCategoryColor = (category) => {
    const colors = {
      'Food & Dining': 'bg-orange-100 text-orange-800',
      'Groceries': 'bg-green-100 text-green-800',
      'Transportation': 'bg-blue-100 text-blue-800',
      'Entertainment': 'bg-purple-100 text-purple-800',
      'Income': 'bg-emerald-100 text-emerald-800',
      'Shopping': 'bg-pink-100 text-pink-800',
      'Bills': 'bg-red-100 text-red-800'
    }
    return colors[category] || 'bg-gray-100 text-gray-800'
  }

  return (
    <AuthenticatedLayout>
      <Head title="Transactions" />
      
      <div className="space-y-4">
        <div>
          {/* Header */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-4">
            <div>
              <h1 className="page-header">
                Transactions
              </h1>
              <p className="page-subtitle">View and manage all your financial transactions</p>
            </div>
            <div className="flex gap-2">
              <Button variant="glass" size="sm">
                <Download className="w-4 h-4 mr-2" />
                Export
              </Button>
              <Button variant="glass" size="sm">
                <Calendar className="w-4 h-4 mr-2" />
                Date Range
              </Button>
            </div>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Transactions
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{filteredTransactions.length}</p>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Spent
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-red-600">
                  {formatCurrency(totalSpent)}
                </p>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Income
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-green-600">
                  {formatCurrency(totalIncome)}
                </p>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Net Change
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className={`text-2xl font-bold ${totalIncome - totalSpent >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {formatCurrency(totalIncome - totalSpent)}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Search and Filters */}
          <Card className="mb-6">
            <CardHeader>
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
                <CardTitle>Search & Filter</CardTitle>
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setShowFilters(!showFilters)}
                >
                  <Filter className="w-4 h-4 mr-2" />
                  {showFilters ? 'Hide' : 'Show'} Filters
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Search Bar */}
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
                  <Input
                    placeholder="Search transactions..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="pl-10"
                  />
                </div>

                {/* Filters */}
                {showFilters && (
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="text-sm font-medium mb-2 block">Category</label>
                      <select 
                        className="w-full p-2 border rounded-md"
                        value={selectedCategory}
                        onChange={(e) => setSelectedCategory(e.target.value)}
                      >
                        {categories.map(cat => (
                          <option key={cat} value={cat}>{cat}</option>
                        ))}
                      </select>
                    </div>
                    
                    <div>
                      <label className="text-sm font-medium mb-2 block">Account</label>
                      <select 
                        className="w-full p-2 border rounded-md"
                        value={selectedAccount}
                        onChange={(e) => setSelectedAccount(e.target.value)}
                      >
                        {accounts.map(acc => (
                          <option key={acc} value={acc}>{acc}</option>
                        ))}
                      </select>
                    </div>
                    
                    <div>
                      <label className="text-sm font-medium mb-2 block">Sort By</label>
                      <div className="flex gap-2">
                        <select 
                          className="flex-1 p-2 border rounded-md"
                          value={sortBy}
                          onChange={(e) => setSortBy(e.target.value)}
                        >
                          <option value="date">Date</option>
                          <option value="amount">Amount</option>
                          <option value="description">Description</option>
                          <option value="category">Category</option>
                        </select>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
                        >
                          <ArrowUpDown className="w-4 h-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Transactions List */}
          <Card>
            <CardHeader>
              <CardTitle>Transaction History</CardTitle>
            </CardHeader>
            <CardContent>
              {filteredTransactions.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">
                    No transactions found matching your criteria.
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {filteredTransactions.map((transaction) => (
                    <div 
                      key={transaction.id}
                      className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors cursor-pointer"
                      onClick={() => setSelectedTransaction(transaction)}
                    >
                      <div className="flex items-center gap-4 flex-1">
                        <div className="w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center">
                          <CreditCard className="w-5 h-5 text-primary" />
                        </div>
                        
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <p className="font-medium text-foreground truncate">
                              {transaction.description}
                            </p>
                            {transaction.pending && (
                              <Badge variant="outline" className="text-xs">
                                Pending
                              </Badge>
                            )}
                          </div>
                          <div className="flex items-center gap-4 text-sm text-muted-foreground">
                            <span>{formatDate(transaction.date)}</span>
                            <span>{transaction.account}</span>
                            <Badge className={`${getCategoryColor(transaction.category)} text-xs`}>
                              {transaction.category}
                            </Badge>
                          </div>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <p className={`font-semibold ${
                            transaction.amount >= 0 ? 'text-green-600' : 'text-red-600'
                          }`}>
                            {transaction.amount >= 0 ? '+' : '-'}{formatCurrency(transaction.amount)}
                          </p>
                          {transaction.tags.length > 0 && (
                            <div className="flex gap-1 mt-1 justify-end">
                              {transaction.tags.slice(0, 2).map(tag => (
                                <span key={tag} className="text-xs bg-gray-100 text-gray-600 px-2 py-1 rounded">
                                  {tag}
                                </span>
                              ))}
                              {transaction.tags.length > 2 && (
                                <span className="text-xs text-muted-foreground">
                                  +{transaction.tags.length - 2}
                                </span>
                              )}
                            </div>
                          )}
                        </div>
                        
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Transaction Detail Modal */}
          {selectedTransaction && (
            <TransactionDetail
              transaction={selectedTransaction}
              onClose={() => setSelectedTransaction(null)}
              onSave={(updatedTransaction) => {
                console.log('Saving transaction:', updatedTransaction)
                // Here you would typically call an API to save the transaction
                setSelectedTransaction(null)
              }}
              onDelete={(transactionId) => {
                console.log('Deleting transaction:', transactionId)
                // Here you would typically call an API to delete the transaction
                setSelectedTransaction(null)
              }}
              onSplit={(transactionId) => {
                console.log('Splitting transaction:', transactionId)
                // Here you would implement transaction splitting logic
              }}
            />
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}