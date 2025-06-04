import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { AccountPageProps } from '@/types'

export default function AccountDetail({ accountId }: AccountPageProps) {
  return (
    <AuthenticatedLayout>
      <Head title="Account Details" />
      
      <div className="p-6 space-y-6">
        <div className="max-w-7xl mx-auto">
          <h1 className="text-3xl font-bold text-foreground mb-6">
            Account Details
          </h1>
          
          <Card>
            <CardHeader>
              <CardTitle>Account Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Account ID</p>
                  <p className="text-foreground">{accountId}</p>
                </div>
                <p className="text-muted-foreground">
                  Account details will be displayed here.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}