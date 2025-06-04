import { useForm, Link, Head } from '@inertiajs/react'
import { FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

interface RegisterForm {
  email: string
  password: string
  organizationName: string
}

export default function Register() {
  const { data, setData, post, processing, errors } = useForm<RegisterForm>({
    email: '',
    password: '',
    organizationName: '',
  })

  const submit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    post('/auth/register')
  }

  return (
    <>
      <Head title="Create Account" />
      
      <div className="min-h-screen flex items-center justify-center bg-background p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="space-y-1">
            <CardTitle className="text-2xl text-center">Create account</CardTitle>
            <CardDescription className="text-center">
              Enter your details to create a new account and organization
            </CardDescription>
          </CardHeader>
          
          <form onSubmit={submit}>
            <CardContent className="space-y-4">
              {errors && (errors as any)['_'] && (
                <div className="rounded-md bg-destructive/15 p-3">
                  <p className="text-sm text-destructive">{(errors as any)['_']}</p>
                </div>
              )}
              
              <div className="space-y-2">
                <Label htmlFor="email">Email address</Label>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={data.email}
                  onChange={(e) => setData('email', e.target.value)}
                  placeholder="you@example.com"
                />
                {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={data.password}
                  onChange={(e) => setData('password', e.target.value)}
                  placeholder="At least 6 characters"
                />
                {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="organizationName">Organization Name</Label>
                <Input
                  id="organizationName"
                  name="organizationName"
                  type="text"
                  required
                  value={data.organizationName}
                  onChange={(e) => setData('organizationName', e.target.value)}
                  placeholder="My Organization"
                />
                {errors.organizationName && <p className="text-sm text-destructive">{errors.organizationName}</p>}
              </div>
            </CardContent>

            <CardFooter className="flex flex-col space-y-4">
              <Button 
                type="submit" 
                className="w-full" 
                disabled={processing}
              >
                {processing ? 'Creating account...' : 'Create account'}
              </Button>
              
              <p className="text-center text-sm text-muted-foreground">
                Already have an account?{' '}
                <Link href="/login" className="font-medium text-primary hover:underline">
                  Sign in here
                </Link>
              </p>
            </CardFooter>
          </form>
        </Card>
      </div>
    </>
  )
}