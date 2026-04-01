/**
 * Типы строго по backend:
 * - Входящие от клиента: service.IncomingEvent (cmd/api + chat.go)
 * - Исходящие broadcast: service.Event (hub Broadcast)
 * - Ответы на SUBSCRIBE/UNSUBSCRIBE и ошибки: main.go WriteJSON / WriteMessage
 */

// --- Client → server (тело WebSocket JSON = service.IncomingEvent: op, event, payload)

export type WsIncomingEventBase = {
  op: number;
  event: string;
  payload: unknown;
};

/** MESSAGE_CREATE — service.IncomingMessageCreatePayload */
export type WsClientMessageCreatePayload = {
  channel_id: number;
  content: string;
};

/** TYPING_START — service.IncomingTypingPayload */
export type WsClientTypingPayload = {
  channel_id: number;
};

/** PRESENCE_UPDATE — service.PresencePayload */
export type WsClientPresencePayload = {
  user_id: string;
  status: string;
};

/** SUBSCRIBE_CHANNEL / UNSUBSCRIBE_CHANNEL — main.go extractChannelID */
export type WsClientChannelPayload = {
  channel_id: number;
};

// --- Server → client (broadcast из chat.go)

export type WsAuthorDTO = {
  id: string;
  username: string;
  avatar: string;
};

/** service.MessagePayload — поле payload при event MESSAGE_CREATE */
export type WsMessagePayload = {
  id: number;
  channel_id: number;
  content: string;
  author: WsAuthorDTO;
  attachments: string[];
};

/** TYPING_START broadcast — map в BroadcastTyping */
export type WsTypingStartPayload = {
  channel_id: number;
  user_id: string;
};

/** PRESENCE_UPDATE broadcast */
export type WsPresenceUpdatePayload = {
  user_id: string;
  status: string;
};

// --- Server → client (служебные ответы main.go)

export type WsChannelAckPayload = {
  channel_id: number;
};

export type WsErrorPayload = {
  message?: string;
  code?: string;
  retry_after?: number;
};

/** Унифицированный ответ с полем event (основной формат) */
export type WsServerEnvelope =
  | { op: number; event: "MESSAGE_CREATE"; payload: WsMessagePayload }
  | { op: number; event: "TYPING_START"; payload: WsTypingStartPayload }
  | { op: number; event: "PRESENCE_UPDATE"; payload: WsPresenceUpdatePayload }
  | { op: number; event: "CHANNEL_SUBSCRIBED"; payload: WsChannelAckPayload }
  | { op: number; event: "CHANNEL_UNSUBSCRIBED"; payload: WsChannelAckPayload }
  | { op: number; event: "ERROR"; payload: WsErrorPayload };

/**
 * Редкий формат из main.go при ошибке до разбора event / chat error:
 * {"type":"error","payload":{"message":"..."}}
 */
export type WsLegacyErrorShape = {
  type: "error";
  payload: { message?: string };
};

export type WsInboundMessage = WsServerEnvelope | WsLegacyErrorShape;

export function isWsLegacyError(msg: WsInboundMessage): msg is WsLegacyErrorShape {
  return "type" in msg && msg.type === "error";
}

export function isWsServerEnvelope(msg: WsInboundMessage): msg is WsServerEnvelope {
  return "event" in msg && typeof msg.event === "string";
}

/** Удобный тип для UI / Zustand (= WsMessagePayload) */
export type ChatMessage = WsMessagePayload;
