export interface ApiResponse<T> {
  data: T;
  meta?: { page: number; limit: number; total: number };
}

export interface ApiError {
  error: { code: string; message: string };
}

export interface DBTransaction {
  id: string;
  account_id: string;
  description: string;
  amount: number;
  posted: number;
  transacted_at: number | null;
  pending: boolean;
  category: string;
  cached_at: string;
  updated_at: string;
}

export interface DBAccount {
  id: string;
  name: string;
  balance: number;
  balance_date: number;
  currency: string;
  org_name: string;
  org_domain: string;
  is_included: boolean;
  first_seen_at: string;
  updated_at: string;
}

export interface Analysis {
  id: number;
  created_at: string;
  period_start: number;
  period_end: number;
  billing_day: number;
  date_range_type: string;
  response_text: string;
  model_used: string;
  is_multi_period: boolean;
  status: string;
}

export interface CategoryEntry {
  merchant_description: string;
  category: string;
  source: string;
  excluded: boolean;
  updated_at: string;
}

export interface SyncLogEntry {
  id: number;
  started_at: string;
  completed_at: string;
  status: string;
  transactions_added: number;
  transactions_updated: number;
  error_message: string;
  api_errors: string;
}

export interface CategoryTrendPeriod {
  label: string;
  start: number;
  end: number;
  totals: Record<string, number>;
}

export interface CategoryTrendData {
  categories: string[];
  periods: CategoryTrendPeriod[];
}

export interface DailyTotalsData {
  days: { date: string; total: number }[];
  start: number;
  end: number;
}

export interface TopMerchantsData {
  merchants: { name: string; total: number; count: number }[];
  start: number;
  end: number;
}

export interface BudgetedCategory {
  category: string;
  amount: number;
  spent: number;
  remaining: number;
  percent: number;
}

export interface UnbudgetedCategory {
  category: string;
  spent: number;
}

export interface BudgetStatusResponse {
  period: { label: string; start: number; end: number };
  budgeted: BudgetedCategory[];
  unbudgeted: UnbudgetedCategory[];
}

export interface DashboardData {
  period: { label: string; start: number; end: number };
  summary: {
    total_spending: number;
    daily_average: number;
    transaction_count: number;
    days_in_period: number;
  };
  category_totals: Record<string, number>;
  recent_transactions: DBTransaction[] | null;
  latest_analysis: Analysis | null;
  trend_data: { label: string; total: number; start: number; end: number }[] | null;
}
