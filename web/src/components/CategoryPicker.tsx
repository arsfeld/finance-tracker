import { useState, useRef, useEffect } from "react";
import { useCategoryNames, useOverrideCategory } from "@/api/queries";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";

// Default categories with emojis for suggestions.
const DEFAULT_CATEGORIES: Record<string, string> = {
  "Groceries": "🛒",
  "Dining": "🍽️",
  "Transportation": "🚗",
  "Gas": "⛽",
  "Entertainment": "🎬",
  "Shopping": "🛍️",
  "Subscriptions": "📱",
  "Healthcare": "🏥",
  "Utilities": "💡",
  "Insurance": "🛡️",
  "Housing": "🏠",
  "Childcare": "👶",
  "Education": "📚",
  "Clothing": "👕",
  "Personal Care": "💇",
  "Fitness": "🏋️",
  "Travel": "✈️",
  "Gifts": "🎁",
  "Pets": "🐾",
  "Home Improvement": "🔨",
  "Electronics": "💻",
  "Pharmacy": "💊",
  "Coffee": "☕",
  "Alcohol": "🍷",
  "Parking": "🅿️",
  "ATM": "🏧",
  "Fees": "💳",
  "Other": "📦",
};

function emojiForCategory(name: string): string {
  // Exact match first.
  if (DEFAULT_CATEGORIES[name]) return DEFAULT_CATEGORIES[name];
  // Case-insensitive match.
  const lower = name.toLowerCase();
  for (const [k, v] of Object.entries(DEFAULT_CATEGORIES)) {
    if (k.toLowerCase() === lower) return v;
  }
  // Fuzzy match on contains.
  for (const [k, v] of Object.entries(DEFAULT_CATEGORIES)) {
    if (lower.includes(k.toLowerCase()) || k.toLowerCase().includes(lower)) return v;
  }
  return "🏷️";
}

interface CategoryPickerProps {
  transactionId: string;
  description: string;
  currentCategory: string;
  onCategoryChanged?: (merchantDescription: string, category: string) => void;
}

export function CategoryPicker({ transactionId, description, currentCategory, onCategoryChanged }: CategoryPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const existingCategories = useCategoryNames();
  const override = useOverrideCategory();
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Build the full category list: existing categories merged with defaults.
  const allCategories = new Map<string, string>();
  // Add existing categories first (they take priority).
  for (const name of existingCategories) {
    allCategories.set(name, emojiForCategory(name));
  }
  // Add defaults that aren't already present.
  for (const [name, emoji] of Object.entries(DEFAULT_CATEGORIES)) {
    if (!allCategories.has(name)) {
      allCategories.set(name, emoji);
    }
  }

  const categories = Array.from(allCategories.entries());

  // Filter by search.
  const filtered = search
    ? categories.filter(([name]) => name.toLowerCase().includes(search.toLowerCase()))
    : categories;

  const isNew = search && !categories.some(([name]) => name.toLowerCase() === search.toLowerCase());

  const handleSelect = (category: string) => {
    override.mutate(
      { txnId: transactionId, category, applyToMerchant: true },
      {
        onSuccess: () => {
          onCategoryChanged?.(description, category);
        },
      },
    );
    setOpen(false);
    setSearch("");
  };

  // Close on escape.
  useEffect(() => {
    if (!open) setSearch("");
  }, [open]);

  const emoji = currentCategory ? emojiForCategory(currentCategory) : "";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          ref={triggerRef}
          className="text-sm px-2 py-1 rounded hover:bg-accent transition-colors cursor-pointer text-left inline-flex items-center gap-1.5 max-w-[200px]"
        >
          {currentCategory ? (
            <>
              <span>{emoji}</span>
              <span className="truncate">{currentCategory}</span>
            </>
          ) : (
            <span className="text-muted-foreground italic">+ Category</span>
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-[220px] p-0" align="start">
        <Command>
          <CommandInput
            placeholder="Search or create..."
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            <CommandEmpty className="py-2 px-3 text-sm text-muted-foreground">
              {search ? "No matching categories" : "Type to search..."}
            </CommandEmpty>
            <CommandGroup>
              {filtered.map(([name, catEmoji]) => (
                <CommandItem
                  key={name}
                  value={name}
                  onSelect={() => handleSelect(name)}
                  className="cursor-pointer"
                >
                  <span className="mr-2">{catEmoji}</span>
                  <span>{name}</span>
                  {name === currentCategory && (
                    <span className="ml-auto text-xs text-muted-foreground">✓</span>
                  )}
                </CommandItem>
              ))}
              {isNew && (
                <CommandItem
                  value={`create-${search}`}
                  onSelect={() => handleSelect(search)}
                  className="cursor-pointer"
                >
                  <span className="mr-2">🏷️</span>
                  <span>Create "{search}"</span>
                </CommandItem>
              )}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
