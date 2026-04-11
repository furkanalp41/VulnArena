export const tierColors: Record<string, string> = {
  'Script Kiddie': '#78716c',
  'Hacker': '#7c9f6b',
  'Pro Hacker': '#8b9dc3',
  'Elite Hacker': '#a78bba',
  'Guru': '#d4a574',
  'Omniscient': '#c9726b',
};

export function getTierColor(title: string): string {
  return tierColors[title] ?? '#d4a574';
}
