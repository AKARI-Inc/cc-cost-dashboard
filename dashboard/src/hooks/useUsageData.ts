import { useState, useEffect } from 'react';

// generator が S3 に出力する per-day × per-key の集計レコード
type Bucket = {
  date: string;
  key: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  request_count: number;
};

export type UsageRow = {
  date?: string;
  model?: string;
  user_email?: string;
  key?: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  request_count: number;
};

type UsageResult = {
  data: UsageRow[] | null;
  loading: boolean;
  error: string | null;
  generated?: string;
};

// groupBy に応じて取得する S3 ファイルパスを返す。
// "day" は per-day-per-model を date で再集計して使う。
function summaryPathFor(groupBy: string): string {
  switch (groupBy) {
    case 'user':
      return '/data/summary/per-day-per-user.json';
    case 'terminal':
      return '/data/summary/per-day-per-terminal.json';
    case 'version':
      return '/data/summary/per-day-per-version.json';
    case 'speed':
      return '/data/summary/per-day-per-speed.json';
    case 'day':
    case 'model':
    default:
      return '/data/summary/per-day-per-model.json';
  }
}

// from <= date <= to の範囲フィルタ (date は YYYY-MM-DD 文字列、辞書順比較)
function filterByDate(buckets: Bucket[], from: string, to: string): Bucket[] {
  return buckets.filter((b) => b.date >= from && b.date <= to);
}

// groupBy に応じて再集計
function aggregate(buckets: Bucket[], groupBy: string): UsageRow[] {
  if (groupBy === 'day') {
    const byDate = new Map<string, UsageRow>();
    for (const b of buckets) {
      const cur = byDate.get(b.date) ?? {
        date: b.date,
        total_cost_usd: 0,
        input_tokens: 0,
        output_tokens: 0,
        request_count: 0,
      };
      cur.total_cost_usd += b.total_cost_usd;
      cur.input_tokens += b.input_tokens;
      cur.output_tokens += b.output_tokens;
      cur.request_count += b.request_count;
      byDate.set(b.date, cur);
    }
    return [...byDate.values()].sort((a, b) => (a.date! < b.date! ? -1 : 1));
  }

  // それ以外 (model/user/terminal/version/speed) は key で集計
  const byKey = new Map<string, UsageRow>();
  for (const b of buckets) {
    const k = b.key;
    const cur = byKey.get(k) ?? {
      total_cost_usd: 0,
      input_tokens: 0,
      output_tokens: 0,
      request_count: 0,
    };
    cur.total_cost_usd += b.total_cost_usd;
    cur.input_tokens += b.input_tokens;
    cur.output_tokens += b.output_tokens;
    cur.request_count += b.request_count;
    if (groupBy === 'model') cur.model = k;
    else if (groupBy === 'user') cur.user_email = k;
    else cur.key = k;
    byKey.set(k, cur);
  }
  return [...byKey.values()].sort((a, b) => b.total_cost_usd - a.total_cost_usd);
}

// 同じ S3 ファイルを複数フックが同時に取得しないようキャッシュ
const cache = new Map<string, Promise<{ buckets: Bucket[]; generated: string }>>();

function fetchSummary(path: string): Promise<{ buckets: Bucket[]; generated: string }> {
  const cached = cache.get(path);
  if (cached) return cached;
  const p = fetch(path)
    .then((res) => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json();
    })
    .then((json) => ({
      buckets: (json.data ?? []) as Bucket[],
      generated: json.generated ?? '',
    }))
    .catch((err) => {
      cache.delete(path);
      throw err;
    });
  cache.set(path, p);
  setTimeout(() => cache.delete(path), 60_000);
  return p;
}

/**
 * useUsageData: generator が S3 に出力した per-day × per-key 集計から
 * from〜to の範囲フィルタ + クライアント側再集計を行う。
 */
export function useUsageData(params: {
  from: string;
  to: string;
  groupBy: string;
}): UsageResult {
  const [data, setData] = useState<UsageRow[] | null>(null);
  const [generated, setGenerated] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    const path = summaryPathFor(params.groupBy);
    fetchSummary(path)
      .then(({ buckets, generated }) => {
        if (cancelled) return;
        const filtered = filterByDate(buckets, params.from, params.to);
        const rows = aggregate(filtered, params.groupBy);
        setData(rows);
        setGenerated(generated);
        setLoading(false);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [params.from, params.to, params.groupBy]);

  return { data, loading, error, generated };
}
