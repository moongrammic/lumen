import { ChatArea } from "@/components/layout/ChatArea";

type ChannelPageProps = {
  params: Promise<{ channelId: string }>;
};

export default async function ChannelPage({ params }: ChannelPageProps) {
  const { channelId } = await params;
  return <ChatArea channelId={channelId} />;
}
