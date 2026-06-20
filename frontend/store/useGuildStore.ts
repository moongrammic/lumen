"use client";



import { create } from "zustand";

import { toast } from "sonner";

import { api } from "@/lib/api";

import { getApiErrorMessage } from "@/lib/api-error";



export type Guild = {

  id: string;

  name: string;

  icon_url: string;

  description: string;

  invite_code: string;

  owner_id: string;

};



export type Channel = { id: string; name: string; type: string; guild_id: string };



export type GuildMember = {

  user_id: string;

  username: string;

  role: string;

  permissions: number;

  is_owner: boolean;

};



type GuildApi = Omit<Guild, "id"> & { id: number | string };

type ChannelApi = Omit<Channel, "id" | "guild_id"> & { id: number | string; guild_id: number | string };

type MemberApi = Omit<GuildMember, "user_id"> & { user_id: string };



function normalizeGuild(raw: GuildApi): Guild {

  return {

    ...raw,

    id: String(raw.id),

    icon_url: raw.icon_url ?? "",

    description: raw.description ?? "",

  };

}



function normalizeChannel(raw: ChannelApi): Channel {

  return {

    id: String(raw.id),

    name: raw.name,

    type: raw.type ?? "text",

    guild_id: String(raw.guild_id),

  };

}



function normalizeMember(raw: MemberApi): GuildMember {

  return {

    user_id: String(raw.user_id),

    username: raw.username,

    role: raw.role,

    permissions: raw.permissions,

    is_owner: raw.is_owner,

  };

}



type CreateGuildResult = {

  guild: Guild;

  firstChannelId: string | null;

};



type UpdateGuildInput = {

  name?: string;

  icon_url?: string;

  description?: string;

};



type UpdateChannelInput = {

  name?: string;

  type?: "text" | "voice";

};



type GuildState = {

  guilds: Guild[];

  channels: Channel[];

  currentGuildId: string | null;

  setCurrentGuild: (guildId: string | null) => void;

  setGuilds: (guilds: Guild[]) => void;

  setChannels: (channels: Channel[]) => void;

  fetchGuilds: () => Promise<Guild[]>;

  /** Pure read: returns a guild's channels WITHOUT mutating store state. */

  fetchChannels: (guildId: string) => Promise<Channel[]>;

  createGuild: (name: string) => Promise<CreateGuildResult | null>;

  updateGuild: (guildId: string, input: UpdateGuildInput) => Promise<Guild | null>;

  deleteGuild: (guildId: string) => Promise<boolean>;

  fetchMembers: (guildId: string) => Promise<GuildMember[]>;

  addMember: (guildId: string, username: string) => Promise<GuildMember | null>;

  removeMember: (guildId: string, userId: string) => Promise<boolean>;

  createChannel: (guildId: string, name: string, type: "text" | "voice") => Promise<Channel | null>;

  updateChannel: (guildId: string, channelId: string, input: UpdateChannelInput) => Promise<Channel | null>;

  deleteChannel: (guildId: string, channelId: string) => Promise<boolean>;

};



export const useGuildStore = create<GuildState>((set, get) => ({

  guilds: [],

  channels: [],

  currentGuildId: null,

  setCurrentGuild: (guildId) =>

    set((state) => (state.currentGuildId === guildId ? state : { currentGuildId: guildId, channels: [] })),

  setGuilds: (guilds) => set({ guilds }),

  setChannels: (channels) => set({ channels }),

  fetchGuilds: async () => {

    try {

      const { data } = await api.get<{ guilds: GuildApi[] }>("/guilds");

      const guilds = data.guilds.map(normalizeGuild);

      set({ guilds });

      return guilds;

    } catch {

      toast.error("Failed to load servers");

      return [];

    }

  },

  fetchChannels: async (guildId) => {

    try {

      const { data } = await api.get<{ channels: ChannelApi[] }>(`/guilds/${guildId}/channels`);

      return data.channels.map(normalizeChannel);

    } catch {

      toast.error("Failed to load channels");

      return [];

    }

  },

  createGuild: async (name) => {

    try {

      const { data } = await api.post<GuildApi>("/guilds", { name });

      const guild = normalizeGuild(data);

      set((state) => ({ guilds: [...state.guilds, guild], currentGuildId: guild.id, channels: [] }));

      toast.success("Server created");

      const channels = await get().fetchChannels(guild.id);

      const firstText = channels.find((channel) => channel.type === "text") ?? channels[0];

      return { guild, firstChannelId: firstText?.id ?? null };

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to create server"));

      return null;

    }

  },

  updateGuild: async (guildId, input) => {

    try {

      const { data } = await api.patch<GuildApi>(`/guilds/${guildId}`, input);

      const guild = normalizeGuild(data);

      set((state) => ({

        guilds: state.guilds.map((g) => (g.id === guildId ? guild : g)),

      }));

      toast.success("Server updated");

      return guild;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to update server"));

      return null;

    }

  },

  deleteGuild: async (guildId) => {

    try {

      await api.delete(`/guilds/${guildId}`);

      set((state) => ({

        guilds: state.guilds.filter((g) => g.id !== guildId),

        currentGuildId: state.currentGuildId === guildId ? null : state.currentGuildId,

        channels: state.currentGuildId === guildId ? [] : state.channels,

      }));

      toast.success("Server deleted");

      return true;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to delete server"));

      return false;

    }

  },

  fetchMembers: async (guildId) => {

    try {

      const { data } = await api.get<{ members: MemberApi[] }>(`/guilds/${guildId}/members`);

      return data.members.map(normalizeMember);

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to load members"));

      return [];

    }

  },

  addMember: async (guildId, username) => {

    try {

      const { data } = await api.post<MemberApi>(`/guilds/${guildId}/members`, { username });

      toast.success("Member added");

      return normalizeMember(data);

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to add member"));

      return null;

    }

  },

  removeMember: async (guildId, userId) => {

    try {

      await api.delete(`/guilds/${guildId}/members/${userId}`);

      toast.success("Member removed");

      return true;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to remove member"));

      return false;

    }

  },

  createChannel: async (guildId, name, type) => {

    try {

      const { data } = await api.post<ChannelApi>(`/guilds/${guildId}/channels`, { name, type });

      const channel = normalizeChannel(data);

      set((state) => {

        if (state.currentGuildId !== guildId) return state;

        return { channels: [...state.channels, channel] };

      });

      toast.success(type === "voice" ? "Voice channel created" : "Channel created");

      return channel;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to create channel"));

      return null;

    }

  },

  updateChannel: async (guildId, channelId, input) => {

    try {

      const { data } = await api.patch<ChannelApi>(`/guilds/${guildId}/channels/${channelId}`, input);

      const channel = normalizeChannel(data);

      set((state) => {

        if (state.currentGuildId !== guildId) return state;

        return { channels: state.channels.map((c) => (c.id === channelId ? channel : c)) };

      });

      toast.success("Channel updated");

      return channel;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to update channel"));

      return null;

    }

  },

  deleteChannel: async (guildId, channelId) => {

    try {

      await api.delete(`/guilds/${guildId}/channels/${channelId}`);

      set((state) => {

        if (state.currentGuildId !== guildId) return state;

        return { channels: state.channels.filter((c) => c.id !== channelId) };

      });

      toast.success("Channel deleted");

      return true;

    } catch (error) {

      toast.error(getApiErrorMessage(error, "Failed to delete channel"));

      return false;

    }

  },

}));


