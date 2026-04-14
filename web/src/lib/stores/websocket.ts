import { writable } from 'svelte/store';
import { PUBLIC_API_URL } from '$env/static/public';

export interface WSEvent {
  type: string;
  [key: string]: string;
}

if (!PUBLIC_API_URL) {
  throw new Error(
    '[VulnArena] PUBLIC_API_URL is not set for WebSocket. ' +
    'Received: ' + JSON.stringify(PUBLIC_API_URL)
  );
}

function getWsUrl(path: string): string {
  return PUBLIC_API_URL.replace(/^http/, 'ws') + path;
}

const WS_URL = getWsUrl('/ws');
const MAX_RECONNECT_DELAY = 30_000;

let socket: WebSocket | null = null;
let reconnectDelay = 1000;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let intentionalClose = false;

export const wsEvents = writable<WSEvent | null>(null);

function onMessage(event: MessageEvent) {
  try {
    // Hub may batch messages separated by newlines
    const messages = (event.data as string).split('\n');
    for (const raw of messages) {
      if (!raw.trim()) continue;
      const parsed = JSON.parse(raw) as WSEvent;
      wsEvents.set(parsed);
    }
  } catch {
    // Ignore malformed messages
  }
}

function scheduleReconnect() {
  if (intentionalClose) return;
  reconnectTimer = setTimeout(() => {
    connect();
    reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
  }, reconnectDelay);
}

export function connect() {
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  intentionalClose = false;

  try {
    socket = new WebSocket(WS_URL);

    socket.onopen = () => {
      reconnectDelay = 1000; // reset on successful connect
    };

    socket.onmessage = onMessage;

    socket.onclose = () => {
      socket = null;
      scheduleReconnect();
    };

    socket.onerror = () => {
      socket?.close();
    };
  } catch {
    scheduleReconnect();
  }
}

export function disconnect() {
  intentionalClose = true;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    socket.close();
    socket = null;
  }
}

// ─── Authenticated Collab WebSocket ───

const COLLAB_WS_URL = getWsUrl('/ws/collab');

let collabSocket: WebSocket | null = null;
let collabReconnectDelay = 1000;
let collabReconnectTimer: ReturnType<typeof setTimeout> | null = null;
let collabIntentionalClose = false;
let collabToken: string | null = null;
let collabUsername: string | null = null;
let collabDisplayName: string | null = null;

export const collabEvents = writable<WSEvent | null>(null);

function onCollabMessage(event: MessageEvent) {
  try {
    const messages = (event.data as string).split('\n');
    for (const raw of messages) {
      if (!raw.trim()) continue;
      const parsed = JSON.parse(raw) as WSEvent;
      collabEvents.set(parsed);
    }
  } catch {
    // Ignore malformed messages
  }
}

function scheduleCollabReconnect() {
  if (collabIntentionalClose) return;
  collabReconnectTimer = setTimeout(() => {
    if (collabToken) {
      connectCollab(collabToken, collabUsername ?? '', collabDisplayName ?? '');
    }
    collabReconnectDelay = Math.min(collabReconnectDelay * 2, MAX_RECONNECT_DELAY);
  }, collabReconnectDelay);
}

export function connectCollab(token: string, username: string = '', displayName: string = '') {
  if (collabSocket && (collabSocket.readyState === WebSocket.OPEN || collabSocket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  collabIntentionalClose = false;
  collabToken = token;
  collabUsername = username;
  collabDisplayName = displayName;

  try {
    const params = new URLSearchParams({ token });
    if (username) params.set('username', username);
    if (displayName) params.set('display_name', displayName);

    collabSocket = new WebSocket(`${COLLAB_WS_URL}?${params.toString()}`);

    collabSocket.onopen = () => {
      collabReconnectDelay = 1000;
    };

    collabSocket.onmessage = onCollabMessage;

    collabSocket.onclose = () => {
      collabSocket = null;
      scheduleCollabReconnect();
    };

    collabSocket.onerror = () => {
      collabSocket?.close();
    };
  } catch {
    scheduleCollabReconnect();
  }
}

export function disconnectCollab() {
  collabIntentionalClose = true;
  collabToken = null;
  if (collabReconnectTimer) {
    clearTimeout(collabReconnectTimer);
    collabReconnectTimer = null;
  }
  if (collabSocket) {
    collabSocket.close();
    collabSocket = null;
  }
}

export function sendCollab(msg: Record<string, unknown>) {
  if (collabSocket && collabSocket.readyState === WebSocket.OPEN) {
    collabSocket.send(JSON.stringify(msg));
  }
}
