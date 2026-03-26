import { Layout } from './components/Layout';
import { DateRangePicker } from './components/DateRangePicker';
import { SummaryCards } from './components/SummaryCards';
import { CostChart } from './components/CostChart';
import { UserCostTable } from './components/UserCostTable';
import { ModelBreakdown } from './components/ModelBreakdown';
import { ProductivityChart } from './components/ProductivityChart';
import { ToolAcceptanceChart } from './components/ToolAcceptanceChart';
import { Spinner } from './components/Spinner';
import { useDateRange } from './hooks/useDateRange';
import { useSheetData } from './hooks/useSheetData';

export default function App() {
  const { startDate, endDate, preset, setPreset, setCustomRange, loading: dateLoading } = useDateRange();
  const { users, dailyCost, modelBreakdown, dailyProductivity, loading, error } = useSheetData(startDate, endDate);

  if (dateLoading) return <Spinner />;

  return (
    <Layout
      controls={
        <DateRangePicker
          startDate={startDate}
          endDate={endDate}
          preset={preset}
          onPreset={setPreset}
          onCustomRange={setCustomRange}
        />
      }
    >
      {error && (
        <div style={{ color: '#ef4444', padding: '1rem', background: '#fef2f2', borderRadius: 8, marginBottom: '1rem' }}>
          {error}
        </div>
      )}

      {loading ? (
        <Spinner />
      ) : (
        <>
          <SummaryCards users={users} />
          <CostChart data={dailyCost} />
          <UserCostTable users={users} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '1.5rem' }}>
            <ModelBreakdown data={modelBreakdown} />
            <ToolAcceptanceChart users={users} />
          </div>
          <ProductivityChart data={dailyProductivity} />
        </>
      )}
    </Layout>
  );
}
