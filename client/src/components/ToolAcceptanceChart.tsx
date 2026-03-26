import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import type { UserSummary } from '../types';
import { formatPercent } from '../utils/format';

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: '#fff',
    border: '1px solid #e5e7eb',
    borderRadius: 12,
    padding: '1.25rem',
    marginBottom: '1.5rem',
    boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
  },
  title: {
    fontSize: '1rem',
    fontWeight: 600,
    color: '#111827',
    margin: '0 0 1rem',
  },
};

interface ToolAcceptanceChartProps {
  users: UserSummary[];
}

function safeRate(accepted: number, rejected: number): number {
  const total = accepted + rejected;
  return total === 0 ? 0 : accepted / total;
}

export function ToolAcceptanceChart({ users }: ToolAcceptanceChartProps) {
  if (users.length === 0) return null;

  const totalEditAccepted = users.reduce((s, u) => s + u.editAccepted, 0);
  const totalEditRejected = users.reduce((s, u) => s + u.editRejected, 0);
  const totalWriteAccepted = users.reduce((s, u) => s + u.writeAccepted, 0);
  const totalWriteRejected = users.reduce((s, u) => s + u.writeRejected, 0);

  const chartData = [
    {
      tool: 'Edit',
      accepted: totalEditAccepted,
      rejected: totalEditRejected,
      rate: safeRate(totalEditAccepted, totalEditRejected),
    },
    {
      tool: 'Write',
      accepted: totalWriteAccepted,
      rejected: totalWriteRejected,
      rate: safeRate(totalWriteAccepted, totalWriteRejected),
    },
  ];

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>ツール承認率</h3>
      <ResponsiveContainer width="100%" height={200}>
        <BarChart data={chartData} layout="vertical" margin={{ top: 5, right: 20, bottom: 5, left: 50 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
          <XAxis type="number" fontSize={12} tick={{ fill: '#6b7280' }} />
          <YAxis type="category" dataKey="tool" fontSize={12} tick={{ fill: '#6b7280' }} />
          <Tooltip
            contentStyle={{ borderRadius: 8, fontSize: '0.8125rem' }}
            formatter={(value: number, name: string) => {
              if (name === '承認率') return formatPercent(value);
              return value;
            }}
          />
          <Legend wrapperStyle={{ fontSize: '0.75rem' }} />
          <Bar dataKey="accepted" name="承認" fill="#10b981" stackId="a" radius={[0, 0, 0, 0]} />
          <Bar dataKey="rejected" name="却下" fill="#ef4444" stackId="a" radius={[0, 2, 2, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
