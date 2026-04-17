import { Fragment, useState } from 'react';
import type { UsageRow } from '../hooks/useUsageData';
import { UserDetail } from './UserDetail';

type Props = { data: UsageRow[]; from: string; to: string };

export function UserSummary({ data, from, to }: Props) {
  const sorted = [...data].sort((a, b) => b.total_cost_usd - a.total_cost_usd);
  const [expanded, setExpanded] = useState<string | null>(null);

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
              <th>詳細</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((r) => {
              const label = r.user_email ?? r.key ?? '-';
              const isOpen = expanded === label;
              return (
                <Fragment key={label}>
                  <tr>
                    <td>{label}</td>
                    <td className="num">{r.request_count.toLocaleString()}</td>
                    <td className="num">{r.input_tokens.toLocaleString()}</td>
                    <td className="num">{r.output_tokens.toLocaleString()}</td>
                    <td className="num">${r.total_cost_usd.toFixed(4)}</td>
                    <td>
                      <button
                        className="btn-detail"
                        onClick={() => setExpanded(isOpen ? null : label)}
                        aria-expanded={isOpen}
                      >
                        {isOpen ? '閉じる' : '▸ 詳細'}
                      </button>
                    </td>
                  </tr>
                  {isOpen && (
                    <tr className="detail-row">
                      <td colSpan={6}>
                        <UserDetail row={r} from={from} to={to} />
                      </td>
                    </tr>
                  )}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
