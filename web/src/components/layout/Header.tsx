import { Button } from "@/components/ui/button";
import { useTheme } from "@/hooks/useTheme";
import { useTriggerSync } from "@/api/queries";

export function Header() {
  const { dark, toggle } = useTheme();
  const sync = useTriggerSync();

  return (
    <header className="h-14 border-b border-border bg-card px-6 flex items-center justify-end gap-2">
      <Button
        variant="outline"
        size="sm"
        onClick={() => sync.mutate()}
        disabled={sync.isPending}
      >
        {sync.isPending ? "Syncing..." : "Sync Now"}
      </Button>
      <Button variant="ghost" size="sm" onClick={toggle}>
        {dark ? "Light" : "Dark"}
      </Button>
    </header>
  );
}
