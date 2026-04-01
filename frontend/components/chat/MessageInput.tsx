"use client";

import { useState } from "react";
import { getSocketClient } from "@/lib/ws";

type MessageInputProps = {
  channelId: string;
};

export function MessageInput({ channelId }: MessageInputProps) {
  const [value, setValue] = useState("");

  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!value.trim()) return;

    const id = Number.parseInt(channelId, 10);
    if (!Number.isFinite(id) || id <= 0) return;

    getSocketClient().sendMessage(id, value);
    setValue("");
  };

  return (
    <form onSubmit={onSubmit} className="border-t border-zinc-800 p-4">
      <input
        value={value}
        onChange={(event) => setValue(event.target.value)}
        placeholder={`Message #${channelId}`}
        className="w-full rounded-md border border-zinc-700 bg-zinc-900 p-3 text-sm text-zinc-100"
      />
    </form>
  );
}
