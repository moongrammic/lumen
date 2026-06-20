"use client";



import { useLayoutEffect, useMemo, useState } from "react";

import Link from "next/link";

import { useParams, useRouter } from "next/navigation";

import { useQuery, useQueryClient } from "@tanstack/react-query";

import { Hash, Pencil, Plus, Settings, Trash2 } from "lucide-react";

import { api } from "@/lib/api";

import { useGuildStore, type Channel } from "@/store/useGuildStore";

import { useAuthStore } from "@/store/useAuthStore";

import { GuildSettingsModal } from "@/components/layout/GuildSettingsModal";



type ChannelApi = Omit<Channel, "id" | "guild_id"> & { id: number | string; guild_id: number | string };



function normalizeChannel(raw: ChannelApi): Channel {

  return {

    id: String(raw.id),

    name: raw.name,

    type: raw.type ?? "text",

    guild_id: String(raw.guild_id),

  };

}



function promptChannelName(label: string, initial = ""): string | null {

  const value = prompt(label, initial);

  if (value === null) return null;

  const trimmed = value.trim();

  if (trimmed.length < 1 || trimmed.length > 64) {

    alert("Название канала: от 1 до 64 символов");

    return null;

  }

  return trimmed;

}



export function ChannelList() {

  const router = useRouter();

  const params = useParams<{ channelId?: string }>();

  const activeChannelId = params.channelId ? String(params.channelId) : null;

  const queryClient = useQueryClient();



  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const channels = useGuildStore((state) => state.channels);

  const guilds = useGuildStore((state) => state.guilds);

  const currentGuildId = useGuildStore((state) => state.currentGuildId);

  const setChannels = useGuildStore((state) => state.setChannels);

  const createChannel = useGuildStore((state) => state.createChannel);

  const updateChannel = useGuildStore((state) => state.updateChannel);

  const deleteChannel = useGuildStore((state) => state.deleteChannel);



  const [settingsOpen, setSettingsOpen] = useState(false);



  const currentGuild = useMemo(

    () => guilds.find((g) => g.id === currentGuildId) ?? null,

    [guilds, currentGuildId],

  );



  const query = useQuery({

    queryKey: ["channels", currentGuildId],

    enabled: isAuthenticated && !!currentGuildId,

    queryFn: async () => {

      const { data } = await api.get<{ channels: ChannelApi[] }>(`/guilds/${currentGuildId}/channels`);

      return data.channels.map(normalizeChannel);

    },

  });



  useLayoutEffect(() => {

    if (query.data) {

      setChannels(query.data);

    }

  }, [query.data, setChannels]);



  if (!currentGuildId || !currentGuild) {

    return null;

  }



  const textChannels = channels.filter((channel) => channel.type === "text");



  const invalidateChannels = () => {

    void queryClient.invalidateQueries({ queryKey: ["channels", currentGuildId] });

  };



  const handleCreateTextChannel = async () => {

    const name = promptChannelName("Название текстового канала:");

    if (!name) return;

    const channel = await createChannel(currentGuildId, name, "text");

    if (channel) {

      invalidateChannels();

      router.push(`/channels/${channel.id}`);

    }

  };



  const handleRenameChannel = async (channel: Channel) => {

    const name = promptChannelName("Новое название канала:", channel.name);

    if (!name || name === channel.name) return;

    const updated = await updateChannel(currentGuildId, channel.id, { name });

    if (updated) invalidateChannels();

  };



  const handleDeleteChannel = async (channel: Channel) => {

    const confirmed = confirm(`Удалить канал #${channel.name}?`);

    if (!confirmed) return;



    const ok = await deleteChannel(currentGuildId, channel.id);

    if (!ok) return;



    invalidateChannels();

    if (activeChannelId === channel.id) {

      const remaining = textChannels.filter((c) => c.id !== channel.id);

      if (remaining[0]) {

        router.push(`/channels/${remaining[0].id}`);

      } else {

        router.push("/guilds");

      }

    }

  };



  return (

    <>

      <div className="border-b border-zinc-800 px-4 py-3">

        <div className="flex items-center justify-between gap-2">

          <h2 className="truncate text-sm font-semibold text-zinc-100" title={currentGuild.name}>

            {currentGuild.name}

          </h2>

          <button

            type="button"

            onClick={() => setSettingsOpen(true)}

            className="shrink-0 rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"

            title="Server settings"

          >

            <Settings size={18} />

          </button>

        </div>

        {currentGuild.description && (

          <p className="mt-1 truncate text-xs text-zinc-500">{currentGuild.description}</p>

        )}

      </div>



      <div className="p-4">

        <div className="mb-2 flex items-center justify-between">

          <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-400">Text Channels</h3>

          <button

            type="button"

            onClick={() => void handleCreateTextChannel()}

            className="rounded p-0.5 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"

            title="Create text channel"

          >

            <Plus size={16} />

          </button>

        </div>



        {query.isPending && <div className="text-sm text-zinc-500">Loading channels...</div>}



        {query.isError && <div className="text-sm text-red-500">Error loading channels</div>}



        {!query.isPending && !query.isError && (

          <div className="space-y-0.5">

            {textChannels.length === 0 ? (

              <div className="px-2 text-sm text-zinc-500">No text channels</div>

            ) : (

              textChannels.map((channel) => (

                <div

                  key={channel.id}

                  className="group flex items-center gap-0.5 rounded-md hover:bg-zinc-800/80"

                >

                  <Link

                    href={`/channels/${channel.id}`}

                    className={`flex min-w-0 flex-1 items-center gap-1.5 px-2 py-1 text-sm ${

                      activeChannelId === channel.id ? "bg-zinc-800 text-zinc-100" : "text-zinc-300"

                    }`}

                  >

                    <Hash size={16} className="shrink-0 text-zinc-500" />

                    <span className="truncate">{channel.name}</span>

                  </Link>

                  <div className="hidden shrink-0 items-center pr-1 group-hover:flex">

                    <button

                      type="button"

                      onClick={() => void handleRenameChannel(channel)}

                      className="rounded p-1 text-zinc-400 hover:text-zinc-100"

                      title="Rename"

                    >

                      <Pencil size={14} />

                    </button>

                    <button

                      type="button"

                      onClick={() => void handleDeleteChannel(channel)}

                      className="rounded p-1 text-zinc-400 hover:text-red-400"

                      title="Delete"

                    >

                      <Trash2 size={14} />

                    </button>

                  </div>

                </div>

              ))

            )}

          </div>

        )}

      </div>



      <GuildSettingsModal guild={currentGuild} open={settingsOpen} onClose={() => setSettingsOpen(false)} />

    </>

  );

}


