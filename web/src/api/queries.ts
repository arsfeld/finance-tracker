import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  ApiResponse,
  DashboardData,
  DBTransaction,
  DBAccount,
  Analysis,
  FilterRule,
  CategoryEntry,
  SyncLogEntry,
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

// Settings (read-only from .env)
export function useSettings() {
  return useQuery({
    queryKey: ["settings"],
    queryFn: () => fetchApi<Record<string, unknown>>("/api/settings"),
  });
}

// Filters
export function useFilters() {
  return useQuery({
    queryKey: ["filters"],
    queryFn: () => fetchApi<FilterRule[]>("/api/filters"),
  });
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
