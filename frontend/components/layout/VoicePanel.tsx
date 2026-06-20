"use client";



import { Pencil, Plus, Trash2, Volume2 } from "lucide-react";

import { useGuildStore, type Channel } from "@/store/useGuildStore";



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



export function VoicePanel() {

  const currentGuildId = useGuildStore((s) => s.currentGuildId);

  const channels = useGuildStore((s) => s.channels);

  const createChannel = useGuildStore((s) => s.createChannel);

  const updateChannel = useGuildStore((s) => s.updateChannel);

  const deleteChannel = useGuildStore((s) => s.deleteChannel);



  if (!currentGuildId) {

    return null;

  }



  const voiceChannels = channels.filter((channel) => channel.type === "voice");



  const handleCreateVoiceChannel = async () => {

    const name = promptChannelName("Название голосового канала:");

    if (!name) return;

    await createChannel(currentGuildId, name, "voice");

  };



  const handleRenameChannel = async (channel: Channel) => {

    const name = promptChannelName("Новое название канала:", channel.name);

    if (!name || name === channel.name) return;

    await updateChannel(currentGuildId, channel.id, { name });

  };



  const handleDeleteChannel = async (channel: Channel) => {

    const confirmed = confirm(`Удалить голосовой канал «${channel.name}»?`);

    if (!confirmed) return;

    await deleteChannel(currentGuildId, channel.id);

  };



  return (

    <div className="border-t border-zinc-800 p-4">

      <div className="mb-2 flex items-center justify-between">

        <p className="text-xs font-semibold uppercase tracking-wide text-zinc-400">Voice Channels</p>

        <button

          type="button"

          onClick={() => void handleCreateVoiceChannel()}

          className="rounded p-0.5 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"

          title="Create voice channel"

        >

          <Plus size={16} />

        </button>

      </div>



      {voiceChannels.length === 0 ? (

        <p className="px-2 text-sm text-zinc-500">No voice channels</p>

      ) : (

        <div className="space-y-0.5">

          {voiceChannels.map((channel) => (

            <div

              key={channel.id}

              className="group flex items-center gap-0.5 rounded-md hover:bg-zinc-800/80"

            >

              <button

                type="button"

                className="flex min-w-0 flex-1 items-center gap-1.5 px-2 py-1 text-left text-sm text-zinc-300"

                title="LiveKit integration placeholder"

              >

                <Volume2 size={16} className="shrink-0 text-zinc-500" />

                <span className="truncate">{channel.name}</span>

              </button>

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

          ))}

        </div>

      )}

    </div>

  );

}


