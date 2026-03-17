import { useState } from "react";
import { useAnalyses, useAnalysis, useTriggerAnalysis } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";

export default function Analysis() {
  const [selectedId, setSelectedId] = useState<number>(0);
  const { data: analyses, isLoading } = useAnalyses();
  const { data: detail } = useAnalysis(selectedId);
  const triggerAnalysis = useTriggerAnalysis();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Analysis History</h1>
        <Button
          onClick={() => triggerAnalysis.mutate()}
          disabled={triggerAnalysis.isPending}
        >
          {triggerAnalysis.isPending ? "Running..." : "Run Analysis"}
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle>Past Analyses</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="text-muted-foreground">Loading...</div>
            ) : analyses && analyses.length > 0 ? (
              <div className="space-y-1">
                {analyses.map((a) => (
                  <Button
                    key={a.id}
                    variant={selectedId === a.id ? "default" : "ghost"}
                    size="sm"
                    className="w-full justify-start text-left"
                    onClick={() => setSelectedId(a.id)}
                  >
                    <div>
                      <div className="text-xs">{new Date(a.created_at).toLocaleDateString()}</div>
                      <div className="text-xs text-muted-foreground">{a.model_used}</div>
                    </div>
                  </Button>
                ))}
              </div>
            ) : (
              <div className="text-muted-foreground text-sm">
                No analyses yet. Run a sync and analysis to see results.
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>
              {detail
                ? `Analysis from ${new Date(detail.created_at).toLocaleDateString()}`
                : "Select an analysis"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {detail ? (
              <MarkdownContent>{detail.response_text}</MarkdownContent>
            ) : (
              <div className="text-muted-foreground text-sm">
                Select an analysis from the list to view its details.
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
