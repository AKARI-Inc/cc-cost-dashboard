type Props = {
  value: string;
  onChange: (v: string) => void;
};

const tabs = [
  { key: 'day', label: 'Day' },
  { key: 'model', label: 'Model' },
  { key: 'user', label: 'User' },
  { key: 'terminal', label: 'Terminal' },
  { key: 'version', label: 'Version' },
  { key: 'speed', label: 'Speed' },
];

export function GroupByTabs({ value, onChange }: Props) {
  return (
    <div className="group-tabs">
      {tabs.map((t) => (
        <button
          key={t.key}
          className={`btn btn-sm ${value === t.key ? 'btn-active' : ''}`}
          onClick={() => onChange(t.key)}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}
