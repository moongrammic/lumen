"use client";

import { useWebSocket } from "@/hooks/useWebSocket";

type WebSocketProviderProps = {
  children: React.ReactNode;
};

/**
 * Глобальный слой WebSocket: reconnect, очередь до OPEN, подписка на канал по маршруту `/channels/:id`.
 * События разбираются в `lib/ws.ts` по контракту backend.
 */
export function WebSocketProvider({ children }: WebSocketProviderProps) {
  useWebSocket();
  return <>{children}</>;
}
