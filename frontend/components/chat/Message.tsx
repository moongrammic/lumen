import type { ChatMessage } from "@/types/backend";
import ReactMarkdown from "react-markdown";

type MessageProps = {
  message: ChatMessage;
};

export function Message({ message }: MessageProps) {
  return (
    <article className="rounded-md px-3 py-2 hover:bg-zinc-900/80">
      <p className="text-sm">
        <span className="font-semibold text-zinc-200">{message.author.username}</span>
      </p>
      <div className="mt-1 max-w-none text-sm text-zinc-100 [&_a]:text-indigo-400 [&_a]:underline [&_code]:rounded [&_code]:bg-zinc-800 [&_code]:px-1 [&_code]:text-[0.9em] [&_h1]:text-base [&_h2]:text-sm [&_li]:my-0.5 [&_ol]:my-1 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:my-1 [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:bg-zinc-900 [&_pre]:p-2 [&_ul]:my-1 [&_ul]:list-disc [&_ul]:pl-5">
        <ReactMarkdown
          components={{
            a: ({ href, children }) => (
              <a href={href} target="_blank" rel="noopener noreferrer">
                {children}
              </a>
            ),
          }}
        >
          {message.content}
        </ReactMarkdown>
      </div>
    </article>
  );
}
