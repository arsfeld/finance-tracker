---
title: "Chat drawer unmount destroys in-flight conversation state"
category: architecture-issues
tags:
  - react
  - context
  - state-management
  - component-lifecycle
  - abort-controller
  - drawer
module: web/chat
symptom: "Closing and reopening the AI chat drawer loses the in-flight AI response; old messages appear from localStorage but loading state is gone and the conversation cannot resume"
root_cause: "Chat state (messages, loading, active fetch) was owned by ChatPanel component mounted inside the drawer. Closing the drawer unmounted the component, destroying state and orphaning the API call."
---

## Problem Statement

The app has a global AI chat drawer (Sheet) accessible from every page. The drawer renders a ChatPanel that manages the conversation.

**What users experienced:**

1. Open chat drawer, send a message
2. While the AI is responding, close the drawer
3. Reopen — old messages appear (from localStorage) but the loading indicator is gone, the AI response never arrives, and the conversation is broken

**Why it happened:**

Closing the drawer unmounted `ChatPanel`. React destroyed all its state — `useState` for messages/loading, `useRef` for AbortController, and the closure over the active `fetch`. The API response had nowhere to land. On reopen, a fresh component rehydrated messages from localStorage but started with `loading: false` and no active connection.

A secondary issue: the drawer and `/chat` page were independent conversations with separate localStorage keys.

## Root Cause Analysis

Three interacting failure modes from one design flaw:

1. **Closing the drawer kills the in-flight fetch.** The `AbortController` ref and `setMessages`/`setLoading` callbacks belonged to the unmounted component. The fetch was orphaned.

2. **Reopening shows stale data without a loading indicator.** Messages were rehydrated from localStorage, but `loading` re-initialized to `false`. No spinner, no response, no indication of failure.

3. **Drawer and /chat page were independent.** Each surface had its own component-local state. A conversation started in the drawer didn't appear on `/chat`.

**Core design flaw:** Component-local state has a lifetime coupled to the component's mount/unmount cycle. State that must survive across mount boundaries cannot live inside the component that gets mounted and unmounted.

## Working Solution

Lift all chat state into a React context provider that sits above the drawer's mount/unmount boundary.

### Architecture

```
AppLayout (never unmounts)
  └── ChatDrawerProvider         ← owns messages, loading, fetch
        ├── children (routes)
        │     ├── /chat page     ← consumes context via useChatDrawer()
        │     ├── /budgets       ← can call open("Help with...")
        │     └── ...
        └── ChatDrawerSheet      ← rendered inside provider, always in tree
              └── ChatPanelView  ← pure render component, receives props
```

**Provider owns:**
- `messages: ChatMessage[]` — conversation history
- `loading: boolean` — whether a fetch is in progress
- `abortRef: MutableRefObject<AbortController | null>` — active request handle
- `send(text)` — appends user message and fires API call
- `clear()` — aborts in-flight request, resets everything
- `open(message?)` — opens drawer, optionally auto-sends a context message
- `close()` — closes drawer (does NOT abort fetch or clear state)

**ChatPanelView owns only:**
- `input: string` — the text being typed (ephemeral UI state)

### Critical behavioral distinction

```
close()  → setIsOpen(false)     // display concern only, fetch continues
clear()  → abort + reset        // explicit user intent to start over
```

Closing the drawer is not a destructive action. The fetch continues, the response lands in context state, and opening the drawer reveals the completed conversation.

### Key code pattern

```tsx
// Provider: state lives here, survives drawer close/open
export function ChatDrawerProvider({ children }) {
  const [messages, setMessages] = useState(loadFromStorage);
  const [loading, setLoading] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const sendToAPI = useCallback(async (msgs) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setLoading(true);
    try {
      const res = await fetch("/api/chat", { signal: controller.signal, ... });
      setMessages([...msgs, { role: "assistant", content: res.data.message }]);
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, []);
  // ...
}

// View: pure render, no fetch logic
function ChatPanelView({ messages, loading, onSend }) {
  const [input, setInput] = useState(""); // only local state
  return <div>...</div>;
}

// Consumer: three lines to get full chat
function ChatPage() {
  const { chat } = useChatDrawer();
  return <ChatPanelView messages={chat.messages} loading={chat.loading} onSend={chat.send} />;
}
```

## Prevention Strategies

### When to use context vs. component state

Use **component-local state** when ALL three hold:
- State is only relevant while the component is mounted
- No async operation depends on the state surviving unmount
- No other component needs to read/write this state

Use **context** when ANY of these is true:
- State must survive mount/unmount cycles (drawer open/close)
- An async operation is in flight and its response must be captured regardless of mount state
- Two or more components share the same live state
- State represents a session-level concept (conversation, auth) not a view-level concept (dropdown open)

**Rule of thumb:** If closing the UI should not cancel the work, the UI cannot own the state.

### Checklist for drawer/modal features with async operations

- [ ] Can the user close the container while an operation is in progress?
- [ ] If yes, should the operation cancel or complete in the background?
- [ ] Is there another place in the app that shows the same data?
- [ ] If the user reopens, should they see results of operations completed while closed?
- [ ] Am I using `useEffect` cleanup to abort a fetch that should keep running?

If any of the first four is yes, lift the state out of the component.

### AbortController rules

- Abort on **user intent** (New Chat, new request), not on unmount
- One controller per logical operation
- Always handle `AbortError` gracefully (`if (err.name === 'AbortError') return`)
- Store in a **ref**, not state (no re-renders needed)
- Clean up on unmount only at the provider level, never in individual consumers

## Related

- Commits: `7786c9a`, `0f83077`, `16c826b`
- Files: `web/src/hooks/useChatDrawer.tsx`, `web/src/components/ChatPanel.tsx`, `web/src/pages/Chat.tsx`
