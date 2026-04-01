import type { ChatMessage } from "@/types/backend";

type MessageProps = {
  message: ChatMessage;
};

export function Message({ message }: MessageProps) {
  return (
    <article className="rounded-md px-3 py-2 hover:bg-zinc-900/80">
      <p className="text-sm">
        <span className="font-semibold text-zinc-200">{message.author.username}</span>
      </p>
      <p className="text-sm text-zinc-100">{message.content}</p>
    </article>
  );
}
