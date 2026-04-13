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
  endpoint: string;
  from: string;
  to: string;
  groupBy: string;
}): UsageResult {
  const [data, setData] = useState<UsageRow[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    const url = `${params.endpoint}?from=${params.from}&to=${params.to}&groupBy=${params.groupBy}`;
    fetch(url)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((json) => {
        if (!cancelled) {
          setData(json.data ?? []);
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
  }, [params.endpoint, params.from, params.to, params.groupBy]);

  return { data, loading, error };
}
