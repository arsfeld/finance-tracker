import { useState, useMemo, useRef } from "react";
import { useNavigate } from "react-router";
import {
  useCategoryTrend,
  useDailyTotals,
  useTopMerchants,
} from "@/api/queries";
import { useChartColors } from "@/lib/chart-colors";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  BarChart,
  Bar,
} from "recharts";

export default function Analytics() {
  const [months, setMonths] = useState(6);
  const navigate = useNavigate();
  const CHART_COLORS = useChartColors();
  const { data: trendData, isLoading: trendLoading } = useCategoryTrend(months);
  const { data: dailyData, isLoading: dailyLoading } = useDailyTotals();
  const { data: merchantData, isLoading: merchantLoading } = useTopMerchants();

  // Transform category trend data for AreaChart
  const { chartData: trendChartData, topCategories } = useMemo(() => {
    if (!trendData) return { chartData: [], topCategories: [] };

    // Find top 8 categories by total spending across all periods
    const totals: Record<string, number> = {};
    for (const period of trendData.periods) {
      for (const [cat, val] of Object.entries(period.totals)) {
        totals[cat] = (totals[cat] || 0) + val;
      }
    }
    const sorted = Object.entries(totals)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 8)
      .map(([name]) => name);

    const chartData = trendData.periods.map((p) => {
      const row: Record<string, string | number> = { name: p.label, _start: p.start, _end: p.end };
      for (const cat of sorted) {
        row[cat] = p.totals[cat] || 0;
      }
      // Group remaining as "Other"
      let other = 0;
      for (const [cat, val] of Object.entries(p.totals)) {
        if (!sorted.includes(cat)) other += val;
      }
      if (other > 0) row["Other"] = other;
      return row;
    });

    const cats = other(trendData, sorted) ? [...sorted, "Other"] : sorted;
    return { chartData, topCategories: cats };
  }, [trendData]);

  // Which stacked area was clicked. The Area sets it, then the chart-level
  // onClick (which knows the active data index) consumes it.
  const pendingCategory = useRef<string | null>(null);

  // Handle category trend click
  const handleTrendClick = (category: string, dataPoint: Record<string, string | number>) => {
    const params = new URLSearchParams({
      category,
      start: String(dataPoint._start),
      end: String(dataPoint._end),
      included_only: "true",
    });
    navigate(`/transactions?${params}`);
  };

  // Handle heatmap day click
  const handleDayClick = (date: string) => {
    const d = new Date(date + "T00:00:00Z");
    const start = Math.floor(d.getTime() / 1000);
    const end = start + 86400 - 1;
    const params = new URLSearchParams({
      start: String(start),
      end: String(end),
      include_positive: "true",
    });
    navigate(`/transactions?${params}`);
  };

  // Handle merchant click
  const handleMerchantClick = (merchantName: string) => {
    const params = new URLSearchParams({
      search: merchantName,
      included_only: "true",
    });
    navigate(`/transactions?${params}`);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Analytics</h1>
        <div className="flex gap-1">
          {[3, 6, 12].map((m) => (
            <Button
              key={m}
              variant={months === m ? "default" : "outline"}
              size="sm"
              onClick={() => setMonths(m)}
            >
              {m}mo
            </Button>
          ))}
        </div>
      </div>

      {/* Category Trend */}
      <Card>
        <CardHeader>
          <CardTitle>Spending by Category Over Time</CardTitle>
        </CardHeader>
        <CardContent>
          {trendLoading ? (
            <div className="h-[350px] flex items-center justify-center text-muted-foreground">Loading...</div>
          ) : trendChartData.length > 0 ? (
            <>
              <ResponsiveContainer width="100%" height={350}>
                <AreaChart
                  data={trendChartData}
                  onClick={(state) => {
                    const category = pendingCategory.current;
                    pendingCategory.current = null;
                    const point = trendChartData[Number(state?.activeIndex)];
                    if (category && point) handleTrendClick(category, point);
                  }}
                >
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip formatter={(v) => `$${Number(v).toFixed(2)}`} />
                  {topCategories.map((cat, i) => (
                    <Area
                      key={cat}
                      type="monotone"
                      dataKey={cat}
                      stackId="1"
                      fill={CHART_COLORS[i % CHART_COLORS.length]}
                      stroke={CHART_COLORS[i % CHART_COLORS.length]}
                      fillOpacity={0.6}
                      className="cursor-pointer"
                      onClick={() => {
                        pendingCategory.current = cat;
                      }}
                    />
                  ))}
                </AreaChart>
              </ResponsiveContainer>
              <p className="text-xs text-muted-foreground text-center mt-2">Click a category area to view transactions</p>
            </>
          ) : (
            <div className="h-[350px] flex items-center justify-center text-muted-foreground">No data available</div>
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Daily Spending Heatmap */}
        <Card>
          <CardHeader>
            <CardTitle>Daily Spending</CardTitle>
          </CardHeader>
          <CardContent>
            {dailyLoading ? (
              <div className="h-[200px] flex items-center justify-center text-muted-foreground">Loading...</div>
            ) : dailyData && dailyData.days && dailyData.days.length > 0 ? (
              <>
                <SpendingHeatmap days={dailyData.days} onDayClick={handleDayClick} />
                <p className="text-xs text-muted-foreground text-center mt-2">Click a day to view transactions</p>
              </>
            ) : (
              <div className="h-[200px] flex items-center justify-center text-muted-foreground">No data available</div>
            )}
          </CardContent>
        </Card>

        {/* Top Merchants */}
        <Card>
          <CardHeader>
            <CardTitle>Top Merchants</CardTitle>
          </CardHeader>
          <CardContent>
            {merchantLoading ? (
              <div className="h-[350px] flex items-center justify-center text-muted-foreground">Loading...</div>
            ) : merchantData && merchantData.merchants && merchantData.merchants.length > 0 ? (
              <>
                <ResponsiveContainer width="100%" height={350}>
                  <BarChart
                    data={merchantData.merchants}
                    layout="vertical"
                    margin={{ left: 10, right: 20 }}
                  >
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis type="number" tickFormatter={(v) => `$${v}`} />
                    <YAxis
                      type="category"
                      dataKey="name"
                      width={150}
                      tick={{ fontSize: 12 }}
                    />
                    <Tooltip formatter={(v) => `$${Number(v).toFixed(2)}`} />
                    <Bar
                      dataKey="total"
                      fill={CHART_COLORS[0]}
                      className="cursor-pointer"
                      onClick={(entry) => {
                        const name = entry.payload?.name;
                        if (name) handleMerchantClick(name);
                      }}
                    />
                  </BarChart>
                </ResponsiveContainer>
                <p className="text-xs text-muted-foreground text-center mt-2">Click a merchant to view transactions</p>
              </>
            ) : (
              <div className="h-[350px] flex items-center justify-center text-muted-foreground">No data available</div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// Helper to check if there are categories beyond the top N
function other(data: { periods: { totals: Record<string, number> }[] }, topCats: string[]): boolean {
  for (const period of data.periods) {
    for (const cat of Object.keys(period.totals)) {
      if (!topCats.includes(cat)) return true;
    }
  }
  return false;
}

// Spending Heatmap component - CSS grid calendar
function SpendingHeatmap({
  days,
  onDayClick,
}: {
  days: { date: string; total: number }[];
  onDayClick: (date: string) => void;
}) {
  const dayMap = useMemo(() => {
    const map = new Map<string, number>();
    for (const d of days) map.set(d.date, d.total);
    return map;
  }, [days]);

  const maxTotal = useMemo(() => Math.max(...days.map((d) => d.total), 1), [days]);

  // Generate all dates in range
  const allDates = useMemo(() => {
    if (days.length === 0) return [];
    const dates: string[] = [];
    const sorted = [...days].sort((a, b) => a.date.localeCompare(b.date));
    const start = new Date(sorted[0].date + "T00:00:00");
    const end = new Date(sorted[sorted.length - 1].date + "T00:00:00");
    const current = new Date(start);
    while (current <= end) {
      dates.push(current.toISOString().slice(0, 10));
      current.setDate(current.getDate() + 1);
    }
    return dates;
  }, [days]);

  // Group by week (starting Sunday)
  const weeks = useMemo(() => {
    if (allDates.length === 0) return [];
    const result: string[][] = [];
    let currentWeek: string[] = [];
    // Pad first week
    const firstDay = new Date(allDates[0] + "T00:00:00").getDay();
    for (let i = 0; i < firstDay; i++) currentWeek.push("");
    for (const date of allDates) {
      const dow = new Date(date + "T00:00:00").getDay();
      if (dow === 0 && currentWeek.length > 0) {
        result.push(currentWeek);
        currentWeek = [];
      }
      currentWeek.push(date);
    }
    if (currentWeek.length > 0) {
      while (currentWeek.length < 7) currentWeek.push("");
      result.push(currentWeek);
    }
    return result;
  }, [allDates]);

  const getColor = (total: number) => {
    if (total === 0) return "bg-muted";
    const intensity = Math.min(total / maxTotal, 1);
    if (intensity < 0.25) return "bg-green-200 dark:bg-green-900";
    if (intensity < 0.5) return "bg-yellow-300 dark:bg-yellow-800";
    if (intensity < 0.75) return "bg-orange-400 dark:bg-orange-700";
    return "bg-red-500 dark:bg-red-700";
  };

  const dayLabels = ["S", "M", "T", "W", "T", "F", "S"];

  // Get month labels for the top
  const monthLabels = useMemo(() => {
    const labels: { label: string; colStart: number }[] = [];
    let lastMonth = "";
    for (let wi = 0; wi < weeks.length; wi++) {
      for (const date of weeks[wi]) {
        if (!date) continue;
        const month = new Date(date + "T00:00:00").toLocaleDateString(undefined, { month: "short" });
        if (month !== lastMonth) {
          labels.push({ label: month, colStart: wi });
          lastMonth = month;
        }
        break;
      }
    }
    return labels;
  }, [weeks]);

  return (
    <div className="overflow-x-auto">
      {/* Month labels */}
      <div className="flex ml-6 mb-1 text-xs text-muted-foreground" style={{ gap: 0 }}>
        {monthLabels.map((m, i) => (
          <span
            key={i}
            style={{
              position: "relative",
              left: `${m.colStart * 16}px`,
              marginRight: i < monthLabels.length - 1 ? `${(monthLabels[i + 1].colStart - m.colStart - 1) * 16}px` : 0,
            }}
          >
            {m.label}
          </span>
        ))}
      </div>
      <div className="flex gap-0.5">
        {/* Day labels */}
        <div className="flex flex-col gap-0.5 mr-1">
          {dayLabels.map((label, i) => (
            <div key={i} className="w-3 h-3 text-[8px] text-muted-foreground flex items-center justify-center">
              {i % 2 === 1 ? label : ""}
            </div>
          ))}
        </div>
        {/* Weeks */}
        {weeks.map((week, wi) => (
          <div key={wi} className="flex flex-col gap-0.5">
            {week.map((date, di) => (
              <div
                key={di}
                className={`w-3 h-3 rounded-sm ${
                  date
                    ? `${getColor(dayMap.get(date) || 0)} cursor-pointer hover:ring-1 hover:ring-foreground transition-all`
                    : "bg-transparent"
                }`}
                title={date ? `${date}: $${(dayMap.get(date) || 0).toFixed(2)}` : ""}
                onClick={() => date && onDayClick(date)}
              />
            ))}
          </div>
        ))}
      </div>
      {/* Legend */}
      <div className="flex items-center gap-1 mt-2 text-xs text-muted-foreground justify-end">
        <span>Less</span>
        <div className="w-3 h-3 rounded-sm bg-muted" />
        <div className="w-3 h-3 rounded-sm bg-green-200 dark:bg-green-900" />
        <div className="w-3 h-3 rounded-sm bg-yellow-300 dark:bg-yellow-800" />
        <div className="w-3 h-3 rounded-sm bg-orange-400 dark:bg-orange-700" />
        <div className="w-3 h-3 rounded-sm bg-red-500 dark:bg-red-700" />
        <span>More</span>
      </div>
    </div>
  );
}
