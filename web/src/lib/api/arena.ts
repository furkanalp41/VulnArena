import { api } from './client';

export interface VulnCategory {
  id: number;
  slug: string;
  name: string;
  owasp_ref?: string;
}

export interface Language {
  id: number;
  slug: string;
  name: string;
}

export interface ChallengeListItem {
  id: string;
  title: string;
  slug: string;
  description: string;
  difficulty: number;
  points: number;
  line_count: number;
  language: Language;
  vuln_category: VulnCategory;
}

export interface Challenge {
  id: string;
  title: string;
  slug: string;
  description: string;
  difficulty: number;
  vulnerable_code: string;
  cve_reference?: string;
  hints: string[];
  points: number;
  line_count: number;
  language: Language;
  vuln_category: VulnCategory;
}

export interface UserProgress {
  user_id: string;
  challenge_id: string;
  status: 'not_started' | 'attempted' | 'solved';
  best_score: number;
  attempt_count: number;
  first_solved_at?: string;
  last_attempted?: string;
}

export interface EvaluationFeedback {
  vuln_identified: boolean;
  vuln_score: number;
  fix_identified: boolean;
  fix_score: number;
  line_accuracy: number;
  overall_score: number;
  passed: boolean;
  terminal_log: string[];
  matched_vuln_terms?: string[];
  matched_fix_terms?: string[];
}

export interface SubmitResult {
  submission: {
    id: string;
    score: number;
    is_correct: boolean;
    target_lines?: number[];
    created_at: string;
  };
  feedback: EvaluationFeedback;
  progress: UserProgress;
  first_blood: boolean;
  bonus_xp?: number;
}

export interface RevealResult {
  solution: {
    target_vulnerability: string;
    conceptual_fix: string;
    vulnerable_lines: number[];
  };
  submission: {
    id: string;
    score: number;
    is_correct: boolean;
    created_at: string;
  };
  progress: UserProgress;
}

interface ChallengeListResponse {
  challenges: ChallengeListItem[] | null;
  total: number;
}

interface ChallengeDetailResponse {
  challenge: Challenge;
  progress: UserProgress | null;
}

export async function listChallenges(params?: {
  language?: string;
  category?: string;
  difficulty_min?: number;
  difficulty_max?: number;
  page?: number;
  limit?: number;
}): Promise<ChallengeListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.language) searchParams.set('language', params.language);
  if (params?.category) searchParams.set('category', params.category);
  if (params?.difficulty_min) searchParams.set('difficulty_min', String(params.difficulty_min));
  if (params?.difficulty_max) searchParams.set('difficulty_max', String(params.difficulty_max));
  if (params?.page) searchParams.set('page', String(params.page));
  if (params?.limit) searchParams.set('limit', String(params.limit));

  const query = searchParams.toString();
  return api<ChallengeListResponse>(`/arena/challenges${query ? `?${query}` : ''}`);
}

export async function getChallenge(id: string): Promise<ChallengeDetailResponse> {
  return api<ChallengeDetailResponse>(`/arena/challenges/${id}`);
}

export async function revealSolution(challengeId: string): Promise<RevealResult> {
  return api<RevealResult>(`/arena/challenges/${challengeId}/reveal`, {
    method: 'POST',
  });
}

export async function submitAnswer(
  challengeId: string,
  answerText: string,
  targetLines?: number[],
  timeSpentSec?: number
): Promise<SubmitResult> {
  return api<SubmitResult>(`/arena/challenges/${challengeId}/submit`, {
    method: 'POST',
    body: {
      answer_text: answerText,
      target_lines: targetLines,
      time_spent_sec: timeSpentSec,
    },
  });
}
