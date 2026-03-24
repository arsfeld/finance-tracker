import { useNavigate } from "react-router";
import { useDashboard } from "@/api/queries";
import { useChartColors } from "@/lib/chart-colors";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MarkdownContent } from "@/components/MarkdownContent";
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";

export default function Dashboard() {
  const { data, isLoading, error } = useDashboard();
  const navigate = useNavigate();
  const COLORS = useChartColors();

  if (isLoading) return <div className="text-muted-foreground">Loading dashboard...</div>;
  if (error) return <div className="text-destructive">Error: {error.message}</div>;
  if (!data) return null;

  const categoryData = Object.entries(data.category_totals || {}).map(([name, value]) => ({
    name,
    value,
  }));

  const trendData = (data.trend_data || []).map((d) => ({
    name: d.label,
    total: d.total,
    start: d.start,
    end: d.end,
  }));

  const handleCategoryClick = (name: string) => {
    const params = new URLSearchParams({
      category: name,
      start: String(data.period.start),
      end: String(data.period.end),
      included_only: "true",
    });
    navigate(`/transactions?${params}`);
  };

  const handleTrendBarClick = (entry: { start: number; end: number }) => {
    const params = new URLSearchParams({
      start: String(entry.start),
      end: String(entry.end),
    });
    navigate(`/transactions?${params}`);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="text-muted-foreground text-sm">{data.period.label}</p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Spending</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${data.summary.total_spending.toFixed(2)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Daily Average</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${data.summary.daily_average.toFixed(2)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Transactions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.summary.transaction_count}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Days in Period</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.summary.days_in_period}</div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {categoryData.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Spending by Category</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={categoryData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={100}
                    label
                    className="cursor-pointer"
                    onClick={(_, idx) => handleCategoryClick(categoryData[idx].name)}
                  >
                    {categoryData.map((_, i) => (
                      <Cell
                        key={i}
                        fill={COLORS[i % COLORS.length]}
                        className="hover:opacity-80 transition-opacity cursor-pointer"
                      />
                    ))}
                  </Pie>
                  <Tooltip formatter={(v) => `$${Number(v).toFixed(2)}`} />
                </PieChart>
              </ResponsiveContainer>
              <p className="text-xs text-muted-foreground text-center mt-2">Click a category to view transactions</p>
            </CardContent>
          </Card>
        )}

        {trendData.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Spending Trend</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={trendData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip formatter={(v) => `$${Number(v).toFixed(2)}`} />
                  <Bar
                    dataKey="total"
                    fill={COLORS[0]}
                    className="cursor-pointer"
                    onClick={(entry) => handleTrendBarClick(entry)}
                  />
                </BarChart>
              </ResponsiveContainer>
              <p className="text-xs text-muted-foreground text-center mt-2">Click a bar to view transactions for that period</p>
            </CardContent>
          </Card>
        )}
      </div>

      {data.recent_transactions && data.recent_transactions.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Recent Transactions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {data.recent_transactions.map((t) => (
                <div key={t.id} className="flex justify-between items-center py-1 border-b border-border last:border-0">
                  <div>
                    <div className="text-sm font-medium">{t.description}</div>
                    <div className="text-xs text-muted-foreground">
                      {new Date(t.posted * 1000).toLocaleDateString()}
                      {t.category && ` · ${t.category}`}
                    </div>
                  </div>
                  <div className={`text-sm font-medium ${t.amount < 0 ? "text-destructive" : "text-green-600"}`}>
                    ${Math.abs(t.amount).toFixed(2)}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {data.latest_analysis && (
        <Card>
          <CardHeader>
            <CardTitle>Latest Analysis</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="max-h-64 overflow-y-auto">
              <MarkdownContent>{data.latest_analysis.response_text}</MarkdownContent>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
