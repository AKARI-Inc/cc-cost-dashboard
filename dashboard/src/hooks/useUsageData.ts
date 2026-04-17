import { useState, useEffect, useMemo } from 'react';

export type UsageRow = {
  date?: string;
  model?: string;
  user_email?: string;
  key?: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  request_count: number;
};

type UsageResult = {
  data: UsageRow[] | null;
  loading: boolean;
  error: string | null;
};

type Bucket = {
  date: string;
  key: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
  request_count: number;
};

const S3_FILES: Record<string, string> = {
  day: '/data/summary/per-day-per-model.json',
  model: '/data/summary/per-day-per-model.json',
  user: '/data/summary/per-day-per-user.json',
  terminal: '/data/summary/per-day-per-terminal.json',
  version: '/data/summary/per-day-per-version.json',
  speed: '/data/summary/per-day-per-speed.json',
};

function toUsageRows(buckets: Bucket[], groupBy: string): UsageRow[] {
  if (groupBy === 'day') {
    const byDate = new Map<string, UsageRow>();
    for (const b of buckets) {
      const existing = byDate.get(b.date);
      if (existing) {
        existing.total_cost_usd += b.total_cost_usd;
        existing.input_tokens += b.input_tokens;
        existing.output_tokens += b.output_tokens;
        existing.cache_read_tokens += b.cache_read_tokens ?? 0;
        existing.cache_creation_tokens += b.cache_creation_tokens ?? 0;
        existing.request_count += b.request_count;
      } else {
        byDate.set(b.date, {
          date: b.date,
          total_cost_usd: b.total_cost_usd,
          input_tokens: b.input_tokens,
          output_tokens: b.output_tokens,
          cache_read_tokens: b.cache_read_tokens ?? 0,
          cache_creation_tokens: b.cache_creation_tokens ?? 0,
          request_count: b.request_count,
        });
      }
    }
    return Array.from(byDate.values());
  }

  return buckets.map((b) => {
    const row: UsageRow = {
      date: b.date,
      key: b.key,
      total_cost_usd: b.total_cost_usd,
      input_tokens: b.input_tokens,
      output_tokens: b.output_tokens,
      cache_read_tokens: b.cache_read_tokens ?? 0,
      cache_creation_tokens: b.cache_creation_tokens ?? 0,
      request_count: b.request_count,
    };
    if (groupBy === 'model') row.model = b.key;
    if (groupBy === 'user') row.user_email = b.key;
    return row;
  });
}

// groupBy ごとに異なるファイル → S3 URL を使い分ける
function aggregateByKey(rows: UsageRow[], groupBy: string): UsageRow[] {
  if (groupBy === 'day') return rows;

  const labelField = groupBy === 'model' ? 'model'
    : groupBy === 'user' ? 'user_email'
    : 'key';

  const byKey = new Map<string, UsageRow>();
  for (const r of rows) {
    const k = String((r as Record<string, unknown>)[labelField] ?? r.key ?? '');
    const existing = byKey.get(k);
    if (existing) {
      existing.total_cost_usd += r.total_cost_usd;
      existing.input_tokens += r.input_tokens;
      existing.output_tokens += r.output_tokens;
      existing.cache_read_tokens += r.cache_read_tokens;
      existing.cache_creation_tokens += r.cache_creation_tokens;
      existing.request_count += r.request_count;
    } else {
      byKey.set(k, { ...r });
    }
  }
  return Array.from(byKey.values());
}

export function useUsageData(params: {
  from: string;
  to: string;
  groupBy: string;
  enabled?: boolean;
}): UsageResult {
  const enabled = params.enabled ?? true;
  const [buckets, setBuckets] = useState<Bucket[] | null>(null);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<string | null>(null);

  const file = S3_FILES[params.groupBy] ?? S3_FILES.day;

  useEffect(() => {
    if (!enabled) {
      setBuckets(null);
      setLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    fetch(file)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((json) => {
        if (!cancelled) {
          setBuckets(json.data ?? []);
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
  }, [file, enabled]);

  const data = useMemo(() => {
    if (!buckets) return null;

    // 日付範囲フィルタ
    const filtered = buckets.filter(
      (b) => b.date >= params.from && b.date <= params.to,
    );

    const rows = toUsageRows(filtered, params.groupBy);
    return aggregateByKey(rows, params.groupBy);
  }, [buckets, params.from, params.to, params.groupBy]);

  return { data, loading, error };
}
