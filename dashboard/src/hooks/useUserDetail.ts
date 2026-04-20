import { useEffect, useMemo, useState } from 'react';

type UserModelBucket = {
  date: string;
  user_email: string;
  model: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  request_count: number;
};

type UserToolBucket = {
  date: string;
  user_email: string;
  tool_name: string;
  request_count: number;
};

type UserTerminalBucket = {
  date: string;
  user_email: string;
  terminal_type: string;
  os_type?: string;
  request_count: number;
  total_cost_usd: number;
};

export type ModelBreakdownRow = {
  model: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  request_count: number;
};

export type ToolBreakdownRow = {
  tool_name: string;
  request_count: number;
};

export type TerminalBreakdownRow = {
  terminal_type: string;
  os_type?: string;
  request_count: number;
  total_cost_usd: number;
};

type DetailResult = {
  models: ModelBreakdownRow[];
  tools: ToolBreakdownRow[];
  terminals: TerminalBreakdownRow[];
  loading: boolean;
  error: string | null;
};

type Cache = {
  models?: UserModelBucket[];
  tools?: UserToolBucket[];
  terminals?: UserTerminalBucket[];
  error?: string;
  loading: boolean;
};

const cache: Cache = { loading: false };
const listeners = new Set<() => void>();

function notify() {
  for (const fn of listeners) fn();
}

function ensureLoaded() {
  if (cache.models && cache.tools) return;
  if (cache.loading) return;
  cache.loading = true;
  notify();

  const load = (path: string) =>
    fetch(path).then((r) => {
      if (!r.ok) throw new Error(`HTTP ${r.status}`);
      return r.json();
    });

  Promise.all([
    load('/data/summary/per-day-per-user-model.json'),
    load('/data/summary/per-day-per-user-tool.json'),
    load('/data/summary/per-day-per-user-terminal.json').catch(() => ({ data: [] })),
  ])
    .then(([m, t, tm]) => {
      cache.models = m.data ?? [];
      cache.tools = t.data ?? [];
      cache.terminals = tm.data ?? [];
      cache.loading = false;
      notify();
    })
    .catch((err) => {
      cache.error = err.message;
      cache.loading = false;
      notify();
    });
}

export function useUserDetail(params: {
  userEmail: string;
  from: string;
  to: string;
}): DetailResult {
  const [, tick] = useState(0);

  useEffect(() => {
    const listener = () => tick((n) => n + 1);
    listeners.add(listener);
    ensureLoaded();
    return () => {
      listeners.delete(listener);
    };
  }, []);

  return useMemo(() => {
    if (cache.loading || (!cache.models && !cache.error)) {
      return { models: [], tools: [], terminals: [], loading: true, error: null };
    }
    if (cache.error) {
      return { models: [], tools: [], terminals: [], loading: false, error: cache.error };
    }

    const inRange = (d: string) => d >= params.from && d <= params.to;
    const matchUser = (email: string) =>
      email === params.userEmail ||
      (params.userEmail === '(unknown)' && !email);

    const modelMap = new Map<string, ModelBreakdownRow>();
    for (const b of cache.models ?? []) {
      if (!inRange(b.date) || !matchUser(b.user_email)) continue;
      const cur = modelMap.get(b.model);
      if (cur) {
        cur.total_cost_usd += b.total_cost_usd;
        cur.input_tokens += b.input_tokens;
        cur.output_tokens += b.output_tokens;
        cur.cache_read_tokens += b.cache_read_tokens;
        cur.cache_creation_tokens += b.cache_creation_tokens;
        cur.request_count += b.request_count;
      } else {
        modelMap.set(b.model, {
          model: b.model,
          total_cost_usd: b.total_cost_usd,
          input_tokens: b.input_tokens,
          output_tokens: b.output_tokens,
          cache_read_tokens: b.cache_read_tokens,
          cache_creation_tokens: b.cache_creation_tokens,
          request_count: b.request_count,
        });
      }
    }

    const toolMap = new Map<string, ToolBreakdownRow>();
    for (const b of cache.tools ?? []) {
      if (!inRange(b.date) || !matchUser(b.user_email)) continue;
      const cur = toolMap.get(b.tool_name);
      if (cur) {
        cur.request_count += b.request_count;
      } else {
        toolMap.set(b.tool_name, {
          tool_name: b.tool_name,
          request_count: b.request_count,
        });
      }
    }

    const termMap = new Map<string, TerminalBreakdownRow>();
    for (const b of cache.terminals ?? []) {
      if (!inRange(b.date) || !matchUser(b.user_email)) continue;
      const key = `${b.terminal_type}::${b.os_type ?? ''}`;
      const cur = termMap.get(key);
      if (cur) {
        cur.request_count += b.request_count;
        cur.total_cost_usd += b.total_cost_usd;
      } else {
        termMap.set(key, {
          terminal_type: b.terminal_type,
          os_type: b.os_type,
          request_count: b.request_count,
          total_cost_usd: b.total_cost_usd,
        });
      }
    }

    const models = Array.from(modelMap.values()).sort(
      (a, b) => b.total_cost_usd - a.total_cost_usd,
    );
    const tools = Array.from(toolMap.values()).sort(
      (a, b) => b.request_count - a.request_count,
    );
    const terminals = Array.from(termMap.values()).sort(
      (a, b) => b.request_count - a.request_count,
    );

    return { models, tools, terminals, loading: false, error: null };
  }, [params.userEmail, params.from, params.to]);
}
