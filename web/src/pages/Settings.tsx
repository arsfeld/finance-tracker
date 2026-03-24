import { useSettings, useUniqueCategories, useSetCategoryExcluded, type CategoryInfoItem } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const CATEGORY_EMOJIS: Record<string, string> = {
  "Groceries": "🛒", "Dining": "🍽️", "Transportation": "🚗", "Gas": "⛽",
  "Entertainment": "🎬", "Shopping": "🛍️", "Subscriptions": "📱", "Healthcare": "🏥",
  "Utilities": "💡", "Insurance": "🛡️", "Housing": "🏠", "Childcare": "👶",
  "Education": "📚", "Clothing": "👕", "Personal Care": "💇", "Fitness": "🏋️",
  "Travel": "✈️", "Gifts": "🎁", "Pets": "🐾", "Home Improvement": "🔨",
  "Electronics": "💻", "Pharmacy": "💊", "Coffee": "☕", "Alcohol": "🍷",
  "Parking": "🅿️", "ATM": "🏧", "Fees": "💳", "Other": "📦",
};

function emojiFor(name: string): string {
  if (CATEGORY_EMOJIS[name]) return CATEGORY_EMOJIS[name];
  for (const [k, v] of Object.entries(CATEGORY_EMOJIS)) {
    if (name.toLowerCase().includes(k.toLowerCase())) return v;
  }
  return "🏷️";
}

export default function Settings() {
  const { data: settings, isLoading } = useSettings();
  const { data: categories } = useUniqueCategories();

  if (isLoading) return <div className="text-muted-foreground">Loading settings...</div>;

  const cfg = settings as Record<string, unknown> | undefined;
  const included = categories?.filter((c) => !c.excluded) || [];
  const excluded = categories?.filter((c) => c.excluded) || [];

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <p className="text-sm text-muted-foreground">
        Configuration loaded from <code className="bg-muted px-1 py-0.5 rounded text-xs">.env</code>. Categories can be included/excluded from analysis below.
      </p>

      <Tabs defaultValue="categories">
        <TabsList>
          <TabsTrigger value="categories">Categories ({categories?.length || 0})</TabsTrigger>
          <TabsTrigger value="config">Configuration</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
        </TabsList>

        <TabsContent value="categories" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Included in Analysis</CardTitle>
              <p className="text-sm text-muted-foreground">
                These categories are included when running spending analysis.
              </p>
            </CardHeader>
            <CardContent>
              {included.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {included.map((c) => (
                    <CategoryChip key={c.name} category={c} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No categories yet. Sync transactions and categorize them first.</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Excluded from Analysis</CardTitle>
              <p className="text-sm text-muted-foreground">
                Transactions in these categories are excluded from LLM analysis.
              </p>
            </CardHeader>
            <CardContent>
              {excluded.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {excluded.map((c) => (
                    <CategoryChip key={c.name} category={c} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No excluded categories. Click a category above to exclude it.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config" className="space-y-4">
          <Card>
            <CardHeader><CardTitle>Server</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <SettingRow label="Listen Address" value={cfg?.listen_addr as string} />
              <SettingRow label="Billing Day" value={String(cfg?.billing_day ?? 15)} />
              <SettingRow label="Sync Schedule" value={cfg?.sync_schedule as string} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>SimpleFin</CardTitle></CardHeader>
            <CardContent>
              <SettingRow label="Bridge URL" value={cfg?.simplefin_configured ? "Configured" : "Not configured"} status={cfg?.simplefin_configured as boolean} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>OpenRouter (LLM)</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <SettingRow label="API Key" value={cfg?.openrouter_configured ? "Configured" : "Not configured"} status={cfg?.openrouter_configured as boolean} />
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
              <SettingRow label="Topic" value={cfg?.ntfy_topic as string || "Not set"} status={!!(cfg?.ntfy_topic)} />
              <SettingRow label="Warning Suffix" value={cfg?.ntfy_warning_suffix as string} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Email (SMTP)</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <SettingRow label="SMTP" value={cfg?.mailer_configured ? "Configured" : "Not configured"} status={cfg?.mailer_configured as boolean} />
              <SettingRow label="From" value={cfg?.mailer_from as string || "Not set"} />
              <SettingRow label="To" value={cfg?.mailer_to as string || "Not set"} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function CategoryChip({ category }: { category: CategoryInfoItem }) {
  const toggle = useSetCategoryExcluded();
  const emoji = emojiFor(category.name);

  return (
    <button
      onClick={() => toggle.mutate({ category: category.name, excluded: !category.excluded })}
      disabled={toggle.isPending}
      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-border text-sm
        hover:bg-accent transition-colors cursor-pointer disabled:opacity-50"
    >
      <span>{emoji}</span>
      <span>{category.name}</span>
      <Badge variant="secondary" className="text-xs px-1.5 py-0">{category.count}</Badge>
    </button>
  );
}

function SettingRow({ label, value, status }: { label: string; value?: string; status?: boolean }) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 py-1">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="flex items-center gap-2">
        <code className="text-sm bg-muted px-2 py-0.5 rounded break-all">{value || "-"}</code>
        {status !== undefined && (
          <span className={`w-2 h-2 rounded-full shrink-0 ${status ? "bg-green-500" : "bg-red-400"}`} />
        )}
      </div>
    </div>
  );
}

function TestNotificationButton() {
  const handleTest = async () => {
    try {
      await fetch("/api/settings/test-notification", { method: "POST" });
    } catch { /* ignore */ }
  };
  return (
    <Button variant="outline" size="sm" onClick={handleTest}>Test</Button>
  );
}
