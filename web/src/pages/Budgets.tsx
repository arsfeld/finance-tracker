import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router";
import {
  useBudgetStatus,
  useUpsertBudget,
  useDeleteBudget,
  useCategoryTransactions,
} from "@/api/queries";
import type {
  BudgetedCategory,
  UnbudgetedCategory,
  DBTransaction,
} from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useChatDrawer } from "@/hooks/useChatDrawer";
import { cn } from "@/lib/utils";
import { ChevronRightIcon } from "lucide-react";
import { Collapsible } from "radix-ui";

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
  suggestedAmount,
  onSave,
}: {
  category: string;
  suggestedAmount?: number;
  onSave: (category: string, amount: number) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(
    suggestedAmount ? String(suggestedAmount) : ""
  );
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
    setValue(suggestedAmount ? String(suggestedAmount) : "");
  };

  if (!editing) {
    return (
      <div className="flex items-center gap-2">
        {suggestedAmount && (
          <Button
            variant="outline"
            size="sm"
            className="text-xs"
            onClick={() => onSave(category, suggestedAmount)}
          >
            Accept {formatCurrency(suggestedAmount)}
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setEditing(true)}
          className="text-xs"
        >
          Custom
        </Button>
      </div>
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
          setValue(suggestedAmount ? String(suggestedAmount) : "");
        }
      }}
      className="text-sm px-2 py-1 rounded border border-ring w-[100px] text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-ring"
    />
  );
}

// --- Transaction Preview (for accordion drill-down) ---

function formatDate(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
}

function TransactionPreview({
  transactions,
  isLoading,
  isError,
  onViewAll,
}: {
  transactions: DBTransaction[] | null | undefined;
  isLoading: boolean;
  isError: boolean;
  onViewAll: () => void;
}) {
  return (
    <div className="pb-3 pl-8">
      {isLoading && (
        <p className="text-xs text-muted-foreground py-2">Loading...</p>
      )}
      {isError && (
        <p className="text-xs text-destructive py-2">
          Failed to load transactions
        </p>
      )}
      {!isLoading && !isError && (!transactions || transactions.length === 0) && (
        <p className="text-xs text-muted-foreground py-2">
          No transactions this period
        </p>
      )}
      {!isLoading && !isError && transactions && transactions.length > 0 && (
        <div className="space-y-1">
          {transactions.map((txn) => (
            <div
              key={txn.id}
              className="flex items-center justify-between text-xs py-1"
            >
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <span className="text-muted-foreground shrink-0">
                  {formatDate(txn.posted)}
                </span>
                <span className="truncate">{txn.description}</span>
              </div>
              <span className="tabular-nums text-muted-foreground shrink-0 ml-3">
                {formatCurrency(txn.amount)}
              </span>
            </div>
          ))}
        </div>
      )}
      <button
        onClick={onViewAll}
        className="text-xs text-primary hover:underline mt-2 cursor-pointer"
      >
        View all transactions
      </button>
    </div>
  );
}

// --- Budgeted Row ---

function BudgetedRow({
  item,
  period,
  onEdit,
  onDelete,
  onFixWithAI,
}: {
  item: BudgetedCategory;
  period: { start: number; end: number };
  onEdit: (category: string, amount: number) => void;
  onDelete: (category: string) => void;
  onFixWithAI: (category: string, spent: number, amount: number) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const navigate = useNavigate();
  const { data: transactions, isLoading, isError } = useCategoryTransactions(
    item.category,
    period.start,
    period.end,
    isOpen
  );
  const isOver = item.percent >= 100;
  const statusText = isOver
    ? `Over by ${formatCurrency(Math.abs(item.remaining))}`
    : `${formatCurrency(item.remaining)} remaining`;

  const handleViewAll = () => {
    const params = new URLSearchParams({
      category: item.category,
      start: String(period.start),
      end: String(period.end),
      included_only: "true",
    });
    navigate(`/transactions?${params}`);
  };

  return (
    <Collapsible.Root open={isOpen} onOpenChange={setIsOpen}>
      <div className="space-y-2 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <Collapsible.Trigger asChild>
              <button
                className="p-0.5 rounded hover:bg-accent transition-colors cursor-pointer"
                aria-label={`Show transactions for ${item.category}`}
              >
                <ChevronRightIcon
                  className={cn(
                    "h-4 w-4 text-muted-foreground transition-transform duration-200",
                    isOpen && "rotate-90"
                  )}
                />
              </button>
            </Collapsible.Trigger>
            <span className="text-sm font-medium">{item.category}</span>
          </div>
          <div className="flex items-center gap-2">
            {isOver && (
              <Button
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={() =>
                  onFixWithAI(item.category, item.spent, item.amount)
                }
              >
                Fix with AI
              </Button>
            )}
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
      <Collapsible.Content className="overflow-hidden data-[state=open]:animate-collapsible-down data-[state=closed]:animate-collapsible-up">
        <TransactionPreview
          transactions={transactions}
          isLoading={isLoading}
          isError={isError}
          onViewAll={handleViewAll}
        />
      </Collapsible.Content>
    </Collapsible.Root>
  );
}

// --- Unbudgeted Row ---

function UnbudgetedRow({
  item,
  period,
  onSetBudget,
  onFixWithAI,
}: {
  item: UnbudgetedCategory;
  period: { start: number; end: number };
  onSetBudget: (category: string, amount: number) => void;
  onFixWithAI: (category: string, spent: number) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const navigate = useNavigate();
  const { data: transactions, isLoading, isError } = useCategoryTransactions(
    item.category,
    period.start,
    period.end,
    isOpen
  );

  const handleViewAll = () => {
    const params = new URLSearchParams({
      category: item.category,
      start: String(period.start),
      end: String(period.end),
      included_only: "true",
    });
    navigate(`/transactions?${params}`);
  };

  return (
    <Collapsible.Root open={isOpen} onOpenChange={setIsOpen}>
      <div className="flex items-center justify-between py-2">
        <div className="flex items-center gap-1">
          <Collapsible.Trigger asChild>
            <button
              className="p-0.5 rounded hover:bg-accent transition-colors cursor-pointer"
              aria-label={`Show transactions for ${item.category}`}
            >
              <ChevronRightIcon
                className={cn(
                  "h-4 w-4 text-muted-foreground transition-transform duration-200",
                  isOpen && "rotate-90"
                )}
              />
            </button>
          </Collapsible.Trigger>
          <span className="text-sm text-muted-foreground">{item.category}</span>
          <span className="text-xs text-muted-foreground tabular-nums ml-2">
            {formatCurrency(item.spent)} spent
          </span>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="text-xs"
            onClick={() => onFixWithAI(item.category, item.spent)}
          >
            Ask AI
          </Button>
          <SetBudgetInput
            category={item.category}
            suggestedAmount={item.suggested_amount}
            onSave={onSetBudget}
          />
        </div>
      </div>
      <Collapsible.Content className="overflow-hidden data-[state=open]:animate-collapsible-down data-[state=closed]:animate-collapsible-up">
        <TransactionPreview
          transactions={transactions}
          isLoading={isLoading}
          isError={isError}
          onViewAll={handleViewAll}
        />
      </Collapsible.Content>
    </Collapsible.Root>
  );
}

// --- Empty State ---

function EmptyState({
  unbudgeted,
  period,
  onSetBudget,
  onFixWithAI,
  onCreateAllWithAI,
}: {
  unbudgeted: UnbudgetedCategory[];
  period: { start: number; end: number };
  onSetBudget: (category: string, amount: number) => void;
  onFixWithAI: (category: string, spent: number) => void;
  onCreateAllWithAI: () => void;
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
        <div className="space-y-4">
          <Button onClick={onCreateAllWithAI} size="lg">
            Create Budget with AI
          </Button>
          <p className="text-xs text-muted-foreground">
            AI will analyze your spending and suggest budgets for each category
          </p>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Or set budgets manually
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="divide-y divide-border">
                {unbudgeted.map((item) => (
                  <UnbudgetedRow
                    key={item.category}
                    item={item}
                    period={period}
                    onSetBudget={onSetBudget}
                    onFixWithAI={onFixWithAI}
                  />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}

// --- Main Page ---

export default function Budgets() {
  const { data, isLoading, error } = useBudgetStatus();
  const upsertBudget = useUpsertBudget();
  const deleteBudget = useDeleteBudget();
  const chatDrawer = useChatDrawer();

  const handleEdit = (category: string, amount: number) => {
    upsertBudget.mutate({ category, amount });
  };

  const handleDelete = (category: string) => {
    deleteBudget.mutate(category);
  };

  const handleFixWithAI = (
    category: string,
    spent: number,
    amount?: number
  ) => {
    const periodLabel = data?.period.label || "this period";
    if (amount) {
      const pct = Math.round((spent / amount) * 100);
      chatDrawer.open(`Help me with my ${category} budget. Current status: ${formatCurrency(spent)} spent of ${formatCurrency(amount)} limit (${pct}%). This billing period: ${periodLabel}.`);
    } else {
      chatDrawer.open(`I'm spending ${formatCurrency(spent)} on ${category} this period but have no budget set. Help me decide on a budget for ${periodLabel}.`);
    }
  };

  const handleCreateAllWithAI = () => {
    const cats = [
      ...(budgeted || []).map((b) => `- ${b.category}: ${formatCurrency(b.spent)} spent (budget: ${formatCurrency(b.amount)})`),
      ...(unbudgeted || []).map((u) => `- ${u.category}: ${formatCurrency(u.spent)} spent (no budget)`),
    ];
    const periodLabel = data?.period.label || "this period";
    chatDrawer.open(
      `Help me set up my budget for ${periodLabel}. Here's my current spending:\n\n${cats.join("\n")}\n\nPlease suggest and create a reasonable budget for each category. Use get_budget_status first to check what's already set, then use set_budget for each category.`
    );
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
        <div className="flex items-center gap-3">
          {hasBudgets && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleCreateAllWithAI}
            >
              Ask AI
            </Button>
          )}
          <span className="text-sm text-muted-foreground">{period.label}</span>
        </div>
      </div>

      {!hasBudgets ? (
        <EmptyState
          unbudgeted={unbudgeted || []}
          period={period}
          onSetBudget={handleEdit}
          onFixWithAI={(cat, spent) => handleFixWithAI(cat, spent)}
          onCreateAllWithAI={handleCreateAllWithAI}
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
                    period={period}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                    onFixWithAI={(cat, spent, amt) =>
                      handleFixWithAI(cat, spent, amt)
                    }
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
                      period={period}
                      onSetBudget={handleEdit}
                      onFixWithAI={(cat, spent) =>
                        handleFixWithAI(cat, spent)
                      }
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
