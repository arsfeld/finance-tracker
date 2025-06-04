import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { RefreshCw, Play, Pause, Square, RotateCcw } from 'lucide-react';

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

interface SyncButtonProps {
  connectionId: string;
  connectionName: string;
  onSyncStart?: (job: SyncJob) => void;
  disabled?: boolean;
}

export function SyncButton({ connectionId, connectionName, onSyncStart, disabled = false }: SyncButtonProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [syncType, setSyncType] = useState<'transactions' | 'full' | 'accounts' | 'test'>('transactions');

  const handleSync = async () => {
    setIsLoading(true);
    
    try {
      const response = await fetch(`/api/v1/connections/${connectionId}/sync`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          type: syncType,
          priority: 'normal',
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to start sync');
      }

      const job: SyncJob = await response.json();
      onSyncStart?.(job);
      
      // Show success message
      // You could use a toast notification here
      
    } catch (error) {
      console.error('Sync failed:', error);
      // Show error message
      // You could use a toast notification here
    } finally {
      setIsLoading(false);
    }
  };

  const getSyncIcon = () => {
    switch (syncType) {
      case 'full':
        return <RefreshCw className="h-4 w-4" />;
      case 'accounts':
        return <Play className="h-4 w-4" />;
      case 'test':
        return <RotateCcw className="h-4 w-4" />;
      default:
        return <RefreshCw className="h-4 w-4" />;
    }
  };

  const getSyncLabel = () => {
    switch (syncType) {
      case 'full':
        return 'Full Sync';
      case 'accounts':
        return 'Sync Accounts';
      case 'test':
        return 'Test Connection';
      default:
        return 'Sync Transactions';
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center space-x-2">
        {/* Sync Type Selector */}
        <select
          value={syncType}
          onChange={(e) => setSyncType(e.target.value as any)}
          disabled={isLoading || disabled}
          className="text-sm border rounded px-2 py-1"
        >
          <option value="transactions">Transactions</option>
          <option value="accounts">Accounts</option>
          <option value="full">Full Sync</option>
          <option value="test">Test Connection</option>
        </select>

        {/* Sync Button */}
        <Button
          onClick={handleSync}
          disabled={isLoading || disabled}
          size="sm"
          className="flex items-center space-x-1"
        >
          {isLoading ? (
            <RefreshCw className="h-4 w-4 animate-spin" />
          ) : (
            getSyncIcon()
          )}
          <span>{isLoading ? 'Starting...' : getSyncLabel()}</span>
        </Button>
      </div>
    </div>
  );
}

interface JobProgressProps {
  job: SyncJob;
  onJobUpdate?: (job: SyncJob) => void;
  onCancel?: () => void;
  onPause?: () => void;
  onResume?: () => void;
  onRetry?: () => void;
}

export function JobProgress({ job, onJobUpdate, onCancel, onPause, onResume, onRetry }: JobProgressProps) {
  const [isUpdating, setIsUpdating] = useState(false);

  const progressPercentage = job.progress_total > 0 
    ? Math.round((job.progress_current / job.progress_total) * 100)
    : 0;

  const getStatusColor = (status: SyncJob['status']) => {
    switch (status) {
      case 'running':
        return 'text-blue-600 bg-blue-50';
      case 'completed':
        return 'text-green-600 bg-green-50';
      case 'failed':
        return 'text-red-600 bg-red-50';
      case 'paused':
        return 'text-yellow-600 bg-yellow-50';
      case 'cancelled':
        return 'text-gray-600 bg-gray-50';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const handleAction = async (action: 'cancel' | 'pause' | 'resume' | 'retry') => {
    if (isUpdating) return;
    
    setIsUpdating(true);
    
    try {
      const response = await fetch(`/api/v1/jobs/${job.id}/${action}`, {
        method: 'POST',
      });

      if (!response.ok) {
        throw new Error(`Failed to ${action} job`);
      }

      const result = await response.json();
      
      // Trigger callbacks
      switch (action) {
        case 'cancel':
          onCancel?.();
          break;
        case 'pause':
          onPause?.();
          break;
        case 'resume':
          onResume?.();
          break;
        case 'retry':
          onRetry?.();
          break;
      }
      
    } catch (error) {
      console.error(`Failed to ${action} job:`, error);
    } finally {
      setIsUpdating(false);
    }
  };

  const formatTime = (dateString?: string) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleTimeString();
  };

  const formatDuration = () => {
    if (!job.started_at) return 'N/A';
    
    const start = new Date(job.started_at);
    const end = job.completed_at ? new Date(job.completed_at) : new Date();
    const diffMs = end.getTime() - start.getTime();
    
    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  };

  return (
    <Card className="w-full">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm font-medium">{job.title}</CardTitle>
            {job.description && (
              <CardDescription className="text-xs">{job.description}</CardDescription>
            )}
          </div>
          <div className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(job.status)}`}>
            {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="space-y-3">
        {/* Progress Bar */}
        {job.status === 'running' && (
          <div className="space-y-1">
            <div className="flex justify-between text-xs text-gray-600">
              <span>{job.progress_message || 'Processing...'}</span>
              <span>{progressPercentage}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${progressPercentage}%` }}
              />
            </div>
            <div className="text-xs text-gray-500">
              {job.progress_current} / {job.progress_total}
            </div>
          </div>
        )}

        {/* Error Message */}
        {job.status === 'failed' && job.error_message && (
          <div className="text-sm text-red-600 bg-red-50 p-2 rounded">
            {job.error_message}
          </div>
        )}

        {/* Job Info */}
        <div className="grid grid-cols-2 gap-2 text-xs text-gray-600">
          <div>
            <span className="font-medium">Started:</span> {formatTime(job.started_at)}
          </div>
          <div>
            <span className="font-medium">Duration:</span> {formatDuration()}
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex space-x-2">
          {job.status === 'running' && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => handleAction('pause')}
                disabled={isUpdating}
                className="flex items-center space-x-1"
              >
                <Pause className="h-3 w-3" />
                <span>Pause</span>
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => handleAction('cancel')}
                disabled={isUpdating}
                className="flex items-center space-x-1 text-red-600 hover:text-red-700"
              >
                <Square className="h-3 w-3" />
                <span>Cancel</span>
              </Button>
            </>
          )}

          {job.status === 'paused' && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => handleAction('resume')}
                disabled={isUpdating}
                className="flex items-center space-x-1"
              >
                <Play className="h-3 w-3" />
                <span>Resume</span>
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => handleAction('cancel')}
                disabled={isUpdating}
                className="flex items-center space-x-1 text-red-600 hover:text-red-700"
              >
                <Square className="h-3 w-3" />
                <span>Cancel</span>
              </Button>
            </>
          )}

          {job.status === 'failed' && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleAction('retry')}
              disabled={isUpdating}
              className="flex items-center space-x-1"
            >
              <RotateCcw className="h-3 w-3" />
              <span>Retry</span>
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}