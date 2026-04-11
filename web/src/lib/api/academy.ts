import { api } from './client';

export interface LessonListItem {
  id: string;
  title: string;
  slug: string;
  category: string;
  description: string;
  difficulty: number;
  read_time_min: number;
  tags: string[];
}

export interface Lesson {
  id: string;
  title: string;
  slug: string;
  category: string;
  description: string;
  content: string;
  difficulty: number;
  read_time_min: number;
  tags: string[];
  created_at: string;
}

interface LessonListResponse {
  lessons: LessonListItem[] | null;
  total: number;
}

export async function listLessons(params?: {
  category?: string;
  page?: number;
  limit?: number;
}): Promise<LessonListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.category) searchParams.set('category', params.category);
  if (params?.page) searchParams.set('page', String(params.page));
  if (params?.limit) searchParams.set('limit', String(params.limit));

  const query = searchParams.toString();
  return api<LessonListResponse>(`/academy/lessons${query ? `?${query}` : ''}`);
}

export async function getLesson(id: string): Promise<Lesson> {
  return api<Lesson>(`/academy/lessons/${id}`);
}
