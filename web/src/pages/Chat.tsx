import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ChatPanel } from "@/components/ChatPanel";

const STORAGE_KEY = "finance-chat-messages";

const suggestions = [
  "What did I spend the most on this month?",
  "Show me my top 10 transactions",
  "Categorize my uncategorized transactions",
  "How much did I spend on dining?",
];

export default function Chat() {
  const [key, setKey] = useState(0);

  return (
    <div className="flex flex-col h-[calc(100dvh-3.5rem)]">
      <div className="px-2 py-3 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Chat</h1>
          <p className="text-sm text-muted-foreground">
            Ask about your spending, categorize transactions, manage budgets, or
            run analysis.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            localStorage.removeItem(STORAGE_KEY);
            setKey((k) => k + 1);
          }}
        >
          New Chat
        </Button>
      </div>

      <ChatPanel
        key={key}
        storageKey={STORAGE_KEY}
        className="flex-1 min-h-0"
        placeholder="Ask about your finances..."
      />
    </div>
  );
}
