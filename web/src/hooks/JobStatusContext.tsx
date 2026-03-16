import { createContext, useContext } from "react";
import { useSSE, type JobStatus } from "./useSSE";

const JobStatusContext = createContext<JobStatus>({
  type: "idle",
  state: "success",
  timestamp: 0,
});

export function JobStatusProvider({ children }: { children: React.ReactNode }) {
  const status = useSSE();
  return (
    <JobStatusContext.Provider value={status}>
      {children}
    </JobStatusContext.Provider>
  );
}

export function useJobStatus() {
  return useContext(JobStatusContext);
}
