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

type UserSkillBucket = {
  date: string;
  user_email: string;
  skill_name: string;
  skill_source?: string;
  plugin_name?: string;
  use_count: number;
};

type UserSessionBucket = {
  date: string;
  user_email: string;
  session_id: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  request_count: number;
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

export type SkillBreakdownRow = {
  skill_name: string;
  skill_source?: string;
  plugin_name?: string;
  use_count: number;
};

export type SessionStats = {
  session_count: number;
  avg_requests: number;
  avg_cost_usd: number;
  max_cost_usd: number;
  total_requests: number;
};

type DetailResult = {
  models: ModelBreakdownRow[];
  tools: ToolBreakdownRow[];
  skills: SkillBreakdownRow[];
  sessions: SessionStats;
  loading: boolean;
  error: string | null;
};

type Cache = {
  models?: UserModelBucket[];
  tools?: UserToolBucket[];
  skills?: UserSkillBucket[];
  sessions?: UserSessionBucket[];
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
    load('/data/summary/per-day-per-user-skill.json').catch(() => ({ data: [] })),
    load('/data/summary/per-day-per-user-session.json').catch(() => ({ data: [] })),
  ])
    .then(([m, t, s, ss]) => {
      cache.models = m.data ?? [];
      cache.tools = t.data ?? [];
      cache.skills = s.data ?? [];
      cache.sessions = ss.data ?? [];
      cache.loading = false;
      notify();
    })
    .catch((err) => {
      cache.error = err.message;
      cache.loading = false;
      notify();
    });
}

const EMPTY_SESSIONS: SessionStats = {
  session_count: 0,
  avg_requests: 0,
  avg_cost_usd: 0,
  max_cost_usd: 0,
  total_requests: 0,
};

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
      return {
        models: [],
        tools: [],
        skills: [],
        sessions: EMPTY_SESSIONS,
        loading: true,
        error: null,
      };
    }
    if (cache.error) {
      return {
        models: [],
        tools: [],
        skills: [],
        sessions: EMPTY_SESSIONS,
        loading: false,
        error: cache.error,
      };
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

    const skillMap = new Map<string, SkillBreakdownRow>();
    for (const b of cache.skills ?? []) {
      if (!inRange(b.date) || !matchUser(b.user_email)) continue;
      const cur = skillMap.get(b.skill_name);
      if (cur) {
        cur.use_count += b.use_count;
        if (!cur.skill_source && b.skill_source) cur.skill_source = b.skill_source;
        if (!cur.plugin_name && b.plugin_name) cur.plugin_name = b.plugin_name;
      } else {
        skillMap.set(b.skill_name, {
          skill_name: b.skill_name,
          skill_source: b.skill_source,
          plugin_name: b.plugin_name,
          use_count: b.use_count,
        });
      }
    }

    const models = Array.from(modelMap.values()).sort(
      (a, b) => b.total_cost_usd - a.total_cost_usd,
    );
    const tools = Array.from(toolMap.values()).sort(
      (a, b) => b.request_count - a.request_count,
    );
    const skills = Array.from(skillMap.values()).sort(
      (a, b) => b.use_count - a.use_count,
    );

    const sessionKeys = new Set<string>();
    const sessionCostByKey = new Map<string, { cost: number; reqs: number }>();
    for (const b of cache.sessions ?? []) {
      if (!inRange(b.date) || !matchUser(b.user_email)) continue;
      // 同じ session_id が複数日にまたがる場合があるので合算する
      const cur = sessionCostByKey.get(b.session_id);
      if (cur) {
        cur.cost += b.total_cost_usd;
        cur.reqs += b.request_count;
      } else {
        sessionCostByKey.set(b.session_id, {
          cost: b.total_cost_usd,
          reqs: b.request_count,
        });
      }
      sessionKeys.add(b.session_id);
    }
    const sessionCount = sessionKeys.size;
    let totalReqs = 0;
    let totalCost = 0;
    let maxCost = 0;
    for (const v of sessionCostByKey.values()) {
      totalReqs += v.reqs;
      totalCost += v.cost;
      if (v.cost > maxCost) maxCost = v.cost;
    }
    const sessions: SessionStats = {
      session_count: sessionCount,
      avg_requests: sessionCount > 0 ? totalReqs / sessionCount : 0,
      avg_cost_usd: sessionCount > 0 ? totalCost / sessionCount : 0,
      max_cost_usd: maxCost,
      total_requests: totalReqs,
    };

    return { models, tools, skills, sessions, loading: false, error: null };
  }, [params.userEmail, params.from, params.to]);
}
