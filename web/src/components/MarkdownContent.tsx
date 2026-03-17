import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkBreaks from "remark-breaks";

export function MarkdownContent({ children, className = "" }: { children: string; className?: string }) {
  return (
    <div className={`prose prose-sm dark:prose-invert max-w-none
      prose-table:w-full prose-table:border-collapse
      prose-th:border prose-th:border-border prose-th:px-3 prose-th:py-1.5 prose-th:bg-muted prose-th:text-left prose-th:text-xs prose-th:font-medium
      prose-td:border prose-td:border-border prose-td:px-3 prose-td:py-1.5 prose-td:text-sm
      prose-p:my-2 prose-ul:my-2 prose-ol:my-2 prose-li:my-0.5
      prose-headings:mt-4 prose-headings:mb-2
      prose-hr:my-3
      [&>*:first-child]:mt-0 [&>*:last-child]:mb-0
      ${className}`}
    >
      <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
        {children}
      </Markdown>
    </div>
  );
}
