export interface DateRange {
  min: string | null;
  max: string | null;
}

export interface UserSummary {
  email: string;
  totalCostCents: number;
  totalTokensInput: number;
  totalTokensOutput: number;
  totalSessions: number;
  linesAdded: number;
  linesRemoved: number;
  commits: number;
  pullRequests: number;
  editAccepted: number;
  editRejected: number;
  writeAccepted: number;
  writeRejected: number;
  activeDays: number;
}

export interface DailyCostEntry {
  date: string;
  email: string;
  costCents: number;
}

export interface ModelBreakdownEntry {
  model: string;
  costCents: number;
  tokensInput: number;
  tokensOutput: number;
}

export interface DailyProductivityEntry {
  date: string;
  sessions: number;
  commits: number;
  pullRequests: number;
}
