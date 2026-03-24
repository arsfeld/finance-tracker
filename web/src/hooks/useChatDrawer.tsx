import { createContext, useContext, useState, useCallback } from "react";
import type { ChatMessage } from "@/components/ChatPanel";

interface ChatDrawerContextValue {
  isOpen: boolean;
  open: (message?: string) => void;
  close: () => void;
}

const ChatDrawerContext = createContext<ChatDrawerContextValue>({
  isOpen: false,
  open: () => {},
  close: () => {},
});

// Shared with /chat page — one conversation everywhere.
export const CHAT_STORAGE_KEY = "finance-chat-messages";

export function ChatDrawerProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [key, setKey] = useState(0);
  const [pendingMessage, setPendingMessage] = useState<string | null>(null);

  const open = useCallback((message?: string) => {
    if (message) {
      // Context-specific: append this message to existing conversation and auto-send
      setPendingMessage(message);
      setKey((k) => k + 1);
    }
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  return (
    <ChatDrawerContext.Provider value={{ isOpen, open, close }}>
      {children}
      <ChatDrawerSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        chatKey={key}
        pendingMessage={pendingMessage}
        onPendingConsumed={() => setPendingMessage(null)}
      />
    </ChatDrawerContext.Provider>
  );
}

export function useChatDrawer() {
  return useContext(ChatDrawerContext);
}

import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ChatPanel } from "@/components/ChatPanel";
import { Button } from "@/components/ui/button";

function ChatDrawerSheet({
  isOpen,
  onOpenChange,
  chatKey,
  pendingMessage,
  onPendingConsumed,
}: {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  chatKey: number;
  pendingMessage: string | null;
  onPendingConsumed: () => void;
}) {
  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-[420px] sm:w-[420px] p-0 flex flex-col"
      >
        <SheetHeader className="p-4 border-b border-border flex flex-row items-center justify-between">
          <SheetTitle>AI Assistant</SheetTitle>
          <Button
            variant="ghost"
            size="sm"
            className="text-xs"
            onClick={() => {
              localStorage.removeItem(CHAT_STORAGE_KEY);
              onOpenChange(true);
              onPendingConsumed();
              // Force remount by closing and reopening
              onOpenChange(false);
              setTimeout(() => onOpenChange(true), 0);
            }}
          >
            New Chat
          </Button>
        </SheetHeader>
        <ChatPanel
          key={chatKey}
          storageKey={CHAT_STORAGE_KEY}
          pendingMessage={pendingMessage}
          onPendingConsumed={onPendingConsumed}
          className="flex-1 min-h-0"
          placeholder="Ask about your finances..."
        />
      </SheetContent>
    </Sheet>
  );
}
