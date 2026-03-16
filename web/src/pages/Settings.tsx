import { useSettings, useFilters } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function Settings() {
  const { data: settings, isLoading } = useSettings();
  const { data: filters } = useFilters();

  if (isLoading) return <div className="text-muted-foreground">Loading settings...</div>;

  const cfg = settings as Record<string, unknown> | undefined;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <p className="text-sm text-muted-foreground">
        Configuration is loaded from the <code className="bg-muted px-1 py-0.5 rounded text-xs">.env</code> file. Edit it and restart the server to apply changes.
      </p>

      <Tabs defaultValue="general">
        <TabsList>
          <TabsTrigger value="general">Configuration</TabsTrigger>
          <TabsTrigger value="filters">Filters ({filters?.length || 0})</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Server</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <SettingRow label="Listen Address" value={cfg?.listen_addr as string} />
              <SettingRow label="Billing Day" value={String(cfg?.billing_day ?? 15)} />
              <SettingRow label="Sync Schedule" value={cfg?.sync_schedule as string} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>SimpleFin</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <SettingRow
                label="Bridge URL"
                value={cfg?.simplefin_configured ? "Configured" : "Not configured"}
                status={cfg?.simplefin_configured as boolean}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>OpenRouter (LLM)</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <SettingRow
                label="API Key"
                value={cfg?.openrouter_configured ? "Configured" : "Not configured"}
                status={cfg?.openrouter_configured as boolean}
              />
              <SettingRow label="URL" value={cfg?.openrouter_url as string || "Not set"} />
              <SettingRow label="Model(s)" value={cfg?.openrouter_model as string || "Not set"} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                Ntfy
                <TestNotificationButton />
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <SettingRow label="Server" value={cfg?.ntfy_server as string} />
              <SettingRow
                label="Topic"
                value={cfg?.ntfy_topic as string || "Not set"}
                status={!!(cfg?.ntfy_topic)}
              />
              <SettingRow label="Warning Suffix" value={cfg?.ntfy_warning_suffix as string} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Email (SMTP)</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <SettingRow
                label="SMTP"
                value={cfg?.mailer_configured ? "Configured" : "Not configured"}
                status={cfg?.mailer_configured as boolean}
              />
              <SettingRow label="From" value={cfg?.mailer_from as string || "Not set"} />
              <SettingRow label="To" value={cfg?.mailer_to as string || "Not set"} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="filters">
          <Card>
            <CardHeader>
              <CardTitle>Transaction Filters</CardTitle>
            </CardHeader>
            <CardContent>
              {filters && filters.length > 0 ? (
                <div className="space-y-2">
                  {filters.map((f) => (
                    <div key={f.id} className="flex items-center gap-2 py-2 border-b border-border last:border-0">
                      <code className="text-sm bg-muted px-2 py-0.5 rounded">{f.pattern}</code>
                      <Badge variant="secondary">{f.match_type}</Badge>
                      <Badge variant={f.is_active ? "default" : "outline"}>
                        {f.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground">No filter rules configured.</div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function SettingRow({ label, value, status }: { label: string; value?: string; status?: boolean }) {
  return (
    <div className="flex items-center justify-between py-1">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="flex items-center gap-2">
        <code className="text-sm bg-muted px-2 py-0.5 rounded">{value || "-"}</code>
        {status !== undefined && (
          <span className={`w-2 h-2 rounded-full ${status ? "bg-green-500" : "bg-red-400"}`} />
        )}
      </div>
    </div>
  );
}

function TestNotificationButton() {
  const handleTest = async () => {
    try {
      await fetch("/api/settings/test-notification", { method: "POST" });
    } catch {
      // ignore
    }
  };

  return (
    <Button variant="outline" size="sm" onClick={handleTest}>
      Test
    </Button>
  );
}
