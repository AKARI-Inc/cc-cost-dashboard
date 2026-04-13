import { useMemo } from 'react';
import type { UsageRow } from '../hooks/useUsageData';

type Props = { data: UsageRow[] };

export function SummaryCards({ data }: Props) {
  const totals = useMemo(() => {
    let cost = 0, input = 0, output = 0, reqs = 0;
    for (const r of data) {
      cost += r.total_cost_usd;
      input += r.input_tokens;
      output += r.output_tokens;
      reqs += r.request_count;
    }
    return { cost, input, output, reqs };
  }, [data]);

  return (
    <div className="summary-cards">
      <div className="card summary-card">
        <div className="summary-label">総コスト</div>
        <div className="summary-value">${totals.cost.toFixed(2)}</div>
      </div>
      <div className="card summary-card">
        <div className="summary-label">総リクエスト数</div>
        <div className="summary-value">{totals.reqs.toLocaleString()}</div>
      </div>
      <div className="card summary-card">
        <div className="summary-label">総入力トークン</div>
        <div className="summary-value">{totals.input.toLocaleString()}</div>
      </div>
      <div className="card summary-card">
        <div className="summary-label">総出力トークン</div>
        <div className="summary-value">{totals.output.toLocaleString()}</div>
      </div>
    </div>
  );
}
