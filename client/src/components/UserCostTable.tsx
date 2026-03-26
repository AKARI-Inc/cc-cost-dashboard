import { useMemo, useState } from 'react';
import type { UserSummary } from '../types';
import { formatCurrency, formatTokens, formatNumber, formatPercent } from '../utils/format';

type SortKey = 'email' | 'totalCostCents' | 'totalTokens' | 'commits' | 'acceptanceRate' | 'activeDays';

const columns: { key: SortKey; label: string }[] = [
  { key: 'email', label: 'ユーザー' },
  { key: 'totalCostCents', label: 'コスト' },
  { key: 'totalTokens', label: 'トークン' },
  { key: 'commits', label: 'コミット' },
  { key: 'acceptanceRate', label: '承認率' },
  { key: 'activeDays', label: '稼働日数' },
];

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: '#fff',
    border: '1px solid #e5e7eb',
    borderRadius: 12,
    padding: '1.25rem',
    marginBottom: '1.5rem',
    boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
    overflowX: 'auto',
  },
  title: {
    fontSize: '1rem',
    fontWeight: 600,
    color: '#111827',
    margin: '0 0 1rem',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse' as const,
    fontSize: '0.875rem',
  },
  th: {
    textAlign: 'left' as const,
    padding: '0.625rem 0.75rem',
    borderBottom: '2px solid #e5e7eb',
    color: '#6b7280',
    fontWeight: 600,
    cursor: 'pointer',
    userSelect: 'none' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '0.625rem 0.75rem',
    borderBottom: '1px solid #f3f4f6',
    color: '#374151',
  },
  emailCell: {
    fontWeight: 500,
    maxWidth: 200,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap' as const,
  },
  numCell: {
    textAlign: 'right' as const,
    fontVariantNumeric: 'tabular-nums',
  },
};

function getAcceptanceRate(u: UserSummary): number {
  const accepted = u.editAccepted + u.writeAccepted;
  const total = accepted + u.editRejected + u.writeRejected;
  return total === 0 ? 0 : accepted / total;
}

interface UserCostTableProps {
  users: UserSummary[];
}

export function UserCostTable({ users }: UserCostTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>('totalCostCents');
  const [sortAsc, setSortAsc] = useState(false);

  const sorted = useMemo(() => {
    return [...users].sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case 'email': cmp = a.email.localeCompare(b.email); break;
        case 'totalCostCents': cmp = a.totalCostCents - b.totalCostCents; break;
        case 'totalTokens': cmp = (a.totalTokensInput + a.totalTokensOutput) - (b.totalTokensInput + b.totalTokensOutput); break;
        case 'commits': cmp = a.commits - b.commits; break;
        case 'acceptanceRate': cmp = getAcceptanceRate(a) - getAcceptanceRate(b); break;
        case 'activeDays': cmp = a.activeDays - b.activeDays; break;
      }
      return sortAsc ? cmp : -cmp;
    });
  }, [users, sortKey, sortAsc]);

  const handleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortAsc(!sortAsc);
    } else {
      setSortKey(key);
      setSortAsc(false);
    }
  };

  if (users.length === 0) return null;

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>ユーザー別サマリー</h3>
      <table style={styles.table}>
        <thead>
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                style={{ ...styles.th, ...(col.key !== 'email' ? styles.numCell : {}) }}
                onClick={() => handleSort(col.key)}
              >
                {col.label} {sortKey === col.key ? (sortAsc ? '▲' : '▼') : ''}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.map((u) => (
            <tr key={u.email}>
              <td style={{ ...styles.td, ...styles.emailCell }}>{u.email}</td>
              <td style={{ ...styles.td, ...styles.numCell }}>{formatCurrency(u.totalCostCents)}</td>
              <td style={{ ...styles.td, ...styles.numCell }}>{formatTokens(u.totalTokensInput + u.totalTokensOutput)}</td>
              <td style={{ ...styles.td, ...styles.numCell }}>{formatNumber(u.commits)}</td>
              <td style={{ ...styles.td, ...styles.numCell }}>{formatPercent(getAcceptanceRate(u))}</td>
              <td style={{ ...styles.td, ...styles.numCell }}>{formatNumber(u.activeDays)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
