import { useEffect, useState, useCallback, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

export type JobStatus = {
  type: "idle" | "sync" | "analysis" | "categorization";
  state: "running" | "success" | "error";
  message?: string;
  timestamp: number;
};

const IDLE: JobStatus = { type: "idle", state: "success", timestamp: 0 };

// Auto-clear success/error after this many ms.
const CLEAR_DELAY = 5000;

export function useSSE() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<JobStatus>(IDLE);
  const clearTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const set = useCallback((s: JobStatus) => {
    setStatus(s);
    if (clearTimer.current) clearTimeout(clearTimer.current);
    if (s.state !== "running") {
      clearTimer.current = setTimeout(() => setStatus(IDLE), CLEAR_DELAY);
    }
  }, []);

  useEffect(() => {
    const es = new EventSource("/api/events");

    es.addEventListener("sync_started", () => {
      set({ type: "sync", state: "running", message: "Syncing transactions...", timestamp: Date.now() });
    });

    es.addEventListener("sync_complete", (e) => {
      let msg = "Sync complete";
      try {
        const d = JSON.parse(e.data);
        if (d.status === "partial") msg = "Sync complete (some accounts had errors)";
      } catch { /* ignore */ }
      set({ type: "sync", state: "success", message: msg, timestamp: Date.now() });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["accounts"] });
      queryClient.invalidateQueries({ queryKey: ["sync-log"] });
    });

    es.addEventListener("sync_error", (e) => {
      let msg = "Sync failed";
      try { msg = `Sync failed: ${JSON.parse(e.data).error}`; } catch { /* ignore */ }
      set({ type: "sync", state: "error", message: msg, timestamp: Date.now() });
    });

    es.addEventListener("analysis_started", () => {
      set({ type: "analysis", state: "running", message: "Running analysis...", timestamp: Date.now() });
    });

    es.addEventListener("analysis_complete", () => {
      set({ type: "analysis", state: "success", message: "Analysis complete", timestamp: Date.now() });
      queryClient.invalidateQueries({ queryKey: ["analyses"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    });

    es.addEventListener("analysis_error", (e) => {
      let msg = "Analysis failed";
      try { msg = `Analysis failed: ${JSON.parse(e.data).error}`; } catch { /* ignore */ }
      set({ type: "analysis", state: "error", message: msg, timestamp: Date.now() });
    });

    es.addEventListener("categories_updated", () => {
      set({ type: "categorization", state: "success", message: "Categories updated", timestamp: Date.now() });
      queryClient.invalidateQueries({ queryKey: ["categories"] });
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    });

    es.addEventListener("budgets_updated", () => {
      queryClient.invalidateQueries({ queryKey: ["budgets"] });
    });

    es.onerror = () => { /* EventSource will auto-reconnect. */ };

    return () => {
      es.close();
      if (clearTimer.current) clearTimeout(clearTimer.current);
    };
  }, [queryClient, set]);

  return status;
}
