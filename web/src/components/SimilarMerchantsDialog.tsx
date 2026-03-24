import { useState, useCallback } from "react";
import { useFindSimilarMerchants, useBulkApplyCategory } from "@/api/queries";
import type { SimilarMerchantInfo } from "@/api/types";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";

interface PendingSuggestion {
  merchantDescription: string;
  category: string;
}

/**
 * SimilarMerchantsDialog manages a queue of propagation suggestions.
 * After a user corrects a merchant's category, call `triggerSimilarityCheck`
 * to find and suggest similar merchants.
 */
export function useSimilarMerchants() {
  const [queue, setQueue] = useState<PendingSuggestion[]>([]);
  const [currentSuggestions, setCurrentSuggestions] = useState<SimilarMerchantInfo[] | null>(null);
  const [currentCategory, setCurrentCategory] = useState("");
  const findSimilar = useFindSimilarMerchants();

  const triggerSimilarityCheck = useCallback(
    (merchantDescription: string, category: string) => {
      // Add to queue.
      const suggestion = { merchantDescription, category };

      if (currentSuggestions !== null) {
        // Already showing a dialog — queue this one.
        setQueue((q) => [...q, suggestion]);
        return;
      }

      // Fetch similarity suggestions.
      findSimilar.mutate(
        { merchant_description: merchantDescription, category },
        {
          onSuccess: (data) => {
            if (data && data.length > 0) {
              setCurrentSuggestions(data);
              setCurrentCategory(category);
            }
          },
          // Silent failure — correction is already saved.
        },
      );
    },
    [findSimilar, currentSuggestions],
  );

  const dismiss = useCallback(() => {
    setCurrentSuggestions(null);
    setCurrentCategory("");

    // Process next in queue.
    setQueue((q) => {
      if (q.length === 0) return q;
      const [next, ...rest] = q;
      findSimilar.mutate(
        { merchant_description: next.merchantDescription, category: next.category },
        {
          onSuccess: (data) => {
            if (data && data.length > 0) {
              setCurrentSuggestions(data);
              setCurrentCategory(next.category);
            }
          },
        },
      );
      return rest;
    });
  }, [findSimilar]);

  return {
    triggerSimilarityCheck,
    currentSuggestions,
    currentCategory,
    dismiss,
    isLoading: findSimilar.isPending,
  };
}

interface SimilarMerchantsDialogProps {
  suggestions: SimilarMerchantInfo[] | null;
  category: string;
  onDismiss: () => void;
}

export function SimilarMerchantsDialog({
  suggestions,
  category,
  onDismiss,
}: SimilarMerchantsDialogProps) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const bulkApply = useBulkApplyCategory();

  const isOpen = suggestions !== null && suggestions.length > 0;

  // Reset selection when suggestions change.
  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setSelected(new Set());
      onDismiss();
    }
  };

  const toggleMerchant = (merchant: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(merchant)) {
        next.delete(merchant);
      } else {
        next.add(merchant);
      }
      return next;
    });
  };

  const selectAll = () => {
    if (!suggestions) return;
    if (selected.size === suggestions.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(suggestions.map((s) => s.merchant_description)));
    }
  };

  const handleApply = () => {
    if (selected.size === 0) return;
    bulkApply.mutate(
      { merchants: Array.from(selected), category },
      {
        onSuccess: () => {
          setSelected(new Set());
          onDismiss();
        },
      },
    );
  };

  if (!isOpen) return null;

  return (
    <Sheet open={isOpen} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="w-[400px] sm:w-[450px]">
        <SheetHeader>
          <SheetTitle>Similar Merchants Found</SheetTitle>
          <SheetDescription>
            Apply <span className="font-medium">"{category}"</span> to these similar merchants?
          </SheetDescription>
        </SheetHeader>

        <div className="py-4 space-y-2 overflow-y-auto max-h-[60vh]">
          <div className="flex items-center gap-2 pb-2 border-b">
            <Checkbox
              checked={selected.size === suggestions.length}
              onCheckedChange={selectAll}
            />
            <span className="text-sm text-muted-foreground">
              Select all ({suggestions.length})
            </span>
          </div>

          {suggestions.map((s) => (
            <label
              key={s.merchant_description}
              className="flex items-start gap-3 p-2 rounded-md hover:bg-accent cursor-pointer"
            >
              <Checkbox
                checked={selected.has(s.merchant_description)}
                onCheckedChange={() => toggleMerchant(s.merchant_description)}
                className="mt-0.5"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium truncate">
                  {s.merchant_description}
                </div>
                <div className="text-xs text-muted-foreground flex items-center gap-2">
                  {s.current_category && (
                    <span>Currently: {s.current_category}</span>
                  )}
                  {s.transaction_count > 0 && (
                    <span>
                      {s.transaction_count} txn{s.transaction_count !== 1 ? "s" : ""}
                    </span>
                  )}
                </div>
              </div>
            </label>
          ))}
        </div>

        <SheetFooter className="flex-row gap-2 sm:justify-between">
          <Button variant="outline" onClick={onDismiss}>
            Dismiss
          </Button>
          <Button
            onClick={handleApply}
            disabled={selected.size === 0 || bulkApply.isPending}
          >
            {bulkApply.isPending
              ? "Applying..."
              : `Apply to ${selected.size} merchant${selected.size !== 1 ? "s" : ""}`}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
