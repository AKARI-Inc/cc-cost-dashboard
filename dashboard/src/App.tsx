import { useState } from 'react';
import { useUsageData } from './hooks/useUsageData';
import { DateRangeFilter } from './components/DateRangeFilter';
import { SummaryCards } from './components/SummaryCards';
import { CostChart } from './components/CostChart';
import { CostEfficiency } from './components/CostEfficiency';
import { ModelBreakdown } from './components/ModelBreakdown';
import { UserSummary } from './components/UserSummary';
import { UserPlanROI } from './components/UserPlanROI';
import { GroupByTabs } from './components/GroupByTabs';
import { RawEventsTable } from './components/RawEventsTable';
import { DataTable } from './components/DataTable';
import { daysAgo, today } from './dateUtil';

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
        {(
          [
            ['claude-code', 'Claude Code'],
            ['raw-events', 'Raw Events'],
          ] as [Tab, string][]
        ).map(([key, label]) => (
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
  from,
  to,
  groupBy,
  onGroupByChange,
}: {
  from: string;
  to: string;
  groupBy: string;
  onGroupByChange: (v: string) => void;
}) {
  const dayData = useUsageData({ from, to, groupBy: 'day' });
  const modelData = useUsageData({ from, to, groupBy: 'model' });
  const otherGroupData = useUsageData({
    from,
    to,
    groupBy,
    enabled: !['day', 'model', 'plan', 'cost'].includes(groupBy),
  });
  const activeGroupData =
    groupBy === 'day' ? dayData : groupBy === 'model' ? modelData : otherGroupData;

  return (
    <div>
      {dayData.loading && <p className="info">読み込み中...</p>}
      {dayData.error && <p className="error">エラー: {dayData.error}</p>}
      {dayData.data && <SummaryCards data={dayData.data} />}
      {dayData.data && <CostChart data={dayData.data} />}

      <GroupByTabs value={groupBy} onChange={onGroupByChange} />

      {activeGroupData.loading && <p className="info">読み込み中...</p>}
      {activeGroupData.error && <p className="error">エラー: {activeGroupData.error}</p>}

      {activeGroupData.data && groupBy === 'model' && (
        <ModelBreakdown data={activeGroupData.data} />
      )}
      {activeGroupData.data && groupBy === 'user' && (
        <UserSummary data={activeGroupData.data} from={from} to={to} />
      )}
      {activeGroupData.data && groupBy === 'day' && (
        <div className="card">
          <h3>日別詳細</h3>
          <DataTable data={activeGroupData.data} labelKey="date" />
        </div>
      )}
      {activeGroupData.data && ['terminal', 'version', 'speed'].includes(groupBy) && (
        <div className="card">
          <h3>{groupBy} 別詳細</h3>
          <DataTable data={activeGroupData.data} labelKey="key" />
        </div>
      )}
      {groupBy === 'cost' && <CostEfficiency data={modelData.data} />}
      {groupBy === 'plan' && <UserPlanROI />}
    </div>
  );
}
