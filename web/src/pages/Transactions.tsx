import { useState } from "react";
import { useTransactions, useOverrideCategory, useCategoryNames } from "@/api/queries";
import type { DBTransaction } from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function Transactions() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [editingTxn, setEditingTxn] = useState<DBTransaction | null>(null);

  const params: Record<string, string> = {
    page: String(page),
    limit: "50",
    sort_by: "posted",
    sort_dir: "desc",
    include_positive: "true",
  };
  if (search) params.search = search;

  const { data, isLoading } = useTransactions(params);
  const transactions = data?.data || [];
  const meta = data?.meta;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Transactions</h1>

      <div className="flex gap-2">
        <Input
          type="text"
          placeholder="Search transactions..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="max-w-xs"
        />
        <Button variant="outline" size="sm" asChild>
          <a href={`/api/transactions/export?search=${encodeURIComponent(search)}&include_positive=true`} download>
            Export CSV
          </a>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {meta ? `${meta.total} transactions` : "Transactions"}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-muted-foreground">Loading...</div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Date</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions && transactions.length > 0 ? transactions.map((t) => (
                    <TableRow key={t.id}>
                      <TableCell className="text-sm whitespace-nowrap">
                        {new Date(t.posted * 1000).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm">{t.description}</TableCell>
                      <TableCell>
                        <button
                          onClick={() => setEditingTxn(t)}
                          className="text-sm px-2 py-0.5 rounded hover:bg-accent transition-colors cursor-pointer text-left"
                        >
                          {t.category ? (
                            <span className="text-foreground">{t.category}</span>
                          ) : (
                            <span className="text-muted-foreground italic">Uncategorized</span>
                          )}
                        </button>
                      </TableCell>
                      <TableCell className={`text-sm text-right font-medium whitespace-nowrap ${t.amount < 0 ? "text-red-600 dark:text-red-400" : "text-green-600 dark:text-green-400"}`}>
                        ${Math.abs(t.amount).toFixed(2)}
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-muted-foreground">
                        No transactions found. Click "Sync Now" to fetch data.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>

              {meta && meta.total > meta.limit && (
                <div className="flex justify-center gap-2 mt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    Previous
                  </Button>
                  <span className="text-sm text-muted-foreground self-center">
                    Page {page} of {Math.ceil(meta.total / meta.limit)}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= Math.ceil(meta.total / meta.limit)}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {editingTxn && (
        <CategoryEditDialog
          transaction={editingTxn}
          onClose={() => setEditingTxn(null)}
        />
      )}
    </div>
  );
}

function CategoryEditDialog({ transaction, onClose }: { transaction: DBTransaction; onClose: () => void }) {
  const [category, setCategory] = useState(transaction.category || "");
  const [applyToAll, setApplyToAll] = useState(false);
  const override = useOverrideCategory();
  const existingCategories = useCategoryNames();

  const handleSave = () => {
    if (!category.trim()) return;
    override.mutate(
      { txnId: transaction.id, category: category.trim(), applyToMerchant: applyToAll },
      { onSuccess: () => onClose() },
    );
  };

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Category</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div>
            <p className="text-sm text-muted-foreground">Transaction</p>
            <p className="text-sm font-medium">{transaction.description}</p>
            <p className="text-sm text-muted-foreground">${Math.abs(transaction.amount).toFixed(2)} on {new Date(transaction.posted * 1000).toLocaleDateString()}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="category">Category</Label>
            <Input
              id="category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="e.g. Groceries, Dining, Transportation..."
              list="category-suggestions"
              autoFocus
            />
            <datalist id="category-suggestions">
              {existingCategories.map((c) => (
                <option key={c} value={c} />
              ))}
            </datalist>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="apply-all"
              checked={applyToAll}
              onCheckedChange={(checked) => setApplyToAll(checked === true)}
            />
            <Label htmlFor="apply-all" className="text-sm font-normal">
              Apply to all transactions from "{transaction.description}"
            </Label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={!category.trim() || override.isPending}>
            {override.isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
