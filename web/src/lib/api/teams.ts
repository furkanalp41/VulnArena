import { api } from './client';

export interface Team {
  id: string;
  name: string;
  tag: string;
  description: string;
  avatar_url: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TeamMember {
  team_id: string;
  user_id: string;
  role: string;
  joined_at: string;
  username: string;
  display_name: string;
}

export interface TeamWithMembers {
  team: Team;
  members: TeamMember[];
  total_xp: number;
  total_solved: number;
}

export interface TeamLeaderboardEntry {
  rank: number;
  team_name: string;
  tag: string;
  member_count: number;
  total_xp: number;
  total_solved: number;
}

export interface CreateTeamInput {
  name: string;
  tag: string;
  description: string;
}

export function listTeams(): Promise<Team[]> {
  return api<Team[]>('/teams');
}

export function getTeam(tag: string): Promise<TeamWithMembers> {
  return api<TeamWithMembers>(`/teams/${tag}`);
}

export function createTeam(input: CreateTeamInput): Promise<TeamWithMembers> {
  return api<TeamWithMembers>('/teams', { method: 'POST', body: input });
}

export function joinTeam(tag: string): Promise<void> {
  return api<void>(`/teams/${tag}/join`, { method: 'POST' });
}

export function leaveTeam(): Promise<void> {
  return api<void>('/teams/leave', { method: 'POST' });
}

export function getMyTeam(): Promise<TeamWithMembers | null> {
  return api<TeamWithMembers | null>('/teams/me');
}

export function getTeamLeaderboard(): Promise<TeamLeaderboardEntry[]> {
  return api<TeamLeaderboardEntry[]>('/teams/leaderboard');
}
