import { daysAgo, today } from '../dateUtil';

type Props = {
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
};

// 「1年」プリセットは generator Lambda の lookbackDays = 90 (応急対応) に合わせて
// 一時的に削除。データ自体は CloudWatch Logs に残っているので
// 根本対応 (docs/generator-redesign.md) 後に復活させる。
const presets = [
  { label: '7日', days: 7 },
  { label: '30日', days: 30 },
  { label: '90日', days: 90 },
];

export function DateRangeFilter({ from, to, onChange }: Props) {
  return (
    <div className="date-range-filter">
      {presets.map((p) => (
        <button
          key={p.days}
          className="btn btn-sm"
          onClick={() => onChange(daysAgo(p.days), today())}
        >
          {p.label}
        </button>
      ))}
      <input type="date" value={from} onChange={(e) => onChange(e.target.value, to)} />
      <span>〜</span>
      <input type="date" value={to} onChange={(e) => onChange(from, e.target.value)} />
    </div>
  );
}
