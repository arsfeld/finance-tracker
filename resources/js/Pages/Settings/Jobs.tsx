import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '@/Layouts/AuthenticatedLayout'
import { JobMonitor } from '@/Components/JobMonitor'

export default function Jobs() {
  return (
    <AuthenticatedLayout>
      <Head title="Sync Jobs" />
      
      <div className="p-6">
        <div className="max-w-7xl mx-auto">
          <JobMonitor />
        </div>
      </div>
    </AuthenticatedLayout>
  )
}