import { api } from './client';
import type { ChallengeListItem } from './arena';
import type { UserAchievement } from './achievements';

export interface RankInfo {
  title: string;
  tier: number;
  total_xp: number;
  next_tier_xp: number;
  progress: number;
}

export interface OverallStats {
  total_solved: number;
  total_available: number;
  total_attempted: number;
  total_points: number;
  average_score: number;
  lessons_read: number;
  current_streak: number;
}

export interface SkillRadarPoint {
  category: string;
  slug: string;
  score: number;
  solved: number;
  total: number;
}

export interface ActivityEntry {
  type: 'challenge_solved' | 'challenge_attempted' | 'lesson_completed';
  title: string;
  points: number;
  score: number;
  occurred_at: string;
}

export interface DashboardProfile {
  rank: RankInfo;
  stats: OverallStats;
  achievements: UserAchievement[] | null;
  skill_radar: SkillRadarPoint[] | null;
  recent_activity: ActivityEntry[] | null;
  next_challenge: ChallengeListItem | null;
}

export async function getDashboardProfile(): Promise<DashboardProfile> {
  return api<DashboardProfile>('/dashboard/profile');
}
