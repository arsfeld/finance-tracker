import { useState, useEffect, useCallback, useMemo } from "react";
import { useSearchParams } from "react-router";
import { useTransactions, useBillingPeriods, useUniqueCategories, type BillingPeriod } from "@/api/queries";
import { CategoryPicker } from "@/components/CategoryPicker";
import { SimilarMerchantsDialog, useSimilarMerchants } from "@/components/SimilarMerchantsDialog";
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

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

export default function Transactions() {
  const [searchParams, setSearchParams] = useSearchParams();

  // URL-driven filter state
  const urlCategory = searchParams.get("category") || "";
  const urlSearch = searchParams.get("search") || "";
  const urlStart = searchParams.get("start") || "";
  const urlEnd = searchParams.get("end") || "";
  const urlPage = parseInt(searchParams.get("page") || "1", 10) || 1;
  const urlIncludedOnly = searchParams.get("included_only") === "true";

  // Local state for sort (not URL-driven) and search input
  const [sortBy, setSortBy] = useState<SortField>("posted");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [searchInput, setSearchInput] = useState(urlSearch);
  const debouncedSearch = useDebounce(searchInput, 300);

  // Sync debounced search to URL
  useEffect(() => {
    if (debouncedSearch !== urlSearch) {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (debouncedSearch) {
          next.set("search", debouncedSearch);
        } else {
          next.delete("search");
        }
        next.set("page", "1");
        return next;
      }, { replace: true });
    }
  }, [debouncedSearch, urlSearch, setSearchParams]);

  // Sync URL search to input when URL changes externally (e.g. browser back)
  useEffect(() => {
    setSearchInput(urlSearch);
  }, [urlSearch]);

  const { data: periods } = useBillingPeriods();
  const { data: uniqueCategories } = useUniqueCategories();

  // Find matching billing period for the URL's start/end
  const matchingPeriodIdx = useMemo(() => {
    if (!periods || !urlStart || !urlEnd) return null;
    const start = parseInt(urlStart, 10);
    const end = parseInt(urlEnd, 10);
    const idx = periods.findIndex((p) => p.start === start && p.end === end);
    return idx >= 0 ? idx : null;
  }, [periods, urlStart, urlEnd]);

  // Active period: from URL match, or default to last period
  const hasCustomRange = urlStart && urlEnd && matchingPeriodIdx === null;
  const activePeriodIdx = matchingPeriodIdx ?? ((!urlStart && !urlEnd && periods) ? periods.length - 1 : null);
  const activePeriod: BillingPeriod | undefined =
    periods && activePeriodIdx !== null ? periods[activePeriodIdx] : undefined;

  // Build API params
  const apiParams: Record<string, string> = {
    page: String(urlPage),
    limit: "50",
    sort_by: sortBy,
    sort_dir: sortDir,
    include_positive: "true",
  };
  if (urlStart) apiParams.start = urlStart;
  else if (activePeriod) apiParams.start = String(activePeriod.start);
  if (urlEnd) apiParams.end = urlEnd;
  else if (activePeriod) apiParams.end = String(activePeriod.end);
  if (urlCategory) apiParams.category = urlCategory;
  if (urlSearch) apiParams.search = urlSearch;
  if (urlIncludedOnly) apiParams.included_only = "true";

  const { data, isLoading } = useTransactions(apiParams);
  const transactions = data?.data || [];
  const meta = data?.meta;

  const updateParams = useCallback(
    (updates: Record<string, string | null>) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        for (const [key, value] of Object.entries(updates)) {
          if (value === null || value === "") {
            next.delete(key);
          } else {
            next.set(key, value);
          }
        }
        return next;
      });
    },
    [setSearchParams],
  );

  const handlePeriodChange = (idx: number) => {
    if (!periods) return;
    const p = periods[idx];
    updateParams({
      start: String(p.start),
      end: String(p.end),
      page: "1",
    });
  };

  const handleCategoryChange = (category: string) => {
    updateParams({
      category: category || null,
      page: "1",
    });
  };

  const handleClearFilters = () => {
    setSearchInput("");
    setSearchParams({});
  };

  const handleSort = (field: SortField) => {
    if (sortBy === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(field);
      setSortDir(field === "posted" ? "desc" : "asc");
    }
    updateParams({ page: "1" });
  };

  const { triggerSimilarityCheck, currentSuggestions, currentCategory: similarCategory, dismiss: dismissSimilar } = useSimilarMerchants();

  const hasActiveFilters = urlCategory || urlSearch || hasCustomRange;

  // Custom range label
  const customRangeLabel = hasCustomRange
    ? `${new Date(parseInt(urlStart, 10) * 1000).toLocaleDateString()} - ${new Date(parseInt(urlEnd, 10) * 1000).toLocaleDateString()}`
    : null;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Transactions</h1>

      {periods && periods.length > 0 && (
        <div className="flex gap-1 flex-wrap items-center">
          {periods.map((p, i) => (
            <Button
              key={i}
              variant={activePeriodIdx === i ? "default" : "outline"}
              size="sm"
              onClick={() => handlePeriodChange(i)}
            >
              {p.label}
            </Button>
          ))}
          {customRangeLabel && (
            <span className="text-sm text-muted-foreground ml-2 flex items-center gap-1">
              Custom: {customRangeLabel}
              <button
                onClick={() => updateParams({ start: null, end: null, page: "1" })}
                className="text-xs hover:text-foreground ml-1"
                title="Clear custom range"
              >
                ✕
              </button>
            </span>
          )}
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-2">
        <Input
          type="text"
          placeholder="Search transactions..."
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          className="sm:max-w-xs"
        />
        <select
          value={urlCategory}
          onChange={(e) => handleCategoryChange(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:max-w-[200px]"
        >
          <option value="">All categories</option>
          {uniqueCategories?.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name} ({c.count})
            </option>
          ))}
        </select>
        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={handleClearFilters}>
            Clear filters
          </Button>
        )}
        <Button variant="outline" size="sm" asChild className="sm:ml-auto">
          <a href={`/api/transactions/export?${new URLSearchParams(apiParams).toString()}`} download>
            Export CSV
          </a>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{meta ? `${meta.total} transactions` : "Transactions"}</span>
            {(activePeriod || customRangeLabel) && (
              <span className="text-sm font-normal text-muted-foreground">
                {customRangeLabel || activePeriod?.label}
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
                            onCategoryChanged={triggerSimilarityCheck}
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
                  <Button variant="outline" size="sm" disabled={urlPage <= 1} onClick={() => updateParams({ page: String(urlPage - 1) })}>
                    Previous
                  </Button>
                  <span className="text-sm text-muted-foreground self-center">
                    Page {urlPage} of {Math.ceil(meta.total / meta.limit)}
                  </span>
                  <Button variant="outline" size="sm" disabled={urlPage >= Math.ceil(meta.total / meta.limit)} onClick={() => updateParams({ page: String(urlPage + 1) })}>
                    Next
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      <SimilarMerchantsDialog
        suggestions={currentSuggestions}
        category={similarCategory}
        onDismiss={dismissSimilar}
      />
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
