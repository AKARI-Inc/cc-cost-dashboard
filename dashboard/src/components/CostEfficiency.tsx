import { useUsageData } from '../hooks/useUsageData';

type Props = { from: string; to: string };

export function CostEfficiency({ from, to }: Props) {
  const { data, loading, error } = useUsageData({ from, to, groupBy: 'model' });

  if (loading) return null;
  if (error) return null;
  if (!data || data.length === 0) return null;

  const rows = [...data].sort((a, b) => b.total_cost_usd - a.total_cost_usd);

  const totalCost = rows.reduce((s, r) => s + r.total_cost_usd, 0);
  const totalReqs = rows.reduce((s, r) => s + r.request_count, 0);
  const totalIn = rows.reduce((s, r) => s + r.input_tokens, 0);
  const totalOut = rows.reduce((s, r) => s + r.output_tokens, 0);
  const totalCacheR = rows.reduce((s, r) => s + r.cache_read_tokens, 0);
  const totalCacheC = rows.reduce((s, r) => s + r.cache_creation_tokens, 0);
  const totalAll = totalIn + totalOut + totalCacheR + totalCacheC;

  const avgCostPerReq = totalReqs > 0 ? totalCost / totalReqs : 0;
  const blended1M = totalAll > 0 ? (totalCost / totalAll) * 1_000_000 : 0;

  return (
    <div className="card">
      <h3>コスト効率</h3>

      <dl className="detail-grid" style={{ marginBottom: 16 }}>
        <div>
          <dt>平均コスト / リクエスト</dt>
          <dd>${avgCostPerReq.toFixed(4)}</dd>
        </div>
        <div>
          <dt>総リクエスト</dt>
          <dd>{totalReqs.toLocaleString()}</dd>
        </div>
        <div>
          <dt>ブレンド単価 / 1M トークン</dt>
          <dd>${blended1M.toFixed(2)}</dd>
        </div>
        <div>
          <dt>総コスト</dt>
          <dd>${totalCost.toFixed(2)}</dd>
        </div>
      </dl>

      <div className="table-wrap">
        <table className="detail-table">
          <thead>
            <tr>
              <th>モデル</th>
              <th className="num">リクエスト</th>
              <th className="num">平均 $/req</th>
              <th className="num">$/1M 入力</th>
              <th className="num">$/1M 出力</th>
              <th className="num">$/1M ブレンド</th>
              <th className="num">コスト比</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => {
              const model = r.model ?? r.key ?? '-';
              const total = r.input_tokens + r.output_tokens + r.cache_read_tokens + r.cache_creation_tokens;
              const avg = r.request_count > 0 ? r.total_cost_usd / r.request_count : 0;
              const per1MIn = r.input_tokens > 0 ? (r.total_cost_usd / r.input_tokens) * 1_000_000 : 0;
              const per1MOut = r.output_tokens > 0 ? (r.total_cost_usd / r.output_tokens) * 1_000_000 : 0;
              const per1MAll = total > 0 ? (r.total_cost_usd / total) * 1_000_000 : 0;
              const share = totalCost > 0 ? (r.total_cost_usd / totalCost) * 100 : 0;
              return (
                <tr key={model}>
                  <td>{model}</td>
                  <td className="num">{r.request_count.toLocaleString()}</td>
                  <td className="num">${avg.toFixed(4)}</td>
                  <td className="num">${per1MIn.toFixed(2)}</td>
                  <td className="num">${per1MOut.toFixed(2)}</td>
                  <td className="num">${per1MAll.toFixed(2)}</td>
                  <td className="num">{share.toFixed(1)}%</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <p className="info" style={{ textAlign: 'left', fontSize: '0.75rem', padding: '8px 0 0' }}>
        ※ $/1M 入力・出力は合計コストを該当トークン数で按分した実効単価（他トークン分は無視）。
        ブレンドは input + output + cache すべてを分母にした単価。
      </p>
    </div>
  );
}
