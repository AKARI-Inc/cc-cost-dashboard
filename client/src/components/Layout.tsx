import type { ReactNode } from 'react';

const styles: Record<string, React.CSSProperties> = {
  wrapper: {
    maxWidth: 1200,
    margin: '0 auto',
    padding: '1.5rem',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    color: '#1f2937',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1.5rem',
    paddingBottom: '1rem',
    borderBottom: '1px solid #e5e7eb',
  },
  title: {
    fontSize: '1.5rem',
    fontWeight: 700,
    color: '#111827',
    margin: 0,
  },
  subtitle: {
    fontSize: '0.875rem',
    color: '#6b7280',
    margin: 0,
  },
};

interface LayoutProps {
  children: ReactNode;
  controls?: ReactNode;
}

export function Layout({ children, controls }: LayoutProps) {
  return (
    <div style={styles.wrapper}>
      <div style={styles.header}>
        <div>
          <h1 style={styles.title}>Claude Code Cost Dashboard</h1>
          <p style={styles.subtitle}>Organization usage analytics</p>
        </div>
        {controls}
      </div>
      {children}
    </div>
  );
}
