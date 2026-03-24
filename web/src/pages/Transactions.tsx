import { useState } from "react";
import { useTransactions, useBillingPeriods, type BillingPeriod } from "@/api/queries";
import { CategoryPicker } from "@/components/CategoryPicker";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type SortField = "posted" | "description" | "amount";
type SortDir = "asc" | "desc";

export default function Transactions() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [selectedPeriodIdx, setSelectedPeriodIdx] = useState<number | null>(null);
  const [sortBy, setSortBy] = useState<SortField>("posted");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const { data: periods } = useBillingPeriods();

  const activePeriod: BillingPeriod | undefined =
    periods && periods.length > 0
      ? periods[selectedPeriodIdx ?? periods.length - 1]
      : undefined;

  const params: Record<string, string> = {
    page: String(page),
    limit: "50",
    sort_by: sortBy,
    sort_dir: sortDir,
    include_positive: "true",
  };
  if (activePeriod) {
    params.start = String(activePeriod.start);
    params.end = String(activePeriod.end);
  }
  if (search) params.search = search;

  const { data, isLoading } = useTransactions(params);
  const transactions = data?.data || [];
  const meta = data?.meta;

  const handlePeriodChange = (idx: number) => {
    setSelectedPeriodIdx(idx);
    setPage(1);
  };

  const handleSort = (field: SortField) => {
    if (sortBy === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(field);
      setSortDir(field === "posted" ? "desc" : "asc");
    }
    setPage(1);
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Transactions</h1>

      {periods && periods.length > 0 && (
        <div className="flex gap-1 flex-wrap">
          {periods.map((p, i) => (
            <Button
              key={i}
              variant={activePeriod === p ? "default" : "outline"}
              size="sm"
              onClick={() => handlePeriodChange(i)}
            >
              {p.label}
            </Button>
          ))}
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-2">
        <Input
          type="text"
          placeholder="Search transactions..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="sm:max-w-xs"
        />
        <Button variant="outline" size="sm" asChild>
          <a href={`/api/transactions/export?${new URLSearchParams(params).toString()}`} download>
            Export CSV
          </a>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{meta ? `${meta.total} transactions` : "Transactions"}</span>
            {activePeriod && (
              <span className="text-sm font-normal text-muted-foreground">
                {activePeriod.label}
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-muted-foreground">Loading...</div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <SortableHead field="posted" label="Date" current={sortBy} dir={sortDir} onSort={handleSort} className="w-[100px]" />
                      <SortableHead field="description" label="Description" current={sortBy} dir={sortDir} onSort={handleSort} />
                      <TableHead className="w-[200px] hidden sm:table-cell">Category</TableHead>
                      <SortableHead field="amount" label="Amount" current={sortBy} dir={sortDir} onSort={handleSort} className="text-right w-[100px]" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {transactions && transactions.length > 0 ? transactions.map((t) => (
                      <TableRow key={t.id}>
                        <TableCell className="text-sm whitespace-nowrap text-muted-foreground">
                          {new Date(t.posted * 1000).toLocaleDateString()}
                        </TableCell>
                        <TableCell className="text-sm">{t.description}</TableCell>
                        <TableCell className="hidden sm:table-cell">
                          <CategoryPicker
                            transactionId={t.id}
                            description={t.description}
                            currentCategory={t.category}
                          />
                        </TableCell>
                        <TableCell className={`text-sm text-right font-medium whitespace-nowrap ${t.amount < 0 ? "text-red-600 dark:text-red-400" : "text-green-600 dark:text-green-400"}`}>
                          ${Math.abs(t.amount).toFixed(2)}
                        </TableCell>
                      </TableRow>
                    )) : (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                          No transactions for this period.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              {meta && meta.total > meta.limit && (
                <div className="flex justify-center gap-2 mt-4">
                  <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                    Previous
                  </Button>
                  <span className="text-sm text-muted-foreground self-center">
                    Page {page} of {Math.ceil(meta.total / meta.limit)}
                  </span>
                  <Button variant="outline" size="sm" disabled={page >= Math.ceil(meta.total / meta.limit)} onClick={() => setPage((p) => p + 1)}>
                    Next
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SortableHead({
  field,
  label,
  current,
  dir,
  onSort,
  className = "",
}: {
  field: SortField;
  label: string;
  current: SortField;
  dir: SortDir;
  onSort: (f: SortField) => void;
  className?: string;
}) {
  const isActive = current === field;
  return (
    <TableHead className={className}>
      <button
        onClick={() => onSort(field)}
        className="inline-flex items-center gap-1 hover:text-foreground transition-colors cursor-pointer"
      >
        {label}
        <span className="text-xs">
          {isActive ? (dir === "asc" ? "▲" : "▼") : "⇅"}
        </span>
      </button>
    </TableHead>
  );
}
