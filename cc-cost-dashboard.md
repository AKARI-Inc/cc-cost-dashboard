# Claude Code コストダッシュボード - 実装プラン

## 概要

組織（Teamプラン）のClaude Code利用状況を、GAS Webアプリで全メンバーが閲覧できるダッシュボードを構築する。

### ゴール

- **誰が**: 燈/akari IP傘下の全メンバー（管理者権限不要）
- **どこで**: GAS Webアプリ（Google Workspace ドメイン制限）
- **何を**: 人別の日次コスト・トークン消費・生産性メトリクス
- **どうやって**: GAS時限トリガーでAdmin API → Sheets蓄積 → Vite単一HTMLで可視化

### データソース

Anthropic Claude Code Analytics API（Teamプラン対応）
```
GET /v1/organizations/usage_report/claude_code?starting_at=YYYY-MM-DD&limit=1000
Header: anthropic-version: 2023-06-01
Header: x-api-key: sk-ant-admin-...
```

### スコープ外

- スキル別利用者数（Enterprise Analytics API が必要 → Teamプランでは不可）
- リアルタイム監視（APIは日次集計のみ）

---

## アーキテクチャ

```
┌─────────────────────────────────────────────────────┐
│                   GAS Web App                        │
│                                                      │
│  ┌────────────┐   UrlFetchApp     ┌───────────────┐ │
│  │ Server     │ ────────────────→ │ Anthropic     │ │
│  │ (GAS)      │ ←──────────────── │ Admin API     │ │
│  │            │                   └───────────────┘ │
│  │ Admin Key  │                                      │
│  │ in Script  │   時限トリガー(日次 JST 9:00)         │
│  │ Properties │ → Google Sheets に蓄積               │
│  └─────┬──────┘                                      │
│        │ google.script.run                           │
│        ▼                                             │
│  ┌────────────┐                                      │
│  │ Frontend   │  Vite + React + recharts             │
│  │ (単一HTML) │  vite-plugin-singlefile でビルド      │
│  └────────────┘                                      │
│                                                      │
│  アクセス: Google Workspace ドメイン制限              │
└─────────────────────────────────────────────────────┘
```

---

## ディレクトリ構成

```
cc-cost-dashboard/
├── README.md
├── package.json
├── .gitignore
│
├── client/                            # フロントエンド
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── types.ts                   # API・Sheet・UI 共通型定義
│       ├── gas.ts                     # google.script.run Promise化ラッパー
│       ├── hooks/
│       │   ├── useSheetData.ts        # GASデータ取得フック
│       │   └── useDateRange.ts        # 日付範囲選択フック
│       ├── components/
│       │   ├── Layout.tsx             # 全体レイアウト
│       │   ├── DateRangePicker.tsx    # 日付範囲 + プリセット
│       │   ├── SummaryCards.tsx       # KPI概要カード（4枚）
│       │   ├── CostChart.tsx          # 日別×ユーザー別コスト推移（折れ線）
│       │   ├── UserCostTable.tsx      # ユーザー別集約テーブル（ソート可）
│       │   ├── ModelBreakdown.tsx     # モデル別コスト割合
│       │   ├── ProductivityChart.tsx  # セッション・コミット・PR推移
│       │   ├── ToolAcceptanceChart.tsx # ツール承認率
│       │   └── Spinner.tsx
│       └── utils/
│           ├── format.ts              # 数値フォーマッタ（通貨・%等）
│           └── colors.ts              # チャートカラーパレット
│
├── appscript/                         # GAS（clasp管理）
│   ├── .clasp.json
│   ├── appsscript.json               # マニフェスト
│   ├── Code.gs                        # doGet（HTML配信）
│   ├── FetchAnalytics.gs             # Admin API呼び出し + ページネーション + リトライ
│   ├── SheetWriter.gs                # Sheets書き込み（フラット化・冪等）
│   ├── WebApp.gs                      # フロント向けデータ取得API
│   ├── Trigger.gs                     # 日次トリガー + バックフィル
│   └── client/                        # ← ビルド出力先
│       └── index.html
│
└── scripts/
    └── build.sh                       # ビルド + clasp push + deploy
```

---

## Google Sheets スキーマ

### シート: `RawData`

1行 = ユーザー × 日 × モデル（`model_breakdown` をフラット化）

| 列 | カラム名 | 型 | ソース |
|----|---------|-----|--------|
| A | `date` | Date | 集計日（UTC） |
| B | `email` | String | `actor.email_address` |
| C | `actor_type` | String | `user_actor` / `api_actor` |
| D | `organization_id` | String | 組織ID |
| E | `customer_type` | String | `api` / `subscription` |
| F | `terminal_type` | String | `vscode`, `iTerm.app` 等 |
| G | `num_sessions` | Number | セッション数 |
| H | `lines_added` | Number | 追加行数 |
| I | `lines_removed` | Number | 削除行数 |
| J | `commits` | Number | コミット数 |
| K | `pull_requests` | Number | PR数 |
| L | `edit_accepted` | Number | edit_tool.accepted |
| M | `edit_rejected` | Number | edit_tool.rejected |
| N | `multi_edit_accepted` | Number | multi_edit_tool.accepted |
| O | `multi_edit_rejected` | Number | multi_edit_tool.rejected |
| P | `write_accepted` | Number | write_tool.accepted |
| Q | `write_rejected` | Number | write_tool.rejected |
| R | `notebook_edit_accepted` | Number | notebook_edit_tool.accepted |
| S | `notebook_edit_rejected` | Number | notebook_edit_tool.rejected |
| T | `model` | String | モデル名（claude-opus-4-6 等） |
| U | `tokens_input` | Number | 入力トークン |
| V | `tokens_output` | Number | 出力トークン |
| W | `tokens_cache_read` | Number | キャッシュ読取トークン |
| X | `tokens_cache_creation` | Number | キャッシュ生成トークン |
| Y | `cost_cents` | Number | 推定コスト（セント） |
| Z | `cost_currency` | String | USD |
| AA | `fetched_at` | DateTime | データ取得日時 |

**設計判断**: `core_metrics` と `tool_actions` はユーザー×日の粒度だが、モデル行に重複して書き込む。フロントのクエリがシンプルになるメリットがSheetsのストレージコスト（無視できる）を上回る。

### シート: `SyncLog`

| 列 | カラム名 | 型 | 説明 |
|----|---------|-----|------|
| A | `date` | Date | 取得対象日 |
| B | `status` | String | `success` / `error` |
| C | `record_count` | Number | 取得レコード数 |
| D | `executed_at` | DateTime | 実行日時 |
| E | `error_message` | String | エラー時のメッセージ |

---

## APIレスポンス構造（参照用）

```json
{
  "data": [
    {
      "date": "2026-03-25T00:00:00Z",
      "actor": {
        "type": "user_actor",
        "email_address": "developer@company.com"
      },
      "organization_id": "uuid",
      "customer_type": "subscription",
      "terminal_type": "vscode",
      "core_metrics": {
        "num_sessions": 5,
        "lines_of_code": { "added": 1543, "removed": 892 },
        "commits_by_claude_code": 12,
        "pull_requests_by_claude_code": 2
      },
      "tool_actions": {
        "edit_tool": { "accepted": 45, "rejected": 5 },
        "multi_edit_tool": { "accepted": 12, "rejected": 2 },
        "write_tool": { "accepted": 8, "rejected": 1 },
        "notebook_edit_tool": { "accepted": 3, "rejected": 0 }
      },
      "model_breakdown": [
        {
          "model": "claude-opus-4-6",
          "tokens": {
            "input": 100000,
            "output": 35000,
            "cache_read": 10000,
            "cache_creation": 5000
          },
          "estimated_cost": {
            "currency": "USD",
            "amount": 1025
          }
        }
      ]
    }
  ],
  "has_more": false,
  "next_page": null
}
```

---

## GAS サーバーサイド実装

### `appsscript.json`

```json
{
  "timeZone": "Asia/Tokyo",
  "dependencies": {},
  "exceptionLogging": "STACKDRIVER",
  "runtimeVersion": "V8",
  "webapp": {
    "executeAs": "USER_DEPLOYING",
    "access": "DOMAIN"
  },
  "oauthScopes": [
    "https://www.googleapis.com/auth/spreadsheets",
    "https://www.googleapis.com/auth/script.external_request",
    "https://www.googleapis.com/auth/script.scriptapp"
  ]
}
```

- `executeAs: "USER_DEPLOYING"` → Admin API Keyにアクセスできるのはデプロイ者のみ
- `access: "DOMAIN"` → Google Workspace組織内に制限

### `Code.gs` - Webアプリエントリ

```javascript
function doGet(e) {
  return HtmlService.createHtmlOutputFromFile('client/index')
    .setTitle('Claude Code Cost Dashboard')
    .setXFrameOptionsMode(HtmlService.XFrameOptionsMode.ALLOWALL);
}
```

### `FetchAnalytics.gs` - API呼び出し

```javascript
var API_BASE = 'https://api.anthropic.com/v1/organizations/usage_report/claude_code';
var MAX_RETRIES = 3;

function fetchDailyAnalytics(dateString) {
  var allData = [];
  var page = null;

  do {
    var url = API_BASE + '?starting_at=' + dateString + '&limit=1000';
    if (page) url += '&page=' + encodeURIComponent(page);

    var result = callAnthropicAPI(url);
    allData = allData.concat(result.data);
    page = result.next_page;
  } while (result.has_more === true);

  return allData;
}

function callAnthropicAPI(url) {
  var apiKey = PropertiesService.getScriptProperties().getProperty('ADMIN_API_KEY');
  var options = {
    method: 'get',
    headers: {
      'anthropic-version': '2023-06-01',
      'x-api-key': apiKey,
      'User-Agent': 'CCDashboard/1.0.0'
    },
    muteHttpExceptions: true
  };

  for (var attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    try {
      var response = UrlFetchApp.fetch(url, options);
      var code = response.getResponseCode();

      if (code === 200) return JSON.parse(response.getContentText());
      if (code === 429) { Utilities.sleep(5000 * attempt); continue; }
      if (code >= 500) { Utilities.sleep(2000 * attempt); continue; }
      if (code === 401 || code === 403) throw new Error('Invalid API key (HTTP ' + code + ')');
      throw new Error('API error: HTTP ' + code);
    } catch (e) {
      if (attempt === MAX_RETRIES) throw e;
      Utilities.sleep(2000);
    }
  }
}
```

### `SheetWriter.gs` - Sheets書き込み

```javascript
function writeAnalyticsToSheet(data, dateString) {
  var ss = SpreadsheetApp.openById(getSpreadsheetId());
  var sheet = ss.getSheetByName('RawData');

  // 冪等性: 同一日付の既存行を削除
  deleteDateRows(sheet, dateString);

  // フラット化してバルク書き込み
  var rows = [];
  var now = new Date();
  data.forEach(function(record) {
    var flatRows = flattenRecord(record, now);
    rows = rows.concat(flatRows);
  });

  if (rows.length > 0) {
    sheet.getRange(sheet.getLastRow() + 1, 1, rows.length, rows[0].length)
      .setValues(rows);
  }
}

function flattenRecord(record, fetchedAt) {
  var base = [
    record.date,
    record.actor.email_address || record.actor.api_key_name || '',
    record.actor.type,
    record.organization_id,
    record.customer_type,
    record.terminal_type || '',
    record.core_metrics.num_sessions,
    record.core_metrics.lines_of_code.added,
    record.core_metrics.lines_of_code.removed,
    record.core_metrics.commits_by_claude_code,
    record.core_metrics.pull_requests_by_claude_code,
    (record.tool_actions.edit_tool || {}).accepted || 0,
    (record.tool_actions.edit_tool || {}).rejected || 0,
    (record.tool_actions.multi_edit_tool || {}).accepted || 0,
    (record.tool_actions.multi_edit_tool || {}).rejected || 0,
    (record.tool_actions.write_tool || {}).accepted || 0,
    (record.tool_actions.write_tool || {}).rejected || 0,
    (record.tool_actions.notebook_edit_tool || {}).accepted || 0,
    (record.tool_actions.notebook_edit_tool || {}).rejected || 0,
  ];

  var models = record.model_breakdown || [];
  if (models.length === 0) {
    return [base.concat(['', 0, 0, 0, 0, 0, 'USD', fetchedAt])];
  }

  return models.map(function(m) {
    return base.concat([
      m.model,
      m.tokens.input,
      m.tokens.output,
      m.tokens.cache_read,
      m.tokens.cache_creation,
      m.estimated_cost.amount,
      m.estimated_cost.currency,
      fetchedAt
    ]);
  });
}

function deleteDateRows(sheet, dateString) {
  var data = sheet.getDataRange().getValues();
  for (var i = data.length - 1; i >= 1; i--) {
    var rowDate = Utilities.formatDate(new Date(data[i][0]), 'UTC', 'yyyy-MM-dd');
    if (rowDate === dateString) sheet.deleteRow(i + 1);
  }
}
```

### `WebApp.gs` - フロント向けAPI

```javascript
function getAvailableDateRange() {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('RawData');
  var dates = sheet.getRange('A2:A' + sheet.getLastRow()).getValues().flat().filter(Boolean);
  if (dates.length === 0) return { min: null, max: null };
  dates.sort(function(a, b) { return new Date(a) - new Date(b); });
  return {
    min: Utilities.formatDate(new Date(dates[0]), 'UTC', 'yyyy-MM-dd'),
    max: Utilities.formatDate(new Date(dates[dates.length - 1]), 'UTC', 'yyyy-MM-dd')
  };
}

function getSummaryByUser(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var summary = {};

  rows.forEach(function(r) {
    var email = r[1];
    if (!summary[email]) {
      summary[email] = {
        email: email,
        totalCostCents: 0, totalTokensInput: 0, totalTokensOutput: 0,
        totalSessions: 0, linesAdded: 0, linesRemoved: 0,
        commits: 0, pullRequests: 0,
        editAccepted: 0, editRejected: 0,
        writeAccepted: 0, writeRejected: 0,
        days: new Set()
      };
    }
    var s = summary[email];
    s.totalCostCents += r[24] || 0;      // cost_cents
    s.totalTokensInput += r[20] || 0;    // tokens_input
    s.totalTokensOutput += r[21] || 0;   // tokens_output
    var dateKey = Utilities.formatDate(new Date(r[0]), 'UTC', 'yyyy-MM-dd');
    if (!s.days.has(dateKey)) {
      s.days.add(dateKey);
      s.totalSessions += r[6] || 0;      // num_sessions
      s.linesAdded += r[7] || 0;
      s.linesRemoved += r[8] || 0;
      s.commits += r[9] || 0;
      s.pullRequests += r[10] || 0;
      s.editAccepted += r[11] || 0;
      s.editRejected += r[12] || 0;
      s.writeAccepted += r[15] || 0;
      s.writeRejected += r[16] || 0;
    }
  });

  // Set は JSON.stringify できないので削除
  return Object.values(summary).map(function(s) {
    s.activeDays = s.days.size;
    delete s.days;
    return s;
  });
}

function getDailyCostTrend(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var trend = {};

  rows.forEach(function(r) {
    var dateKey = Utilities.formatDate(new Date(r[0]), 'UTC', 'yyyy-MM-dd');
    var email = r[1];
    var key = dateKey + '|' + email;
    if (!trend[key]) trend[key] = { date: dateKey, email: email, costCents: 0 };
    trend[key].costCents += r[24] || 0;
  });

  return Object.values(trend).sort(function(a, b) {
    return a.date.localeCompare(b.date) || a.email.localeCompare(b.email);
  });
}

function getFilteredRows(startDate, endDate) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('RawData');
  var allData = sheet.getDataRange().getValues();
  var start = new Date(startDate);
  var end = new Date(endDate);
  end.setHours(23, 59, 59);

  return allData.slice(1).filter(function(row) {
    var d = new Date(row[0]);
    return d >= start && d <= end;
  });
}

function getSpreadsheetId() {
  return PropertiesService.getScriptProperties().getProperty('SPREADSHEET_ID');
}
```

### `Trigger.gs` - 日次同期

```javascript
function setupDailyTrigger() {
  // 既存トリガー削除（重複防止）
  ScriptApp.getProjectTriggers().forEach(function(trigger) {
    if (trigger.getHandlerFunction() === 'dailySync') {
      ScriptApp.deleteTrigger(trigger);
    }
  });

  ScriptApp.newTrigger('dailySync')
    .timeBased()
    .atHour(9)
    .everyDays(1)
    .inTimezone('Asia/Tokyo')
    .create();
}

function dailySync() {
  var yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  var dateString = Utilities.formatDate(yesterday, 'UTC', 'yyyy-MM-dd');

  try {
    var data = fetchDailyAnalytics(dateString);
    writeAnalyticsToSheet(data, dateString);
    logSync(dateString, 'success', data.length, '');
  } catch (e) {
    logSync(dateString, 'error', 0, e.message);
    MailApp.sendEmail(
      Session.getActiveUser().getEmail(),
      '[CC Dashboard] Daily sync failed',
      'Date: ' + dateString + '\nError: ' + e.message
    );
  }
}

function backfillData(startDateStr, endDateStr) {
  var current = new Date(startDateStr);
  var end = new Date(endDateStr);

  while (current <= end) {
    var dateStr = Utilities.formatDate(current, 'UTC', 'yyyy-MM-dd');
    if (!isAlreadySynced(dateStr)) {
      try {
        var data = fetchDailyAnalytics(dateStr);
        writeAnalyticsToSheet(data, dateStr);
        logSync(dateStr, 'success', data.length, '');
      } catch (e) {
        logSync(dateStr, 'error', 0, e.message);
      }
      Utilities.sleep(1000);
    }
    current.setDate(current.getDate() + 1);
  }
}

function isAlreadySynced(dateString) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('SyncLog');
  var data = sheet.getDataRange().getValues();
  return data.some(function(row) {
    return Utilities.formatDate(new Date(row[0]), 'UTC', 'yyyy-MM-dd') === dateString
      && row[1] === 'success';
  });
}

function logSync(dateString, status, recordCount, errorMessage) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('SyncLog');
  sheet.appendRow([dateString, status, recordCount, new Date(), errorMessage]);
}
```

---

## フロントエンド実装

### 技術スタック

| ライブラリ | バージョン | 用途 |
|-----------|----------|------|
| React | 19.x | UI |
| recharts | 2.15.x | チャート |
| date-fns | 4.x | 日付操作 |
| vite-plugin-singlefile | 2.x | 単一HTML化 |

### `vite.config.ts`

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { viteSingleFile } from 'vite-plugin-singlefile';

export default defineConfig({
  plugins: [react(), viteSingleFile()],
  build: {
    outDir: '../appscript/client',
    emptyOutDir: true,
  },
});
```

### `gas.ts` - GAS通信ラッパー

```typescript
declare const google: {
  script: {
    run: {
      withSuccessHandler: (cb: (result: any) => void) => any;
      withFailureHandler: (cb: (error: Error) => void) => any;
      [key: string]: any;
    };
  };
};

const IS_DEV = typeof google === 'undefined' || !google.script;

export function callGas<T>(functionName: string, ...args: unknown[]): Promise<T> {
  if (IS_DEV) return callMock<T>(functionName, ...args);

  return new Promise((resolve, reject) => {
    const runner = google.script.run
      .withSuccessHandler(resolve)
      .withFailureHandler(reject);
    (runner as any)[functionName](...args);
  });
}

// 開発用モックデータ
function callMock<T>(functionName: string, ...args: unknown[]): Promise<T> {
  // モックデータを返す（開発時のみ）
  const mocks: Record<string, unknown> = {
    getAvailableDateRange: { min: '2026-01-01', max: '2026-03-25' },
    getSummaryByUser: [/* mock data */],
    getDailyCostTrend: [/* mock data */],
  };
  return Promise.resolve(mocks[functionName] as T);
}
```

### コンポーネント責務

| コンポーネント | 表示内容 | データ元 |
|--------------|---------|---------|
| `DateRangePicker` | 日付範囲 + プリセット（7日/30日/今月/先月） | `getAvailableDateRange` |
| `SummaryCards` | 合計コスト / 合計トークン / アクティブユーザー数 / コミット+PR | `getSummaryByUser` を集約 |
| `CostChart` | 日別×ユーザー別コスト推移（折れ線） | `getDailyCostTrend` |
| `UserCostTable` | ユーザー別テーブル（コスト,トークン,LOC,コミット,承認率）ソート可 | `getSummaryByUser` |
| `ModelBreakdown` | モデル別コスト割合（円グラフ/棒グラフ） | `getSummaryByUser` |
| `ProductivityChart` | セッション・コミット・PR日次推移 | `getDailyCostTrend` 拡張 |
| `ToolAcceptanceChart` | ツール種別ごと承認率（横棒） | `getSummaryByUser` |

---

## 実装フェーズ

### Phase 1: 基盤構築

- [ ] リポジトリ作成 `cc-cost-dashboard`
- [ ] ディレクトリ構成作成
- [ ] Google Sheets 作成（RawData + SyncLog ヘッダー行）
- [ ] GASプロジェクト初期化（`clasp create --type webapp`）
- [ ] Script Properties 設定（ADMIN_API_KEY, SPREADSHEET_ID）
- [ ] Viteプロジェクト初期化（React + TypeScript + singlefile）

### Phase 2: データ取得パイプライン

- [ ] `FetchAnalytics.gs` 実装（API呼び出し + ページネーション + リトライ）
- [ ] `SheetWriter.gs` 実装（フラット化 + 冪等書き込み）
- [ ] `Trigger.gs` 実装（dailySync + backfillData）
- [ ] GASエディタから `dailySync()` 手動実行 → Sheetsにデータ入ることを確認

### Phase 3: サーバーAPI

- [ ] `Code.gs` 実装（doGet）
- [ ] `WebApp.gs` 実装（getAvailableDateRange, getSummaryByUser, getDailyCostTrend）
- [ ] GASエディタから各関数の動作確認

### Phase 4: フロントエンド

- [ ] `types.ts` + `gas.ts`（型定義 + GASラッパー + モック）
- [ ] `App.tsx` + `Layout.tsx`（骨格 + 状態管理）
- [ ] `DateRangePicker.tsx`
- [ ] `SummaryCards.tsx`
- [ ] `UserCostTable.tsx`（最重要ビュー）
- [ ] `CostChart.tsx`
- [ ] `ModelBreakdown.tsx`, `ProductivityChart.tsx`, `ToolAcceptanceChart.tsx`

### Phase 5: 統合デプロイ

- [ ] `scripts/build.sh` 作成
- [ ] `npm run build` → `appscript/client/index.html` 生成確認
- [ ] `clasp push -f` + `clasp deploy`
- [ ] WebアプリURLでE2E動作確認
- [ ] `setupDailyTrigger()` 実行
- [ ] `backfillData()` で過去データ投入
- [ ] 組織メンバーにURL共有

---

## 注意点

### GAS固有の制限

| 制限 | 値 | 対策 |
|------|-----|------|
| スクリプト実行時間 | 6分/実行 | バックフィルは1日ずつ + sleep |
| UrlFetchApp | 20,000リクエスト/日（Workspace） | 日数×ページ数で十分少ない |
| Spreadsheet | 10,000,000セル | 365日×20人×3モデル≒22,000行/年 |
| 単一HTMLサイズ | 実質制限なし | tree-shaking + minify、500KB以下目標 |

### セキュリティ

- Admin API Key → Script Properties のみ（フロントからアクセス不可）
- `executeAs: "USER_DEPLOYING"` → デプロイ者の権限で実行
- `access: "DOMAIN"` → Google Workspace 組織内のみアクセス可

### デプロイ手順（`scripts/build.sh`）

```bash
#!/bin/bash
set -e
cd client && npm run build
cd ../appscript && npx clasp push -f && npx clasp deploy --description "v$(date +%Y%m%d-%H%M%S)"
echo "Deploy complete!"
```

### 初回セットアップ手順

```bash
# 1. clasp ログイン
npm i @google/clasp -g
clasp login

# 2. フロントエンド初期化
cd client
npm create vite@latest . -- --template react-ts
npm install recharts date-fns
npm install -D vite-plugin-singlefile

# 3. GAS初期化
cd ../appscript
clasp create --type webapp --title "CC Dashboard"

# 4. Script Properties設定（GASエディタで）
# ADMIN_API_KEY = sk-ant-admin-...
# SPREADSHEET_ID = <Sheets ID>

# 5. 初回デプロイ
cd .. && bash scripts/build.sh

# 6. トリガー設定（GASエディタで setupDailyTrigger() を実行）
# 7. バックフィル（GASエディタで backfillData('2026-01-01', '2026-03-25') を実行）
```
