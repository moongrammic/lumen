"use client";

import { useGuildStore } from "@/store/useGuildStore";

export function ServerList() {
  const guilds = useGuildStore((state) => state.guilds);
  const currentGuildId = useGuildStore((state) => state.currentGuildId);
  const setCurrentGuild = useGuildStore((state) => state.setCurrentGuild);

  return (
    <nav className="flex flex-col gap-3 border-r border-zinc-800 bg-zinc-900 p-3">
      {guilds.map((guild) => (
        <button
          key={guild.id}
          type="button"
          onClick={() => setCurrentGuild(guild.id)}
          className={`h-12 rounded-2xl text-sm font-semibold ${
            currentGuildId === guild.id ? "bg-indigo-600 text-white" : "bg-zinc-800 text-zinc-200"
          }`}
        >
          {guild.name.slice(0, 2).toUpperCase()}
        </button>
      ))}
    </nav>
  );
}
