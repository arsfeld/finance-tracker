import { useSyncLog } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useState } from "react";
import { Button } from "@/components/ui/button";

export default function SyncHistory() {
  const { data: logs, isLoading } = useSyncLog();
  const [expandedId, setExpandedId] = useState<number | null>(null);

  const formatDuration = (start: string, end: string | null) => {
    if (!end) return "Running...";
    const startDate = new Date(start);
    const endDate = new Date(end);
    const diffMs = endDate.getTime() - startDate.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return `${diffSec}s`;
    const diffMin = Math.floor(diffSec / 60);
    const remainingSec = diffSec % 60;
    return `${diffMin}m ${remainingSec}s`;
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case "success":
        return "text-green-600 dark:text-green-400";
      case "partial":
        return "text-yellow-600 dark:text-yellow-400";
      case "error":
        return "text-red-600 dark:text-red-400";
      default:
        return "text-blue-600 dark:text-blue-400";
    }
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Sync History</h1>
      <Card>
        <CardHeader>
          <CardTitle>Recent Sync Attempts</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-muted-foreground">Loading sync history...</div>
          ) : !logs || logs.length === 0 ? (
            <div className="text-muted-foreground py-8 text-center">
              No sync attempts found.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Started At</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead className="text-right">Added</TableHead>
                    <TableHead className="text-right">Updated</TableHead>
                    <TableHead className="w-[100px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => {
                    const isExpanded = expandedId === log.id;
                    const apiErrors = log.api_errors ? JSON.parse(log.api_errors) : [];
                    const hasDetails = log.error_message || apiErrors.length > 0;

                    return (
                      <>
                        <TableRow key={log.id}>
                          <TableCell className="text-sm whitespace-nowrap text-muted-foreground">
                            {new Date(log.started_at).toLocaleString()}
                          </TableCell>
                          <TableCell className={`text-sm font-medium capitalize ${getStatusColor(log.status)}`}>
                            {log.status}
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {formatDuration(log.started_at, log.completed_at)}
                          </TableCell>
                          <TableCell className="text-sm text-right font-medium">
                            {log.transactions_added}
                          </TableCell>
                          <TableCell className="text-sm text-right font-medium">
                            {log.transactions_updated}
                          </TableCell>
                          <TableCell>
                            {hasDetails && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setExpandedId(isExpanded ? null : log.id)}
                              >
                                {isExpanded ? "Hide Details" : "View Details"}
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                        {isExpanded && (
                          <TableRow className="bg-muted/30">
                            <TableCell colSpan={6} className="p-4">
                              <div className="space-y-3">
                                {log.error_message && (
                                  <div>
                                    <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1">
                                      Error Message
                                    </div>
                                    <div className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-2 rounded border border-red-100 dark:border-red-900/30">
                                      {log.error_message}
                                    </div>
                                  </div>
                                )}
                                {apiErrors.length > 0 && (
                                  <div>
                                    <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1">
                                      Account Status & API Errors
                                    </div>
                                    <ul className="space-y-1">
                                      {apiErrors.map((err: string, i: number) => (
                                        <li key={i} className="text-sm flex items-start gap-2">
                                          <span className="text-red-500 mt-1">•</span>
                                          <span>{err}</span>
                                        </li>
                                      ))}
                                    </ul>
                                  </div>
                                )}
                                {!log.error_message && apiErrors.length === 0 && (
                                  <div className="text-sm text-muted-foreground">
                                    No detailed error information available.
                                  </div>
                                )}
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
