import { api } from './client';
import type { RankInfo, SkillRadarPoint, ActivityEntry } from './dashboard';
import type { UserAchievement } from './achievements';

export interface LeaderboardEntry {
  rank: number;
  username: string;
  display_name: string;
  avatar_url: string;
  rank_title: string;
  tier: number;
  total_xp: number;
  total_solved: number;
}

export interface PublicStats {
  total_solved: number;
  total_points: number;
}

export interface PublicProfile {
  username: string;
  display_name: string;
  avatar_url: string;
  joined_at: string;
  rank: RankInfo;
  stats: PublicStats;
  achievements: UserAchievement[] | null;
  skill_radar: SkillRadarPoint[] | null;
  recent_activity: ActivityEntry[] | null;
}

export async function getLeaderboard(): Promise<LeaderboardEntry[]> {
  return api<LeaderboardEntry[]>('/leaderboard');
}

export async function getPublicProfile(username: string): Promise<PublicProfile> {
  return api<PublicProfile>(`/users/${encodeURIComponent(username)}/profile`);
}
