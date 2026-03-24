import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  ApiResponse,
  DashboardData,
  DBTransaction,
  DBAccount,
  Analysis,
  CategoryEntry,
  SyncLogEntry,
  CategoryTrendData,
  DailyTotalsData,
  TopMerchantsData,
  BudgetStatusResponse,
} from "./types";

async function fetchApi<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
  const json: ApiResponse<T> = await res.json();
  return json.data;
}

async function postApi<T>(url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
  const json: ApiResponse<T> = await res.json();
  return json.data;
}

async function deleteApi(url: string): Promise<void> {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    const err = await res
      .json()
      .catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
}

// Billing periods
export interface BillingPeriod {
  label: string;
  start: number;
  end: number;
}

export function useBillingPeriods() {
  return useQuery({
    queryKey: ["billing-periods"],
    queryFn: () => fetchApi<BillingPeriod[]>("/api/billing-periods"),
  });
}

// Dashboard
export function useDashboard() {
  return useQuery({
    queryKey: ["dashboard"],
    queryFn: () => fetchApi<DashboardData>("/api/dashboard"),
  });
}

// Transactions
export function useTransactions(params?: Record<string, string>) {
  const searchParams = new URLSearchParams(params);
  return useQuery({
    queryKey: ["transactions", params],
    queryFn: async () => {
      const res = await fetch(`/api/transactions?${searchParams}`);
      const json = await res.json();
      return { data: json.data as DBTransaction[] | null, meta: json.meta };
    },
  });
}

// Accounts
export function useAccounts() {
  return useQuery({
    queryKey: ["accounts"],
    queryFn: () => fetchApi<DBAccount[] | null>("/api/accounts"),
  });
}

// Analyses
export function useAnalyses(limit = 20) {
  return useQuery({
    queryKey: ["analyses", limit],
    queryFn: () => fetchApi<Analysis[] | null>(`/api/analyses?limit=${limit}`),
  });
}

export function useAnalysis(id: number) {
  return useQuery({
    queryKey: ["analysis", id],
    queryFn: () => fetchApi<Analysis>(`/api/analyses/${id}`),
    enabled: id > 0,
  });
}

export function useLatestAnalysis() {
  return useQuery({
    queryKey: ["analysis", "latest"],
    queryFn: () => fetchApi<Analysis | null>("/api/analyses/latest"),
  });
}

// Categories
export function useCategories() {
  return useQuery({
    queryKey: ["categories"],
    queryFn: () => fetchApi<CategoryEntry[] | null>("/api/categories"),
  });
}

export interface CategoryInfoItem {
  name: string;
  excluded: boolean;
  count: number;
}

export function useUniqueCategories() {
  return useQuery({
    queryKey: ["categories", "unique"],
    queryFn: () => fetchApi<CategoryInfoItem[]>("/api/categories/unique"),
  });
}

export function useSetCategoryExcluded() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { category: string; excluded: boolean }) =>
      postApi("/api/categories/exclude", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["categories"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });
}

// Settings (read-only from .env)
export function useSettings() {
  return useQuery({
    queryKey: ["settings"],
    queryFn: () => fetchApi<Record<string, unknown>>("/api/settings"),
  });
}

// Category override
export function useOverrideCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ txnId, category, applyToMerchant }: { txnId: string; category: string; applyToMerchant: boolean }) => {
      const res = await fetch(`/api/transactions/${txnId}/category`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ category, apply_to_merchant: applyToMerchant }),
      });
      if (!res.ok) throw new Error("Failed to update category");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["transactions"] });
      qc.invalidateQueries({ queryKey: ["categories"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });
}

// Unique categories (for autocomplete)
export function useCategoryNames() {
  const { data } = useCategories();
  const names = new Set<string>();
  if (data) {
    for (const e of data) {
      names.add(e.category);
    }
  }
  return Array.from(names).sort();
}

// Sync
export function useTriggerSync() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => postApi("/api/sync"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      qc.invalidateQueries({ queryKey: ["transactions"] });
      qc.invalidateQueries({ queryKey: ["accounts"] });
    },
  });
}

export function useSyncLog() {
  return useQuery({
    queryKey: ["sync-log"],
    queryFn: () => fetchApi<SyncLogEntry[] | null>("/api/sync/log"),
  });
}

// Analytics
export function useCategoryTrend(months = 6) {
  return useQuery({
    queryKey: ["analytics", "category-trend", months],
    queryFn: () => fetchApi<CategoryTrendData>(`/api/analytics/category-trend?months=${months}`),
  });
}

export function useDailyTotals(start?: number, end?: number) {
  const params = new URLSearchParams();
  if (start) params.set("start", String(start));
  if (end) params.set("end", String(end));
  return useQuery({
    queryKey: ["analytics", "daily-totals", start, end],
    queryFn: () => fetchApi<DailyTotalsData>(`/api/analytics/daily-totals?${params}`),
  });
}

export function useTopMerchants(start?: number, end?: number, limit = 10) {
  const params = new URLSearchParams();
  if (start) params.set("start", String(start));
  if (end) params.set("end", String(end));
  params.set("limit", String(limit));
  return useQuery({
    queryKey: ["analytics", "top-merchants", start, end, limit],
    queryFn: () => fetchApi<TopMerchantsData>(`/api/analytics/top-merchants?${params}`),
  });
}

// Trigger Analysis
export function useTriggerAnalysis() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => postApi("/api/analyses/run"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["analyses"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });
}

// Budgets
export function useBudgetStatus() {
  return useQuery({
    queryKey: ["budgets", "status"],
    queryFn: () => fetchApi<BudgetStatusResponse>("/api/budgets/status"),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUpsertBudget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { category: string; amount: number }) =>
      postApi("/api/budgets", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["budgets"] });
    },
  });
}

export function useDeleteBudget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (category: string) =>
      deleteApi(`/api/budgets/${encodeURIComponent(category)}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["budgets"] });
    },
  });
}
