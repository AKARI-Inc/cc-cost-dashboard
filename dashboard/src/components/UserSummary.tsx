import type { UsageRow } from '../hooks/useUsageData';

type Props = { data: UsageRow[] };

export function UserSummary({ data }: Props) {
  const sorted = [...data].sort((a, b) => b.total_cost_usd - a.total_cost_usd);

  return (
    <div className="card">
      <h3>ユーザー別サマリー</h3>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>ユーザー</th>
              <th className="num">リクエスト数</th>
              <th className="num">入力トークン</th>
              <th className="num">出力トークン</th>
              <th className="num">コスト</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((r, i) => (
              <tr key={i}>
                <td>{r.user_email ?? r.key ?? '-'}</td>
                <td className="num">{r.request_count.toLocaleString()}</td>
                <td className="num">{r.input_tokens.toLocaleString()}</td>
                <td className="num">{r.output_tokens.toLocaleString()}</td>
                <td className="num">${r.total_cost_usd.toFixed(4)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
