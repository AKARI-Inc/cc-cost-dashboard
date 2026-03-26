import { useMemo } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import type { DailyCostEntry } from '../types';
import { getUserColorMap } from '../utils/colors';
import { formatCurrency } from '../utils/format';

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

interface CostChartProps {
  data: DailyCostEntry[];
}

export function CostChart({ data }: CostChartProps) {
  const { chartData, emails, colorMap } = useMemo(() => {
    const emails = [...new Set(data.map((d) => d.email))];
    const colorMap = getUserColorMap(emails);

    const byDate: Record<string, Record<string, number>> = {};
    for (const entry of data) {
      if (!byDate[entry.date]) byDate[entry.date] = {};
      byDate[entry.date]![entry.email] = (byDate[entry.date]![entry.email] ?? 0) + entry.costCents;
    }

    const chartData = Object.entries(byDate)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, users]) => ({
        date: date.slice(5), // MM-DD
        ...Object.fromEntries(
          emails.map((e) => [e, (users[e] ?? 0) / 100])
        ),
      }));

    return { chartData, emails, colorMap };
  }, [data]);

  if (data.length === 0) return null;

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>日別コスト推移</h3>
      <ResponsiveContainer width="100%" height={350}>
        <LineChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
          <XAxis dataKey="date" fontSize={12} tick={{ fill: '#6b7280' }} />
          <YAxis fontSize={12} tick={{ fill: '#6b7280' }} tickFormatter={(v: number) => `$${v}`} />
          <Tooltip
            formatter={(value: number) => formatCurrency(value * 100)}
            labelFormatter={(label: string) => `日付: ${label}`}
            contentStyle={{ borderRadius: 8, fontSize: '0.8125rem' }}
          />
          <Legend wrapperStyle={{ fontSize: '0.75rem' }} />
          {emails.map((email) => (
            <Line
              key={email}
              type="monotone"
              dataKey={email}
              stroke={colorMap.get(email)}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
