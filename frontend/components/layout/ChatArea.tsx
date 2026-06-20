"use client";

import { useEffect, useRef } from "react";
import { ChatErrorBoundary } from "@/components/chat/ChatErrorBoundary";
import { MessageInput } from "@/components/chat/MessageInput";
import { MessageList } from "@/components/chat/MessageList";
import { useGuildStore } from "@/store/useGuildStore";
import { useAuthStore } from "@/store/useAuthStore";

type ChatAreaProps = {
  channelId: string;
};

export function ChatArea({ channelId }: ChatAreaProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const channel = useGuildStore((state) => state.channels.find((item) => item.id === channelId));
  const title = channel?.name ?? channelId;

  // Resolve which guild owns this channel exactly ONCE per channelId.
  // Uses getState() instead of reactive deps so it cannot self-trigger a render/effect loop.
  const resolvedForRef = useRef<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) return;
    if (resolvedForRef.current === channelId) return;
    resolvedForRef.current = channelId;

    const store = useGuildStore.getState();

    const known = store.channels.find((item) => item.id === channelId);
    if (known) {
      if (store.currentGuildId !== known.guild_id) {
        store.setCurrentGuild(known.guild_id);
      }
      return;
    }

    // Already inside a guild context: ChannelList will load its channels.
    if (store.currentGuildId) return;

    // Deep link without context: resolve the owning guild a single time.
    void (async () => {
      let guildList = useGuildStore.getState().guilds;
      if (guildList.length === 0) {
        guildList = await useGuildStore.getState().fetchGuilds();
      }
      for (const guild of guildList) {
        const channels = await useGuildStore.getState().fetchChannels(guild.id);
        if (channels.some((item) => item.id === channelId)) {
          useGuildStore.getState().setCurrentGuild(guild.id);
          return;
        }
      }
    })();
  }, [channelId, isAuthenticated]);

  return (
    <div className="flex h-screen flex-col">
      <header className="border-b border-zinc-800 px-4 py-3 text-sm font-medium"># {title}</header>
      <ChatErrorBoundary>
        <MessageList channelId={channelId} />
      </ChatErrorBoundary>
      <MessageInput channelId={channelId} />
    </div>
  );
}
