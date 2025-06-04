import { Link, Head } from '@inertiajs/react'
import { Button } from '@/components/ui/button'

export default function Home() {
  return (
    <>
      <Head title="Welcome" />
      
      <div className="min-h-screen bg-gradient-to-br from-background to-muted">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col items-center justify-center min-h-screen">
            <div className="text-center space-y-6">
              <h1 className="text-6xl font-bold text-foreground mb-4">
                WalletMind
              </h1>
              <p className="text-xl text-muted-foreground mb-8 max-w-2xl">
                Your AI-Powered Financial Intelligence
              </p>
              
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Button asChild size="lg">
                  <Link href="/login">
                    Sign In
                  </Link>
                </Button>
                <Button asChild variant="outline" size="lg">
                  <Link href="/register">
                    Create Account
                  </Link>
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}