import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Brain, TrendingUp, AlertTriangle, Sparkles, Target, DollarSign } from 'lucide-react'

interface Insight {
  type: string
  title: string
  description: string
  suggestion: string
  confidence: number
  priority?: 'high' | 'medium' | 'low'
}

interface Anomaly {
  type: string
  transaction: string
  date: string
  reason: string
  severity: 'high' | 'medium' | 'low'
}

interface AIInsightsProps {
  className?: string
}

export default function AIInsights({ className = '' }: AIInsightsProps) {
  const [insights, setInsights] = useState<Insight[]>([])
  const [anomalies, setAnomalies] = useState<Anomaly[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'insights' | 'anomalies' | 'trends'>('insights')

  useEffect(() => {
    loadAIData()
  }, [])

  const loadAIData = async () => {
    setLoading(true)
    try {
      // Load insights and anomalies in parallel
      const [insightsResponse, anomaliesResponse] = await Promise.all([
        fetch('/api/v1/ai/insights/spending'),
        fetch('/api/v1/ai/insights/anomalies')
      ])

      if (insightsResponse.ok) {
        const insightsData = await insightsResponse.json()
        setInsights(insightsData.insights || [])
      }

      if (anomaliesResponse.ok) {
        const anomaliesData = await anomaliesResponse.json()
        setAnomalies(anomaliesData.anomalies || [])
      }
    } catch (error) {
      console.error('Failed to load AI insights:', error)
    } finally {
      setLoading(false)
    }
  }

  const getInsightIcon = (type: string) => {
    switch (type) {
      case 'spending_pattern':
        return <TrendingUp className="w-4 h-4" />
      case 'budget_alert':
        return <AlertTriangle className="w-4 h-4" />
      case 'savings_opportunity':
        return <Target className="w-4 h-4" />
      default:
        return <Sparkles className="w-4 h-4" />
    }
  }

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'text-green-600'
    if (confidence >= 0.6) return 'text-yellow-600'
    return 'text-red-600'
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'high': return 'bg-red-100 text-red-800'
      case 'medium': return 'bg-yellow-100 text-yellow-800'
      case 'low': return 'bg-blue-100 text-blue-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center">
            <Brain className="w-5 h-5 mr-2" />
            AI Financial Insights
          </div>
          <Button variant="outline" size="sm" onClick={loadAIData} disabled={loading}>
            {loading ? 'Analyzing...' : 'Refresh'}
          </Button>
        </CardTitle>
        
        {/* Tab Navigation */}
        <div className="flex space-x-1">
          {(['insights', 'anomalies', 'trends'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                activeTab === tab
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
              {tab === 'anomalies' && anomalies.length > 0 && (
                <Badge variant="secondary" className="ml-1 h-4 text-xs">
                  {anomalies.length}
                </Badge>
              )}
            </button>
          ))}
        </div>
      </CardHeader>

      <CardContent>
        {loading ? (
          <div className="text-center py-8">
            <Brain className="w-8 h-8 mx-auto text-muted-foreground mb-2 animate-pulse" />
            <p className="text-muted-foreground">Analyzing your financial data...</p>
          </div>
        ) : (
          <>
            {/* Insights Tab */}
            {activeTab === 'insights' && (
              <div className="space-y-4">
                {insights.length > 0 ? (
                  insights.map((insight, index) => (
                    <div key={index} className="border rounded-lg p-4 space-y-2">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center space-x-2">
                          {getInsightIcon(insight.type)}
                          <h4 className="font-medium">{insight.title}</h4>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className={`text-xs font-medium ${getConfidenceColor(insight.confidence)}`}>
                            {Math.round(insight.confidence * 100)}% confidence
                          </span>
                        </div>
                      </div>
                      
                      <p className="text-sm text-muted-foreground">{insight.description}</p>
                      
                      <div className="bg-blue-50 p-3 rounded-md">
                        <div className="flex items-start space-x-2">
                          <Sparkles className="w-4 h-4 text-blue-600 mt-0.5 flex-shrink-0" />
                          <div>
                            <p className="text-sm font-medium text-blue-900">AI Suggestion</p>
                            <p className="text-sm text-blue-800">{insight.suggestion}</p>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="text-center py-8">
                    <TrendingUp className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                    <p className="text-muted-foreground">No insights available yet.</p>
                    <p className="text-sm text-muted-foreground">
                      Add more transactions to get personalized insights.
                    </p>
                  </div>
                )}
              </div>
            )}

            {/* Anomalies Tab */}
            {activeTab === 'anomalies' && (
              <div className="space-y-4">
                {anomalies.length > 0 ? (
                  anomalies.map((anomaly, index) => (
                    <div key={index} className="border rounded-lg p-4 space-y-2">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center space-x-2">
                          <AlertTriangle className="w-4 h-4 text-orange-500" />
                          <h4 className="font-medium">{anomaly.transaction}</h4>
                        </div>
                        <Badge className={getSeverityColor(anomaly.severity)}>
                          {anomaly.severity}
                        </Badge>
                      </div>
                      
                      <p className="text-sm text-muted-foreground">{anomaly.date}</p>
                      <p className="text-sm">{anomaly.reason}</p>
                    </div>
                  ))
                ) : (
                  <div className="text-center py-8">
                    <AlertTriangle className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                    <p className="text-muted-foreground">No anomalies detected.</p>
                    <p className="text-sm text-muted-foreground">
                      Your spending patterns look normal.
                    </p>
                  </div>
                )}
              </div>
            )}

            {/* Trends Tab */}
            {activeTab === 'trends' && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="border rounded-lg p-4">
                    <div className="flex items-center space-x-2 mb-2">
                      <TrendingUp className="w-4 h-4 text-green-600" />
                      <h4 className="font-medium">Monthly Trend</h4>
                    </div>
                    <p className="text-2xl font-bold text-green-600">-5.0%</p>
                    <p className="text-sm text-muted-foreground">vs last month</p>
                  </div>
                  
                  <div className="border rounded-lg p-4">
                    <div className="flex items-center space-x-2 mb-2">
                      <DollarSign className="w-4 h-4 text-blue-600" />
                      <h4 className="font-medium">Avg Transaction</h4>
                    </div>
                    <p className="text-2xl font-bold">$47.23</p>
                    <p className="text-sm text-muted-foreground">per transaction</p>
                  </div>
                </div>

                <div className="space-y-2">
                  <h4 className="font-medium">Category Trends</h4>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between p-2 border rounded">
                      <span className="text-sm">Groceries</span>
                      <div className="flex items-center space-x-2">
                        <TrendingUp className="w-3 h-3 text-red-500" />
                        <span className="text-sm text-red-600">+12.5%</span>
                      </div>
                    </div>
                    <div className="flex items-center justify-between p-2 border rounded">
                      <span className="text-sm">Entertainment</span>
                      <div className="flex items-center space-x-2">
                        <TrendingUp className="w-3 h-3 text-green-500 rotate-180" />
                        <span className="text-sm text-green-600">-8.3%</span>
                      </div>
                    </div>
                    <div className="flex items-center justify-between p-2 border rounded">
                      <span className="text-sm">Transportation</span>
                      <div className="flex items-center space-x-2">
                        <TrendingUp className="w-3 h-3 text-gray-500" />
                        <span className="text-sm text-gray-600">+2.1%</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}