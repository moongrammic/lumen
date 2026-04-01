"use client";

import Link from "next/link";
import { useGuildStore } from "@/store/useGuildStore";

export function ChannelList() {
  const channels = useGuildStore((state) => state.channels);

  return (
    <div className="p-4">
      <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-zinc-400">Channels</h2>
      <div className="space-y-1">
        {channels.map((channel) => (
          <Link
            key={channel.id}
            href={`/channels/${channel.id}`}
            className="block rounded-md px-2 py-1 text-sm text-zinc-200 hover:bg-zinc-800"
          >
            # {channel.name}
          </Link>
        ))}
      </div>
    </div>
  );
}
