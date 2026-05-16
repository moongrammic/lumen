"use client";

import { useLayoutEffect } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useGuildStore, type Channel } from "@/store/useGuildStore";
import { useAuthStore } from "@/store/useAuthStore";

type ChannelsApiResponse = {
  channels: Channel[];
};

export function ChannelList() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const channels = useGuildStore((state) => state.channels);
  const currentGuildId = useGuildStore((state) => state.currentGuildId);
  const setChannels = useGuildStore((state) => state.setChannels);

  const query = useQuery({
    queryKey: ["channels", currentGuildId],
    enabled: isAuthenticated && !!currentGuildId,
    queryFn: async () => {
      const { data } = await api.get<ChannelsApiResponse>(`/guilds/${currentGuildId}/channels`);
      return data.channels;
    },
  });

  useLayoutEffect(() => {
    if (query.data) {
      setChannels(query.data);
    }
  }, [query.data, setChannels]);

  if (!currentGuildId) {
    return null;
  }

  return (
    <div className="p-4">
      <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-zinc-400">Channels</h2>

      {query.isPending && (
        <div className="text-sm text-zinc-500">Loading channels...</div>
      )}

      {query.isError && (
        <div className="text-sm text-red-500">Error loading channels</div>
      )}

      {!query.isPending && !query.isError && (
        <div className="space-y-1">
          {channels.length === 0 ? (
            <div className="text-sm text-zinc-500 px-2">No channels</div>
          ) : (
            channels.map((channel) => (
              <Link
                key={channel.id}
                href={`/channels/${channel.id}`}
                className="block rounded-md px-2 py-1 text-sm text-zinc-200 hover:bg-zinc-800"
              >
                # {channel.name}
              </Link>
            ))
          )}
        </div>
      )}
    </div>
  );
}
