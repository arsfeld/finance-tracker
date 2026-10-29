import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  useAccounts,
  useImportPreview,
  useImportStatement,
} from "@/api/queries";
import type { DBAccount, ImportPreview, ImportResult } from "@/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function currentAccounts(accounts: DBAccount[] | null | undefined): DBAccount[] {
  const byIdentity = new Map<string, DBAccount>();
  for (const account of accounts ?? []) {
    if (!account.is_current) continue;
    const identity = account.card_key || account.id;
    if (!byIdentity.has(identity)) byIdentity.set(identity, account);
  }
  return [...byIdentity.values()].sort((a, b) => a.name.localeCompare(b.name));
}

function formatDate(ts?: number) {
  if (!ts) return "—";
  return new Date(ts * 1000).toLocaleDateString();
}

function formatMoney(value: number | undefined, currency = "CAD") {
  if (value === undefined || value === null) return "—";
  return value.toLocaleString(undefined, { style: "currency", currency });
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

export default function Import() {
  const { data: accounts } = useAccounts();
  const previewMutation = useImportPreview();
  const commitMutation = useImportStatement();

  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [selected, setSelected] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragging, setDragging] = useState(false);

  const accountsForSelect = useMemo(() => currentAccounts(accounts), [accounts]);
  const currency = preview?.statement.currency || "CAD";

  const runPreview = (nextFile: File, accountID?: string) => {
    setResult(null);
    const form = new FormData();
    form.append("file", nextFile);
    if (accountID) form.append("account_id", accountID);
    previewMutation.mutate(form, {
      onSuccess: (data) => {
        setPreview(data);
        setSelected(data.account?.id || data.candidates[0]?.account_id || "");
      },
    });
  };

  const chooseFile = (nextFile: File | undefined) => {
    if (!nextFile) return;
    setFile(nextFile);
    setPreview(null);
    setSelected("");
    runPreview(nextFile);
  };

  const chooseAccount = (accountID: string) => {
    setSelected(accountID);
    if (file) runPreview(file, accountID);
  };

  const commit = () => {
    if (!file || !selected) return;
    const form = new FormData();
    form.append("file", file);
    form.append("account_id", selected);
    commitMutation.mutate(form, {
      onSuccess: (data) => {
        setResult(data);
        setPreview(null);
      },
    });
  };

  const busy = previewMutation.isPending || commitMutation.isPending;
  const canImport = !!file && !!selected && !busy;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Import</h1>
        <p className="text-muted-foreground text-sm">
          Upload an OFX, QFX, QBO or ASO statement from Quicken, MS Money, QuickBooks or Simply Accounting.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Statement file</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <label
            onDragOver={(e) => {
              e.preventDefault();
              setDragging(true);
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={(e) => {
              e.preventDefault();
              setDragging(false);
              chooseFile(e.dataTransfer.files?.[0]);
            }}
            className={`flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-10 text-center transition-colors ${
              dragging ? "border-primary bg-accent/40" : "border-border hover:bg-accent/20"
            }`}
          >
            <input
              type="file"
              accept=".ofx,.qfx,.qbo,.aso"
              className="sr-only"
              onChange={(e) => chooseFile(e.target.files?.[0])}
            />
            <div className="font-medium">
              {file ? file.name : "Drop a statement here, or click to choose one"}
            </div>
            <div className="text-muted-foreground mt-1 text-sm">
              OFX, QFX, QBO or ASO · one account per file
            </div>
          </label>

          {(previewMutation.isPending || commitMutation.isPending) && (
            <div className="text-muted-foreground text-sm">
              {commitMutation.isPending ? "Importing statement..." : "Reading statement..."}
            </div>
          )}
        </CardContent>
      </Card>

      {previewMutation.error && (
        <Card>
          <CardContent className="text-destructive py-4 text-sm">
            {errorMessage(previewMutation.error)}
          </CardContent>
        </Card>
      )}

      {preview && (
        <Card>
          <CardHeader>
            <CardTitle>Review</CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
              <div>
                <div className="text-muted-foreground">Account number</div>
                <div className="font-mono">{preview.statement.account_id}</div>
              </div>
              <div>
                <div className="text-muted-foreground">Period</div>
                <div>
                  {formatDate(preview.statement.start)} – {formatDate(preview.statement.end)}
                </div>
              </div>
              <div>
                <div className="text-muted-foreground">Ending balance</div>
                <div className="font-medium">{formatMoney(preview.statement.ledger_balance, currency)}</div>
              </div>
              <div>
                <div className="text-muted-foreground">Transactions</div>
                <div>{preview.statement.transaction_count}</div>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="import-account">
                Import into
              </label>
              <select
                id="import-account"
                value={selected}
                onChange={(e) => chooseAccount(e.target.value)}
                className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              >
                <option value="">Choose an account</option>
                {accountsForSelect.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.name} — {account.org_name}
                  </option>
                ))}
              </select>
              {preview.candidates.length > 0 && (
                <p className="text-muted-foreground text-xs">
                  Suggested from the statement: {preview.candidates[0].name}
                </p>
              )}
            </div>

            <div className="flex flex-wrap gap-4 text-sm">
              <span>
                <span className="font-semibold text-green-600 dark:text-green-400">{preview.new_count}</span>{" "}
                new
              </span>
              <span>
                <span className="font-semibold">{preview.duplicate_count}</span> already imported
              </span>
              <span>
                <span className="font-semibold">{preview.updated_count}</span> changed
              </span>
            </div>

            {preview.warnings && preview.warnings.length > 0 && (
              <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800 dark:border-yellow-900/40 dark:bg-yellow-900/20 dark:text-yellow-200">
                <ul className="list-inside list-disc space-y-1">
                  {preview.warnings.map((warning) => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </div>
            )}

            {preview.rows.length > 0 && (
              <div className="max-h-96 overflow-y-auto rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Date</TableHead>
                      <TableHead>Description</TableHead>
                      <TableHead className="text-right">Amount</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {preview.rows.map((row, index) => (
                      <TableRow key={`${row.fitid || row.date}-${index}`}>
                        <TableCell className="whitespace-nowrap text-muted-foreground">
                          {formatDate(row.date)}
                        </TableCell>
                        <TableCell>{row.description}</TableCell>
                        <TableCell
                          className={`text-right whitespace-nowrap ${
                            row.amount < 0 ? "" : "text-green-600 dark:text-green-400"
                          }`}
                        >
                          {formatMoney(row.amount, currency)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              row.status === "new"
                                ? "default"
                                : row.status === "updated"
                                  ? "outline"
                                  : "secondary"
                            }
                          >
                            {row.status}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            <div className="flex items-center gap-3">
              <Button onClick={commit} disabled={!canImport}>
                {preview.new_count > 0
                  ? `Import ${preview.new_count} transaction${preview.new_count === 1 ? "" : "s"}`
                  : "Update balance"}
              </Button>
              {preview.new_count === 0 && preview.updated_count === 0 && (
                <span className="text-muted-foreground text-sm">
                  Nothing new; the statement balance will still be applied.
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {commitMutation.error && (
        <Card>
          <CardContent className="text-destructive py-4 text-sm">
            {errorMessage(commitMutation.error)}
          </CardContent>
        </Card>
      )}

      {result && (
        <Card>
          <CardHeader>
            <CardTitle>Imported</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <span className="font-semibold text-green-600 dark:text-green-400">{result.added}</span> added,{" "}
              <span className="font-semibold">{result.updated}</span> updated,{" "}
              <span className="font-semibold">{result.duplicates}</span> already present.
            </div>
            <div className="text-muted-foreground">
              {result.account.name} · balance now{" "}
              {formatMoney(result.account.balance, result.account.currency || currency)}
            </div>
            {result.warnings && result.warnings.length > 0 && (
              <ul className="text-muted-foreground list-inside list-disc space-y-1">
                {result.warnings.map((warning) => (
                  <li key={warning}>{warning}</li>
                ))}
              </ul>
            )}
            <div className="flex gap-3">
              <Button asChild variant="outline" size="sm">
                <Link to="/transactions">View transactions</Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to="/sync-history">View history</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
