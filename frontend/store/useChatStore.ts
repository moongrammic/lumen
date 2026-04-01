"use client";

import { create } from "zustand";
import type { ChatMessage } from "@/types/backend";

function channelKey(channelId: number): string {
  return String(channelId);
}

type ChatState = {
  messagesByChannel: Record<string, ChatMessage[]>;
  upsertMessage: (message: ChatMessage) => void;
  setTypingUser: (channelId: number, userId: string | null) => void;
  typingByChannel: Record<string, string | null>;
};

export const useChatStore = create<ChatState>((set) => ({
  messagesByChannel: {},
  typingByChannel: {},
  upsertMessage: (message) =>
    set((state) => {
      const key = channelKey(message.channel_id);
      const items = state.messagesByChannel[key] ?? [];
      const exists = items.some((item) => item.id === message.id);
      return {
        messagesByChannel: {
          ...state.messagesByChannel,
          [key]: exists ? items : [...items, message],
        },
      };
    }),
  setTypingUser: (channelId, userId) =>
    set((state) => ({
      typingByChannel: {
        ...state.typingByChannel,
        [channelKey(channelId)]: userId,
      },
    })),
}));
