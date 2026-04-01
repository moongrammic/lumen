import {
  isWsLegacyError,
  isWsServerEnvelope,
  type WsInboundMessage,
  type WsServerEnvelope,
} from "@/types/backend";
import { useAuthStore } from "@/store/useAuthStore";
import { useChatStore } from "@/store/useChatStore";
import { toast } from "sonner";

/**
 * WebSocket только на NEXT_PUBLIC_WS_URL. Аутентификация: первое сообщение IDENTIFY (op 2) с JWT в теле,
 * затем сервер шлёт READY — без токена в query string.
 */
class SocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private readonly wsUrl = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";
  private manualClose = false;
  private identified = false;
  private readonly connectListeners = new Set<() => void>();
  private readonly outboundQueue: string[] = [];

  onConnect(listener: () => void): () => void {
    this.connectListeners.add(listener);
    return () => this.connectListeners.delete(listener);
  }

  /** Сессия готова к SUBSCRIBE / MESSAGE_CREATE (получен READY). */
  isSessionReady(): boolean {
    return this.identified && this.ws?.readyState === WebSocket.OPEN;
  }

  connect() {
    if (typeof window === "undefined" || this.ws?.readyState === WebSocket.OPEN) return;

    this.manualClose = false;
    this.identified = false;

    const token = useAuthStore.getState().token;
    if (!token) {
      return;
    }

    this.ws = new WebSocket(this.wsUrl);

    this.ws.onerror = () => {
      this.ws?.close();
    };

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      const t = useAuthStore.getState().token;
      if (!t || !this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      this.ws.send(
        JSON.stringify({
          op: 2,
          event: "IDENTIFY",
          payload: { token: t },
        }),
      );
    };

    this.ws.onmessage = (event) => {
      this.dispatchInbound(event.data);
    };

    this.ws.onclose = () => {
      this.identified = false;
      if (!this.manualClose) {
        this.scheduleReconnect();
      }
    };
  }

  disconnect() {
    this.manualClose = true;
    this.identified = false;
    this.outboundQueue.length = 0;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
    this.ws?.close();
    this.ws = null;
  }

  subscribeToChannel(channelId: number) {
    this.sendFrame({
      op: 0,
      event: "SUBSCRIBE_CHANNEL",
      payload: { channel_id: channelId },
    });
  }

  unsubscribeFromChannel(channelId: number) {
    this.sendFrame({
      op: 0,
      event: "UNSUBSCRIBE_CHANNEL",
      payload: { channel_id: channelId },
    });
  }

  sendMessage(channelId: number, content: string) {
    this.sendFrame({
      op: 0,
      event: "MESSAGE_CREATE",
      payload: { channel_id: channelId, content },
    });
  }

  private scheduleReconnect() {
    this.reconnectAttempts += 1;
    const delay = Math.min(30_000, 500 * 2 ** this.reconnectAttempts);
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  private sendFrame(frame: { op: number; event: string; payload: object }) {
    const raw = JSON.stringify(frame);
    if (this.ws?.readyState === WebSocket.OPEN && this.identified) {
      this.ws.send(raw);
      return;
    }
    if (this.ws?.readyState === WebSocket.CONNECTING || (this.ws?.readyState === WebSocket.OPEN && !this.identified)) {
      this.outboundQueue.push(raw);
      return;
    }
  }

  private flushOutbound() {
    if (this.ws?.readyState !== WebSocket.OPEN || !this.identified) return;
    while (this.outboundQueue.length > 0) {
      const raw = this.outboundQueue.shift();
      if (raw) this.ws.send(raw);
    }
  }

  private dispatchInbound(raw: string) {
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      toast.error("Malformed WebSocket message");
      return;
    }

    if (!parsed || typeof parsed !== "object") return;

    const msg = parsed as WsInboundMessage;

    if (isWsLegacyError(msg)) {
      toast.error(msg.payload.message ?? "WebSocket error");
      return;
    }

    if (!isWsServerEnvelope(msg)) {
      return;
    }

    this.handleServerEnvelope(msg);
  }

  private handleServerEnvelope(msg: WsServerEnvelope) {
    switch (msg.event) {
      case "READY":
        this.identified = true;
        this.flushOutbound();
        this.connectListeners.forEach((fn) => fn());
        break;
      case "MESSAGE_CREATE":
        useChatStore.getState().upsertMessage(msg.payload);
        break;
      case "TYPING_START":
        useChatStore.getState().setTypingUser(msg.payload.channel_id, msg.payload.user_id);
        break;
      case "PRESENCE_UPDATE":
        break;
      case "CHANNEL_SUBSCRIBED":
        break;
      case "CHANNEL_UNSUBSCRIBED":
        break;
      case "ERROR": {
        const p = msg.payload;
        if (p.code === "IDENTIFY_FAILED") {
          toast.error(p.message ?? "Identify failed");
          this.disconnect();
          void useAuthStore.getState().logout();
          if (typeof window !== "undefined") {
            window.location.assign("/login");
          }
          return;
        }
        const text =
          p.message ??
          (p.code ? `${p.code}${p.retry_after != null ? ` (retry ${p.retry_after}s)` : ""}` : "Unknown error");
        toast.error(text);
        break;
      }
      default:
        break;
    }
  }
}

const socketClient = new SocketClient();

export function getSocketClient() {
  return socketClient;
}
