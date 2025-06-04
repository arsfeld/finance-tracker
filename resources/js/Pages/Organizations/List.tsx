import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { OrganizationPageProps } from '@/types'

export default function OrganizationsList({ organizations = [] }: OrganizationPageProps) {
  return (
    <AuthenticatedLayout>
      <Head title="Organizations" />
      
      <div className="p-6 space-y-6">
        <div className="max-w-7xl mx-auto">
          <h1 className="text-3xl font-bold text-foreground mb-6">
            Organizations
          </h1>
          
          <Card>
            <CardHeader>
              <CardTitle>Switch Organization</CardTitle>
            </CardHeader>
            <CardContent>
              {organizations.length > 0 ? (
                <div className="space-y-4">
                  {organizations.map((org) => (
                    <Card key={org.id} className="hover:shadow-md transition-shadow">
                      <CardContent className="p-4">
                        <h3 className="text-lg font-medium text-foreground">{org.name}</h3>
                        <p className="text-sm text-muted-foreground">ID: {org.id}</p>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">
                    No organizations found.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}