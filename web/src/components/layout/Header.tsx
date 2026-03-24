import { Button } from "@/components/ui/button";
import { useTheme } from "@/hooks/useTheme";
import { useJobStatus } from "@/hooks/JobStatusContext";
import { useTriggerSync } from "@/api/queries";
import { useChatDrawer } from "@/hooks/useChatDrawer";

export function Header({ onMenuToggle }: { onMenuToggle?: () => void }) {
  const { dark, toggle } = useTheme();
  const sync = useTriggerSync();
  const job = useJobStatus();
  const chatDrawer = useChatDrawer();

  const isRunning = job.state === "running";

  return (
    <header className="border-b border-border bg-card">
      <div className="h-14 px-4 md:px-6 flex items-center gap-2">
        <button
          onClick={onMenuToggle}
          className="md:hidden p-2 -ml-2 rounded-md hover:bg-accent transition-colors"
          aria-label="Toggle menu"
        >
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <span className="md:hidden font-semibold text-sm">Finance Tracker</span>
        <div className="flex-1" />
        <Button
          variant="outline"
          size="sm"
          onClick={() => chatDrawer.open()}
        >
          AI Chat
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => sync.mutate()}
          disabled={sync.isPending || isRunning}
        >
          {isRunning && job.type === "sync" ? "Syncing..." : "Sync Now"}
        </Button>
        <Button variant="ghost" size="sm" onClick={toggle}>
          {dark ? "Light" : "Dark"}
        </Button>
      </div>

      {job.type !== "idle" && (
        <StatusBanner state={job.state} message={job.message} />
      )}
    </header>
  );
}

function StatusBanner({ state, message }: { state: string; message?: string }) {
  const bg =
    state === "running"
      ? "bg-blue-500/10 text-blue-700 dark:text-blue-400 border-blue-500/20"
      : state === "success"
        ? "bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/20"
        : "bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20";

  return (
    <div className={`px-6 py-2 text-sm flex items-center gap-2 border-b ${bg}`}>
      {state === "running" && <Spinner />}
      {state === "success" && <span>&#10003;</span>}
      {state === "error" && <span>&#10007;</span>}
      <span>{message}</span>
    </div>
  );
}

function Spinner() {
  return (
    <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
  );
}
