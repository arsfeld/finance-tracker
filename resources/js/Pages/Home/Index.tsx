import { Link, Head } from '@inertiajs/react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

export default function Home() {
  return (
    <>
      <Head title="Finaro - AI-Powered Financial Intelligence" />
      
      <div className="min-h-screen">
        {/* Navigation */}
        <nav className="nav-glass fixed top-0 left-0 right-0 z-50">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between h-16">
              <div className="flex items-center space-x-3">
                {/* Logo Icon */}
                <div className="w-10 h-10 relative">
                  <svg viewBox="0 0 120 120" className="w-full h-full">
                    <defs>
                      <linearGradient id="logoGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" stopColor="#667eea" />
                        <stop offset="100%" stopColor="#764ba2" />
                      </linearGradient>
                    </defs>
                    <g transform="translate(60, 60)">
                      <path d="M -20,-15 Q -25,-20 -15,-25 L -10,-25 L -5,-30 L 5,-30 L 10,-25 L 15,-25 
                               Q 25,-20 20,-15 L 20,-10 L 25,-5 L 25,5 L 20,10 L 20,15 
                               Q 25,20 15,20 L 10,20 L 5,25 L -5,25 L -10,20 L -15,20 
                               Q -25,15 -20,10 L -20,5 L -25,0 L -25,-10 L -20,-15 Z" 
                            fill="none" 
                            stroke="url(#logoGradient)" 
                            strokeWidth="2"/>
                      <rect x="-12" y="-7.5" width="24" height="15" rx="2" ry="2" 
                            fill="rgba(102,126,234,0.1)" 
                            stroke="url(#logoGradient)" 
                            strokeWidth="1.5"/>
                      <rect x="-9" y="-4.5" width="5" height="4" rx="0.5" ry="0.5" 
                            fill="url(#logoGradient)"/>
                    </g>
                  </svg>
                </div>
                <span className="text-xl font-bold gradient-text">Finaro</span>
              </div>
              <div className="flex items-center space-x-4">
                <Button asChild variant="outline" size="sm">
                  <Link href="/login">Sign In</Link>
                </Button>
                <Button asChild size="sm" className="btn-primary">
                  <Link href="/register">Get Started</Link>
                </Button>
              </div>
            </div>
          </div>
        </nav>

        {/* Hero Section */}
        <section className="pt-32 pb-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto text-center max-w-6xl">
            <div className="mb-12">
              <h1 className="text-5xl md:text-7xl font-extrabold mb-6">
                <span className="gradient-text">AI-Powered</span><br />
                <span className="text-white">Financial Intelligence</span>
              </h1>
              <p className="text-xl md:text-2xl text-white/80 mb-8 max-w-3xl mx-auto leading-relaxed">
                Transform how you understand and manage money with conversational AI. 
                Get insights as simple as asking a friend, powered by sophisticated financial analysis.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
                <Button asChild size="lg" className="btn-primary text-lg px-8 py-4">
                  <Link href="/register">Start Free Trial</Link>
                </Button>
                <Button asChild variant="outline" size="lg" className="btn-outline text-lg px-8 py-4">
                  <Link href="#demo">Watch Demo</Link>
                </Button>
              </div>
            </div>
            
            {/* Hero Visual */}
            <div className="relative max-w-4xl mx-auto">
              <div className="card-glass p-8 animate-float">
                <div className="bg-gradient-to-r from-purple-600/20 to-blue-600/20 rounded-2xl p-6">
                  <div className="text-left">
                    <div className="flex items-center mb-4">
                      <div className="w-3 h-3 bg-red-400 rounded-full mr-2"></div>
                      <div className="w-3 h-3 bg-yellow-400 rounded-full mr-2"></div>
                      <div className="w-3 h-3 bg-green-400 rounded-full mr-2"></div>
                      <span className="text-white/60 text-sm ml-4">Finaro Chat</span>
                    </div>
                    <div className="space-y-4">
                      <div className="bg-white/10 rounded-lg p-3 max-w-xs">
                        <p className="text-white text-sm">"How much did I spend on dining out last month?"</p>
                      </div>
                      <div className="bg-gradient-to-r from-purple-600/40 to-blue-600/40 rounded-lg p-4 ml-8">
                        <p className="text-white text-sm mb-2">You spent <strong>$487</strong> on dining out last month, which is 23% more than your average. Here's the breakdown:</p>
                        <div className="text-xs text-white/80 space-y-1">
                          <div>• Restaurants: $312 (64%)</div>
                          <div>• Takeout: $175 (36%)</div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Features Section */}
        <section id="features" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <div className="text-center mb-16">
              <h2 className="text-4xl md:text-5xl font-bold mb-6">
                <span className="gradient-text">Why Choose Finaro?</span>
              </h2>
              <p className="text-xl text-white/80 max-w-3xl mx-auto">
                Not just another budgeting app. Finaro combines AI intelligence with financial expertise 
                to give you insights that actually matter.
              </p>
            </div>

            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
              {/* Feature 1 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-purple-600 to-blue-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Conversational AI</h3>
                <p className="text-white/70">Ask questions in plain English and get instant, intelligent answers about your finances.</p>
              </div>

              {/* Feature 2 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-green-600 to-teal-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Real-time Sync</h3>
                <p className="text-white/70">Automatic synchronization with all your bank accounts for up-to-the-minute insights.</p>
              </div>

              {/* Feature 3 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-pink-600 to-purple-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Team Collaboration</h3>
                <p className="text-white/70">Share insights with family or business partners while maintaining privacy and control.</p>
              </div>

              {/* Feature 4 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-orange-600 to-red-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v4a2 2 0 01-2 2h-2a2 2 0 00-2-2z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Smart Analytics</h3>
                <p className="text-white/70">Advanced pattern recognition and trend analysis that learns from your financial behavior.</p>
              </div>

              {/* Feature 5 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Bank-Level Security</h3>
                <p className="text-white/70">End-to-end encryption and read-only access ensure your data stays safe and private.</p>
              </div>

              {/* Feature 6 */}
              <div className="card-glass p-8 text-center group hover:scale-105 transition-transform duration-300">
                <div className="w-16 h-16 mx-auto mb-6 bg-gradient-to-r from-cyan-600 to-blue-600 rounded-2xl flex items-center justify-center">
                  <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-5 5v-5zM9 17H4l5 5v-5zM21 3.6v16.8a.6.6 0 01-.6.6H3.6a.6.6 0 01-.6-.6V3.6a.6.6 0 01.6-.6h16.8a.6.6 0 01.6.6z" />
                  </svg>
                </div>
                <h3 className="text-xl font-bold text-white mb-4">Multi-Account Support</h3>
                <p className="text-white/70">Connect unlimited bank accounts, credit cards, and financial institutions in one place.</p>
              </div>
            </div>
          </div>
        </section>

        {/* Pricing Section */}
        <section id="pricing" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <div className="text-center mb-16">
              <h2 className="text-4xl md:text-5xl font-bold mb-6">
                <span className="gradient-text">Simple, Transparent Pricing</span>
              </h2>
              <p className="text-xl text-white/80 max-w-3xl mx-auto">
                Start free and upgrade when you need more. No hidden fees, no surprises.
              </p>
            </div>

            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
              {/* Free Tier */}
              <div className="card-glass p-8 text-center relative">
                <h3 className="text-xl font-bold text-white mb-2">Free</h3>
                <div className="mb-6">
                  <span className="text-4xl font-bold text-white">$0</span>
                  <span className="text-white/60">/month</span>
                </div>
                <ul className="text-left space-y-3 mb-8 text-white/80">
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    1 bank connection
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    3 months history
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    5 AI questions/month
                  </li>
                </ul>
                <Button asChild className="w-full btn-outline">
                  <Link href="/register">Get Started</Link>
                </Button>
              </div>

              {/* Personal Tier */}
              <div className="card-glass p-8 text-center relative border-2 border-purple-400">
                <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                  <span className="bg-gradient-to-r from-purple-600 to-blue-600 text-white px-4 py-1 rounded-full text-sm font-semibold">
                    Most Popular
                  </span>
                </div>
                <h3 className="text-xl font-bold text-white mb-2">Personal</h3>
                <div className="mb-6">
                  <span className="text-4xl font-bold text-white">$9</span>
                  <span className="text-white/60">/month</span>
                </div>
                <ul className="text-left space-y-3 mb-8 text-white/80">
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Unlimited connections
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Full history
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Unlimited AI chat
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Email notifications
                  </li>
                </ul>
                <Button asChild className="w-full btn-primary">
                  <Link href="/register">Start Free Trial</Link>
                </Button>
              </div>

              {/* Family Tier */}
              <div className="card-glass p-8 text-center relative">
                <h3 className="text-xl font-bold text-white mb-2">Family</h3>
                <div className="mb-6">
                  <span className="text-4xl font-bold text-white">$15</span>
                  <span className="text-white/60">/month</span>
                </div>
                <ul className="text-left space-y-3 mb-8 text-white/80">
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Everything in Personal
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    5 users per org
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Shared categories
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Family insights
                  </li>
                </ul>
                <Button asChild className="w-full btn-outline">
                  <Link href="/register">Start Free Trial</Link>
                </Button>
              </div>

              {/* Business Tier */}
              <div className="card-glass p-8 text-center relative">
                <h3 className="text-xl font-bold text-white mb-2">Business</h3>
                <div className="mb-6">
                  <span className="text-4xl font-bold text-white">$29</span>
                  <span className="text-white/60">/month</span>
                </div>
                <ul className="text-left space-y-3 mb-8 text-white/80">
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Everything in Family
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Unlimited users
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    Admin controls
                  </li>
                  <li className="flex items-center">
                    <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    API access
                  </li>
                </ul>
                <Button asChild className="w-full btn-outline">
                  <Link href="/register">Start Free Trial</Link>
                </Button>
              </div>
            </div>
          </div>
        </section>

        {/* CTA Section */}
        <section className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-4xl text-center">
            <div className="card-glass p-12">
              <h2 className="text-4xl md:text-5xl font-bold mb-6">
                <span className="gradient-text">Ready to Transform</span><br />
                <span className="text-white">Your Financial Life?</span>
              </h2>
              <p className="text-xl text-white/80 mb-8 max-w-2xl mx-auto">
                Join thousands of users who've already discovered the power of AI-driven financial insights.
                Start your free trial today.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Button asChild size="lg" className="btn-primary text-lg px-8 py-4">
                  <Link href="/register">Start Free Trial</Link>
                </Button>
                <Button asChild variant="outline" size="lg" className="btn-outline text-lg px-8 py-4">
                  <Link href="/login">Already have an account?</Link>
                </Button>
              </div>
              <p className="text-sm text-white/60 mt-4">No credit card required • 14-day free trial • Cancel anytime</p>
            </div>
          </div>
        </section>

        {/* Footer */}
        <footer className="py-12 px-4 sm:px-6 lg:px-8 border-t border-white/10">
          <div className="container mx-auto max-w-6xl">
            <div className="text-center">
              <div className="flex items-center justify-center space-x-3 mb-4">
                <div className="w-8 h-8">
                  <svg viewBox="0 0 120 120" className="w-full h-full">
                    <defs>
                      <linearGradient id="footerLogoGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" stopColor="#667eea" />
                        <stop offset="100%" stopColor="#764ba2" />
                      </linearGradient>
                    </defs>
                    <g transform="translate(60, 60)">
                      <path d="M -20,-15 Q -25,-20 -15,-25 L -10,-25 L -5,-30 L 5,-30 L 10,-25 L 15,-25 
                               Q 25,-20 20,-15 L 20,-10 L 25,-5 L 25,5 L 20,10 L 20,15 
                               Q 25,20 15,20 L 10,20 L 5,25 L -5,25 L -10,20 L -15,20 
                               Q -25,15 -20,10 L -20,5 L -25,0 L -25,-10 L -20,-15 Z" 
                            fill="none" 
                            stroke="url(#footerLogoGradient)" 
                            strokeWidth="2"/>
                    </g>
                  </svg>
                </div>
                <span className="text-xl font-bold gradient-text">Finaro</span>
              </div>
              <p className="text-white/60 mb-4">AI-Powered Financial Intelligence</p>
              <p className="text-white/40 text-sm">
                © 2024 Finaro. All rights reserved. • Privacy Policy • Terms of Service
              </p>
            </div>
          </div>
        </footer>
      </div>
    </>
  )
}