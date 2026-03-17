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

export default function Transactions() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [selectedPeriodIdx, setSelectedPeriodIdx] = useState<number | null>(null);

  const { data: periods } = useBillingPeriods();

  // Default to the latest (current) period once loaded.
  const activePeriod: BillingPeriod | undefined =
    periods && periods.length > 0
      ? periods[selectedPeriodIdx ?? periods.length - 1]
      : undefined;

  const params: Record<string, string> = {
    page: String(page),
    limit: "50",
    sort_by: "posted",
    sort_dir: "desc",
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

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Transactions</h1>

      {/* Period selector */}
      {periods && periods.length > 0 && (
        <div className="flex gap-1 flex-wrap">
          {periods.map((p, i) => {
            const isActive = activePeriod === p;
            return (
              <Button
                key={i}
                variant={isActive ? "default" : "outline"}
                size="sm"
                onClick={() => handlePeriodChange(i)}
              >
                {p.label}
              </Button>
            );
          })}
        </div>
      )}

      <div className="flex gap-2">
        <Input
          type="text"
          placeholder="Search transactions..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="max-w-xs"
        />
        <Button variant="outline" size="sm" asChild>
          <a
            href={`/api/transactions/export?${new URLSearchParams(params).toString()}`}
            download
          >
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
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[100px]">Date</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="w-[200px]">Category</TableHead>
                    <TableHead className="text-right w-[100px]">Amount</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions && transactions.length > 0 ? transactions.map((t) => (
                    <TableRow key={t.id}>
                      <TableCell className="text-sm whitespace-nowrap text-muted-foreground">
                        {new Date(t.posted * 1000).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm">{t.description}</TableCell>
                      <TableCell>
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
    </div>
  );
}
