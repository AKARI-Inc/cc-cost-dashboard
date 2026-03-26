const PALETTE = [
  '#6366f1', // indigo
  '#f59e0b', // amber
  '#10b981', // emerald
  '#ef4444', // red
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
  '#84cc16', // lime
];

export function getColor(index: number): string {
  return PALETTE[index % PALETTE.length]!;
}

export function getUserColorMap(emails: string[]): Map<string, string> {
  const sorted = [...emails].sort();
  return new Map(sorted.map((email, i) => [email, getColor(i)]));
}
