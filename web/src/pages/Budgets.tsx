import { useState, useRef, useEffect } from "react";
import {
  useBudgetStatus,
  useUpsertBudget,
  useDeleteBudget,
} from "@/api/queries";
import type { BudgetedCategory, UnbudgetedCategory } from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ChatPanel, type ChatMessage } from "@/components/ChatPanel";
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

// --- Budgeted Row ---

function BudgetedRow({
  item,
  onEdit,
  onDelete,
  onFixWithAI,
}: {
  item: BudgetedCategory;
  onEdit: (category: string, amount: number) => void;
  onDelete: (category: string) => void;
  onFixWithAI: (category: string, spent: number, amount: number) => void;
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
  );
}

// --- Unbudgeted Row ---

function UnbudgetedRow({
  item,
  onSetBudget,
  onFixWithAI,
}: {
  item: UnbudgetedCategory;
  onSetBudget: (category: string, amount: number) => void;
  onFixWithAI: (category: string, spent: number) => void;
}) {
  return (
    <div className="flex items-center justify-between py-2">
      <div className="flex items-center gap-3">
        <span className="text-sm text-muted-foreground">{item.category}</span>
        <span className="text-xs text-muted-foreground tabular-nums">
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
  );
}

// --- Empty State ---

function EmptyState({
  unbudgeted,
  onSetBudget,
  onFixWithAI,
  onCreateAllWithAI,
}: {
  unbudgeted: UnbudgetedCategory[];
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

  // Drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerKey, setDrawerKey] = useState(0);
  const [drawerMessages, setDrawerMessages] = useState<ChatMessage[]>([]);

  const handleEdit = (category: string, amount: number) => {
    upsertBudget.mutate({ category, amount });
  };

  const handleDelete = (category: string) => {
    deleteBudget.mutate(category);
  };

  const openDrawer = (message: string) => {
    setDrawerMessages([{ role: "user", content: message }]);
    setDrawerKey((k) => k + 1);
    setDrawerOpen(true);
  };

  const handleFixWithAI = (
    category: string,
    spent: number,
    amount?: number
  ) => {
    const periodLabel = data?.period.label || "this period";
    if (amount) {
      const pct = Math.round((spent / amount) * 100);
      openDrawer(`Help me with my ${category} budget. Current status: ${formatCurrency(spent)} spent of ${formatCurrency(amount)} limit (${pct}%). This billing period: ${periodLabel}.`);
    } else {
      openDrawer(`I'm spending ${formatCurrency(spent)} on ${category} this period but have no budget set. Help me decide on a budget for ${periodLabel}.`);
    }
  };

  const handleCreateAllWithAI = () => {
    const cats = [
      ...(budgeted || []).map((b) => `- ${b.category}: ${formatCurrency(b.spent)} spent (budget: ${formatCurrency(b.amount)})`),
      ...(unbudgeted || []).map((u) => `- ${u.category}: ${formatCurrency(u.spent)} spent (no budget)`),
    ];
    const periodLabel = data?.period.label || "this period";
    openDrawer(
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

      {/* AI Budget Assistant Drawer */}
      <Sheet open={drawerOpen} onOpenChange={setDrawerOpen}>
        <SheetContent side="right" className="w-[420px] sm:w-[420px] p-0 flex flex-col">
          <SheetHeader className="p-4 border-b border-border">
            <SheetTitle>Budget Assistant</SheetTitle>
          </SheetHeader>
          <ChatPanel
            key={drawerKey}
            initialMessages={drawerMessages}
            autoSendFirst={true}
            className="flex-1 min-h-0"
            placeholder="Ask about this budget..."
          />
        </SheetContent>
      </Sheet>
    </div>
  );
}
