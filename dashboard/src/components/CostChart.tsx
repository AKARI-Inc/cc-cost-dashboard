import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import type { UsageRow } from '../hooks/useUsageData';

type Props = { data: UsageRow[] };

export function CostChart({ data }: Props) {
  const sorted = [...data].sort((a, b) => (a.date ?? '').localeCompare(b.date ?? ''));

  return (
    <div className="card">
      <h3>日別コスト推移</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={sorted}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" fontSize={12} />
          <YAxis fontSize={12} tickFormatter={(v: number) => `$${v.toFixed(2)}`} />
          <Tooltip
            formatter={(value: number, name: string) => {
              if (name === 'total_cost_usd') return [`$${value.toFixed(4)}`, 'コスト'];
              return [value.toLocaleString(), name];
            }}
            labelFormatter={(label: string) => `日付: ${label}`}
          />
          <Line type="monotone" dataKey="total_cost_usd" stroke="#6366f1" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
