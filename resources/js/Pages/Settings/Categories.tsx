import { Head, router } from '@inertiajs/react'
import { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'

interface Category {
  id: string
  name: string
  parent_id?: string
  color?: string
  description?: string
  created_at: string
  is_default: boolean
}

interface Rule {
  id: string
  name: string
  description?: string
  category_id: string
  category_name: string
  conditions: RuleCondition[]
  priority: number
  enabled: boolean
  match_count: number
  created_at: string
}

interface RuleCondition {
  field: 'description' | 'amount' | 'account_id' | 'merchant'
  operator: 'contains' | 'equals' | 'starts_with' | 'ends_with' | 'greater_than' | 'less_than'
  value: string
  case_sensitive?: boolean
}

interface CategoriesPageProps {
  categories: Category[]
  rules: Rule[]
}

export default function Categories({ categories: initialCategories, rules: initialRules }: CategoriesPageProps) {
  const [categories, setCategories] = useState<Category[]>(initialCategories || [])
  const [rules, setRules] = useState<Rule[]>(initialRules || [])
  const [selectedTab, setSelectedTab] = useState<'categories' | 'rules' | 'patterns'>('categories')
  const [showNewCategoryForm, setShowNewCategoryForm] = useState(false)
  const [showNewRuleForm, setShowNewRuleForm] = useState(false)
  const [loading, setLoading] = useState(false)
  const [patterns, setPatterns] = useState<any[]>([])
  const [loadingPatterns, setLoadingPatterns] = useState(false)
  const [showBatchCategorization, setShowBatchCategorization] = useState(false)
  const [batchEstimate, setBatchEstimate] = useState<any>(null)
  const [loadingEstimate, setLoadingEstimate] = useState(false)

  // Load data when component mounts or tab changes
  useEffect(() => {
    if (selectedTab === 'categories' && categories.length === 0) {
      loadCategories()
    } else if (selectedTab === 'rules' && rules.length === 0) {
      loadRules()
    } else if (selectedTab === 'patterns') {
      loadPatterns()
    }
  }, [selectedTab])

  const loadCategories = async () => {
    try {
      const response = await fetch('/api/v1/categories')
      if (response.ok) {
        const data = await response.json()
        setCategories(data || [])
      }
    } catch (error) {
      console.error('Failed to load categories:', error)
    }
  }

  const loadRules = async () => {
    try {
      const response = await fetch('/api/v1/categorization/rules')
      if (response.ok) {
        const data = await response.json()
        setRules(data || [])
      }
    } catch (error) {
      console.error('Failed to load rules:', error)
    }
  }

  const loadPatterns = async () => {
    setLoadingPatterns(true)
    try {
      const response = await fetch('/api/v1/categorization/patterns')
      if (response.ok) {
        const data = await response.json()
        setPatterns(data || [])
      }
    } catch (error) {
      console.error('Failed to load patterns:', error)
    } finally {
      setLoadingPatterns(false)
    }
  }

  // New category form state
  const [newCategory, setNewCategory] = useState({
    name: '',
    description: '',
    color: '#3B82F6',
    parent_id: ''
  })

  // New rule form state
  const [newRule, setNewRule] = useState({
    name: '',
    description: '',
    category_id: '',
    priority: 100,
    conditions: [{ field: 'description' as const, operator: 'contains' as const, value: '', case_sensitive: false }]
  })

  const handleCreateCategory = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/v1/categories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newCategory)
      })
      
      if (response.ok) {
        const category = await response.json()
        setCategories([...categories, category])
        setNewCategory({ name: '', description: '', color: '#3B82F6', parent_id: '' })
        setShowNewCategoryForm(false)
      }
    } catch (error) {
      console.error('Failed to create category:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateRule = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/v1/categorization/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newRule)
      })
      
      if (response.ok) {
        const rule = await response.json()
        setRules([...rules, rule])
        setNewRule({
          name: '',
          description: '',
          category_id: '',
          priority: 100,
          conditions: [{ field: 'description', operator: 'contains', value: '', case_sensitive: false }]
        })
        setShowNewRuleForm(false)
      }
    } catch (error) {
      console.error('Failed to create rule:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteRule = async (ruleId: string) => {
    try {
      const response = await fetch(`/api/v1/categorization/rules/${ruleId}`, {
        method: 'DELETE'
      })
      
      if (response.ok) {
        setRules(rules.filter(rule => rule.id !== ruleId))
      }
    } catch (error) {
      console.error('Failed to delete rule:', error)
    }
  }

  const handleTestRule = async (ruleId: string) => {
    try {
      const response = await fetch(`/api/v1/categorization/rules/${ruleId}/test`, {
        method: 'POST'
      })
      
      if (response.ok) {
        const result = await response.json()
        alert(`Rule would match ${result.match_count} transactions`)
      }
    } catch (error) {
      console.error('Failed to test rule:', error)
    }
  }

  const addCondition = () => {
    setNewRule({
      ...newRule,
      conditions: [...newRule.conditions, { field: 'description', operator: 'contains', value: '', case_sensitive: false }]
    })
  }

  const removeCondition = (index: number) => {
    setNewRule({
      ...newRule,
      conditions: newRule.conditions.filter((_, i) => i !== index)
    })
  }

  const updateCondition = (index: number, updates: Partial<RuleCondition>) => {
    const updatedConditions = newRule.conditions.map((condition, i) => 
      i === index ? { ...condition, ...updates } : condition
    )
    setNewRule({ ...newRule, conditions: updatedConditions })
  }

  const handleBatchCostEstimate = async () => {
    setLoadingEstimate(true)
    try {
      const response = await fetch('/api/v1/ai/categorization/estimate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          transaction_ids: [], // Empty means all uncategorized
          force_recategorize: false,
          model_preference: 'cost',
          use_rag: true
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        setBatchEstimate(data)
      } else {
        console.error('Failed to estimate cost:', response.statusText)
      }
    } catch (error) {
      console.error('Failed to estimate batch cost:', error)
    } finally {
      setLoadingEstimate(false)
    }
  }

  const handleBatchCategorize = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/v1/ai/categorization/batch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          transaction_ids: [], // Empty means all uncategorized
          force_recategorize: false,
          model_preference: 'cost',
          use_rag: true
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        alert(`Batch categorization started! Processing ${data.transaction_count} transactions. Check the Jobs page for progress.`)
        setShowBatchCategorization(false)
        setBatchEstimate(null)
      } else {
        console.error('Failed to start categorization:', response.statusText)
        alert('Failed to start batch categorization. Please try again.')
      }
    } catch (error) {
      console.error('Failed to start batch categorization:', error)
      alert('Failed to start batch categorization. Please check your connection.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthenticatedLayout>
      <Head title="Categories & Rules" />
      
      <div className="space-y-4">
        <div className="max-w-6xl mx-auto">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="page-header">
                Categories & Rules
              </h1>
              <p className="page-subtitle">Manage transaction categories and automated rules</p>
            </div>
            <Button
              onClick={() => router.visit('/settings')}
              variant="glass"
            >
              ← Back to Settings
            </Button>
          </div>
          
          {/* Tab Navigation */}
          <div className="tab-list">
            {(['categories', 'rules', 'patterns'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setSelectedTab(tab)}
                className={`tab-button ${selectedTab === tab ? 'active' : ''}`}
              >
                {tab}
              </button>
            ))}
          </div>

          {/* Categories Tab */}
          {selectedTab === 'categories' && (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">Transaction Categories</h2>
                <Button onClick={() => setShowNewCategoryForm(true)}>
                  + Add Category
                </Button>
              </div>

              {/* New Category Form */}
              {showNewCategoryForm && (
                <Card>
                  <CardHeader>
                    <CardTitle>Create New Category</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="category-name">Name</Label>
                        <Input
                          id="category-name"
                          value={newCategory.name}
                          onChange={(e) => setNewCategory({ ...newCategory, name: e.target.value })}
                          placeholder="e.g., Groceries"
                        />
                      </div>
                      <div>
                        <Label htmlFor="category-color">Color</Label>
                        <Input
                          id="category-color"
                          type="color"
                          value={newCategory.color}
                          onChange={(e) => setNewCategory({ ...newCategory, color: e.target.value })}
                        />
                      </div>
                    </div>
                    <div>
                      <Label htmlFor="category-description">Description</Label>
                      <Input
                        id="category-description"
                        value={newCategory.description}
                        onChange={(e) => setNewCategory({ ...newCategory, description: e.target.value })}
                        placeholder="Optional description"
                      />
                    </div>
                    <div className="flex space-x-2">
                      <Button onClick={handleCreateCategory} disabled={loading || !newCategory.name.trim()}>
                        {loading ? 'Creating...' : 'Create Category'}
                      </Button>
                      <Button variant="outline" onClick={() => setShowNewCategoryForm(false)}>
                        Cancel
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Categories List */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {categories.map((category) => (
                  <Card key={category.id}>
                    <CardContent className="p-4">
                      <div className="flex items-center space-x-3">
                        <div
                          className="w-4 h-4 rounded-full"
                          style={{ backgroundColor: category.color || '#3B82F6' }}
                        />
                        <div className="flex-1">
                          <h3 className="font-medium">{category.name}</h3>
                          {category.description && (
                            <p className="text-sm text-muted-foreground">{category.description}</p>
                          )}
                        </div>
                        {category.is_default && (
                          <Badge variant="secondary">Default</Badge>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Rules Tab */}
          {selectedTab === 'rules' && (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">Categorization Rules</h2>
                <Button onClick={() => setShowNewRuleForm(true)}>
                  + Add Rule
                </Button>
              </div>

              {/* New Rule Form */}
              {showNewRuleForm && (
                <Card>
                  <CardHeader>
                    <CardTitle>Create Categorization Rule</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="rule-name">Rule Name</Label>
                        <Input
                          id="rule-name"
                          value={newRule.name}
                          onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                          placeholder="e.g., Grocery Store Purchases"
                        />
                      </div>
                      <div>
                        <Label htmlFor="rule-category">Category</Label>
                        <select
                          id="rule-category"
                          value={newRule.category_id}
                          onChange={(e) => setNewRule({ ...newRule, category_id: e.target.value })}
                          className="w-full px-3 py-2 border border-input rounded-md bg-background"
                        >
                          <option value="">Select a category</option>
                          {categories.map((category) => (
                            <option key={category.id} value={category.id}>
                              {category.name}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>
                    
                    <div>
                      <Label htmlFor="rule-description">Description</Label>
                      <Input
                        id="rule-description"
                        value={newRule.description}
                        onChange={(e) => setNewRule({ ...newRule, description: e.target.value })}
                        placeholder="Optional description"
                      />
                    </div>

                    {/* Conditions */}
                    <div>
                      <div className="flex items-center justify-between mb-3">
                        <Label>Conditions</Label>
                        <Button type="button" variant="outline" size="sm" onClick={addCondition}>
                          + Add Condition
                        </Button>
                      </div>
                      
                      {newRule.conditions.map((condition, index) => (
                        <div key={index} className="flex items-center space-x-2 mb-3 p-3 border border-border rounded-lg">
                          <select
                            value={condition.field}
                            onChange={(e) => updateCondition(index, { field: e.target.value as RuleCondition['field'] })}
                            className="px-3 py-2 border border-input rounded-md bg-background"
                          >
                            <option value="description">Description</option>
                            <option value="merchant">Merchant</option>
                            <option value="amount">Amount</option>
                            <option value="account_id">Account</option>
                          </select>
                          
                          <select
                            value={condition.operator}
                            onChange={(e) => updateCondition(index, { operator: e.target.value as RuleCondition['operator'] })}
                            className="px-3 py-2 border border-input rounded-md bg-background"
                          >
                            <option value="contains">Contains</option>
                            <option value="equals">Equals</option>
                            <option value="starts_with">Starts with</option>
                            <option value="ends_with">Ends with</option>
                            {condition.field === 'amount' && (
                              <>
                                <option value="greater_than">Greater than</option>
                                <option value="less_than">Less than</option>
                              </>
                            )}
                          </select>
                          
                          <Input
                            value={condition.value}
                            onChange={(e) => updateCondition(index, { value: e.target.value })}
                            placeholder="Value"
                            className="flex-1"
                          />
                          
                          {newRule.conditions.length > 1 && (
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => removeCondition(index)}
                            >
                              Remove
                            </Button>
                          )}
                        </div>
                      ))}
                    </div>

                    <div className="flex space-x-2">
                      <Button 
                        onClick={handleCreateRule} 
                        disabled={loading || !newRule.name.trim() || !newRule.category_id}
                      >
                        {loading ? 'Creating...' : 'Create Rule'}
                      </Button>
                      <Button variant="outline" onClick={() => setShowNewRuleForm(false)}>
                        Cancel
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Rules List */}
              <div className="space-y-4">
                {rules.map((rule) => (
                  <Card key={rule.id}>
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-3 mb-2">
                            <h3 className="font-medium">{rule.name}</h3>
                            <Badge variant={rule.enabled ? "default" : "secondary"}>
                              {rule.enabled ? 'Active' : 'Disabled'}
                            </Badge>
                            <Badge variant="outline">{rule.category_name}</Badge>
                          </div>
                          {rule.description && (
                            <p className="text-sm text-muted-foreground mb-2">{rule.description}</p>
                          )}
                          <div className="text-sm text-muted-foreground">
                            {rule.conditions.length} condition(s) • Priority: {rule.priority} • Matched: {rule.match_count} transactions
                          </div>
                        </div>
                        <div className="flex space-x-2">
                          <Button variant="outline" size="sm" onClick={() => handleTestRule(rule.id)}>
                            Test
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => handleDeleteRule(rule.id)}>
                            Delete
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>

              {rules.length === 0 && (
                <Card>
                  <CardContent className="p-8 text-center">
                    <p className="text-muted-foreground mb-4">No categorization rules created yet.</p>
                    <Button onClick={() => setShowNewRuleForm(true)}>
                      Create Your First Rule
                    </Button>
                  </CardContent>
                </Card>
              )}
            </div>
          )}

          {/* Patterns Tab */}
          {selectedTab === 'patterns' && (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">Spending Patterns</h2>
                <div className="flex space-x-2">
                  <Button onClick={loadPatterns} variant="outline" disabled={loadingPatterns}>
                    {loadingPatterns ? 'Loading...' : 'Refresh Patterns'}
                  </Button>
                  <Button onClick={() => setShowBatchCategorization(true)}>
                    AI Batch Categorization
                  </Button>
                </div>
              </div>

              {/* Batch Categorization Form */}
              {showBatchCategorization && (
                <Card>
                  <CardHeader>
                    <CardTitle>AI Batch Categorization</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-muted-foreground">
                      Use AI to automatically categorize uncategorized transactions. This will analyze transaction descriptions and suggest appropriate categories.
                    </p>
                    
                    {!batchEstimate ? (
                      <div className="space-y-4">
                        <div className="bg-blue-50 p-4 rounded-lg">
                          <h4 className="font-medium text-blue-900 mb-2">Enhanced AI Categorization:</h4>
                          <ul className="text-sm text-blue-800 space-y-1">
                            <li>• <strong>RAG-Powered:</strong> Uses your transaction history for context</li>
                            <li>• <strong>Smart Analysis:</strong> Considers patterns, merchants, amounts, and timing</li>
                            <li>• <strong>Learning System:</strong> Improves accuracy from your corrections</li>
                            <li>• <strong>Cost Optimized:</strong> Selects best model for your budget</li>
                            <li>• <strong>Background Processing:</strong> Monitor progress via Jobs page</li>
                          </ul>
                        </div>
                        
                        <div className="flex space-x-2">
                          <Button onClick={handleBatchCostEstimate} disabled={loadingEstimate}>
                            {loadingEstimate ? 'Calculating...' : 'Get Cost Estimate'}
                          </Button>
                          <Button variant="outline" onClick={() => setShowBatchCategorization(false)}>
                            Cancel
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <div className="space-y-4">
                        <div className="bg-green-50 p-4 rounded-lg">
                          <h4 className="font-medium text-green-900 mb-2">AI Processing Estimate</h4>
                          <div className="text-sm text-green-800 space-y-2">
                            <div className="flex justify-between">
                              <span>Transactions to process:</span>
                              <span className="font-medium">{batchEstimate.transaction_count || 0}</span>
                            </div>
                            <div className="flex justify-between">
                              <span>AI Model:</span>
                              <span className="font-medium">{batchEstimate.model || 'cost-optimized'}</span>
                            </div>
                            <div className="flex justify-between">
                              <span>Estimated cost:</span>
                              <span className="font-medium">${batchEstimate.estimated_cost || '0.00'}</span>
                            </div>
                            <div className="flex justify-between">
                              <span>Processing mode:</span>
                              <span className="font-medium">RAG-Enhanced</span>
                            </div>
                            <div className="mt-2 pt-2 border-t border-green-200">
                              <p className="text-xs text-green-700">
                                Uses your transaction history for context-aware categorization
                              </p>
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex space-x-2">
                          <Button onClick={handleBatchCategorize} disabled={loading}>
                            {loading ? 'Starting...' : 'Start Categorization'}
                          </Button>
                          <Button variant="outline" onClick={() => setBatchEstimate(null)}>
                            Back
                          </Button>
                          <Button variant="outline" onClick={() => setShowBatchCategorization(false)}>
                            Cancel
                          </Button>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}

              {loadingPatterns ? (
                <Card>
                  <CardContent className="p-8 text-center">
                    <p className="text-muted-foreground">Loading spending patterns...</p>
                  </CardContent>
                </Card>
              ) : patterns.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {patterns.map((pattern, index) => (
                    <Card key={index}>
                      <CardHeader>
                        <CardTitle className="text-base">{pattern.merchant || pattern.pattern_key}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Frequency:</span>
                            <span>{pattern.frequency || pattern.count} transactions</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Average Amount:</span>
                            <span>${pattern.average_amount || '0.00'}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Total Spent:</span>
                            <span>${pattern.total_amount || '0.00'}</span>
                          </div>
                          {pattern.suggested_category && (
                            <div className="flex justify-between text-sm">
                              <span className="text-muted-foreground">Suggested Category:</span>
                              <Badge variant="outline">{pattern.suggested_category}</Badge>
                            </div>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <Card>
                  <CardContent className="p-8 text-center">
                    <p className="text-muted-foreground mb-4">
                      No spending patterns found. Patterns are generated from your transaction history to help identify recurring expenses.
                    </p>
                    <div className="space-y-2">
                      <p className="text-sm text-muted-foreground">
                        Patterns help you:
                      </p>
                      <ul className="text-sm text-muted-foreground space-y-1">
                        <li>• Identify recurring merchants and subscriptions</li>
                        <li>• Understand spending habits by category</li>
                        <li>• Set up automatic categorization rules</li>
                        <li>• Spot unusual or new spending patterns</li>
                      </ul>
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}