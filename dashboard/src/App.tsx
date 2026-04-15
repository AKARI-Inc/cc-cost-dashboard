import { useState } from 'react';
import { useUsageData } from './hooks/useUsageData';
import { DateRangeFilter } from './components/DateRangeFilter';
import { SummaryCards } from './components/SummaryCards';
import { CostChart } from './components/CostChart';
import { ModelBreakdown } from './components/ModelBreakdown';
import { UserSummary } from './components/UserSummary';
import { GroupByTabs } from './components/GroupByTabs';
import { RawEventsTable } from './components/RawEventsTable';

// ローカルタイムゾーン基準の YYYY-MM-DD を返す（toISOString は UTC で
// JST 早朝に前日扱いになるバグの対応）
function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return formatLocalDate(d);
}

function today(): string {
  return formatLocalDate(new Date());
}

type Tab = 'claude-code' | 'raw-events';

export function App() {
  const [tab, setTab] = useState<Tab>('claude-code');
  const [from, setFrom] = useState(daysAgo(30));
  const [to, setTo] = useState(today());
  const [groupBy, setGroupBy] = useState('day');

  const handleDateChange = (f: string, t: string) => {
    setFrom(f);
    setTo(t);
  };

  return (
    <div className="container">
      <header className="header">
        <h1>Claude Code 利用量ダッシュボード</h1>
        <DateRangeFilter from={from} to={to} onChange={handleDateChange} />
      </header>

      <nav className="main-tabs">
        {([
          ['claude-code', 'Claude Code'],
          ['raw-events', 'Raw Events'],
        ] as [Tab, string][]).map(([key, label]) => (
          <button
            key={key}
            className={`btn ${tab === key ? 'btn-active' : ''}`}
            onClick={() => setTab(key)}
          >
            {label}
          </button>
        ))}
      </nav>

      {tab === 'claude-code' && (
        <ClaudeCodeView from={from} to={to} groupBy={groupBy} onGroupByChange={setGroupBy} />
      )}
      {tab === 'raw-events' && <RawEventsTable from={from} to={to} />}
    </div>
  );
}

function ClaudeCodeView({
  from, to, groupBy, onGroupByChange,
}: {
  from: string; to: string; groupBy: string; onGroupByChange: (v: string) => void;
}) {
  const dayData = useUsageData({ from, to, groupBy: 'day' });
  const groupData = useUsageData({ from, to, groupBy });

  return (
    <div>
      {dayData.loading && <p className="info">読み込み中...</p>}
      {dayData.error && <p className="error">エラー: {dayData.error}</p>}
      {dayData.data && <SummaryCards data={dayData.data} />}
      {dayData.data && <CostChart data={dayData.data} />}

      <GroupByTabs value={groupBy} onChange={onGroupByChange} />

      {groupData.loading && <p className="info">読み込み中...</p>}
      {groupData.error && <p className="error">エラー: {groupData.error}</p>}

      {groupData.data && groupBy === 'model' && <ModelBreakdown data={groupData.data} />}
      {groupData.data && groupBy === 'user' && <UserSummary data={groupData.data} />}
      {groupData.data && groupBy === 'day' && (
        <div className="card">
          <h3>日別詳細</h3>
          <DataTable data={groupData.data} labelKey="date" />
        </div>
      )}
      {groupData.data && ['terminal', 'version', 'speed'].includes(groupBy) && (
        <div className="card">
          <h3>{groupBy} 別詳細</h3>
          <DataTable data={groupData.data} labelKey="key" />
        </div>
      )}
    </div>
  );
}

function DataTable({ data, labelKey }: {
  data: { [k: string]: unknown; total_cost_usd: number; input_tokens: number; output_tokens: number; request_count: number }[];
  labelKey: string;
}) {
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
          {sorted.map((r, i) => (
            <tr key={i}>
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
