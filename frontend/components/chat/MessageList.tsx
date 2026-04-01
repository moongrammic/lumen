"use client";

import { useQuery } from "@tanstack/react-query";
import { useLayoutEffect, useRef } from "react";
import { Message } from "@/components/chat/Message";
import { api } from "@/lib/api";
import { useAuthStore, type AuthState } from "@/store/useAuthStore";
import { useChatStore } from "@/store/useChatStore";
import type { ChatMessage } from "@/types/backend";

type MessageListProps = {
  channelId: string;
};

type MessagesApiResponse = {
  messages: ChatMessage[];
};

export function MessageList({ channelId }: MessageListProps) {
  const isAuthenticated = useAuthStore((s: AuthState) => s.isAuthenticated);
  const replaceChannelMessages = useChatStore((s) => s.replaceChannelMessages);
  const messages = useChatStore((s) => s.messagesByChannel[channelId] ?? []);
  const typingUserId = useChatStore((s) => s.typingByChannel[channelId] ?? null);
  const endRef = useRef<HTMLDivElement>(null);

  const numericId = Number.parseInt(channelId, 10);
  const validChannel = Number.isFinite(numericId) && numericId > 0;

  const query = useQuery({
    queryKey: ["channel-messages", channelId],
    enabled: isAuthenticated && validChannel,
    queryFn: async () => {
      const { data } = await api.get<MessagesApiResponse>(`/channels/${numericId}/messages`);
      return data.messages;
    },
  });

  const lastSynced = useRef<string>("");
  useLayoutEffect(() => {
    if (!query.data || !validChannel) return;
    const key = `${channelId}:${query.dataUpdatedAt}:${query.data.length}`;
    if (lastSynced.current === key) return;
    lastSynced.current = key;
    replaceChannelMessages(numericId, query.data);
  }, [query.data, query.dataUpdatedAt, channelId, numericId, validChannel, replaceChannelMessages]);

  useLayoutEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, query.isFetching]);

  if (!validChannel) {
    return (
      <div className="flex flex-1 items-center justify-center p-4 text-sm text-zinc-500">
        Некорректный id канала
      </div>
    );
  }

  if (query.isPending) {
    return (
      <div className="flex flex-1 flex-col gap-2 overflow-hidden p-4">
        <div className="h-4 w-2/3 animate-pulse rounded bg-zinc-800" />
        <div className="h-4 w-1/2 animate-pulse rounded bg-zinc-800" />
        <div className="h-4 w-3/4 animate-pulse rounded bg-zinc-800" />
        <p className="text-xs text-zinc-500">Загрузка сообщений…</p>
      </div>
    );
  }

  if (query.isError) {
    const err = query.error;
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-sm text-zinc-400">
        <p>{err instanceof Error ? err.message : "Не удалось загрузить сообщения"}</p>
        <button
          type="button"
          className="rounded-md bg-zinc-800 px-3 py-1.5 text-zinc-100 hover:bg-zinc-700"
          onClick={() => query.refetch()}
        >
          Повторить
        </button>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      {query.isFetching && !query.isPending ? (
        <div className="shrink-0 border-b border-zinc-800/80 px-4 py-1 text-xs text-zinc-500">Обновление…</div>
      ) : null}
      <div className="min-h-0 flex-1 space-y-1 overflow-y-auto p-4">
        {messages.map((message) => (
          <Message key={message.id} message={message} />
        ))}
        {typingUserId ? (
          <p className="px-3 text-xs italic text-zinc-500">Кто-то печатает…</p>
        ) : null}
        <div ref={endRef} />
      </div>
    </div>
  );
}
