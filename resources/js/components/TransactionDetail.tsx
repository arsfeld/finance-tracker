import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { 
  Edit3, 
  Tag, 
  Calendar,
  CreditCard,
  Building,
  MapPin,
  Receipt,
  X,
  Save,
  Trash2,
  Split
} from 'lucide-react'

interface Transaction {
  id: string
  date: string
  description: string
  merchant: string
  amount: number
  category: string
  account: string
  pending: boolean
  tags: string[]
  memo?: string
  location?: string
  receiptUrl?: string
}

interface TransactionDetailProps {
  transaction: Transaction
  onClose: () => void
  onSave?: (transaction: Transaction) => void
  onDelete?: (transactionId: string) => void
  onSplit?: (transactionId: string) => void
}

const categories = [
  'Food & Dining', 'Groceries', 'Transportation', 'Entertainment', 
  'Shopping', 'Bills', 'Healthcare', 'Education', 'Travel', 'Income'
]

export default function TransactionDetail({ 
  transaction, 
  onClose, 
  onSave,
  onDelete,
  onSplit 
}: TransactionDetailProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editedTransaction, setEditedTransaction] = useState(transaction)
  const [newTag, setNewTag] = useState('')

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(Math.abs(amount))
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    })
  }

  const getCategoryColor = (category: string) => {
    const colors = {
      'Food & Dining': 'bg-orange-100 text-orange-800',
      'Groceries': 'bg-green-100 text-green-800',
      'Transportation': 'bg-blue-100 text-blue-800',
      'Entertainment': 'bg-purple-100 text-purple-800',
      'Income': 'bg-emerald-100 text-emerald-800',
      'Shopping': 'bg-pink-100 text-pink-800',
      'Bills': 'bg-red-100 text-red-800',
      'Healthcare': 'bg-cyan-100 text-cyan-800',
      'Education': 'bg-indigo-100 text-indigo-800',
      'Travel': 'bg-yellow-100 text-yellow-800'
    }
    return colors[category] || 'bg-gray-100 text-gray-800'
  }

  const handleSave = () => {
    onSave?.(editedTransaction)
    setIsEditing(false)
  }

  const handleCancel = () => {
    setEditedTransaction(transaction)
    setIsEditing(false)
  }

  const addTag = () => {
    if (newTag.trim() && !editedTransaction.tags.includes(newTag.trim())) {
      setEditedTransaction({
        ...editedTransaction,
        tags: [...editedTransaction.tags, newTag.trim()]
      })
      setNewTag('')
    }
  }

  const removeTag = (tagToRemove: string) => {
    setEditedTransaction({
      ...editedTransaction,
      tags: editedTransaction.tags.filter(tag => tag !== tagToRemove)
    })
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Receipt className="w-5 h-5" />
              Transaction Details
            </CardTitle>
            <div className="flex items-center gap-2">
              {!isEditing && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setIsEditing(true)}
                >
                  <Edit3 className="w-4 h-4 mr-2" />
                  Edit
                </Button>
              )}
              <Button 
                variant="ghost" 
                size="sm"
                onClick={onClose}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </CardHeader>
        
        <CardContent className="space-y-6">
          {/* Main Transaction Info */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">Description</label>
                {isEditing ? (
                  <Input
                    value={editedTransaction.description}
                    onChange={(e) => setEditedTransaction({
                      ...editedTransaction,
                      description: e.target.value
                    })}
                    className="mt-1"
                  />
                ) : (
                  <p className="font-medium text-lg">{transaction.description}</p>
                )}
              </div>

              <div>
                <label className="text-sm font-medium text-muted-foreground">Merchant</label>
                {isEditing ? (
                  <Input
                    value={editedTransaction.merchant}
                    onChange={(e) => setEditedTransaction({
                      ...editedTransaction,
                      merchant: e.target.value
                    })}
                    className="mt-1"
                  />
                ) : (
                  <div className="flex items-center gap-2 mt-1">
                    <Building className="w-4 h-4 text-muted-foreground" />
                    <span>{transaction.merchant}</span>
                  </div>
                )}
              </div>

              <div>
                <label className="text-sm font-medium text-muted-foreground">Category</label>
                {isEditing ? (
                  <select 
                    value={editedTransaction.category}
                    onChange={(e) => setEditedTransaction({
                      ...editedTransaction,
                      category: e.target.value
                    })}
                    className="w-full mt-1 p-2 border rounded-md"
                  >
                    {categories.map(cat => (
                      <option key={cat} value={cat}>{cat}</option>
                    ))}
                  </select>
                ) : (
                  <div className="mt-1">
                    <Badge className={getCategoryColor(transaction.category)}>
                      {transaction.category}
                    </Badge>
                  </div>
                )}
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">Amount</label>
                {isEditing ? (
                  <Input
                    type="number"
                    step="0.01"
                    value={Math.abs(editedTransaction.amount)}
                    onChange={(e) => setEditedTransaction({
                      ...editedTransaction,
                      amount: transaction.amount < 0 ? -parseFloat(e.target.value) : parseFloat(e.target.value)
                    })}
                    className="mt-1"
                  />
                ) : (
                  <p className={`text-2xl font-bold mt-1 ${
                    transaction.amount >= 0 ? 'text-green-600' : 'text-red-600'
                  }`}>
                    {transaction.amount >= 0 ? '+' : '-'}{formatCurrency(transaction.amount)}
                  </p>
                )}
              </div>

              <div>
                <label className="text-sm font-medium text-muted-foreground">Date</label>
                {isEditing ? (
                  <Input
                    type="date"
                    value={editedTransaction.date}
                    onChange={(e) => setEditedTransaction({
                      ...editedTransaction,
                      date: e.target.value
                    })}
                    className="mt-1"
                  />
                ) : (
                  <div className="flex items-center gap-2 mt-1">
                    <Calendar className="w-4 h-4 text-muted-foreground" />
                    <span>{formatDate(transaction.date)}</span>
                  </div>
                )}
              </div>

              <div>
                <label className="text-sm font-medium text-muted-foreground">Account</label>
                <div className="flex items-center gap-2 mt-1">
                  <CreditCard className="w-4 h-4 text-muted-foreground" />
                  <span>{transaction.account}</span>
                  {transaction.pending && (
                    <Badge variant="outline" className="text-xs">
                      Pending
                    </Badge>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Tags Section */}
          <div>
            <label className="text-sm font-medium text-muted-foreground">Tags</label>
            <div className="mt-2 space-y-2">
              <div className="flex flex-wrap gap-2">
                {editedTransaction.tags.map(tag => (
                  <div key={tag} className="flex items-center gap-1 bg-secondary text-secondary-foreground px-2 py-1 rounded-md text-sm">
                    <Tag className="w-3 h-3" />
                    <span>{tag}</span>
                    {isEditing && (
                      <button
                        onClick={() => removeTag(tag)}
                        className="ml-1 hover:text-red-500"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
              
              {isEditing && (
                <div className="flex gap-2">
                  <Input
                    placeholder="Add a tag..."
                    value={newTag}
                    onChange={(e) => setNewTag(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && addTag()}
                    className="flex-1"
                  />
                  <Button onClick={addTag} size="sm" variant="outline">
                    Add
                  </Button>
                </div>
              )}
            </div>
          </div>

          {/* Memo Section */}
          <div>
            <label className="text-sm font-medium text-muted-foreground">Notes</label>
            {isEditing ? (
              <textarea
                value={editedTransaction.memo || ''}
                onChange={(e) => setEditedTransaction({
                  ...editedTransaction,
                  memo: e.target.value
                })}
                placeholder="Add notes about this transaction..."
                className="w-full mt-1 p-2 border rounded-md h-20 resize-none"
              />
            ) : (
              <p className="mt-1 text-sm text-muted-foreground">
                {transaction.memo || 'No notes added'}
              </p>
            )}
          </div>

          {/* Location Section */}
          {transaction.location && (
            <div>
              <label className="text-sm font-medium text-muted-foreground">Location</label>
              <div className="flex items-center gap-2 mt-1">
                <MapPin className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm">{transaction.location}</span>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2 pt-4 border-t">
            {isEditing ? (
              <>
                <Button onClick={handleSave} className="flex-1">
                  <Save className="w-4 h-4 mr-2" />
                  Save Changes
                </Button>
                <Button onClick={handleCancel} variant="outline" className="flex-1">
                  Cancel
                </Button>
              </>
            ) : (
              <>
                <Button onClick={() => setIsEditing(true)} variant="outline" className="flex-1">
                  <Edit3 className="w-4 h-4 mr-2" />
                  Edit
                </Button>
                {onSplit && (
                  <Button onClick={() => onSplit(transaction.id)} variant="outline" className="flex-1">
                    <Split className="w-4 h-4 mr-2" />
                    Split
                  </Button>
                )}
                {onDelete && (
                  <Button 
                    onClick={() => onDelete(transaction.id)} 
                    variant="outline" 
                    className="flex-1 text-red-600 hover:text-red-700"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    Delete
                  </Button>
                )}
              </>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}