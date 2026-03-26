import type { UserSummary } from '../types';
import { formatCurrency, formatTokens, formatNumber } from '../utils/format';

const styles: Record<string, React.CSSProperties> = {
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
    gap: '1rem',
    marginBottom: '1.5rem',
  },
  card: {
    background: '#fff',
    border: '1px solid #e5e7eb',
    borderRadius: 12,
    padding: '1.25rem',
    boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
  },
  label: {
    fontSize: '0.75rem',
    fontWeight: 600,
    color: '#6b7280',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    margin: 0,
  },
  value: {
    fontSize: '1.75rem',
    fontWeight: 700,
    color: '#111827',
    margin: '0.25rem 0 0',
  },
  sub: {
    fontSize: '0.75rem',
    color: '#9ca3af',
    margin: '0.25rem 0 0',
  },
};

interface SummaryCardsProps {
  users: UserSummary[];
}

export function SummaryCards({ users }: SummaryCardsProps) {
  const totalCost = users.reduce((s, u) => s + u.totalCostCents, 0);
  const totalTokens = users.reduce((s, u) => s + u.totalTokensInput + u.totalTokensOutput, 0);
  const activeUsers = users.length;
  const totalCommits = users.reduce((s, u) => s + u.commits, 0);
  const totalPRs = users.reduce((s, u) => s + u.pullRequests, 0);

  const cards = [
    { label: '合計コスト', value: formatCurrency(totalCost), sub: `${users.length} ユーザー` },
    { label: '合計トークン', value: formatTokens(totalTokens), sub: `入力+出力` },
    { label: 'アクティブユーザー', value: formatNumber(activeUsers), sub: `期間内` },
    { label: 'コミット / PR', value: `${formatNumber(totalCommits)} / ${formatNumber(totalPRs)}`, sub: 'Claude Code経由' },
  ];

  return (
    <div style={styles.grid}>
      {cards.map((c) => (
        <div key={c.label} style={styles.card}>
          <p style={styles.label}>{c.label}</p>
          <p style={styles.value}>{c.value}</p>
          <p style={styles.sub}>{c.sub}</p>
        </div>
      ))}
    </div>
  );
}
