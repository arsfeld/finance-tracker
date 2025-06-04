import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { JobProgress } from './SyncButton';
import { RefreshCw, Settings, TrendingUp, Users, Clock, CheckCircle, XCircle, Pause } from 'lucide-react';

interface SyncJob {
  id: string;
  type: 'sync_transactions' | 'sync_accounts' | 'full_sync' | 'test_connection';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused';
  title: string;
  description?: string;
  progress_current: number;
  progress_total: number;
  progress_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  provider_connection_id?: string;
}

interface JobStats {
  by_status: Record<string, { count: number; avg_duration_seconds?: number }>;
  total: number;
}

interface WorkerStats {
  by_status: Record<string, { count: number; jobs: number; capacity: number }>;
  total_workers: number;
  total_jobs: number;
  total_capacity: number;
  utilization_percent: number;
}

interface Worker {
  id: string;
  hostname: string;
  pid: number;
  started_at: string;
  last_heartbeat: string;
  status: string;
  max_concurrent_jobs: number;
  current_job_count: number;
  version?: string;
  worker_type?: string;
}

export function JobMonitor() {
  const [jobs, setJobs] = useState<SyncJob[]>([]);
  const [jobStats, setJobStats] = useState<JobStats | null>(null);
  const [workerStats, setWorkerStats] = useState<WorkerStats | null>(null);
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [filter, setFilter] = useState<'all' | 'running' | 'failed' | 'completed'>('all');

  const fetchData = async () => {
    try {
      const [jobsRes, jobStatsRes, workerStatsRes, workersRes] = await Promise.all([
        fetch('/api/v1/jobs?limit=20'),
        fetch('/api/v1/jobs/stats'),
        fetch('/api/v1/workers/stats'),
        fetch('/api/v1/workers'),
      ]);

      if (jobsRes.ok) {
        const jobsData = await jobsRes.json();
        setJobs(jobsData.jobs || []);
      }

      if (jobStatsRes.ok) {
        const statsData = await jobStatsRes.json();
        setJobStats(statsData);
      }

      if (workerStatsRes.ok) {
        const workerStatsData = await workerStatsRes.json();
        setWorkerStats(workerStatsData);
      }

      if (workersRes.ok) {
        const workersData = await workersRes.json();
        setWorkers(workersData.workers || []);
      }
    } catch (error) {
      console.error('Failed to fetch job monitor data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(fetchData, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const filteredJobs = jobs.filter(job => {
    if (filter === 'all') return true;
    return job.status === filter;
  });

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-600" />;
      case 'running':
        return <RefreshCw className="h-4 w-4 text-blue-600 animate-spin" />;
      case 'paused':
        return <Pause className="h-4 w-4 text-yellow-600" />;
      default:
        return <Clock className="h-4 w-4 text-gray-600" />;
    }
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return 'N/A';
    
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  };

  const isWorkerHealthy = (worker: Worker) => {
    const lastHeartbeat = new Date(worker.last_heartbeat);
    const now = new Date();
    const diffMinutes = (now.getTime() - lastHeartbeat.getTime()) / (1000 * 60);
    return diffMinutes < 2; // Consider healthy if heartbeat within 2 minutes
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="h-8 w-8 animate-spin" />
        <span className="ml-2">Loading job monitor...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Sync Job Monitor</h2>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={autoRefresh ? 'bg-green-50 text-green-700' : ''}
          >
            <RefreshCw className={`h-4 w-4 ${autoRefresh ? 'animate-spin' : ''}`} />
            <span className="ml-1">{autoRefresh ? 'Auto Refresh' : 'Manual Refresh'}</span>
          </Button>
          <Button variant="outline" size="sm" onClick={fetchData}>
            <RefreshCw className="h-4 w-4" />
            <span className="ml-1">Refresh Now</span>
          </Button>
        </div>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Job Stats */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center">
              <TrendingUp className="h-4 w-4 mr-2" />
              Job Stats (7 days)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{jobStats?.total || 0}</div>
            <div className="text-xs text-gray-600 space-y-1">
              {jobStats?.by_status && Object.entries(jobStats.by_status).map(([status, data]) => (
                <div key={status} className="flex justify-between">
                  <span className="capitalize">{status}:</span>
                  <span>{data.count}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Worker Stats */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center">
              <Users className="h-4 w-4 mr-2" />
              Workers
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{workerStats?.total_workers || 0}</div>
            <div className="text-xs text-gray-600 space-y-1">
              <div className="flex justify-between">
                <span>Active Jobs:</span>
                <span>{workerStats?.total_jobs || 0}</span>
              </div>
              <div className="flex justify-between">
                <span>Capacity:</span>
                <span>{workerStats?.total_capacity || 0}</span>
              </div>
              <div className="flex justify-between">
                <span>Utilization:</span>
                <span>{Math.round(workerStats?.utilization_percent || 0)}%</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recent Performance */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center">
              <Clock className="h-4 w-4 mr-2" />
              Avg Duration
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {jobStats?.by_status?.completed?.avg_duration_seconds
                ? formatDuration(jobStats.by_status.completed.avg_duration_seconds)
                : 'N/A'}
            </div>
            <div className="text-xs text-gray-600">
              Completed jobs average
            </div>
          </CardContent>
        </Card>

        {/* Success Rate */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center">
              <CheckCircle className="h-4 w-4 mr-2" />
              Success Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {jobStats?.total && jobStats.by_status?.completed
                ? Math.round((jobStats.by_status.completed.count / jobStats.total) * 100)
                : 0}%
            </div>
            <div className="text-xs text-gray-600">
              Last 7 days
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Workers Status */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Active Workers</CardTitle>
          <CardDescription>
            Workers currently processing or available for jobs
          </CardDescription>
        </CardHeader>
        <CardContent>
          {workers.length === 0 ? (
            <div className="text-center text-gray-500 py-4">
              No active workers found
            </div>
          ) : (
            <div className="space-y-2">
              {workers.map((worker) => (
                <div
                  key={worker.id}
                  className="flex items-center justify-between p-3 border rounded-lg"
                >
                  <div className="flex items-center space-x-3">
                    <div className={`w-2 h-2 rounded-full ${
                      isWorkerHealthy(worker) ? 'bg-green-500' : 'bg-red-500'
                    }`} />
                    <div>
                      <div className="font-medium text-sm">{worker.id}</div>
                      <div className="text-xs text-gray-600">
                        {worker.hostname} (PID: {worker.pid})
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center space-x-4 text-sm">
                    <div>
                      <span className="text-gray-600">Jobs:</span>{' '}
                      <span className="font-medium">
                        {worker.current_job_count}/{worker.max_concurrent_jobs}
                      </span>
                    </div>
                    <Badge variant={worker.status === 'active' ? 'default' : 'secondary'}>
                      {worker.status}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Job Filter */}
      <div className="flex items-center space-x-2">
        <span className="text-sm font-medium">Filter:</span>
        <div className="flex space-x-1">
          {(['all', 'running', 'failed', 'completed'] as const).map((status) => (
            <Button
              key={status}
              variant={filter === status ? 'default' : 'outline'}
              size="sm"
              onClick={() => setFilter(status)}
              className="capitalize"
            >
              {status}
            </Button>
          ))}
        </div>
      </div>

      {/* Recent Jobs */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent Jobs</CardTitle>
          <CardDescription>
            Latest sync jobs and their status
          </CardDescription>
        </CardHeader>
        <CardContent>
          {filteredJobs.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              No jobs found
            </div>
          ) : (
            <div className="space-y-4">
              {filteredJobs.map((job) => (
                <JobProgress
                  key={job.id}
                  job={job}
                  onJobUpdate={(updatedJob) => {
                    setJobs(prev => prev.map(j => j.id === updatedJob.id ? updatedJob : j));
                  }}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}