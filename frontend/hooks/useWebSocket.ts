"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { getSocketClient } from "@/lib/ws";
import { useAuthStore, type AuthState } from "@/store/useAuthStore";

function parseChannelIdFromPath(pathname: string): number | null {
  if (!pathname.startsWith("/channels/")) return null;
  const segment = pathname.split("/").pop();
  if (!segment) return null;
  const n = Number.parseInt(segment, 10);
  if (!Number.isFinite(n) || n <= 0) return null;
  return n;
}

export function useWebSocket() {
  const pathname = usePathname();
  const isAuthenticated = useAuthStore((state: AuthState) => state.isAuthenticated);
  const token = useAuthStore((state: AuthState) => state.token);

  useEffect(() => {
    const client = getSocketClient();
    if (!isAuthenticated || !token) {
      client.disconnect();
      return;
    }
    client.connect();
    return () => client.disconnect();
  }, [isAuthenticated, token]);

  useEffect(() => {
    if (!isAuthenticated || !token) return;

    const channelId = parseChannelIdFromPath(pathname);
    if (channelId == null) return;

    const client = getSocketClient();

    const resubscribe = () => client.subscribeToChannel(channelId);
    const offConnect = client.onConnect(resubscribe);
    resubscribe();

    return () => {
      offConnect();
      client.unsubscribeFromChannel(channelId);
    };
  }, [pathname, isAuthenticated, token]);
}
