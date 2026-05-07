# ジェネレーター集計システム再設計

Step Functions + 日次パーティションによるインクリメンタル集計への移行設計。

## 目次

1. [背景と問題](#1-背景と問題)
2. [ゴール / ノンゴール](#2-ゴール--ノンゴール)
3. [現状サイジング](#3-現状サイジング)
4. [アーキテクチャ概要](#4-アーキテクチャ概要)
5. [データ意味論](#5-データ意味論)
6. [データスキーマ](#6-データスキーマ)
7. [Lambda 個別仕様](#7-lambda-個別仕様)
8. [Step Functions ASL](#8-step-functions-asl)
9. [CloudWatch 読み取り最適化](#9-cloudwatch-読み取り最適化)
10. [障害モードと復旧](#10-障害モードと復旧)
11. [IAM 設計](#11-iam-設計)
12. [コスト試算](#12-コスト試算)
13. [観測性](#13-観測性)
14. [テスト戦略](#14-テスト戦略)
15. [移行計画](#15-移行計画)
16. [未決事項](#16-未決事項)

---

## 1. 背景と問題

現行のジェネレーター Lambda (`cc-cost-dashboard-dev-generator`) は EventBridge `rate(5 minutes)` で起動し、CloudWatch Logs `/otel/claude-code` から **過去 365 日分のイベントを毎回全件メモリに展開** して 10 種類のサマリを生成し S3 に書き出す構造になっている。

実装は [lambda/generator/main.go](../lambda/generator/main.go) の `handler()` で、

```go
events, err := reader.ReadOtelEvents(ctx, from, now, nil)  // from = now - 365days
// events は []model.OtelEvent に全件 append される
// その後 10 個の bucketize 関数が events を順番にループ
```

データ量増加に伴い 2026-04-30 17:46 以降 OOM (`Runtime.OutOfMemory`) と timeout (300s) を継続発生。応急対応で memory 8192MB / timeout 900s に引き上げ済みだが、現在の伸び率（平日 60〜100MB/日）では数ヶ月で再 OOM が確定する**線形に詰むパス**。

最終成果物（`data/summary/*.json` 計 ~2MB）に対して、毎回 ~13GB の中間状態を作るのが構造的に歪。

---

## 2. ゴール / ノンゴール

### ゴール

- データ量が今後 5〜10 倍に増えても OOM / timeout で停止しない
- フロントエンド（[useUsageData.ts](../dashboard/src/hooks/useUsageData.ts), [useUserDetail.ts](../dashboard/src/hooks/useUserDetail.ts), [RawEventsTable.tsx](../dashboard/src/components/RawEventsTable.tsx)）への破壊的変更ゼロ
- 障害時の復旧オペが「特定日の partial を消す」程度で済む構造
- ロールバックが Terraform / git revert 1 アクションで可能

### ノンゴール

- タイムゾーン（UTC → JST）の変更：別 PR で扱う
- フロントエンド側のサマリスキーマ変更：別 PR で扱う
- raw_attributes の重複保存問題：別 PR で扱う（[内部メモ](#16-未決事項)参照）

---

## 3. 現状サイジング

### 3.1 イベントボリューム

| 指標 | 値 |
|---|---|
| 1 イベントの JSON サイズ（CloudWatch Logs 1 件） | 約 1.3〜1.7 KB（`raw_attributes` が大半） |
| 平日のイベント数 | 30,000〜70,000 件／日 |
| 直近 14 日の蓄積 | ~50 万件 / ~740 MB |
| 365 日窓に外挿 | **~9〜12M 件 / 13〜18 GB** |
| Go 構造体にデコード後のヒープ消費 | データの 3〜5 倍 |

### 3.2 出力サマリの実サイズ（2026-05-01 18:06 時点）

| ファイル | サイズ |
|---|---|
| per-day-per-user-session.json | **779 KB**（最大） |
| per-day-per-user-tool.json | 421 KB |
| per-day-per-user-model.json | 258 KB |
| per-day-per-user-terminal.json | 97 KB |
| per-day-per-user.json | 100 KB |
| per-day-per-version.json | 46 KB |
| per-day-per-terminal.json | 26 KB |
| per-day-per-model.json | 21 KB |
| per-day-per-user-skill.json | 17 KB |
| per-day-per-speed.json | 6 KB |
| **summary 9 ファイル合計** | **~2 MB** |
| recent.json (raw 10,000 件) | 7.8 MB |

**出力は 2 MB しかない。13 GB 読んで 2 MB を作っているのが構造的歪み。**

---

## 4. アーキテクチャ概要

### 4.1 S3 レイアウト

```
s3://cc-cost-dashboard-dev-front-bucket/
├── data/
│   ├── daily/                              # ← 新設。日次の中間集計（過去日は不変）
│   │   ├── 2025-05-02/partials.json
│   │   ├── 2025-05-03/partials.json
│   │   ├── ... (365 日分)
│   │   ├── 2026-04-30/partials.json
│   │   └── 2026-05-01/partials.json        # 「今日」は毎ティック上書き
│   ├── summary/                            # ← 既存。形式・ファイル名はそのまま維持
│   │   └── per-day-per-{model,user,...}.json
│   └── events/
│       └── recent.json                     # ← 既存。直近 14 日のみ読んで再構築
└── (フロントエンド静的アセット)
```

### 4.2 中核アイデア

> **「過去日の集計結果は不変。毎回計算するのは『今日』の差分だけ。」**

- 過去日 → S3 に確定保存（一度集計したら再計算しない）
- 今日 → CloudWatch から毎ティック読んで `partials.json` を上書き（冪等）
- サマリ統合 → 365 個の partial を結合して既存形式の 10 ファイルを再構築

### 4.3 オーケストレーション

EventBridge `rate(15 minutes)` → Step Functions State Machine → 4 つの Lambda

```
ListMissingDays  →  BackfillMap (Map, max_concurrency=10)
                                        ↓
                             AggregateToday
                                        ↓
                  ┌─────────────────────┴─────────────────────┐
                  ↓                                           ↓
            MergeSummaries                           RebuildRecent
                                  (Parallel)
```

---

## 5. データ意味論

### 5.1 「日付」の定義

イベント本体の `timestamp` は OTel コレクタが `Z` 付き UTC で書く。
[internal/model/otel_event.go](../internal/model/otel_event.go) の `ExtractDate(ts)` は文字列の先頭 10 文字を切る実装で、**UTC 日付でグループ化**している。

新設計でも UTC 日付を踏襲する（挙動を変えない）。JST に変えるとフロントの date フィルタも全件ずれて互換が崩れるので、別 PR で扱う。

### 5.2 イベント時刻 vs CloudWatch ingestion 時刻

新設計で**最も間違えやすいポイント**。

- `FilterLogEvents` の `StartTime/EndTime` は **PutLogEvents 時の timestamp**（書き込み時刻）でフィルタする
- イベント本体の `timestamp` フィールドは **発生時刻**
- コレクタ側のバッファ（[internal/storage/cloudwatch_writer.go](../internal/storage/cloudwatch_writer.go) の 500件 / 5 秒 flush）と OTel SDK のバッチ送信込みで現実的に **数十秒〜数分のスキュー**が発生する

#### 対策：読み取り範囲にバッファを取り、本体タイムスタンプで再判定

```go
// AggregateDay(targetDay = "2026-04-30")
const buffer = 10 * time.Minute
start  := targetDay.Add(-buffer)              // 2026-04-29T23:50:00Z
end    := targetDay.Add(24*time.Hour + buffer) // 2026-05-01T00:10:00Z
events := reader.ReadOtelEvents(start, end, opts)

for _, ev := range events {
    if model.ExtractDate(ev.Timestamp) != targetDay.Format("2006-01-02") {
        continue  // 別の日に属するイベントは捨てる（その日の run で拾われる）
    }
    aggregate(ev)
}
```

これで日跨ぎイベントの**重複も漏れも構造的に発生しない**。

### 5.3 「過去日 = 不変」のルール

過去日 partial を再集計しないルールは「ある日の終了から十分時間が経てば、その日のイベントは全て CloudWatch に到達済み」という前提に依存する。

- **安全マージン**: 「日 N の partial を作るのは day N+1 の 01:00 UTC 以降」とする
- 実装：`list-missing-days` Lambda で `closeable_today := now.UTC().Add(-1*time.Hour).Truncate(24h)` を「最後の閉じている日」と定義
- 00:00 UTC ジャストに backfill すると late ingest を取りこぼすため、1 時間バッファで吸収

---

## 6. データスキーマ

### 6.1 partials.json

10 種類の bucketize を 1 ファイルに集約。フィールドは現行の `Bucket` 系構造体から `date` を外側に括り出すだけ（1 日 1 ファイルなので冗長）。

```jsonc
{
  "schema_version": 1,
  "date": "2026-04-30",                   // UTC 日付
  "generated_at": "2026-05-01T01:05:23Z",
  "events_processed": 64412,              // CloudWatch から読んだ件数（バッファ含む）
  "events_kept": 28934,                   // うち target day 範囲内のもの

  // ── 単一 key 系（5 種類） ─────────────────────────
  "by_model":    [{ "key": "sonnet-4-6", "total_cost_usd": 12.34, "input_tokens": 12345, "output_tokens": 234, "cache_read_tokens": 1234, "cache_creation_tokens": 234, "request_count": 89 }],
  "by_user":     [{ "key": "user@x.com", "...": "..." }],
  "by_terminal": [{ "key": "vscode",     "...": "..." }],
  "by_version":  [{ "key": "2.1.105",    "...": "..." }],
  "by_speed":    [{ "key": "fast",       "...": "..." }],

  // ── 複合 key 系（5 種類） ─────────────────────────
  "by_user_model":    [{ "user_email": "u@x.com", "model": "sonnet-4-6", "total_cost_usd": 0, "input_tokens": 0, "output_tokens": 0, "cache_read_tokens": 0, "cache_creation_tokens": 0, "request_count": 0 }],
  "by_user_tool":     [{ "user_email": "u@x.com", "tool_name": "Read", "request_count": 17 }],
  "by_user_terminal": [{ "user_email": "u@x.com", "terminal_type": "vscode", "os_type": "darwin", "request_count": 0, "total_cost_usd": 0 }],
  "by_user_skill":    [{ "user_email": "u@x.com", "skill_name": "init", "skill_source": "config", "plugin_name": "", "use_count": 3 }],
  "by_user_session":  [{ "user_email": "u@x.com", "session_id": "abc-123", "total_cost_usd": 0, "input_tokens": 0, "output_tokens": 0, "request_count": 0 }]
}
```

サイズ予測：1 日 50K events → partial **150〜400 KB**（gzip 後 ~50KB）、365 日で **~100 MB / S3**。

### 6.2 schema_version の運用

- `merge-summaries` は読み込んだ partial の `schema_version` をチェック
- 期待バージョンと違う partial があったらその日付を `data/daily/{date}/partials.json` から削除して missing 扱い → 次回 cron で再集計
- スキーマ変更は **`schema_version` インクリメント + Lambda コード差し替え + 古い partial 削除** の 3 手順で完結

### 6.3 出力サマリの互換性保証

`data/summary/per-day-per-*.json` の JSON キー名・配列構造・meta envelope（`{ data, meta: {from, to, groupBy}, generated, eventCount }`）は**完全に維持**する。フロント側 [useUsageData.ts](../dashboard/src/hooks/useUsageData.ts), [useUserDetail.ts](../dashboard/src/hooks/useUserDetail.ts), [RawEventsTable.tsx](../dashboard/src/components/RawEventsTable.tsx) は無改修。

---

## 7. Lambda 個別仕様

### 7.1 list-missing-days (256 MB / 30 s)

```jsonc
// Input
{}

// Output
{
  "today": "2026-05-01",
  "missing_days": ["2025-05-02", ..., "2026-04-30"]
}
```

**ロジック**:

1. `now = time.Now().UTC()`
2. `closeable_today = now.Add(-1*time.Hour).Truncate(24h)` （late-arrival ガード）
3. `today = now.Truncate(24h)` （AggregateToday に渡す UTC today）
4. `expected_days = [closeable_today - 365days, closeable_today - 1day]`
5. `S3 ListObjectsV2(Prefix="data/daily/", Delimiter="/")` で既存日付セットを取得
6. `missing_days = expected_days - existing_set` （昇順）

定常運用では `missing_days = []`。初回デプロイ時のみ 365 件入る。

### 7.2 aggregate-day (1024 MB / 120 s)

```jsonc
// Input
{ "date": "2026-04-30" }

// Output
{
  "date": "2026-04-30",
  "events_processed": 64412,
  "events_kept": 28934,
  "s3_key": "data/daily/2026-04-30/partials.json"
}
```

**ロジック**:

1. `targetDay = parse(input.date)`
2. `events = reader.ReadOtelEvents(targetDay - 10min, targetDay + 24h + 10min, opts)`
   - opts には api_request / tool_decision / tool_result / skill_activated の OR フィルタを設定（[5.1 参照](#91-server-side-filter-pattern-を活用)）
3. ストリーム的に bucketize:
   ```go
   for ev := range events {
       if model.ExtractDate(ev.Timestamp) != targetDayStr { continue }
       aggregator.Add(ev)
   }
   partial := aggregator.Build(targetDay)
   ```
4. `s3.PutObject("data/daily/{targetDay}/partials.json", partial)`

**メモリ予算**: events 100 MB + bucketize maps 30 MB = ~150 MB（1024 MB は十分余裕）

**書き込み戦略**: 過去日は backfill 1 回のみ。今日は毎回上書きで OK（PUT は強整合）。

### 7.3 merge-summaries (1024 MB / 60 s)

```jsonc
// Input
{ "today": "2026-05-01" }

// Output
{ "summaries_written": 10, "total_bytes": 2034567 }
```

**ロジック**:

1. `S3 ListObjectsV2(Prefix="data/daily/")` → 全日付（365 日 + today）
2. **並列 GET** (errgroup, 並列度 50) で全 partial を取得
3. `schema_version` チェック。不一致 partial があれば warn ログ（merge から除外）
4. 10 種類の summary 配列を再構成（concat → sort）
5. 現行と同じ envelope で wrap：`{ data, meta, generated, eventCount }`
6. `s3.PutObject("data/summary/per-day-per-*.json")` × 10

**メモリ予算**: 365 partials × 300 KB = ~110 MB → ~250 MB (Go の JSON parse オーバーヘッド込み)

### 7.4 rebuild-recent (2048 MB / 180 s)

```jsonc
// Input
{ "today": "2026-05-01", "days": 14 }

// Output
{ "events_count": 9876, "s3_key": "data/events/recent.json" }
```

**ロジック**:

1. `from = today - 14d`, `to = today + 24h`
2. `events = reader.ReadOtelEvents(from, to, opts={Limit: 50000})`
3. `slim = slimEvents(events, 10000)` — RawAttributes 削除、新しい順、上限 10K
4. `s3.PutObject("data/events/recent.json", payload)`

**メモリ予算**: 14 日 × 60K events = 840K events × 1.5 KB = **~1.2 GB** → 2048 MB が必要（ここだけ重い）

---

## 8. Step Functions ASL

### 8.1 全文

```json
{
  "Comment": "cc-cost-dashboard generator (incremental daily partials)",
  "StartAt": "ListMissingDays",
  "States": {
    "ListMissingDays": {
      "Type": "Task",
      "Resource": "arn:aws:states:::lambda:invoke",
      "Parameters": {
        "FunctionName": "${list_missing_days_arn}",
        "Payload": {}
      },
      "ResultSelector": {
        "today.$": "$.Payload.today",
        "missing_days.$": "$.Payload.missing_days"
      },
      "Retry": [
        {
          "ErrorEquals": ["States.ALL"],
          "IntervalSeconds": 5,
          "MaxAttempts": 3,
          "BackoffRate": 2.0
        }
      ],
      "Next": "BackfillMap"
    },
    "BackfillMap": {
      "Type": "Map",
      "ItemsPath": "$.missing_days",
      "MaxConcurrency": 10,
      "ItemSelector": { "date.$": "$$.Map.Item.Value" },
      "ItemProcessor": {
        "ProcessorConfig": { "Mode": "INLINE" },
        "StartAt": "AggregateBackfillDay",
        "States": {
          "AggregateBackfillDay": {
            "Type": "Task",
            "Resource": "arn:aws:states:::lambda:invoke",
            "Parameters": {
              "FunctionName": "${aggregate_day_arn}",
              "Payload": { "date.$": "$.date" }
            },
            "Retry": [
              {
                "ErrorEquals": ["States.ALL"],
                "IntervalSeconds": 10,
                "MaxAttempts": 3,
                "BackoffRate": 2.0
              }
            ],
            "End": true
          }
        }
      },
      "ResultPath": null,
      "Next": "AggregateToday"
    },
    "AggregateToday": {
      "Type": "Task",
      "Resource": "arn:aws:states:::lambda:invoke",
      "Parameters": {
        "FunctionName": "${aggregate_day_arn}",
        "Payload": { "date.$": "$.today" }
      },
      "Retry": [
        {
          "ErrorEquals": ["States.ALL"],
          "IntervalSeconds": 10,
          "MaxAttempts": 3,
          "BackoffRate": 2.0
        }
      ],
      "ResultPath": null,
      "Next": "Parallel"
    },
    "Parallel": {
      "Type": "Parallel",
      "Branches": [
        {
          "StartAt": "MergeSummaries",
          "States": {
            "MergeSummaries": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${merge_summaries_arn}",
                "Payload": { "today.$": "$.today" }
              },
              "Retry": [
                {
                  "ErrorEquals": ["States.ALL"],
                  "IntervalSeconds": 5,
                  "MaxAttempts": 3
                }
              ],
              "End": true
            }
          }
        },
        {
          "StartAt": "RebuildRecent",
          "States": {
            "RebuildRecent": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${rebuild_recent_arn}",
                "Payload": { "today.$": "$.today", "days": 14 }
              },
              "Retry": [
                {
                  "ErrorEquals": ["States.ALL"],
                  "IntervalSeconds": 5,
                  "MaxAttempts": 3
                }
              ],
              "End": true
            }
          }
        }
      ],
      "End": true
    }
  }
}
```

### 8.2 設計ポイント

- `MergeSummaries` と `RebuildRecent` は依存なしで `Parallel` 同時実行（壁時計時間半減）
- `BackfillMap` の `MaxConcurrency: 10` は CloudWatch Logs `FilterLogEvents` のアカウント TPS 制限（5 TPS）に配慮。10 並列でも各 Lambda は数十回 paginate するためバースト平滑化される
- 全ステップに指数バックオフ Retry。失敗時は Step Functions Console で失敗した日付が一目

### 8.3 ワークフロータイプ

**Standard** を選ぶ。理由：

- 初回 backfill で実行時間が 5 分超える可能性（Express は 5 分上限）
- Execution History が UI で確認可能（Express は CloudWatch Logs にしか残らない）
- 月数千実行なのでコスト差は無視できる

---

## 9. CloudWatch 読み取り最適化

### 9.1 Server-side filter pattern を活用

`AggregateDay` で読むイベントのうち、bucketize に使われるのは **api_request / tool_decision / tool_result / skill_activated** の 4 種だけ。`user_prompt`（しばしば最大の `raw_attributes` を持つ）は使われない。

filter pattern で server-side に落とす：

```
{ ($.event_name = "claude_code.api_request") || ($.event_name = "claude_code.tool_decision") || ($.event_name = "claude_code.tool_result") || ($.event_name = "claude_code.skill_activated") }
```

`internal/storage/cloudwatch_reader.go` の `buildFilterPattern` は現状 OR をサポートしないので、`ReadOptions` に `EventNames []string` を追加して OR を組み立てるよう拡張する。

→ AggregateDay の入力データ量が **30〜40% 減**。

`rebuild-recent` ではこのフィルタを使わない（フロントの RawEventsTable は全イベントを表示するため）。

### 9.2 ページング上限

`FilterLogEvents` は 1MB or 10,000 events のうち先に到達した方で response を返す。1 日 70K events なら 7〜10 ページ、各 ~200ms とすると **1 日読み取り = 1.5〜2 秒**。365 日 backfill 並列 10 並列で 60〜90 秒。

### 9.3 `internal/storage` への追加 API

```go
// 既存
ReadOtelEvents(ctx, from, to time.Time, opts *ReadOptions) ([]model.OtelEvent, error)

// 追加
ReadDay(ctx context.Context, day time.Time, opts *ReadOptions) ([]model.OtelEvent, error)
// 内部で from = day - 10min, to = day + 24h + 10min を使い、
// 戻り値の中で ExtractDate(ts) == day の event のみを返す

// ReadOptions 拡張
type ReadOptions struct {
    EventName  string   // 既存（単一）。後方互換のため残す
    EventNames []string // 追加（OR）。両方指定時は EventNames が優先
    UserEmail  string
    Limit      int
}
```

---

## 10. 障害モードと復旧

| シナリオ | 検出 | 自動復旧 | 手動オペ |
|---|---|---|---|
| AggregateDay timeout | Step Functions Execution 失敗、CloudWatch alarm | 次回 cron で missing 再検出 → 再 backfill | 不要 |
| MergeSummaries 失敗 | 同上 | partial は冪等。次回 cron で再 merge | 不要 |
| 部分書き込み失敗（partial 書けたが下流失敗） | 同上 | 過去日は不変、今日は次回上書き | 不要 |
| late-arrival で過去日 partial に漏れ | 検知困難 | なし | `aws s3 rm s3://.../data/daily/{date}/partials.json` → 次回 cron で再生成 |
| schema_version 不一致 | merge-summaries が warn ログ + 該当日除外 | なし | 同上、対象日 partial を S3 から削除 |
| CloudWatch Logs API rate limit | Throttling 例外 | Step Functions Retry で吸収 | 不要 |
| EventBridge ルール無効化 | summary の `generated` 時刻が古い | なし | EventBridge を有効化 |
| Lambda コードバグ | 全実行失敗 | なし | 旧バージョンに rollback（Lambda alias 切替 or git revert + 再デプロイ） |

### 10.1 オペレーション Runbook（手動再集計）

**特定日の再集計**:
```bash
# 1. 該当日の partial を削除
aws s3 rm s3://cc-cost-dashboard-dev-front-bucket/data/daily/2026-04-25/partials.json

# 2. 次回 cron 起動を待つ（最大 15 分）か、手動実行
aws stepfunctions start-execution \
  --state-machine-arn arn:aws:states:ap-northeast-1:050721760927:stateMachine:cc-cost-dashboard-dev-aggregator
```

**全日再集計**（schema 変更後など）:
```bash
aws s3 rm --recursive s3://cc-cost-dashboard-dev-front-bucket/data/daily/
aws stepfunctions start-execution \
  --state-machine-arn arn:aws:states:ap-northeast-1:050721760927:stateMachine:cc-cost-dashboard-dev-aggregator
```

---

## 11. IAM 設計

現行は単一の `aws_iam_role.lambda_role` にすべて寄せているが、新設計では **Lambda 別ロールに分離**する。Read 専用 Lambda が S3 への書き込み権限を持たない構造になり、事故耐性が上がる。

### 11.1 各 Lambda の最小権限

```hcl
# list-missing-days
{
  Action   = ["s3:ListBucket"]
  Resource = aws_s3_bucket.frontend.arn
  Condition = { "StringLike": { "s3:prefix": "data/daily/*" } }
}

# aggregate-day
{ Action = ["logs:FilterLogEvents", "logs:DescribeLogStreams"]
  Resource = aws_cloudwatch_log_group.otel_logs.arn },
{ Action = ["s3:PutObject"]
  Resource = "${aws_s3_bucket.frontend.arn}/data/daily/*" }

# merge-summaries
{ Action = ["s3:ListBucket"]
  Resource = aws_s3_bucket.frontend.arn
  Condition = { "StringLike": { "s3:prefix": "data/daily/*" } } },
{ Action = ["s3:GetObject"]
  Resource = "${aws_s3_bucket.frontend.arn}/data/daily/*" },
{ Action = ["s3:PutObject"]
  Resource = "${aws_s3_bucket.frontend.arn}/data/summary/*" }

# rebuild-recent
{ Action = ["logs:FilterLogEvents", "logs:DescribeLogStreams"]
  Resource = aws_cloudwatch_log_group.otel_logs.arn },
{ Action = ["s3:PutObject"]
  Resource = "${aws_s3_bucket.frontend.arn}/data/events/*" }

# State Machine 実行ロール
{ Action = ["lambda:InvokeFunction"]
  Resource = [4 つの Lambda ARN] }
```

各 Lambda は CloudWatch Logs へのログ書き込み権限（`AWSLambdaBasicExecutionRole`）も別途付与。

---

## 12. コスト試算

dev 環境、月額。EventBridge `rate(15 minutes)` = 1 日 96 回 = 月 2,880 回 を前提。

| 項目 | 計算 | コスト |
|---|---|---|
| Lambda 実行 | 4 Lambda × 平均 60s × 1024MB × 2880 run | ~$3.5 |
| Step Functions Standard 状態遷移 | ~15 transitions/run × 2880 = 43,200/月 | ~$1 |
| S3 PUT (data/daily) | 1 PUT/run + 過去日初回 365 = ~3,000/月 | <$0.01 |
| S3 GET (merge 時) | 365 GETs/run × 2880 = 1.05M/月 | ~$0.4 |
| S3 ストレージ (data/daily) | ~100 MB | <$0.01 |
| CloudWatch Logs FilterLogEvents | $0.005/GB scanned, 1 日 100MB 読み × 2880 ≒ 280GB/月 | ~$1.4 |
| **合計** | | **~$6/月** |

旧構成（毎回 5 min で 365 日読み + メモリ 8GB Lambda）と比較して、Lambda 実行コストはむしろ半減、CloudWatch Logs scan コストは 1/15 に。

---

## 13. 観測性

### 13.1 CloudWatch Alarms

| Alarm | Metric | Threshold | Action |
|---|---|---|---|
| `aggregator-execution-failed` | StepFunctions `ExecutionsFailed` | > 0 (5 min window) | SNS 通知 |
| `aggregator-execution-stale` | StepFunctions `ExecutionsSucceeded` | < 1 (90 min window) | SNS 通知（cron 停止検知） |
| `aggregate-day-throttled` | Lambda `Throttles` (function=aggregate-day) | > 0 | SNS 通知 |
| `cloudwatch-logs-throttled` | Custom (FilterLogEvents 4xx) | > 0 | SNS 通知 |

### 13.2 Custom Metrics（put 推奨）

```go
// aggregate-day 終了時に EMF で出す
{
  "_aws": {
    "CloudWatchMetrics": [{
      "Namespace": "ccCostDashboard/Aggregator",
      "Dimensions": [["FunctionName"]],
      "Metrics": [
        { "Name": "EventsProcessed",  "Unit": "Count" },
        { "Name": "EventsKept",       "Unit": "Count" },
        { "Name": "PartialSizeBytes", "Unit": "Bytes" }
      ]
    }]
  },
  "FunctionName": "aggregate-day",
  "EventsProcessed": 64412,
  "EventsKept": 28934,
  "PartialSizeBytes": 312456
}
```

ダッシュボード: `EventsKept` の日次推移を見れば異常値（極端に少ない / 多い）が早期検知できる。

---

## 14. テスト戦略

### 14.1 ユニットテスト

新設パッケージ `internal/aggregator/` に bucketize 関数を**純粋関数として切り出し**、ユニットテストで以下を保証：

- 同一入力 → 同一出力（決定性）
- 空入力 → 空配列を返す
- 単一 event → 1 行のバケット
- `event_name` 違いの混在 → 各 bucketize 関数が正しい event のみ拾う
- `user_email` 空 → `(unknown)` に集約

`internal/aggregator/merge.go` も同様にテスト：

- 空配列 → 空 summary
- 同一日の partial 2 つ → エラーまたはマージ（仕様で決める。提案：**エラー**にして上流の取り違えを早期検知）
- `schema_version` 不一致 → 該当 partial を skip + warn

### 14.2 並走検証 (Parity Test)

カットオーバー前に **dev で 24 時間並走**：

1. 旧 generator はそのまま `data/summary/*.json` に書く
2. 新パイプラインは `data/summary-v2/*.json` に書く（環境変数 `SUMMARY_PREFIX=data/summary-v2/` で切替）
3. 24h 後に diff を実行：
   ```bash
   for f in per-day-per-{model,user,terminal,version,speed,user-model,user-tool,user-terminal,user-skill,user-session}; do
     aws s3 cp s3://.../data/summary/${f}.json - | jq -S 'del(.generated, .eventCount, .meta.generated)' > old.json
     aws s3 cp s3://.../data/summary-v2/${f}.json - | jq -S 'del(.generated, .eventCount, .meta.generated)' > new.json
     diff old.json new.json && echo "${f}: OK" || echo "${f}: DIFF"
   done
   ```
4. 全 10 ファイルが OK ならカットオーバー実行（参照 prefix を `data/summary/` に切替）

期待差分は `generated`, `eventCount` のみ。`data` 配列は完全一致すべき。

### 14.3 ローカル開発

`internal/storage` の `FileReader` を使って dev container 内でフルパイプラインを再現可能にする。`scripts/generate_mock_data.go` のモックデータでテスト → CI に組み込む。

---

## 15. 移行計画

### 15.1 PR 分割

| PR | 内容 | リスク | 検証 |
|---|---|---|---|
| #1 | `internal/aggregator/` パッケージ新設、bucketize 関数を切り出し、ユニットテスト追加 | 低（純粋リファクタ） | `go test ./internal/aggregator` |
| #2 | `internal/storage` に `ReadDay` + `EventNames []string` opt 追加 | 低（追加のみ） | mock-seed → ローカル test |
| #3 | `lambda/{list-missing-days,aggregate-day,merge-summaries,rebuild-recent}/` 4 Lambda 実装 | 中 | 単体起動でローカル確認 |
| #4 | Terraform: Step Functions + IAM + 4 Lambda + EventBridge 追加。**旧 generator 並走**、出力 prefix は `data/summary-v2/` | 中 | dev で State Machine 手動実行 → backfill 365 日が ~10 分で完了 |
| #5 | 24h parity 検証 → 出力 prefix を `data/summary/` に切替 | 中 | parity diff |
| #6 | 旧 generator Lambda + EventBridge を Terraform から削除 | 低 | 旧 cron が止まること確認 |
| #7 | EventBridge を `rate(15 minutes)` に変更、Lambda memory/timeout を必要分まで戻す | 低 | 1 週間運用後 |

### 15.2 デプロイ順序（Day ベース目安）

```
Day 0  PR #1 merge        # 純粋リファクタ
Day 1  PR #2 merge        # ストレージ層拡張
Day 2  PR #3 merge        # Lambda 実装
Day 3  PR #4 merge        # 並走デプロイ。dev で手動実行 → backfill 完了
Day 4  parity 検証開始
Day 5  PR #5 merge        # cutover
Day 6  PR #6 merge        # 旧 generator 削除
Day 13 PR #7 merge        # cron 緩和
```

### 15.3 ロールバック

| シナリオ | 手順 |
|---|---|
| カットオーバー後に問題発覚 | PR #5 を git revert（出力 prefix を v2 に戻す）。フロント参照先は `data/summary/` のままなので**フロント側の操作不要**で旧 generator の出力に戻る |
| Step Functions が完全停止 | Terraform で旧 generator + EventBridge を一時復活（PR #6 を revert） |
| backfill が暴走（コスト想定超過） | EventBridge ルールを無効化、Step Functions の進行中実行を `aws stepfunctions stop-execution` で停止 |

---

## 16. 未決事項

実装着手前に確認したい設計判断：

1. **タイムゾーン**: UTC 日付運用のままで良いか（JST 移行は別 PR で扱う）
2. **`schema_version` 管理**: 「不一致 → 再集計」方式で良いか
3. **late-arrival バッファ**: 1 時間で十分か（OTel SDK のリトライ設定に依存）
4. **IAM 分離**: Lambda 別ロールにする方針で良いか
5. **EventBridge レート**: `rate(15 minutes)` で良いか（フロントの cache-control が 60s なので 5 分は元々過剰、コメントでも `rate(1 hour)` 想定）
6. **並走検証期間**: 24h で十分か、もっと長く取るか

### 関連する別 PR の候補

新設計の延長線上で扱える小改善：

- **`raw_attributes` の重複保存削減**: 現サンプルを見ると `user_email` などの typed field と `raw_attributes["user.email"]` が完全重複。1 イベント ~600 byte 削減可能 → 取り込み量 30〜40% 減
- **JST タイムゾーン対応**: フロントの date filter と `ExtractDate` を JST 基準に統一
- **集計関数のテスト網羅**: 現状ゼロ
