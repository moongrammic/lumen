"use client";

import { MessageInput } from "@/components/chat/MessageInput";
import { MessageList } from "@/components/chat/MessageList";

type ChatAreaProps = {
  channelId: string;
};

export function ChatArea({ channelId }: ChatAreaProps) {
  return (
    <div className="flex h-screen flex-col">
      <header className="border-b border-zinc-800 px-4 py-3 text-sm font-medium"># {channelId}</header>
      <MessageList channelId={channelId} />
      <MessageInput channelId={channelId} />
    </div>
  );
}
