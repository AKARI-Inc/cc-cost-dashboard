import type { Preset } from '../hooks/useDateRange';

const presets: { value: Preset; label: string }[] = [
  { value: '7d', label: '7日間' },
  { value: '30d', label: '30日間' },
  { value: 'this_month', label: '今月' },
  { value: 'last_month', label: '先月' },
  { value: 'custom', label: 'カスタム' },
];

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    flexWrap: 'wrap',
  },
  btn: {
    padding: '0.375rem 0.75rem',
    fontSize: '0.8125rem',
    border: '1px solid #d1d5db',
    borderRadius: 6,
    cursor: 'pointer',
    background: '#fff',
    color: '#374151',
    transition: 'all 0.15s',
  },
  btnActive: {
    background: '#6366f1',
    color: '#fff',
    borderColor: '#6366f1',
  },
  input: {
    padding: '0.375rem 0.5rem',
    fontSize: '0.8125rem',
    border: '1px solid #d1d5db',
    borderRadius: 6,
    color: '#374151',
  },
};

interface DateRangePickerProps {
  startDate: string | null;
  endDate: string | null;
  preset: Preset;
  onPreset: (p: Preset) => void;
  onCustomRange: (start: string, end: string) => void;
}

export function DateRangePicker({ startDate, endDate, preset, onPreset, onCustomRange }: DateRangePickerProps) {
  return (
    <div style={styles.container}>
      {presets.map((p) => (
        <button
          key={p.value}
          style={{ ...styles.btn, ...(preset === p.value ? styles.btnActive : {}) }}
          onClick={() => onPreset(p.value)}
        >
          {p.label}
        </button>
      ))}
      {preset === 'custom' && (
        <>
          <input
            type="date"
            style={styles.input}
            value={startDate ?? ''}
            onChange={(e) => onCustomRange(e.target.value, endDate ?? '')}
          />
          <span style={{ color: '#9ca3af' }}>~</span>
          <input
            type="date"
            style={styles.input}
            value={endDate ?? ''}
            onChange={(e) => onCustomRange(startDate ?? '', e.target.value)}
          />
        </>
      )}
    </div>
  );
}
