"use client";

import { create } from "zustand";
import type { ChatMessage } from "@/types/backend";

function channelKey(channelId: number): string {
  return String(channelId);
}

const typingTimers = new Map<string, ReturnType<typeof setTimeout>>();

type ChatState = {
  messagesByChannel: Record<string, ChatMessage[]>;
  upsertMessage: (message: ChatMessage) => void;
  replaceChannelMessages: (channelId: number, messages: ChatMessage[]) => void;
  setTypingUser: (channelId: number, userId: string | null) => void;
  typingByChannel: Record<string, string | null>;
};

export const useChatStore = create<ChatState>((set) => ({
  messagesByChannel: {},
  typingByChannel: {},
  replaceChannelMessages: (channelId, messages) =>
    set((state) => ({
      messagesByChannel: {
        ...state.messagesByChannel,
        [channelKey(channelId)]: messages,
      },
    })),
  upsertMessage: (message) =>
    set((state) => {
      const key = channelKey(message.channel_id);
      const items = state.messagesByChannel[key] ?? [];
      const index = items.findIndex((item) => item.id === message.id);
      if (index >= 0) {
        const next = [...items];
        next[index] = message;
        return { messagesByChannel: { ...state.messagesByChannel, [key]: next } };
      }
      return {
        messagesByChannel: {
          ...state.messagesByChannel,
          [key]: [...items, message],
        },
      };
    }),
  setTypingUser: (channelId, userId) => {
    const key = channelKey(channelId);
    const existing = typingTimers.get(key);
    if (existing) clearTimeout(existing);

    set((state) => ({
      typingByChannel: {
        ...state.typingByChannel,
        [key]: userId,
      },
    }));

    if (userId) {
      typingTimers.set(
        key,
        setTimeout(() => {
          typingTimers.delete(key);
          set((state) => ({
            typingByChannel: {
              ...state.typingByChannel,
              [key]: null,
            },
          }));
        }, 5000),
      );
    }
  },
}));
