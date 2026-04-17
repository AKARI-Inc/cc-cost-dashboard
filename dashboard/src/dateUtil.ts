/** ローカルタイムゾーン基準の YYYY-MM-DD を返す（toISOString は UTC なので JST 早朝に前日扱いになる） */
export function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return formatLocalDate(d);
}

export function today(): string {
  return formatLocalDate(new Date());
}
