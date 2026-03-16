import { useState } from "react";
import { useSettings, useUpdateSettings, useFilters } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function Settings() {
  const { data: settings, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();
  const { data: filters } = useFilters();
  const [editValues, setEditValues] = useState<Record<string, string>>({});

  const fields = [
    { key: "billing_day", label: "Billing Day (1-28)", placeholder: "15" },
    { key: "sync_schedule", label: "Sync Schedule (cron)", placeholder: "0 0 */6 * * *" },
    { key: "simplefin_bridge_url", label: "SimpleFin Bridge URL", placeholder: "https://..." },
    { key: "openrouter_url", label: "OpenRouter URL", placeholder: "https://openrouter.ai/api/v1/chat/completions" },
    { key: "openrouter_api_key", label: "OpenRouter API Key", placeholder: "sk-..." },
    { key: "openrouter_model", label: "OpenRouter Model(s)", placeholder: "deepseek/deepseek-r1:free" },
    { key: "ntfy_topic", label: "Ntfy Topic", placeholder: "finance" },
    { key: "mailer_url", label: "SMTP URL", placeholder: "smtp://user:pass@host:587" },
    { key: "mailer_from", label: "Email From", placeholder: "finance@example.com" },
    { key: "mailer_to", label: "Email To", placeholder: "you@example.com" },
  ];

  const getValue = (key: string) => editValues[key] ?? settings?.[key] ?? "";

  const handleSave = () => {
    if (Object.keys(editValues).length > 0) {
      updateSettings.mutate(editValues, {
        onSuccess: () => setEditValues({}),
      });
    }
  };

  if (isLoading) return <div className="text-muted-foreground">Loading settings...</div>;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Settings</h1>

      <Tabs defaultValue="general">
        <TabsList>
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="filters">Filters ({filters?.length || 0})</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {fields.map((f) => (
                <div key={f.key}>
                  <label className="text-sm font-medium text-muted-foreground block mb-1">{f.label}</label>
                  <input
                    type={f.key.includes("key") || f.key.includes("password") ? "password" : "text"}
                    value={getValue(f.key)}
                    onChange={(e) => setEditValues((prev) => ({ ...prev, [f.key]: e.target.value }))}
                    placeholder={f.placeholder}
                    className="w-full px-3 py-2 border border-input rounded-md bg-background text-sm"
                  />
                </div>
              ))}
              <Button onClick={handleSave} disabled={updateSettings.isPending || Object.keys(editValues).length === 0}>
                {updateSettings.isPending ? "Saving..." : "Save Settings"}
              </Button>
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
                    <div key={f.id} className="flex items-center gap-2 py-1 border-b border-border last:border-0">
                      <span className="text-sm font-mono">{f.pattern}</span>
                      <span className="text-xs text-muted-foreground">{f.match_type}</span>
                      <span className={`text-xs ${f.is_active ? "text-green-600" : "text-muted-foreground"}`}>
                        {f.is_active ? "Active" : "Inactive"}
                      </span>
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
