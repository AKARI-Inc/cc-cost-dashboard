import { useCallback, useEffect, useState } from 'react';
import { callGas } from '../gas';
import type {
  DailyCostEntry,
  DailyProductivityEntry,
  ModelBreakdownEntry,
  UserSummary,
} from '../types';

interface SheetData {
  users: UserSummary[];
  dailyCost: DailyCostEntry[];
  modelBreakdown: ModelBreakdownEntry[];
  dailyProductivity: DailyProductivityEntry[];
  loading: boolean;
  error: string | null;
}

export function useSheetData(startDate: string | null, endDate: string | null): SheetData {
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [dailyCost, setDailyCost] = useState<DailyCostEntry[]>([]);
  const [modelBreakdown, setModelBreakdown] = useState<ModelBreakdownEntry[]>([]);
  const [dailyProductivity, setDailyProductivity] = useState<DailyProductivityEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    setError(null);

    try {
      const [usersData, costData, modelData, prodData] = await Promise.all([
        callGas<UserSummary[]>('getSummaryByUser', startDate, endDate),
        callGas<DailyCostEntry[]>('getDailyCostTrend', startDate, endDate),
        callGas<ModelBreakdownEntry[]>('getModelBreakdown', startDate, endDate),
        callGas<DailyProductivityEntry[]>('getDailyProductivity', startDate, endDate),
      ]);

      setUsers(usersData);
      setDailyCost(costData);
      setModelBreakdown(modelData);
      setDailyProductivity(prodData);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'データ取得に失敗しました');
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  return { users, dailyCost, modelBreakdown, dailyProductivity, loading, error };
}
