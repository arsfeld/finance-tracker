import { Button } from "@/components/ui/button";
import { ChatPanelView } from "@/components/ChatPanel";
import { useChatDrawer } from "@/hooks/useChatDrawer";

export default function Chat() {
  const { chat } = useChatDrawer();

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
        <Button variant="outline" size="sm" onClick={chat.clear}>
          New Chat
        </Button>
      </div>

      <ChatPanelView
        messages={chat.messages}
        loading={chat.loading}
        onSend={chat.send}
        className="flex-1 min-h-0"
        placeholder="Ask about your finances..."
      />
    </div>
  );
}
