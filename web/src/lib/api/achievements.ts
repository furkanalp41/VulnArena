import { api } from './client';

export interface Achievement {
  id: string;
  slug: string;
  name: string;
  description: string;
  icon_svg: string;
  category: 'combat' | 'dedication' | 'mastery' | 'special';
  xp_reward: number;
  created_at: string;
}

export interface UserAchievement {
  achievement: Achievement;
  unlocked_at: string;
}

export async function getAllAchievements(): Promise<Achievement[]> {
  return api<Achievement[]>('/achievements');
}
