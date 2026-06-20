"use client";

import Link from "next/link";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useGuildStore } from "@/store/useGuildStore";
import { useAuthStore } from "@/store/useAuthStore";

function GuildIcon({ name, iconUrl }: { name: string; iconUrl: string }) {
  const [imageFailed, setImageFailed] = useState(false);
  const url = iconUrl.trim();

  if (url && !imageFailed) {
    return (
      <img
        src={url}
        alt=""
        className="h-full w-full rounded-full object-cover"
        onError={() => setImageFailed(true)}
      />
    );
  }

  return (
    <span className="flex h-full w-full items-center justify-center text-sm font-bold">
      {name.slice(0, 2).toUpperCase()}
    </span>
  );
}

export function ServerList() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const guilds = useGuildStore((state) => state.guilds);
  const currentGuildId = useGuildStore((state) => state.currentGuildId);
  const fetchGuilds = useGuildStore((state) => state.fetchGuilds);
  const fetchChannels = useGuildStore((state) => state.fetchChannels);
  const createGuild = useGuildStore((state) => state.createGuild);
  const setCurrentGuild = useGuildStore((state) => state.setCurrentGuild);

  useEffect(() => {
    if (isAuthenticated) {
      void fetchGuilds();
    }
  }, [fetchGuilds, isAuthenticated]);

  const openGuild = async (guildId: string) => {
    setCurrentGuild(guildId);
    const channels = await fetchChannels(guildId);
    const firstText = channels.find((channel) => channel.type === "text") ?? channels[0];
    if (firstText) {
      router.push(`/channels/${firstText.id}`);
    } else {
      router.push("/guilds");
    }
  };

  const handleCreateGuild = async () => {
    const name = prompt("Название сервера:");
    if (!name?.trim()) return;

    const trimmed = name.trim();
    if (trimmed.length < 3 || trimmed.length > 32) {
      alert("Название сервера: от 3 до 32 символов");
      return;
    }

    const result = await createGuild(trimmed);
    if (result?.firstChannelId) {
      router.push(`/channels/${result.firstChannelId}`);
    }
  };

  return (
    <nav className="flex w-20 flex-col items-center gap-2 overflow-y-auto bg-[#202225] py-3">
      <Link
        href="/guilds"
        className="flex h-12 w-12 cursor-pointer items-center justify-center rounded-full bg-[#36393f] transition-all hover:rounded-2xl"
        title="Home"
      >
        🏠
      </Link>

      <div className="my-2 h-[2px] w-8 bg-[#292b2f]" />

      {guilds.map((guild) => (
        <button
          key={guild.id}
          type="button"
          onClick={() => void openGuild(guild.id)}
          className={`flex h-12 w-12 cursor-pointer items-center justify-center overflow-hidden rounded-full text-white transition-all hover:rounded-2xl ${
            currentGuildId === guild.id ? "rounded-2xl bg-[#5865f2]" : "bg-[#5865f2]/80 hover:bg-[#5865f2]"
          }`}
          title={guild.name}
        >
          <GuildIcon name={guild.name} iconUrl={guild.icon_url} />
        </button>
      ))}

      <button
        type="button"
        onClick={() => void handleCreateGuild()}
        className="mt-2 flex h-12 w-12 cursor-pointer items-center justify-center rounded-full bg-[#3ba55c] text-white transition-all hover:rounded-2xl hover:bg-[#3ba55c]/80"
        title="Create server"
      >
        <Plus size={28} />
      </button>
    </nav>
  );
}
