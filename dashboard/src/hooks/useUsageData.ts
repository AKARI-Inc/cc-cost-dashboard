import { useState, useEffect } from 'react';

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
};

export function useUsageData(params: {
  from: string;
  to: string;
  groupBy: string;
  enabled?: boolean;
}): UsageResult {
  const enabled = params.enabled ?? true;
  const [data, setData] = useState<UsageRow[] | null>(null);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) {
      setData(null);
      setLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    const qs = new URLSearchParams({
      from: params.from,
      to: params.to,
      groupBy: params.groupBy,
    });

    fetch(`/api/claude-code/usage?${qs}`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((json) => {
        if (!cancelled) {
          setData((json.data ?? []) as UsageRow[]);
          setLoading(false);
        }
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
  }, [params.from, params.to, params.groupBy, enabled]);

  return { data, loading, error };
}
