import { useState, useRef, useEffect } from "react";
import {
  useBudgetStatus,
  useUpsertBudget,
  useDeleteBudget,
} from "@/api/queries";
import type { BudgetedCategory, UnbudgetedCategory } from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const currencyFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

function formatCurrency(amount: number): string {
  return currencyFormatter.format(Math.abs(amount));
}

function getBarColor(percent: number): string {
  if (percent >= 100) return "bg-red-600";
  if (percent >= 90) return "bg-red-500";
  if (percent >= 75) return "bg-amber-500";
  return "bg-emerald-500";
}

function getTrackColor(percent: number): string {
  if (percent >= 100) return "bg-red-100 dark:bg-red-950";
  return "bg-muted";
}

// --- Progress Bar ---

function BudgetProgressBar({
  category,
  percent,
  statusText,
}: {
  category: string;
  percent: number;
  statusText: string;
}) {
  return (
    <div
      role="progressbar"
      aria-valuenow={Math.round(percent)}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={`${category} budget: ${Math.round(percent)}% used. ${statusText}`}
      className={cn(
        "h-2.5 w-full rounded-full overflow-hidden",
        getTrackColor(percent)
      )}
    >
      <div
        className={cn(
          "h-full rounded-full transition-all duration-500 ease-out",
          getBarColor(percent)
        )}
        style={{ width: `${Math.min(percent, 100)}%` }}
      />
    </div>
  );
}

// --- Inline Editor ---

function InlineBudgetEditor({
  currentAmount,
  onSave,
}: {
  currentAmount: number;
  onSave: (amount: number) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(String(currentAmount));
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.select();
    }
  }, [editing]);

  const commit = () => {
    const parsed = parseFloat(value);
    if (!isNaN(parsed) && parsed > 0 && parsed <= 1_000_000) {
      onSave(parsed);
    } else {
      setValue(String(currentAmount));
    }
    setEditing(false);
  };

  if (!editing) {
    return (
      <button
        onClick={() => {
          setValue(String(currentAmount));
          setEditing(true);
        }}
        className="text-sm px-2 py-1 rounded hover:bg-accent transition-colors cursor-pointer tabular-nums text-right min-w-[80px]"
      >
        {formatCurrency(currentAmount)}
      </button>
    );
  }

  return (
    <input
      ref={inputRef}
      type="number"
      min={1}
      max={1000000}
      step={1}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit();
        if (e.key === "Escape") {
          setValue(String(currentAmount));
          setEditing(false);
        }
      }}
      className="text-sm px-2 py-1 rounded border border-ring w-[100px] text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-ring"
    />
  );
}

// --- Set Budget (for unbudgeted categories) ---

function SetBudgetInput({
  category,
  onSave,
}: {
  category: string;
  onSave: (category: string, amount: number) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
    }
  }, [editing]);

  const commit = () => {
    const parsed = parseFloat(value);
    if (!isNaN(parsed) && parsed > 0) {
      onSave(category, parsed);
    }
    setEditing(false);
    setValue("");
  };

  if (!editing) {
    return (
      <Button
        variant="outline"
        size="sm"
        onClick={() => setEditing(true)}
        className="text-xs"
      >
        Set Budget
      </Button>
    );
  }

  return (
    <input
      ref={inputRef}
      type="number"
      min={1}
      step={1}
      placeholder="Amount"
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit();
        if (e.key === "Escape") {
          setEditing(false);
          setValue("");
        }
      }}
      className="text-sm px-2 py-1 rounded border border-ring w-[100px] text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-ring"
    />
  );
}

// --- Budgeted Row ---

function BudgetedRow({
  item,
  onEdit,
  onDelete,
}: {
  item: BudgetedCategory;
  onEdit: (category: string, amount: number) => void;
  onDelete: (category: string) => void;
}) {
  const isOver = item.percent >= 100;
  const statusText = isOver
    ? `Over by ${formatCurrency(Math.abs(item.remaining))}`
    : `${formatCurrency(item.remaining)} remaining`;

  return (
    <div className="space-y-2 py-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{item.category}</span>
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "text-sm tabular-nums",
              isOver ? "text-red-600 font-semibold" : "text-muted-foreground"
            )}
          >
            {formatCurrency(item.spent)} /
          </span>
          <InlineBudgetEditor
            currentAmount={item.amount}
            onSave={(amount) => onEdit(item.category, amount)}
          />
          <button
            onClick={() => onDelete(item.category)}
            className="text-muted-foreground hover:text-destructive text-xs px-1 transition-colors"
            title="Remove budget"
          >
            ✕
          </button>
        </div>
      </div>
      <BudgetProgressBar
        category={item.category}
        percent={item.percent}
        statusText={statusText}
      />
      <p
        className={cn(
          "text-xs",
          isOver ? "text-red-600 font-medium" : "text-muted-foreground"
        )}
      >
        {statusText} ({Math.round(item.percent)}%)
      </p>
    </div>
  );
}

// --- Unbudgeted Row ---

function UnbudgetedRow({
  item,
  onSetBudget,
}: {
  item: UnbudgetedCategory;
  onSetBudget: (category: string, amount: number) => void;
}) {
  return (
    <div className="flex items-center justify-between py-2">
      <div className="flex items-center gap-3">
        <span className="text-sm text-muted-foreground">{item.category}</span>
        <span className="text-xs text-muted-foreground tabular-nums">
          {formatCurrency(item.spent)} spent
        </span>
      </div>
      <SetBudgetInput category={item.category} onSave={onSetBudget} />
    </div>
  );
}

// --- Empty State ---

function EmptyState({
  unbudgeted,
  onSetBudget,
}: {
  unbudgeted: UnbudgetedCategory[];
  onSetBudget: (category: string, amount: number) => void;
}) {
  return (
    <div className="text-center py-12">
      <div className="text-4xl mb-3">📊</div>
      <h3 className="text-lg font-semibold mb-2">No budgets set yet</h3>
      <p className="text-muted-foreground max-w-sm mx-auto mb-6">
        Set spending limits for your categories to track how your actual spending
        compares to your goals.
      </p>
      {unbudgeted.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Your categories</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="divide-y divide-border">
              {unbudgeted.map((item) => (
                <UnbudgetedRow
                  key={item.category}
                  item={item}
                  onSetBudget={onSetBudget}
                />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// --- Main Page ---

export default function Budgets() {
  const { data, isLoading, error } = useBudgetStatus();
  const upsertBudget = useUpsertBudget();
  const deleteBudget = useDeleteBudget();

  const handleEdit = (category: string, amount: number) => {
    upsertBudget.mutate({ category, amount });
  };

  const handleDelete = (category: string) => {
    deleteBudget.mutate(category);
  };

  if (isLoading) {
    return (
      <div className="p-6 text-muted-foreground">Loading budgets...</div>
    );
  }

  if (error) {
    return (
      <div className="p-6 text-destructive">Error: {error.message}</div>
    );
  }

  if (!data) return null;

  const { period, budgeted, unbudgeted } = data;
  const hasBudgets = budgeted && budgeted.length > 0;

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Budgets</h1>
        <span className="text-sm text-muted-foreground">{period.label}</span>
      </div>

      {!hasBudgets ? (
        <EmptyState
          unbudgeted={unbudgeted || []}
          onSetBudget={handleEdit}
        />
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Budget Progress</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="divide-y divide-border">
                {budgeted.map((item) => (
                  <BudgetedRow
                    key={item.category}
                    item={item}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </CardContent>
          </Card>

          {unbudgeted && unbudgeted.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base text-muted-foreground">
                  Unbudgeted Categories
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="divide-y divide-border">
                  {unbudgeted.map((item) => (
                    <UnbudgetedRow
                      key={item.category}
                      item={item}
                      onSetBudget={handleEdit}
                    />
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
