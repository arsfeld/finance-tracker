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

const STORAGE_KEY = "finance-chat-drawer-messages";

export function ChatDrawerProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [key, setKey] = useState(0);
  const [initialMessages, setInitialMessages] = useState<ChatMessage[]>([]);
  const [autoSend, setAutoSend] = useState(false);

  const open = useCallback((message?: string) => {
    if (message) {
      // Context-specific: start a new conversation with this message
      localStorage.removeItem(STORAGE_KEY);
      setInitialMessages([{ role: "user", content: message }]);
      setAutoSend(true);
      setKey((k) => k + 1);
    } else {
      // General: resume existing conversation
      setInitialMessages([]);
      setAutoSend(false);
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
        initialMessages={initialMessages}
        autoSend={autoSend}
      />
    </ChatDrawerContext.Provider>
  );
}

export function useChatDrawer() {
  return useContext(ChatDrawerContext);
}

// Lazy import to avoid circular deps — inline the Sheet here
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ChatPanel } from "@/components/ChatPanel";

function ChatDrawerSheet({
  isOpen,
  onOpenChange,
  chatKey,
  initialMessages,
  autoSend,
}: {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  chatKey: number;
  initialMessages: ChatMessage[];
  autoSend: boolean;
}) {
  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-[420px] sm:w-[420px] p-0 flex flex-col"
      >
        <SheetHeader className="p-4 border-b border-border">
          <SheetTitle>AI Assistant</SheetTitle>
        </SheetHeader>
        <ChatPanel
          key={chatKey}
          storageKey={STORAGE_KEY}
          initialMessages={initialMessages}
          autoSendFirst={autoSend}
          className="flex-1 min-h-0"
          placeholder="Ask about your finances..."
        />
      </SheetContent>
    </Sheet>
  );
}
