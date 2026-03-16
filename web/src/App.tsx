import { Routes, Route } from "react-router";
import { AppLayout } from "@/components/layout/AppLayout";
import { JobStatusProvider } from "@/hooks/JobStatusContext";
import Dashboard from "@/pages/Dashboard";
import Transactions from "@/pages/Transactions";
import Analysis from "@/pages/Analysis";
import Settings from "@/pages/Settings";

export default function App() {
  return (
    <JobStatusProvider>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="transactions" element={<Transactions />} />
          <Route path="analysis" element={<Analysis />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </JobStatusProvider>
  );
}
