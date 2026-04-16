type Props = {
  data: { [k: string]: unknown; total_cost_usd: number; input_tokens: number; output_tokens: number; request_count: number }[];
  labelKey: string;
};

export function DataTable({ data, labelKey }: Props) {
  const sorted = [...data].sort((a, b) => b.total_cost_usd - a.total_cost_usd);
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{labelKey}</th>
            <th className="num">リクエスト数</th>
            <th className="num">入力トークン</th>
            <th className="num">出力トークン</th>
            <th className="num">コスト</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((r) => (
            <tr key={String(r[labelKey] ?? '')}>
              <td>{String(r[labelKey] ?? '-')}</td>
              <td className="num">{r.request_count.toLocaleString()}</td>
              <td className="num">{r.input_tokens.toLocaleString()}</td>
              <td className="num">{r.output_tokens.toLocaleString()}</td>
              <td className="num">${r.total_cost_usd.toFixed(4)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
