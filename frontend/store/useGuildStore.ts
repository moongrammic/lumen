"use client";

import { create } from "zustand";

type Guild = { id: string; name: string };
type Channel = { id: string; name: string };

type GuildState = {
  guilds: Guild[];
  channels: Channel[];
  currentGuildId: string | null;
  setCurrentGuild: (guildId: string) => void;
};

export const useGuildStore = create<GuildState>((set) => ({
  guilds: [
    { id: "g-1", name: "Lumen" },
    { id: "g-2", name: "Team" },
  ],
  channels: [
    { id: "1", name: "general" },
    { id: "2", name: "random" },
  ],
  currentGuildId: "g-1",
  setCurrentGuild: (guildId) => set({ currentGuildId: guildId }),
}));
