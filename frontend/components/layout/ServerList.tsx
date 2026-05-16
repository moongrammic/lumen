"use client";

import { useLayoutEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useGuildStore, type Guild } from "@/store/useGuildStore";
import { useAuthStore } from "@/store/useAuthStore";

type GuildsApiResponse = {
  guilds: Guild[];
};

export function ServerList() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const guilds = useGuildStore((state) => state.guilds);
  const currentGuildId = useGuildStore((state) => state.currentGuildId);
  const setCurrentGuild = useGuildStore((state) => state.setCurrentGuild);
  const setGuilds = useGuildStore((state) => state.setGuilds);

  const query = useQuery({
    queryKey: ["guilds"],
    enabled: isAuthenticated,
    queryFn: async () => {
      const { data } = await api.get<GuildsApiResponse>("/guilds");
      return data.guilds;
    },
  });

  useLayoutEffect(() => {
    if (query.data) {
      setGuilds(query.data);
    }
  }, [query.data, setGuilds]);

  if (query.isPending) {
    return (
      <nav className="flex flex-col gap-3 border-r border-zinc-800 bg-zinc-900 p-3">
        <div className="h-12 w-12 animate-pulse rounded-2xl bg-zinc-800" />
        <div className="h-12 w-12 animate-pulse rounded-2xl bg-zinc-800" />
      </nav>
    );
  }

  if (query.isError) {
    return (
      <nav className="flex flex-col gap-3 border-r border-zinc-800 bg-zinc-900 p-3">
        <div className="text-xs text-red-500 text-center">Error</div>
      </nav>
    );
  }

  return (
    <nav className="flex flex-col gap-3 border-r border-zinc-800 bg-zinc-900 p-3">
      {guilds.map((guild) => (
        <button
          key={guild.id}
          type="button"
          onClick={() => setCurrentGuild(guild.id)}
          className={`h-12 w-12 rounded-2xl text-sm font-semibold transition-colors flex items-center justify-center ${
            currentGuildId === guild.id ? "bg-indigo-600 text-white" : "bg-zinc-800 text-zinc-200 hover:bg-indigo-500 hover:text-white"
          }`}
        >
          {guild.name.slice(0, 2).toUpperCase()}
        </button>
      ))}
    </nav>
  );
}
