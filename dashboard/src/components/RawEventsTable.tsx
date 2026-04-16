import { useState, useEffect, Fragment } from 'react';

// UTC タイムスタンプを JST (Asia/Tokyo) 表示に変換
function toJST(ts: string): string {
  if (!ts) return '';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return ts;
  return new Intl.DateTimeFormat('ja-JP', {
    timeZone: 'Asia/Tokyo',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(d);
}

type RawEvent = {
  timestamp: string;
  event_name: string;
  user_email: string;
  model: string;
  cost_usd: number;
  raw_attributes: Record<string, unknown>;
};

type Props = {
  from: string;
  to: string;
};

const ORDER_OPTIONS = [
  { value: 'desc', label: '新しい順' },
  { value: 'asc', label: '古い順' },
];

export function RawEventsTable({ from, to }: Props) {
  const [events, setEvents] = useState<RawEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [eventName, setEventName] = useState('');
  const [userEmail, setUserEmail] = useState('');
  const [order, setOrder] = useState('desc');
  const [limit, setLimit] = useState(500);
  const [expanded, setExpanded] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams({
      from,
      to,
      limit: String(limit),
      order,
    });
    if (eventName) params.set('eventName', eventName);
    if (userEmail) params.set('userEmail', userEmail);

    fetch(`/api/claude-code/events?${params}`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((json) => {
        if (!cancelled) {
          setEvents(json.data ?? []);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [from, to, eventName, userEmail, order, limit]);

  return (
    <div className="card">
      <h3>Raw Events</h3>
      <div className="filter-row" style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', alignItems: 'center' }}>
        <label>
          イベント名:
          <input
            type="text"
            value={eventName}
            onChange={(e) => setEventName(e.target.value)}
            placeholder="例: claude_code.api_request"
          />
        </label>
        <label>
          ユーザー:
          <input
            type="text"
            value={userEmail}
            onChange={(e) => setUserEmail(e.target.value)}
            placeholder="例: alice@example.com"
          />
        </label>
        <label>
          並び順:
          <select value={order} onChange={(e) => setOrder(e.target.value)}>
            {ORDER_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </label>
        <label>
          件数:
          <select value={limit} onChange={(e) => setLimit(Number(e.target.value))}>
            <option value={100}>100</option>
            <option value={500}>500</option>
            <option value={1000}>1000</option>
            <option value={5000}>5000</option>
          </select>
        </label>
      </div>

      {loading && <p className="info">読み込み中...</p>}
      {error && <p className="error">エラー: {error}</p>}

      {!loading && !error && (
        <>
          <p className="info" style={{ marginTop: 8 }}>
            表示中: {events.length} 件 ({order === 'desc' ? '新しい順' : '古い順'})
          </p>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>timestamp (JST)</th>
                  <th>event_name</th>
                  <th>user_email</th>
                  <th>model</th>
                  <th className="num">cost_usd</th>
                  <th>detail</th>
                </tr>
              </thead>
              <tbody>
                {events.map((ev, i) => (
                  <Fragment key={`${ev.timestamp}-${i}`}>
                    <tr>
                      <td>{toJST(ev.timestamp)}</td>
                      <td>{ev.event_name}</td>
                      <td>{ev.user_email}</td>
                      <td>{ev.model}</td>
                      <td className="num">${(ev.cost_usd ?? 0).toFixed(4)}</td>
                      <td>
                        <button className="btn btn-sm" onClick={() => setExpanded(expanded === i ? null : i)}>
                          {expanded === i ? '閉じる' : '展開'}
                        </button>
                      </td>
                    </tr>
                    {expanded === i && (
                      <tr>
                        <td colSpan={6}>
                          <pre className="json-detail">{JSON.stringify(ev.raw_attributes, null, 2)}</pre>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                ))}
                {events.length === 0 && (
                  <tr><td colSpan={6} style={{ textAlign: 'center' }}>データなし</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
