import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import type { ModelBreakdownEntry } from '../types';
import { getColor } from '../utils/colors';
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

interface ModelBreakdownProps {
  data: ModelBreakdownEntry[];
}

export function ModelBreakdown({ data }: ModelBreakdownProps) {
  if (data.length === 0) return null;

  const chartData = data.map((d) => ({
    name: d.model,
    value: d.costCents / 100,
  }));

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>モデル別コスト割合</h3>
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            outerRadius={100}
            dataKey="value"
            label={({ name, percent }: { name: string; percent: number }) =>
              `${name} (${(percent * 100).toFixed(0)}%)`
            }
            labelLine={true}
          >
            {chartData.map((_, i) => (
              <Cell key={i} fill={getColor(i)} />
            ))}
          </Pie>
          <Tooltip
            formatter={(value: number) => formatCurrency(value * 100)}
            contentStyle={{ borderRadius: 8, fontSize: '0.8125rem' }}
          />
          <Legend wrapperStyle={{ fontSize: '0.75rem' }} />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
