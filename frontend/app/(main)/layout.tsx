import { ChannelList } from "@/components/layout/ChannelList";
import { ServerList } from "@/components/layout/ServerList";
import { VoicePanel } from "@/components/layout/VoicePanel";

type MainLayoutProps = {
  children: React.ReactNode;
};

export default function MainLayout({ children }: MainLayoutProps) {
  return (
    <div className="grid min-h-screen grid-cols-[72px_260px_1fr] bg-zinc-950 text-zinc-100">
      <ServerList />
      <aside className="border-x border-zinc-800 bg-zinc-900">
        <ChannelList />
        <VoicePanel />
      </aside>
      <section className="bg-zinc-950">{children}</section>
    </div>
  );
}
