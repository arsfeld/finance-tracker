import { useForm, Link, Head } from '@inertiajs/react'
import { FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

interface LoginForm {
  email: string
  password: string
}

export default function Login() {
  const { data, setData, post, processing, errors } = useForm<LoginForm>({
    email: '',
    password: '',
  })

  const submit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    post('/auth/login')
  }

  return (
    <>
      <Head title="Sign In - Finaro" />
      
      <div className="min-h-screen flex items-center justify-center p-4" style={{ background: 'var(--color-bg-primary)' }}>
        <Card className="w-full max-w-md animate-float">
          <CardHeader className="space-y-3 text-center">
            <div className="mx-auto mb-4">
              {/* Finaro Logo */}
              <svg viewBox="0 0 80 80" xmlns="http://www.w3.org/2000/svg" width="64" height="64">
                <defs>
                  <linearGradient id="loginIconGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" style={{ stopColor: '#667eea', stopOpacity: 1 }} />
                    <stop offset="100%" style={{ stopColor: '#764ba2', stopOpacity: 1 }} />
                  </linearGradient>
                </defs>
                
                <g transform="translate(40, 40)">
                  <path d="M -20,-15 Q -25,-20 -15,-25 L -10,-25 L -5,-30 L 5,-30 L 10,-25 L 15,-25 
                           Q 25,-20 20,-15 L 20,-10 L 25,-5 L 25,5 L 20,10 L 20,15 
                           Q 25,20 15,20 L 10,20 L 5,25 L -5,25 L -10,20 L -15,20 
                           Q -25,15 -20,10 L -20,5 L -25,0 L -25,-10 L -20,-15 Z" 
                        fill="none" 
                        stroke="url(#loginIconGradient)" 
                        strokeWidth="2"/>
                  
                  <rect x="-12" y="-7.5" width="24" height="15" rx="2" ry="2" 
                        fill="rgba(102,126,234,0.1)" 
                        stroke="url(#loginIconGradient)" 
                        strokeWidth="1.5"/>
                  
                  <rect x="-9" y="-4.5" width="5" height="4" rx="0.5" ry="0.5" 
                        fill="url(#loginIconGradient)"/>
                </g>
              </svg>
            </div>
            <CardTitle className="text-3xl font-extrabold gradient-text">Welcome to Finaro</CardTitle>
            <CardDescription className="text-gray-600">
              Sign in to access your AI-powered financial intelligence
            </CardDescription>
          </CardHeader>
          
          <form onSubmit={submit}>
            <CardContent className="space-y-4">
              {errors && (errors as any)['_'] && (
                <div className="rounded-xl p-3" style={{ background: 'var(--color-danger-gradient)' }}>
                  <p className="text-sm text-white font-medium">{(errors as any)['_']}</p>
                </div>
              )}
              
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={data.email}
                  onChange={(e) => setData('email', e.target.value)}
                  placeholder="Enter your email"
                />
                {errors.email && <p className="text-sm text-red-600">{errors.email}</p>}
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={data.password}
                  onChange={(e) => setData('password', e.target.value)}
                  placeholder="Enter your password"
                />
                {errors.password && <p className="text-sm text-red-600">{errors.password}</p>}
              </div>
            </CardContent>

            <CardFooter className="flex flex-col space-y-4">
              <Button 
                type="submit" 
                className="w-full" 
                disabled={processing}
              >
                {processing ? 'Signing in...' : 'Sign in'}
              </Button>
              
              <p className="text-center text-sm text-gray-600">
                Don't have an account?{' '}
                <Link href="/register" className="font-semibold gradient-text hover:underline">
                  Create one here
                </Link>
              </p>
            </CardFooter>
          </form>
        </Card>
      </div>
    </>
  )
}