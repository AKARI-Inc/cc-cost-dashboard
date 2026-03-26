import { useCallback, useEffect, useState } from 'react';
import { format, subDays, startOfMonth, endOfMonth, subMonths } from 'date-fns';
import { callGas } from '../gas';
import type { DateRange } from '../types';

export type Preset = '7d' | '30d' | 'this_month' | 'last_month' | 'custom';

interface DateRangeState {
  startDate: string | null;
  endDate: string | null;
  availableRange: DateRange;
  preset: Preset;
  setPreset: (preset: Preset) => void;
  setCustomRange: (start: string, end: string) => void;
  loading: boolean;
}

export function useDateRange(): DateRangeState {
  const [availableRange, setAvailableRange] = useState<DateRange>({ min: null, max: null });
  const [startDate, setStartDate] = useState<string | null>(null);
  const [endDate, setEndDate] = useState<string | null>(null);
  const [preset, setPresetState] = useState<Preset>('30d');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    callGas<DateRange>('getAvailableDateRange').then((range) => {
      setAvailableRange(range);
      if (range.max) {
        applyPreset('30d', range.max);
      }
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const applyPreset = useCallback((p: Preset, refDate?: string) => {
    const ref = refDate ?? endDate;
    if (!ref) return;
    const today = new Date(ref);

    switch (p) {
      case '7d':
        setStartDate(format(subDays(today, 6), 'yyyy-MM-dd'));
        setEndDate(format(today, 'yyyy-MM-dd'));
        break;
      case '30d':
        setStartDate(format(subDays(today, 29), 'yyyy-MM-dd'));
        setEndDate(format(today, 'yyyy-MM-dd'));
        break;
      case 'this_month':
        setStartDate(format(startOfMonth(today), 'yyyy-MM-dd'));
        setEndDate(format(today, 'yyyy-MM-dd'));
        break;
      case 'last_month': {
        const lastMonth = subMonths(today, 1);
        setStartDate(format(startOfMonth(lastMonth), 'yyyy-MM-dd'));
        setEndDate(format(endOfMonth(lastMonth), 'yyyy-MM-dd'));
        break;
      }
      case 'custom':
        break;
    }
  }, [endDate]);

  const setPreset = useCallback((p: Preset) => {
    setPresetState(p);
    if (p !== 'custom') applyPreset(p);
  }, [applyPreset]);

  const setCustomRange = useCallback((start: string, end: string) => {
    setPresetState('custom');
    setStartDate(start);
    setEndDate(end);
  }, []);

  return { startDate, endDate, availableRange, preset, setPreset, setCustomRange, loading };
}
