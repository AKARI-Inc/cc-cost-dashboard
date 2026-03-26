import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import type { DailyProductivityEntry } from '../types';

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

interface ProductivityChartProps {
  data: DailyProductivityEntry[];
}

export function ProductivityChart({ data }: ProductivityChartProps) {
  if (data.length === 0) return null;

  const chartData = data.map((d) => ({
    ...d,
    date: d.date.slice(5),
  }));

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>セッション / コミット / PR 推移</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
          <XAxis dataKey="date" fontSize={12} tick={{ fill: '#6b7280' }} />
          <YAxis fontSize={12} tick={{ fill: '#6b7280' }} />
          <Tooltip contentStyle={{ borderRadius: 8, fontSize: '0.8125rem' }} />
          <Legend wrapperStyle={{ fontSize: '0.75rem' }} />
          <Bar dataKey="sessions" name="セッション" fill="#6366f1" radius={[2, 2, 0, 0]} />
          <Bar dataKey="commits" name="コミット" fill="#10b981" radius={[2, 2, 0, 0]} />
          <Bar dataKey="pullRequests" name="PR" fill="#f59e0b" radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
