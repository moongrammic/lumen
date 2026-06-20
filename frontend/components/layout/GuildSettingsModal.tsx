"use client";



import { useEffect, useState } from "react";

import { useRouter } from "next/navigation";

import { X, Trash2, UserMinus, UserPlus } from "lucide-react";

import { useGuildStore, type Guild, type GuildMember } from "@/store/useGuildStore";

import { useAuthStore } from "@/store/useAuthStore";

import { Button } from "@/components/ui/button";



type Tab = "overview" | "members";



type GuildSettingsModalProps = {

  guild: Guild;

  open: boolean;

  onClose: () => void;

};



export function GuildSettingsModal({ guild, open, onClose }: GuildSettingsModalProps) {

  const router = useRouter();

  const userId = useAuthStore((s) => s.userId);

  const updateGuild = useGuildStore((s) => s.updateGuild);

  const deleteGuild = useGuildStore((s) => s.deleteGuild);

  const fetchMembers = useGuildStore((s) => s.fetchMembers);

  const addMember = useGuildStore((s) => s.addMember);

  const removeMember = useGuildStore((s) => s.removeMember);



  const [tab, setTab] = useState<Tab>("overview");

  const [name, setName] = useState(guild.name);

  const [iconUrl, setIconUrl] = useState(guild.icon_url);

  const [description, setDescription] = useState(guild.description);

  const [saving, setSaving] = useState(false);

  const [deleting, setDeleting] = useState(false);



  const [members, setMembers] = useState<GuildMember[]>([]);

  const [membersLoading, setMembersLoading] = useState(false);

  const [newUsername, setNewUsername] = useState("");

  const [addingMember, setAddingMember] = useState(false);



  const isOwner = userId === guild.owner_id;



  useEffect(() => {

    if (!open) return;

    setTab("overview");

    setName(guild.name);

    setIconUrl(guild.icon_url);

    setDescription(guild.description);

    setNewUsername("");

  }, [open, guild]);



  useEffect(() => {

    if (!open || tab !== "members") return;



    let cancelled = false;

    setMembersLoading(true);

    void fetchMembers(guild.id).then((list) => {

      if (!cancelled) {

        setMembers(list);

        setMembersLoading(false);

      }

    });



    return () => {

      cancelled = true;

    };

  }, [open, tab, guild.id, fetchMembers]);



  if (!open) return null;



  const handleSaveOverview = async () => {

    const trimmedName = name.trim();

    if (trimmedName.length < 3 || trimmedName.length > 32) {

      alert("Название сервера: от 3 до 32 символов");

      return;

    }



    setSaving(true);

    const updated = await updateGuild(guild.id, {

      name: trimmedName,

      icon_url: iconUrl.trim(),

      description: description.trim(),

    });

    setSaving(false);

    if (updated) {

      onClose();

    }

  };



  const handleDeleteGuild = async () => {

    if (!isOwner) return;

    const confirmed = confirm(

      `Удалить сервер «${guild.name}»? Это действие необратимо.`,

    );

    if (!confirmed) return;



    setDeleting(true);

    const ok = await deleteGuild(guild.id);

    setDeleting(false);

    if (ok) {

      onClose();

      router.push("/guilds");

    }

  };



  const handleAddMember = async () => {

    const username = newUsername.trim();

    if (username.length < 3) {

      alert("Username: минимум 3 символа");

      return;

    }



    setAddingMember(true);

    const member = await addMember(guild.id, username);

    setAddingMember(false);

    if (member) {

      setMembers((prev) => {

        if (prev.some((m) => m.user_id === member.user_id)) return prev;

        return [...prev, member];

      });

      setNewUsername("");

    }

  };



  const handleRemoveMember = async (member: GuildMember) => {

    if (member.is_owner) return;

    const confirmed = confirm(`Удалить участника @${member.username}?`);

    if (!confirmed) return;



    const ok = await removeMember(guild.id, member.user_id);

    if (ok) {

      setMembers((prev) => prev.filter((m) => m.user_id !== member.user_id));

    }

  };



  return (

    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">

      <div

        role="dialog"

        aria-modal="true"

        aria-labelledby="guild-settings-title"

        className="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-xl border border-zinc-700 bg-zinc-900 shadow-xl"

      >

        <div className="flex items-center justify-between border-b border-zinc-800 px-5 py-4">

          <h2 id="guild-settings-title" className="text-lg font-semibold text-zinc-100">

            Настройки сервера

          </h2>

          <button

            type="button"

            onClick={onClose}

            className="rounded-md p-1 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"

            aria-label="Close"

          >

            <X size={20} />

          </button>

        </div>



        <div className="flex border-b border-zinc-800 px-5">

          <button

            type="button"

            onClick={() => setTab("overview")}

            className={`border-b-2 px-3 py-2 text-sm font-medium transition-colors ${

              tab === "overview"

                ? "border-[#5865f2] text-zinc-100"

                : "border-transparent text-zinc-400 hover:text-zinc-200"

            }`}

          >

            Обзор

          </button>

          <button

            type="button"

            onClick={() => setTab("members")}

            className={`border-b-2 px-3 py-2 text-sm font-medium transition-colors ${

              tab === "members"

                ? "border-[#5865f2] text-zinc-100"

                : "border-transparent text-zinc-400 hover:text-zinc-200"

            }`}

          >

            Участники

          </button>

        </div>



        <div className="flex-1 overflow-y-auto px-5 py-4">

          {tab === "overview" && (

            <div className="space-y-4">

              <div>

                <label htmlFor="guild-name" className="mb-1 block text-xs font-medium uppercase text-zinc-400">

                  Название

                </label>

                <input

                  id="guild-name"

                  value={name}

                  onChange={(e) => setName(e.target.value)}

                  maxLength={32}

                  className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-[#5865f2]"

                />

              </div>



              <div>

                <label htmlFor="guild-icon" className="mb-1 block text-xs font-medium uppercase text-zinc-400">

                  URL иконки

                </label>

                <input

                  id="guild-icon"

                  value={iconUrl}

                  onChange={(e) => setIconUrl(e.target.value)}

                  placeholder="https://..."

                  maxLength={512}

                  className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-[#5865f2]"

                />

                {iconUrl.trim() && (

                  <img

                    src={iconUrl.trim()}

                    alt=""

                    className="mt-2 h-16 w-16 rounded-full object-cover"

                    onError={(e) => {

                      (e.target as HTMLImageElement).style.display = "none";

                    }}

                  />

                )}

              </div>



              <div>

                <label htmlFor="guild-desc" className="mb-1 block text-xs font-medium uppercase text-zinc-400">

                  Описание

                </label>

                <textarea

                  id="guild-desc"

                  value={description}

                  onChange={(e) => setDescription(e.target.value)}

                  maxLength={512}

                  rows={3}

                  className="w-full resize-none rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-[#5865f2]"

                />

              </div>



              <Button onClick={() => void handleSaveOverview()} disabled={saving} className="w-full">

                {saving ? "Сохранение..." : "Сохранить"}

              </Button>



              {isOwner && (

                <div className="border-t border-zinc-800 pt-4">

                  <p className="mb-2 text-xs text-zinc-500">Опасная зона</p>

                  <Button

                    variant="destructive"

                    onClick={() => void handleDeleteGuild()}

                    disabled={deleting}

                    className="w-full gap-2"

                  >

                    <Trash2 size={16} />

                    {deleting ? "Удаление..." : "Удалить сервер"}

                  </Button>

                </div>

              )}

            </div>

          )}



          {tab === "members" && (

            <div className="space-y-4">

              <div className="flex gap-2">

                <input

                  value={newUsername}

                  onChange={(e) => setNewUsername(e.target.value)}

                  placeholder="username"

                  maxLength={32}

                  className="min-w-0 flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-[#5865f2]"

                  onKeyDown={(e) => {

                    if (e.key === "Enter") void handleAddMember();

                  }}

                />

                <Button

                  onClick={() => void handleAddMember()}

                  disabled={addingMember}

                  size="sm"

                  className="shrink-0 gap-1"

                >

                  <UserPlus size={14} />

                  Добавить

                </Button>

              </div>



              {membersLoading ? (

                <p className="text-sm text-zinc-500">Загрузка...</p>

              ) : members.length === 0 ? (

                <p className="text-sm text-zinc-500">Нет участников</p>

              ) : (

                <ul className="space-y-1">

                  {members.map((member) => (

                    <li

                      key={member.user_id}

                      className="flex items-center justify-between rounded-md px-2 py-2 hover:bg-zinc-800/60"

                    >

                      <div>

                        <span className="text-sm font-medium text-zinc-100">{member.username}</span>

                        {member.is_owner && (

                          <span className="ml-2 rounded bg-[#5865f2]/20 px-1.5 py-0.5 text-xs text-[#5865f2]">

                            owner

                          </span>

                        )}

                      </div>

                      {!member.is_owner && (

                        <button

                          type="button"

                          onClick={() => void handleRemoveMember(member)}

                          className="rounded p-1 text-zinc-400 hover:bg-red-500/10 hover:text-red-400"

                          title="Remove member"

                        >

                          <UserMinus size={16} />

                        </button>

                      )}

                    </li>

                  ))}

                </ul>

              )}

            </div>

          )}

        </div>

      </div>

    </div>

  );

}


