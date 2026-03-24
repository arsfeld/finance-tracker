import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ChatPanel } from "@/components/ChatPanel";
import { CHAT_STORAGE_KEY } from "@/hooks/useChatDrawer";

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
            localStorage.removeItem(CHAT_STORAGE_KEY);
            setKey((k) => k + 1);
          }}
        >
          New Chat
        </Button>
      </div>

      <ChatPanel
        key={key}
        storageKey={CHAT_STORAGE_KEY}
        className="flex-1 min-h-0"
        placeholder="Ask about your finances..."
      />
    </div>
  );
}
