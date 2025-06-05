import { useState, useRef, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Brain, Send, MessageCircle, Sparkles } from 'lucide-react'

interface ChatMessage {
  id: string
  type: 'user' | 'ai'
  message: string
  timestamp: string
  metadata?: {
    suggestions?: string[]
    charts?: any[]
    insights?: any[]
  }
}

interface AIChatProps {
  title?: string
  placeholder?: string
  height?: string
  context?: string
  onMessageSent?: (message: string) => void
  initialMessages?: ChatMessage[]
  quickQuestions?: string[]
}

export default function AIChat({
  title = "AI Financial Assistant",
  placeholder = "Ask about your spending, budgets, or savings...",
  height = "h-[600px]",
  context = "",
  onMessageSent,
  initialMessages = [],
  quickQuestions = [
    "Monthly spending summary",
    "Budget recommendations", 
    "Expense trends",
    "Savings goals"
  ]
}: AIChatProps) {
  const [chatMessage, setChatMessage] = useState('')
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>(initialMessages)
  const [isLoading, setIsLoading] = useState(false)
  const chatEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
  }, [chatHistory])

  const generateAIResponse = (userMessage: string): string => {
    const lowerMessage = userMessage.toLowerCase()
    
    // Pattern matching for common financial questions
    if (lowerMessage.includes('spend') && lowerMessage.includes('month')) {
      return `Based on your recent transactions, you spent approximately $2,847 this month across various categories. Your highest spending categories were:

• **Food & Dining**: $854 (30%)
• **Groceries**: $642 (22.5%) 
• **Bills**: $432 (15.3%)
• **Transportation**: $398 (14%)

This represents a 12.5% decrease from last month, which is great progress! Would you like me to break down any specific category further?`
    }
    
    if (lowerMessage.includes('budget') || lowerMessage.includes('recommend')) {
      return `Based on your spending patterns, here are my budget recommendations:

**Suggested Monthly Budget:**
• Food & Dining: $600 (vs current $854) - Try meal planning
• Groceries: $650 (vs current $642) - You're doing great here!
• Transportation: $400 (vs current $398) - Well managed
• Entertainment: $200 (vs current $285) - Consider streaming service audit

**Savings Goal**: Aim to save $500/month (15% of income)

Would you like me to help you set up automatic savings transfers or budget alerts?`
    }
    
    if (lowerMessage.includes('trend') || lowerMessage.includes('pattern')) {
      return `I've analyzed your spending trends over the past 6 months:

**Positive Trends:**
• Transportation costs down 8% (great job!)
• Subscription management improved
• Grocery spending more consistent

**Areas to Watch:**
• Dining out increased 15% recently
• Weekend spending spikes detected
• Small recurring charges may be adding up

**Seasonal Patterns:**
Your spending typically increases 20% during holiday months. Consider setting aside extra budget for November-December.

Want me to dive deeper into any specific trend?`
    }
    
    if (lowerMessage.includes('save') || lowerMessage.includes('goal')) {
      return `Let's talk about your savings goals! Here's what I see:

**Current Savings Rate**: ~12% of income
**Target Recommendation**: 20% for optimal financial health

**Quick Wins to Boost Savings:**
1. **Automate**: Set up automatic transfer of $300/month
2. **Optimize**: Reduce dining out by 2x/week = $120/month saved
3. **Review**: Cancel unused subscriptions = $45/month saved

**Bigger Goals:**
• Emergency fund: $10,000 (6 months expenses)
• Vacation fund: $3,000 by December
• Investment account: Start with $500/month

Which goal would you like to prioritize first?`
    }
    
    if (lowerMessage.includes('category') || lowerMessage.includes('breakdown')) {
      return `Here's your complete spending breakdown for this month:

**Essential Expenses (65%)**
• Groceries: $642 (22.5%)
• Bills & Utilities: $432 (15.3%)
• Transportation: $398 (14%)
• Healthcare: $165 (5.8%)
• Insurance: $178 (6.3%)

**Discretionary Expenses (35%)**
• Food & Dining: $854 (30%)
• Entertainment: $285 (10%)
• Shopping: $235 (8.2%)
• Personal Care: $87 (3.1%)

Your essential-to-discretionary ratio is healthy, but dining expenses are higher than typical recommendations (10-15%). Consider setting a weekly dining budget to optimize this category.`
    }
    
    // Default response for unmatched questions
    return `I understand you're asking about "${userMessage}". Based on your financial data, I can help you with:

• **Spending Analysis**: Track where your money goes
• **Budget Planning**: Create realistic spending plans  
• **Savings Goals**: Build toward your financial targets
• **Trend Analysis**: Spot patterns in your finances
• **Bill Optimization**: Find ways to reduce recurring costs

Could you be more specific about what aspect of your finances you'd like to explore? For example:
- "How much did I spend on [category] last month?"
- "What's my average weekly spending?"
- "How can I save more money?"
- "Show me my biggest expenses"`
  }

  const handleSendMessage = async () => {
    if (!chatMessage.trim()) return

    const newUserMessage: ChatMessage = {
      id: Date.now().toString(),
      type: 'user',
      message: chatMessage,
      timestamp: new Date().toLocaleString()
    }

    setChatHistory(prev => [...prev, newUserMessage])
    const currentMessage = chatMessage
    setChatMessage('')
    setIsLoading(true)

    // Call callback if provided
    onMessageSent?.(currentMessage)

    try {
      // Call the enhanced AI API with RAG
      const response = await fetch('/api/v1/ai/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          message: currentMessage,
          conversation_history: chatHistory.slice(-10).map(msg => ({
            role: msg.type === 'user' ? 'user' : 'assistant',
            content: msg.message
          })),
          use_rag: true,
          context: context
        })
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()
      
      const aiResponse: ChatMessage = {
        id: (Date.now() + 1).toString(),
        type: 'ai',
        message: data.response,
        timestamp: new Date().toLocaleString(),
        metadata: {
          suggestions: data.suggestions || ["Tell me more", "Show breakdown", "What's next?"],
          insights: data.insights || [],
          charts: data.charts || []
        }
      }
      
      setChatHistory(prev => [...prev, aiResponse])
      
    } catch (error) {
      console.error('Failed to get AI response:', error)
      
      // Fallback to mock response if API fails
      const aiResponse: ChatMessage = {
        id: (Date.now() + 1).toString(),
        type: 'ai',
        message: generateAIResponse(currentMessage),
        timestamp: new Date().toLocaleString(),
        metadata: {
          suggestions: ["Try again", "Check connection", "Contact support"]
        }
      }
      
      setChatHistory(prev => [...prev, aiResponse])
    } finally {
      setIsLoading(false)
    }
  }

  const handleQuickQuestion = (question: string) => {
    setChatMessage(question)
  }

  return (
    <Card className={`${height} flex flex-col`}>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center">
          <Brain className="w-5 h-5 mr-2" />
          {title}
        </CardTitle>
        <p className="text-sm text-muted-foreground">
          Ask me anything about your finances
        </p>
      </CardHeader>
      
      <CardContent className="flex-1 flex flex-col">
        {/* Chat History */}
        <div className="flex-1 overflow-y-auto space-y-4 mb-4">
          {chatHistory.length === 0 && (
            <div className="text-center py-8">
              <Brain className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-2">
                Hi! I'm your AI financial assistant.
              </p>
              <p className="text-sm text-muted-foreground">
                Ask me about your spending, budgets, or savings goals.
              </p>
            </div>
          )}
          
          {chatHistory.map((message) => (
            <div 
              key={message.id}
              className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div 
                className={`max-w-[85%] p-3 rounded-lg ${
                  message.type === 'user' 
                    ? 'bg-primary text-primary-foreground' 
                    : 'bg-muted'
                }`}
              >
                <div className="flex items-start gap-2">
                  {message.type === 'ai' && (
                    <Sparkles className="w-4 h-4 mt-0.5 text-primary flex-shrink-0" />
                  )}
                  <div className="flex-1">
                    <p className="text-sm whitespace-pre-wrap leading-relaxed">
                      {message.message}
                    </p>
                    <p className={`text-xs mt-2 opacity-70`}>
                      {message.timestamp}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          ))}
          
          {isLoading && (
            <div className="flex justify-start">
              <div className="bg-muted p-3 rounded-lg">
                <div className="flex items-center gap-2">
                  <Sparkles className="w-4 h-4 text-primary" />
                  <div className="flex gap-1">
                    <div className="w-2 h-2 bg-primary/60 rounded-full animate-pulse"></div>
                    <div className="w-2 h-2 bg-primary/60 rounded-full animate-pulse delay-100"></div>
                    <div className="w-2 h-2 bg-primary/60 rounded-full animate-pulse delay-200"></div>
                  </div>
                  <span className="text-sm text-muted-foreground">Analyzing...</span>
                </div>
              </div>
            </div>
          )}
          
          <div ref={chatEndRef} />
        </div>

        {/* Chat Input */}
        <div className="flex gap-2">
          <Input
            placeholder={placeholder}
            value={chatMessage}
            onChange={(e) => setChatMessage(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && !e.shiftKey && handleSendMessage()}
            disabled={isLoading}
            className="flex-1"
          />
          <Button 
            onClick={handleSendMessage}
            disabled={isLoading || !chatMessage.trim()}
            size="sm"
          >
            <Send className="w-4 h-4" />
          </Button>
        </div>

        {/* Quick Questions */}
        {quickQuestions.length > 0 && (
          <div className="mt-3">
            <p className="text-xs text-muted-foreground mb-2">Quick questions:</p>
            <div className="flex flex-wrap gap-1">
              {quickQuestions.map(question => (
                <Button 
                  key={question}
                  variant="outline" 
                  size="sm"
                  className="text-xs h-7"
                  onClick={() => handleQuickQuestion(question)}
                  disabled={isLoading}
                >
                  {question}
                </Button>
              ))}
            </div>
          </div>
        )}

        {/* Context Info */}
        {context && (
          <div className="mt-2 p-2 bg-muted/50 rounded text-xs text-muted-foreground">
            Context: {context}
          </div>
        )}
      </CardContent>
    </Card>
  )
}