import { writable, get } from 'svelte/store';
import { collabEvents, sendCollab } from './websocket';

// ─── Types ───

export interface RemoteCursor {
  userId: string;
  username: string;
  displayName: string;
  line: number;
  column: number;
  color: string;
}

export interface RemoteSelection {
  userId: string;
  username: string;
  lines: Set<number>;
  color: string;
}

export interface RoomMember {
  userId: string;
  username: string;
  displayName: string;
  color: string;
}

// ─── Color palette for remote users ───

const CURSOR_COLORS = [
  '#3b82f6', // blue
  '#a855f7', // purple
  '#f97316', // orange
  '#06b6d4', // cyan
  '#ec4899', // pink
  '#84cc16', // lime
];

function colorForUser(userId: string): string {
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = ((hash << 5) - hash + userId.charCodeAt(i)) | 0;
  }
  return CURSOR_COLORS[Math.abs(hash) % CURSOR_COLORS.length];
}

// ─── Stores ───

export const remoteCursors = writable<Map<string, RemoteCursor>>(new Map());
export const remoteSelections = writable<Map<string, RemoteSelection>>(new Map());
export const roomMembers = writable<RoomMember[]>([]);
export const currentRoomKey = writable<string | null>(null);

// ─── Throttled cursor broadcast ───

let cursorThrottleTimer: ReturnType<typeof setTimeout> | null = null;
let pendingCursor: { line: number; column: number } | null = null;

export function broadcastCursor(line: number, column: number) {
  pendingCursor = { line, column };
  if (!cursorThrottleTimer) {
    cursorThrottleTimer = setTimeout(() => {
      if (pendingCursor) {
        sendCollab({
          type: 'CURSOR_MOVE',
          line: pendingCursor.line,
          column: pendingCursor.column,
        });
        pendingCursor = null;
      }
      cursorThrottleTimer = null;
    }, 100); // ~10 Hz
  }
}

export function broadcastLineSelect(line: number, selected: boolean) {
  sendCollab({
    type: 'LINE_SELECT',
    line,
    selected,
  });
}

// ─── Room management ───

export function joinAuditRoom(challengeId: string, teamId: string) {
  sendCollab({
    type: 'JOIN_ROOM',
    challenge_id: challengeId,
    team_id: teamId,
  });
  currentRoomKey.set(`${challengeId}:${teamId}`);
}

export function leaveAuditRoom() {
  sendCollab({ type: 'LEAVE_ROOM' });
  currentRoomKey.set(null);
  remoteCursors.set(new Map());
  remoteSelections.set(new Map());
  roomMembers.set([]);
}

// ─── Event listener ───

let unsubscribe: (() => void) | null = null;

export function startCollabListener() {
  if (unsubscribe) return;

  unsubscribe = collabEvents.subscribe((event) => {
    if (!event) return;

    switch (event.type) {
      case 'REMOTE_CURSOR': {
        const userId = event.user_id as string;
        const cursor: RemoteCursor = {
          userId,
          username: event.username as string,
          displayName: (event.display_name as string) || (event.username as string),
          line: Number(event.line),
          column: Number(event.column),
          color: colorForUser(userId),
        };
        remoteCursors.update((map) => {
          const next = new Map(map);
          next.set(userId, cursor);
          return next;
        });
        break;
      }

      case 'REMOTE_LINE_SELECT': {
        const userId = event.user_id as string;
        const line = Number(event.line);
        const selected = String((event as any).selected) === 'true';
        remoteSelections.update((map) => {
          const next = new Map(map);
          const existing = next.get(userId);
          if (existing) {
            const lines = new Set(existing.lines);
            if (selected) {
              lines.add(line);
            } else {
              lines.delete(line);
            }
            next.set(userId, { ...existing, lines });
          } else if (selected) {
            next.set(userId, {
              userId,
              username: event.username as string,
              lines: new Set([line]),
              color: colorForUser(userId),
            });
          }
          return next;
        });
        break;
      }

      case 'ROOM_USER_JOINED': {
        const userId = event.user_id as string;
        const member: RoomMember = {
          userId,
          username: event.username as string,
          displayName: (event.display_name as string) || (event.username as string),
          color: colorForUser(userId),
        };
        roomMembers.update((members) => {
          if (members.some((m) => m.userId === userId)) return members;
          return [...members, member];
        });
        break;
      }

      case 'ROOM_USER_LEFT': {
        const userId = event.user_id as string;
        roomMembers.update((members) => members.filter((m) => m.userId !== userId));
        remoteCursors.update((map) => {
          const next = new Map(map);
          next.delete(userId);
          return next;
        });
        remoteSelections.update((map) => {
          const next = new Map(map);
          next.delete(userId);
          return next;
        });
        break;
      }

      case 'ROOM_STATE': {
        // Initial state with existing room members
        const members = (event as any).members as Array<{
          user_id: string;
          username: string;
          display_name: string;
        }>;
        if (Array.isArray(members)) {
          roomMembers.set(
            members.map((m) => ({
              userId: m.user_id,
              username: m.username,
              displayName: m.display_name || m.username,
              color: colorForUser(m.user_id),
            }))
          );
        }
        break;
      }
    }
  });
}

export function stopCollabListener() {
  if (unsubscribe) {
    unsubscribe();
    unsubscribe = null;
  }
  remoteCursors.set(new Map());
  remoteSelections.set(new Map());
  roomMembers.set([]);
  currentRoomKey.set(null);
}
