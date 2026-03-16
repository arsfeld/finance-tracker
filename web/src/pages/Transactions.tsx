import { useState } from "react";
import { useTransactions } from "@/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
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

  const params: Record<string, string> = {
    page: String(page),
    limit: "50",
    sort_by: "posted",
    sort_dir: "desc",
  };
  if (search) params.search = search;

  const { data, isLoading } = useTransactions(params);
  const transactions = data?.data || [];
  const meta = data?.meta;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Transactions</h1>

      <div className="flex gap-2">
        <input
          type="text"
          placeholder="Search transactions..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="px-3 py-2 border border-input rounded-md bg-background text-sm flex-1 max-w-xs"
        />
        <Button variant="outline" size="sm" asChild>
          <a href={`/api/transactions/export?search=${encodeURIComponent(search)}`} download>
            Export CSV
          </a>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex justify-between items-center">
            <span>
              {meta ? `${meta.total} transactions` : "Transactions"}
            </span>
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
                      <TableCell className="text-sm">
                        {new Date(t.posted * 1000).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm">{t.description}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{t.category || "-"}</TableCell>
                      <TableCell className={`text-sm text-right font-medium ${t.amount < 0 ? "text-destructive" : "text-green-600"}`}>
                        ${Math.abs(t.amount).toFixed(2)}
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-muted-foreground">
                        No transactions found. Sync to fetch data.
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
