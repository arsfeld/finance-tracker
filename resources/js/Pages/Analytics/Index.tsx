import { Head } from '@inertiajs/react'
import { useState } from 'react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import AIChat from '@/components/AIChat'
import AIInsights from '@/components/AIInsights'
import { 
  TrendingUp,
  TrendingDown,
  PieChart,
  Calendar,
  Target,
  AlertTriangle,
  Lightbulb,
  DollarSign,
  BarChart3,
  Download,
  RefreshCw,
  Sparkles
} from 'lucide-react'

// Mock data for analytics
const mockAnalytics = {
  monthlySpending: 2847.32,
  monthlyChange: -12.5,
  averageTransaction: 42.15,
  transactionCount: 67,
  topCategories: [
    { name: 'Food & Dining', amount: 854.20, percentage: 30, color: 'bg-orange-500' },
    { name: 'Groceries', amount: 642.15, percentage: 22.5, color: 'bg-green-500' },
    { name: 'Transportation', amount: 398.50, percentage: 14, color: 'bg-blue-500' },
    { name: 'Entertainment', amount: 285.30, percentage: 10, color: 'bg-purple-500' },
    { name: 'Shopping', amount: 234.80, percentage: 8.2, color: 'bg-pink-500' },
    { name: 'Bills', amount: 432.37, percentage: 15.3, color: 'bg-red-500' }
  ],
  weeklyTrends: [
    { week: 'Week 1', amount: 687.45 },
    { week: 'Week 2', amount: 823.12 },
    { week: 'Week 3', amount: 567.89 },
    { week: 'Week 4', amount: 768.86 }
  ],
  insights: [
    {
      type: 'warning',
      title: 'High Dining Spending',
      description: 'Your dining expenses are 15% higher than last month. Consider meal planning to reduce costs.',
      impact: '$128.50',
      category: 'Food & Dining'
    },
    {
      type: 'positive',
      title: 'Great Savings Progress',
      description: 'You\'ve saved $200 more than your target this month!',
      impact: '+$200.00',
      category: 'Savings'
    },
    {
      type: 'neutral',
      title: 'Recurring Subscription',
      description: 'Netflix subscription renewed. Consider reviewing all subscriptions quarterly.',
      impact: '$15.99',
      category: 'Entertainment'
    }
  ]
}

const initialChatMessages = [
  {
    id: '1',
    type: 'user' as const,
    message: 'How much did I spend on dining last month?',
    timestamp: '2024-01-15 10:30'
  },
  {
    id: '2',
    type: 'ai' as const,
    message: 'You spent $854.20 on dining last month, which represents 30% of your total expenses. This is slightly higher than your 3-month average of $742. The main contributors were:\n\n• Restaurant meals: $654.20\n• Coffee shops: $128.50\n• Food delivery: $71.50\n\nWould you like some suggestions to optimize your dining expenses?',
    timestamp: '2024-01-15 10:30'
  },
  {
    id: '3',
    type: 'user' as const,
    message: 'What are my biggest expense categories?',
    timestamp: '2024-01-15 10:32'
  },
  {
    id: '4',
    type: 'ai' as const,
    message: 'Your biggest expense categories this month are:\n\n1. **Food & Dining** - $854.20 (30%)\n2. **Groceries** - $642.15 (22.5%)\n3. **Bills** - $432.37 (15.3%)\n4. **Transportation** - $398.50 (14%)\n5. **Entertainment** - $285.30 (10%)\n\nFood-related expenses (dining + groceries) account for over 52% of your spending. This is quite high compared to the recommended 25-30%.',
    timestamp: '2024-01-15 10:32'
  }
]

export default function Analytics() {
  const [selectedInsight, setSelectedInsight] = useState(null)

  const handleChatMessage = (message: string) => {
    console.log('Analytics context - User asked:', message)
    // Here you could trigger analytics-specific actions
    // like updating charts, generating reports, etc.
  }

  const formatCurrency = (amount) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(amount)
  }

  const getInsightIcon = (type) => {
    switch (type) {
      case 'warning': return <AlertTriangle className="w-5 h-5 text-orange-500" />
      case 'positive': return <TrendingUp className="w-5 h-5 text-green-500" />
      default: return <Lightbulb className="w-5 h-5 text-blue-500" />
    }
  }

  const getInsightColor = (type) => {
    switch (type) {
      case 'warning': return 'border-orange-200 bg-orange-50'
      case 'positive': return 'border-green-200 bg-green-50'
      default: return 'border-blue-200 bg-blue-50'
    }
  }

  return (
    <AuthenticatedLayout>
      <Head title="AI Analytics" />
      
      <div className="p-6 space-y-6">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold text-foreground mb-2">
                AI Analytics
              </h1>
              <p className="text-muted-foreground">
                Get intelligent insights about your financial health
              </p>
            </div>
            <div className="flex gap-2 mt-4 sm:mt-0">
              <Button variant="outline" size="sm">
                <Download className="w-4 h-4 mr-2" />
                Export Report
              </Button>
              <Button variant="outline" size="sm">
                <RefreshCw className="w-4 h-4 mr-2" />
                Refresh Data
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Left Column - Analytics Overview */}
            <div className="lg:col-span-2 space-y-6">
              {/* Key Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center">
                      <DollarSign className="w-4 h-4 mr-1" />
                      Monthly Spending
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">
                      {formatCurrency(mockAnalytics.monthlySpending)}
                    </p>
                    <div className="flex items-center mt-1">
                      <TrendingDown className="w-4 h-4 text-green-500 mr-1" />
                      <span className="text-sm text-green-600 font-medium">
                        {Math.abs(mockAnalytics.monthlyChange)}% vs last month
                      </span>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center">
                      <BarChart3 className="w-4 h-4 mr-1" />
                      Avg Transaction
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">
                      {formatCurrency(mockAnalytics.averageTransaction)}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {mockAnalytics.transactionCount} transactions
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center">
                      <PieChart className="w-4 h-4 mr-1" />
                      Top Category
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-lg font-bold">Food & Dining</p>
                    <p className="text-sm text-muted-foreground">
                      {formatCurrency(mockAnalytics.topCategories[0].amount)} (30%)
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center">
                      <Target className="w-4 h-4 mr-1" />
                      Budget Status
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-lg font-bold text-green-600">On Track</p>
                    <p className="text-sm text-muted-foreground">
                      $152.68 under budget
                    </p>
                  </CardContent>
                </Card>
              </div>

              {/* Category Breakdown */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center">
                    <PieChart className="w-5 h-5 mr-2" />
                    Spending by Category
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {mockAnalytics.topCategories.map((category, index) => (
                      <div key={category.name} className="flex items-center justify-between">
                        <div className="flex items-center gap-3 flex-1">
                          <div className={`w-3 h-3 rounded-full ${category.color}`}></div>
                          <span className="font-medium">{category.name}</span>
                        </div>
                        <div className="flex items-center gap-4">
                          <div className="w-24 bg-gray-200 rounded-full h-2">
                            <div 
                              className={`h-2 rounded-full ${category.color}`}
                              style={{ width: `${category.percentage}%` }}
                            ></div>
                          </div>
                          <span className="text-sm text-muted-foreground w-12">
                            {category.percentage}%
                          </span>
                          <span className="font-semibold w-20 text-right">
                            {formatCurrency(category.amount)}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* AI Insights */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center">
                    <Sparkles className="w-5 h-5 mr-2" />
                    AI Insights
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {mockAnalytics.insights.map((insight, index) => (
                      <div 
                        key={index}
                        className={`p-4 rounded-lg border cursor-pointer transition-colors hover:bg-muted/50 ${getInsightColor(insight.type)}`}
                        onClick={() => setSelectedInsight(insight)}
                      >
                        <div className="flex items-start gap-3">
                          {getInsightIcon(insight.type)}
                          <div className="flex-1">
                            <div className="flex items-center justify-between mb-1">
                              <h4 className="font-medium">{insight.title}</h4>
                              <Badge variant="outline" className="text-xs">
                                {insight.impact}
                              </Badge>
                            </div>
                            <p className="text-sm text-muted-foreground">
                              {insight.description}
                            </p>
                            <Badge className="mt-2 text-xs" variant="secondary">
                              {insight.category}
                            </Badge>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Right Column - AI Features */}
            <div className="space-y-6">
              <AIInsights />
              
              <AIChat
                title="AI Financial Assistant"
                placeholder="Ask about your spending, budgets, or analytics..."
                height="h-[600px]"
                context="Analytics Dashboard"
                onMessageSent={handleChatMessage}
                initialMessages={initialChatMessages}
                quickQuestions={[
                  "Monthly spending summary",
                  "Budget recommendations", 
                  "Show expense trends",
                  "Category breakdown",
                  "Savings optimization",
                  "Compare to last month"
                ]}
              />

              {/* Quick Actions */}
              <Card>
                <CardHeader>
                  <CardTitle>Quick Actions</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <Button variant="outline" className="w-full justify-start" size="sm">
                    <Calendar className="w-4 h-4 mr-2" />
                    Generate Monthly Report
                  </Button>
                  <Button variant="outline" className="w-full justify-start" size="sm">
                    <Target className="w-4 h-4 mr-2" />
                    Set Budget Goals
                  </Button>
                  <Button variant="outline" className="w-full justify-start" size="sm">
                    <TrendingUp className="w-4 h-4 mr-2" />
                    Forecast Next Month
                  </Button>
                  <Button variant="outline" className="w-full justify-start" size="sm">
                    <AlertTriangle className="w-4 h-4 mr-2" />
                    Setup Alerts
                  </Button>
                </CardContent>
              </Card>
            </div>
          </div>

          {/* Insight Detail Modal */}
          {selectedInsight && (
            <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
              <Card className="w-full max-w-lg">
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {getInsightIcon(selectedInsight.type)}
                      {selectedInsight.title}
                    </div>
                    <Button 
                      variant="ghost" 
                      size="sm"
                      onClick={() => setSelectedInsight(null)}
                    >
                      ×
                    </Button>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <p className="text-muted-foreground">
                      {selectedInsight.description}
                    </p>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Impact</label>
                      <p className="font-semibold text-lg">{selectedInsight.impact}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Category</label>
                      <Badge variant="secondary">{selectedInsight.category}</Badge>
                    </div>
                  </div>
                  
                  <div className="flex gap-2 pt-4">
                    <Button className="flex-1">
                      Take Action
                    </Button>
                    <Button variant="outline" className="flex-1">
                      Learn More
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}