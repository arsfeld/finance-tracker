import { useAccounts, useSetAccountsIncluded } from "@/api/queries";
import type { DBAccount } from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";

function groupByInstitution(accounts: DBAccount[]): [string, DBAccount[]][] {
  const groups = new Map<string, DBAccount[]>();
  for (const a of accounts) {
    const org = a.org_name || "Other";
    groups.set(org, [...(groups.get(org) ?? []), a]);
  }
  return [...groups.entries()].sort(([x], [y]) => x.localeCompare(y));
}

function formatBalance(balance: number): string {
  return `${balance < 0 ? "-" : ""}$${Math.abs(balance).toFixed(2)}`;
}

export function AccountsSettings() {
  const { data: accounts, isLoading } = useAccounts();
  const setIncluded = useSetAccountsIncluded();

  if (isLoading) return <p className="text-sm text-muted-foreground">Loading accounts...</p>;

  // One entry per identity: the row reporting now. Toggling it covers the
  // older rows a re-auth left behind.
  const groups = groupByInstitution((accounts ?? []).filter((a) => a.is_current));
  if (groups.length === 0) {
    return <p className="text-sm text-muted-foreground">No accounts yet. Run a sync first.</p>;
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Excluded accounts are left out of spending, analysis and connection alerts. They stay connected in
        SimpleFin and are still checked for payments toward your cards.
      </p>
      {setIncluded.error && <p className="text-sm text-destructive">{setIncluded.error.message}</p>}
      {groups.map(([org, accts]) => (
        <Card key={org}>
          <CardHeader>
            <CardTitle className="flex items-center gap-3">
              <Checkbox
                checked={accts.some((a) => a.is_included)}
                disabled={setIncluded.isPending}
                onCheckedChange={(v) => setIncluded.mutate({ ids: accts.map((a) => a.id), included: v === true })}
                aria-label={`Include ${org} accounts`}
              />
              {org}
            </CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            {accts.map((a) => (
              <div key={a.id} className="flex items-center gap-3 py-2">
                <Checkbox
                  id={`account-${a.id}`}
                  checked={a.is_included}
                  disabled={setIncluded.isPending}
                  onCheckedChange={(v) => setIncluded.mutate({ ids: [a.id], included: v === true })}
                />
                <label htmlFor={`account-${a.id}`} className="flex-1 min-w-0 text-sm cursor-pointer">
                  <span className={a.is_included ? "font-medium" : "font-medium text-muted-foreground line-through"}>
                    {a.name}
                  </span>
                  {a.is_credit_card && (
                    <Badge variant="secondary" className="ml-2 text-xs px-1.5 py-0">card</Badge>
                  )}
                  <span className="block text-xs text-muted-foreground">
                    updated {new Date(a.balance_date * 1000).toLocaleDateString()}
                  </span>
                </label>
                <span className="text-sm tabular-nums">{formatBalance(a.balance)}</span>
              </div>
            ))}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
