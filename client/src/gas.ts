declare const google: {
  script: {
    run: {
      withSuccessHandler: (cb: (result: unknown) => void) => {
        withFailureHandler: (cb: (error: Error) => void) => Record<string, (...args: unknown[]) => void>;
      };
    };
  };
};

const IS_DEV = typeof google === 'undefined' || !google.script;

export function callGas<T>(functionName: string, ...args: unknown[]): Promise<T> {
  if (IS_DEV) return callMock<T>(functionName);

  return new Promise((resolve, reject) => {
    const runner = google.script.run
      .withSuccessHandler((result) => resolve(result as T))
      .withFailureHandler(reject);
    runner[functionName]!(...args);
  });
}

// 開発用モックデータ
function callMock<T>(functionName: string): Promise<T> {
  const emails = ['alice@example.com', 'bob@example.com', 'carol@example.com', 'dave@example.com'];
  const models = ['claude-opus-4-6', 'claude-sonnet-4-6', 'claude-haiku-4-5-20251001'];

  const mocks: Record<string, unknown> = {
    getAvailableDateRange: { min: '2026-03-01', max: '2026-03-25' },
    getSummaryByUser: emails.map((email, i) => ({
      email,
      totalCostCents: (4 - i) * 2500 + Math.floor(Math.random() * 1000),
      totalTokensInput: (4 - i) * 500000,
      totalTokensOutput: (4 - i) * 150000,
      totalSessions: (4 - i) * 30 + Math.floor(Math.random() * 20),
      linesAdded: (4 - i) * 1200,
      linesRemoved: (4 - i) * 600,
      commits: (4 - i) * 15,
      pullRequests: (4 - i) * 5,
      editAccepted: (4 - i) * 40,
      editRejected: (4 - i) * 5,
      writeAccepted: (4 - i) * 10,
      writeRejected: (4 - i) * 2,
      activeDays: Math.min(25, (4 - i) * 8),
    })),
    getDailyCostTrend: Array.from({ length: 25 }, (_, day) =>
      emails.map((email, i) => ({
        date: `2026-03-${String(day + 1).padStart(2, '0')}`,
        email,
        costCents: Math.floor((4 - i) * 100 + Math.random() * 200),
      }))
    ).flat(),
    getModelBreakdown: models.map((model, i) => ({
      model,
      costCents: (3 - i) * 5000 + Math.floor(Math.random() * 2000),
      tokensInput: (3 - i) * 1000000,
      tokensOutput: (3 - i) * 300000,
    })),
    getDailyProductivity: Array.from({ length: 25 }, (_, day) => ({
      date: `2026-03-${String(day + 1).padStart(2, '0')}`,
      sessions: Math.floor(5 + Math.random() * 15),
      commits: Math.floor(3 + Math.random() * 10),
      pullRequests: Math.floor(Math.random() * 5),
    })),
  };

  return new Promise((resolve) => setTimeout(() => resolve(mocks[functionName] as T), 300));
}
