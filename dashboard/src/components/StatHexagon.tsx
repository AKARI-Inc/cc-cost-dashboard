import type {
  SessionStats,
  SkillBreakdownRow,
  ToolBreakdownRow,
} from '../hooks/useUserDetail';
import type { UsageRow } from '../hooks/useUsageData';

type Props = {
  row: UsageRow;
  tools: ToolBreakdownRow[];
  skills: SkillBreakdownRow[];
  sessions: SessionStats;
};

type Bracket = { A: number; B: number; C: number; D: number };

const THRESHOLDS: Record<string, Bracket> = {
  ioRatio: { A: 0.95, B: 0.92, C: 0.87, D: 0.8 },
  cacheRate: { A: 0.97, B: 0.96, C: 0.95, D: 0.93 },
  sessionDensity: { A: 100, B: 70, C: 45, D: 25 },
  tokensPerReq: { A: 700, B: 600, C: 500, D: 400 },
  toolRate: { A: 2.8, B: 2.5, C: 2.2, D: 1.9 },
  skillRate: { A: 0.005, B: 0.002, C: 0.001, D: 0.0001 },
};

function gradeOf(value: number, t: Bracket): string {
  if (value >= t.A) return 'A';
  if (value >= t.B) return 'B';
  if (value >= t.C) return 'C';
  if (value >= t.D) return 'D';
  return 'E';
}

function scoreFor(value: number, t: Bracket): number {
  return Math.max(0, Math.min(1, value / t.A));
}

export function StatHexagon({ row, tools, skills, sessions }: Props) {
  const totalTokens = row.input_tokens + row.output_tokens;
  const totalInputLike =
    row.input_tokens + row.cache_read_tokens + row.cache_creation_tokens;
  const totalToolCalls = tools.reduce((s, t) => s + t.request_count, 0);
  const totalSkillCalls = skills.reduce((s, x) => s + x.use_count, 0);

  const ioRatio = totalTokens > 0 ? row.output_tokens / totalTokens : 0;
  const cacheRate = totalInputLike > 0 ? row.cache_read_tokens / totalInputLike : 0;
  const sessionDensity =
    sessions.session_count > 0 ? row.request_count / sessions.session_count : 0;
  const tokensPerReq = row.request_count > 0 ? totalTokens / row.request_count : 0;
  const toolRate = row.request_count > 0 ? totalToolCalls / row.request_count : 0;
  const skillRate = row.request_count > 0 ? totalSkillCalls / row.request_count : 0;

  const stats = [
    {
      label: '入出力比',
      value: ioRatio,
      display: ioRatio.toFixed(2),
      threshold: THRESHOLDS.ioRatio,
    },
    {
      label: 'キャッシュ率',
      value: cacheRate,
      display: `${(cacheRate * 100).toFixed(0)}%`,
      threshold: THRESHOLDS.cacheRate,
    },
    {
      label: 'セッション密度',
      value: sessionDensity,
      display: `${sessionDensity.toFixed(1)} req`,
      threshold: THRESHOLDS.sessionDensity,
    },
    {
      label: '1req トークン',
      value: tokensPerReq,
      display: `${Math.round(tokensPerReq).toLocaleString()} tok`,
      threshold: THRESHOLDS.tokensPerReq,
    },
    {
      label: 'ツール率',
      value: toolRate,
      display: toolRate.toFixed(2),
      threshold: THRESHOLDS.toolRate,
    },
    {
      label: 'Skill 率',
      value: skillRate,
      display: skillRate.toFixed(2),
      threshold: THRESHOLDS.skillRate,
    },
  ];

  const cx = 230;
  const cy = 230;
  const R = 140;
  const angleAt = (i: number) => -Math.PI / 2 + (i * Math.PI) / 3;
  const pointAt = (i: number, scale: number): [number, number] => {
    const a = angleAt(i);
    return [cx + R * scale * Math.cos(a), cy + R * scale * Math.sin(a)];
  };

  const hexPath = (scale: number) =>
    stats
      .map((_, i) => pointAt(i, scale))
      .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`)
      .join(' ') + ' Z';

  const dataPath =
    stats
      .map((s, i) => pointAt(i, Math.max(0.03, scoreFor(s.value, s.threshold))))
      .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`)
      .join(' ') + ' Z';

  return (
    <div className="hex-panel">
      <svg viewBox="0 0 460 470" className="stat-hex" role="img" aria-label="能力値">
        {[1, 0.8, 0.6, 0.4, 0.2].map((l) => (
          <path key={l} d={hexPath(l)} className="hex-grid" />
        ))}
        {stats.map((_, i) => {
          const [x, y] = pointAt(i, 1);
          return (
            <line key={i} x1={cx} y1={cy} x2={x} y2={y} className="hex-axis" />
          );
        })}
        <path d={dataPath} className="hex-data" />
        {stats.map((s, i) => {
          const [px, py] = pointAt(i, Math.max(0.03, scoreFor(s.value, s.threshold)));
          return <circle key={i} cx={px} cy={py} r={4.5} className="hex-dot" />;
        })}
        {stats.map((s, i) => {
          const a = angleAt(i);
          const [lx, ly] = pointAt(i, 1.28);
          const anchor: 'start' | 'middle' | 'end' =
            Math.abs(Math.cos(a)) < 0.15 ? 'middle' : Math.cos(a) > 0 ? 'start' : 'end';
          const isTop = Math.sin(a) < -0.5;
          const isBottom = Math.sin(a) > 0.5;
          const nameY = isTop ? ly - 18 : isBottom ? ly + 14 : ly - 8;
          const gradeY = nameY + 26;
          return (
            <g key={s.label}>
              <text x={lx} y={nameY} textAnchor={anchor} className="hex-label-name">
                {s.label}
              </text>
              <text x={lx} y={gradeY} textAnchor={anchor} className="hex-label-grade">
                {gradeOf(s.value, s.threshold)}
              </text>
            </g>
          );
        })}
      </svg>
      <div className="hex-legend">
        {stats.map((s) => (
          <span key={s.label}>
            <strong>{s.label}</strong>
            {s.display}
          </span>
        ))}
      </div>
    </div>
  );
}
