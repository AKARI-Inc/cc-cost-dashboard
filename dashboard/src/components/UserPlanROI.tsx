import { useState } from 'react';
import { useUsageData } from '../hooks/useUsageData';

const STANDARD_USD_PER_MONTH = 30;
const PREMIUM_USD_PER_MONTH = 125;
const BAR_MAX_USD = PREMIUM_USD_PER_MONTH * 2;

type Judgement = 'recovered' | 'none';

const JUDGEMENT_LABEL: Record<Judgement, string> = {
  recovered: '元取り',
  none: '未達',
};

function currentMonth(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

function monthRange(ym: string): { from: string; to: string; days: number } {
  const [y, m] = ym.split('-').map(Number);
  const last = new Date(y, m, 0).getDate();
  const pad = (n: number) => String(n).padStart(2, '0');
  return {
    from: `${ym}-01`,
    to: `${ym}-${pad(last)}`,
    days: last,
  };
}

function shiftMonth(ym: string, delta: number): string {
  const [y, m] = ym.split('-').map(Number);
  const d = new Date(y, m - 1 + delta, 1);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

export function UserPlanROI() {
  const [month, setMonth] = useState<string>(currentMonth());
  const { from, to, days } = monthRange(month);
  const { data, loading, error } = useUsageData({ from, to, groupBy: 'user' });

  const standardThreshold = (STANDARD_USD_PER_MONTH * days) / 30;
  const premiumThreshold = (PREMIUM_USD_PER_MONTH * days) / 30;
  const barMax = (BAR_MAX_USD * days) / 30;
  const standardMarkPct = (standardThreshold / barMax) * 100;
  const premiumMarkPct = (premiumThreshold / barMax) * 100;

  const rows = (data ?? [])
    .map((r) => {
      const label = r.user_email ?? r.key ?? '-';
      const cost = r.total_cost_usd;
      const judgement: Judgement = cost >= premiumThreshold ? 'recovered' : 'none';
      const roi = premiumThreshold > 0 ? cost / premiumThreshold : 0;
      const diff = cost - premiumThreshold;
      return { label, cost, judgement, roi, diff };
    })
    .sort((a, b) => b.cost - a.cost);

  const totalUsers = rows.length;
  const recovered = rows.filter((r) => r.judgement === 'recovered').length;
  const totalCost = rows.reduce((s, r) => s + r.cost, 0);
  const teamPlanCost = premiumThreshold * totalUsers;
  const teamDiff = totalCost - teamPlanCost;

  return (
    <div className="card">
      <div className="plan-header">
        <h3 style={{ margin: 0 }}>
          Premium プラン 元取り状況{' '}
          <span className="muted">
            (基準 Premium ${premiumThreshold.toFixed(2)} /人 × {days}日)
          </span>
        </h3>
        <div className="plan-month-picker">
          <button
            className="btn btn-sm"
            onClick={() => setMonth(shiftMonth(month, -1))}
            aria-label="前の月"
          >
            ◀
          </button>
          <input
            type="month"
            value={month}
            onChange={(e) => setMonth(e.target.value || currentMonth())}
            className="plan-month-input"
          />
          <button
            className="btn btn-sm"
            onClick={() => setMonth(shiftMonth(month, 1))}
            aria-label="次の月"
          >
            ▶
          </button>
          <button
            className="btn btn-sm"
            onClick={() => setMonth(currentMonth())}
          >
            今月
          </button>
        </div>
      </div>

      {loading && <p className="info">読み込み中...</p>}
      {error && <p className="error">エラー: {error}</p>}

      {!loading && !error && rows.length === 0 && (
        <p className="info">この月のデータはありません</p>
      )}

      {!loading && !error && rows.length > 0 && (
        <>
          <dl className="detail-grid" style={{ margin: '16px 0' }}>
            <div>
              <dt>元取り人数</dt>
              <dd>
                {recovered} / {totalUsers}
              </dd>
            </div>
            <div>
              <dt>元取り率</dt>
              <dd>
                {totalUsers > 0
                  ? ((recovered / totalUsers) * 100).toFixed(0)
                  : 0}
                %
              </dd>
            </div>
            <div>
              <dt>実利用総額</dt>
              <dd>${totalCost.toFixed(2)}</dd>
            </div>
            <div>
              <dt>Premium チーム総額</dt>
              <dd>${teamPlanCost.toFixed(2)}</dd>
            </div>
            <div>
              <dt>チーム差額</dt>
              <dd className={teamDiff >= 0 ? 'roi-positive' : 'roi-negative'}>
                {teamDiff >= 0 ? '+' : ''}${teamDiff.toFixed(2)}
              </dd>
            </div>
          </dl>

          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ユーザー</th>
                  <th className="num">利用金額</th>
                  <th className="num">差額</th>
                  <th className="num">元取り率</th>
                  <th className="plan-bar-col">
                    元取り可視化{' '}
                    <span className="muted" style={{ fontWeight: 400 }}>
                      (Standard ┃ Premium)
                    </span>
                  </th>
                  <th>判定</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => {
                  const fillPct = Math.min((r.cost / barMax) * 100, 100);
                  return (
                    <tr key={r.label}>
                      <td>{r.label}</td>
                      <td className="num">${r.cost.toFixed(2)}</td>
                      <td
                        className={`num ${r.diff >= 0 ? 'roi-positive' : 'roi-negative'}`}
                      >
                        {r.diff >= 0 ? '+' : ''}${r.diff.toFixed(2)}
                      </td>
                      <td className="num">{(r.roi * 100).toFixed(0)}%</td>
                      <td className="plan-bar-col">
                        <div className="plan-bar-track">
                          <div
                            className="plan-bar-mark plan-bar-mark-secondary"
                            style={{ left: `${standardMarkPct}%` }}
                            title={`Standard $${standardThreshold.toFixed(2)}`}
                          />
                          <div
                            className="plan-bar-mark"
                            style={{ left: `${premiumMarkPct}%` }}
                            title={`Premium $${premiumThreshold.toFixed(2)}`}
                          />
                          <div
                            className={`plan-bar-fill judgement-${r.judgement}`}
                            style={{ width: `${fillPct}%` }}
                          />
                        </div>
                      </td>
                      <td>
                        <span className={`roi-badge judgement-${r.judgement}`}>
                          {JUDGEMENT_LABEL[r.judgement]}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}

      <p
        className="info"
        style={{ textAlign: 'left', fontSize: '0.75rem', padding: '8px 0 0' }}
      >
        ※ 判定は Premium ${PREMIUM_USD_PER_MONTH}{' '}
        /席/月 を日割りした金額を超えているかで決定。左縦線が Standard ($
        {STANDARD_USD_PER_MONTH}/月)、右縦線が Premium (${PREMIUM_USD_PER_MONTH}/月)。
      </p>
    </div>
  );
}
