import { useState, useRef, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

interface ChatPanelProps {
  initialMessages?: ChatMessage[];
  storageKey?: string | null;
  className?: string;
  placeholder?: string;
  autoSendFirst?: boolean;
}

export function ChatPanel({
  initialMessages,
  storageKey = null,
  className = "",
  placeholder = "Ask about your finances...",
  autoSendFirst = false,
}: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>(() => {
    if (storageKey) {
      try {
        const raw = localStorage.getItem(storageKey);
        return raw ? JSON.parse(raw) : [];
      } catch {
        return [];
      }
    }
    return [];
  });
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const autoSentRef = useRef(false);

  // Persist messages on change (only if storageKey is set).
  useEffect(() => {
    if (storageKey) {
      localStorage.setItem(storageKey, JSON.stringify(messages));
    }
  }, [messages, storageKey]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  const sendMessages = async (msgs: ChatMessage[]) => {
    setLoading(true);
    try {
      const res = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          messages: msgs.map((m) => ({ role: m.role, content: m.content })),
        }),
      });
      const json = await res.json();
      if (json.data?.message) {
        setMessages([...msgs, { role: "assistant", content: json.data.message }]);
      } else if (json.error) {
        setMessages([...msgs, { role: "assistant", content: `Error: ${json.error.message}` }]);
      }
    } catch (err) {
      setMessages([...msgs, { role: "assistant", content: `Error: ${err}` }]);
    } finally {
      setLoading(false);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  };

  // Auto-send the first initial message if requested.
  useEffect(() => {
    if (
      autoSendFirst &&
      !autoSentRef.current &&
      initialMessages &&
      initialMessages.length > 0 &&
      messages.length === 0
    ) {
      autoSentRef.current = true;
      setMessages(initialMessages);
      sendMessages(initialMessages);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialMessages, autoSendFirst]);

  const send = async () => {
    const text = input.trim();
    if (!text || loading) return;

    const userMsg: ChatMessage = { role: "user", content: text };
    const newMessages = [...messages, userMsg];
    setMessages(newMessages);
    setInput("");
    sendMessages(newMessages);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  };

  return (
    <div className={`flex flex-col ${className}`}>
      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-3 space-y-3 py-3">
        {messages.length === 0 && !loading && (
          <div className="text-center text-muted-foreground py-8">
            <p className="text-sm">Ask me about your budgets and spending.</p>
          </div>
        )}

        {messages.map((msg, i) => (
          <div
            key={i}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <Card
              className={`max-w-[85%] px-3 py-2 ${
                msg.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-card"
              }`}
            >
              {msg.role === "assistant" ? (
                <MarkdownContent>{msg.content}</MarkdownContent>
              ) : (
                <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
              )}
            </Card>
          </div>
        ))}

        {loading && (
          <div className="flex justify-start">
            <Card className="px-3 py-2 bg-card">
              <div className="flex items-center gap-1.5">
                <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce [animation-delay:0ms]" />
                <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce [animation-delay:150ms]" />
                <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce [animation-delay:300ms]" />
              </div>
            </Card>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="border-t border-border p-3 bg-card">
        <div className="flex gap-2 items-end">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            className="flex-1 resize-none bg-background border border-input rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            style={{ minHeight: "40px", maxHeight: "120px" }}
            onInput={(e) => {
              const el = e.currentTarget;
              el.style.height = "auto";
              el.style.height = Math.min(el.scrollHeight, 120) + "px";
            }}
          />
          <Button onClick={send} disabled={loading || !input.trim()} size="sm">
            Send
          </Button>
        </div>
      </div>
    </div>
  );
}
