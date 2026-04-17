import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import type { UsageRow } from '../hooks/useUsageData';

type Props = { data: UsageRow[] };

export function ModelBreakdown({ data }: Props) {
  const sorted = [...data].sort((a, b) => b.total_cost_usd - a.total_cost_usd);

  return (
    <div className="card">
      <h3>モデル別コスト</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={sorted} layout="vertical">
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" tickFormatter={(v: number) => `$${v.toFixed(2)}`} fontSize={12} />
          <YAxis type="category" dataKey="model" width={200} fontSize={11} />
          <Tooltip formatter={(value: number) => [`$${value.toFixed(4)}`, 'コスト']} />
          <Bar dataKey="total_cost_usd" fill="#6366f1" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
