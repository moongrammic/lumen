"use client";

import { Message } from "@/components/chat/Message";
import { useChatStore } from "@/store/useChatStore";

type MessageListProps = {
  channelId: string;
};

export function MessageList({ channelId }: MessageListProps) {
  const messages = useChatStore((state) => state.messagesByChannel[channelId] ?? []);

  return (
    <div className="flex-1 space-y-1 overflow-y-auto p-4">
      {messages.map((message) => (
        <Message key={message.id} message={message} />
      ))}
    </div>
  );
}
