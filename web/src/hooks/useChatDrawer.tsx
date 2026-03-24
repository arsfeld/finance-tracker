import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
} from "react";

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

interface ChatState {
  messages: ChatMessage[];
  loading: boolean;
  send: (text: string) => void;
  clear: () => void;
}

interface ChatDrawerContextValue {
  isOpen: boolean;
  open: (message?: string) => void;
  close: () => void;
  chat: ChatState;
}

const ChatDrawerContext = createContext<ChatDrawerContextValue>(null!);

export const CHAT_STORAGE_KEY = "finance-chat-messages";

function loadMessages(): ChatMessage[] {
  try {
    const raw = localStorage.getItem(CHAT_STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

export function ChatDrawerProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>(loadMessages);
  const [loading, setLoading] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  // Persist messages to localStorage.
  useEffect(() => {
    localStorage.setItem(CHAT_STORAGE_KEY, JSON.stringify(messages));
  }, [messages]);

  const sendToAPI = useCallback(async (msgs: ChatMessage[]) => {
    // Cancel any in-flight request.
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    try {
      const res = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          messages: msgs.map((m) => ({ role: m.role, content: m.content })),
        }),
        signal: controller.signal,
      });
      const json = await res.json();
      if (json.data?.message) {
        setMessages([
          ...msgs,
          { role: "assistant", content: json.data.message },
        ]);
      } else if (json.error) {
        setMessages([
          ...msgs,
          { role: "assistant", content: `Error: ${json.error.message}` },
        ]);
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setMessages([
        ...msgs,
        {
          role: "assistant",
          content: `Error: ${err instanceof Error ? err.message : String(err)}`,
        },
      ]);
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false);
      }
    }
  }, []);

  const send = useCallback(
    (text: string) => {
      if (!text.trim() || loading) return;
      const userMsg: ChatMessage = { role: "user", content: text.trim() };
      const updated = [...messages, userMsg];
      setMessages(updated);
      sendToAPI(updated);
    },
    [messages, loading, sendToAPI]
  );

  const clear = useCallback(() => {
    abortRef.current?.abort();
    setMessages([]);
    setLoading(false);
    localStorage.removeItem(CHAT_STORAGE_KEY);
  }, []);

  const open = useCallback(
    (message?: string) => {
      setIsOpen(true);
      if (message) {
        // Context-specific: append and auto-send.
        const userMsg: ChatMessage = { role: "user", content: message };
        const updated = [...messages, userMsg];
        setMessages(updated);
        sendToAPI(updated);
      }
    },
    [messages, sendToAPI]
  );

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const chat: ChatState = { messages, loading, send, clear };

  return (
    <ChatDrawerContext.Provider value={{ isOpen, open, close, chat }}>
      {children}
      <ChatDrawerSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        chat={chat}
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
import { Button } from "@/components/ui/button";
import { ChatPanelView } from "@/components/ChatPanel";

function ChatDrawerSheet({
  isOpen,
  onOpenChange,
  chat,
}: {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  chat: ChatState;
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
            onClick={chat.clear}
          >
            New Chat
          </Button>
        </SheetHeader>
        <ChatPanelView
          messages={chat.messages}
          loading={chat.loading}
          onSend={chat.send}
          className="flex-1 min-h-0"
          placeholder="Ask about your finances..."
        />
      </SheetContent>
    </Sheet>
  );
}
