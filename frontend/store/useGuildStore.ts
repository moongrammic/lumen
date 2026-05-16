"use client";

import { create } from "zustand";

export type Guild = { id: string; name: string; invite_code: string; owner_id: string };
export type Channel = { id: string; name: string; type?: string; guild_id?: number };

type GuildState = {
  guilds: Guild[];
  channels: Channel[];
  currentGuildId: string | null;
  setCurrentGuild: (guildId: string | null) => void;
  setGuilds: (guilds: Guild[]) => void;
  setChannels: (channels: Channel[]) => void;
};

export const useGuildStore = create<GuildState>((set) => ({
  guilds: [],
  channels: [],
  currentGuildId: null,
  setCurrentGuild: (guildId) => set({ currentGuildId: guildId }),
  setGuilds: (guilds) => set({ guilds }),
  setChannels: (channels) => set({ channels }),
}));
