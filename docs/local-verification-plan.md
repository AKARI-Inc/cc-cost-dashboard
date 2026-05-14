# Claude Code 利用量ダッシュボード アーキテクチャ

## 概要

OTel テレメトリの受信から可視化までを、Lambda 2 つ + 静的フロントエンドで実現する。

```
Claude Code (OTel)
    │
    ▼
POST /v1/logs (CloudFront → API Gateway → Collector Lambda)
    │
    ▼
CloudWatch Logs (/otel/claude-code)
    │
    ▼  EventBridge (15分間隔)
Generator Lambda
    │
    ▼
S3 (静的 JSON)
    │
    ▼
CloudFront → ブラウザ (React SPA)
```

---

## コンポーネント

| コンポーネント | 役割 | トリガー |
|---|---|---|
| **Collector Lambda** | OTel protobuf → デコード → CloudWatch Logs に書き込み | API Gateway `POST /v1/logs` |
| **Generator Lambda** | CloudWatch Logs → 集計 JSON + Raw Events JSON を S3 に書き出し | EventBridge スケジュール (15分) |
| **フロントエンド** | S3 の静的 JSON を fetch → クライアントサイドでフィルタ・集計・可視化 | ユーザーアクセス |

API Lambda は不要。ダッシュボードのデータはすべて S3 の静的 JSON から読み取る。

---

## ディレクトリ構成

```
cc-cost-dashboard/
├── go.mod / go.sum
├── docker-compose.yml
├── .mise.toml
│
├── cmd/
│   └── collector/              # ローカル開発用エントリポイント (:4318)
│       ├── main.go
│       └── Dockerfile
│
├── lambda/
│   ├── collector/              # 本番 Lambda (OTel 受信)
│   │   ├── main.go
│   │   └── Dockerfile
│   └── generator/              # 本番 Lambda (集計 → S3)
│       ├── main.go
│       └── Dockerfile
│
├── internal/
│   ├── collector/              # protobuf デコード + イベント抽出
│   │   ├── decoder.go
│   │   └── extractor.go
│   ├── storage/                # CloudWatch Logs 読み書き + 集計
│   │   ├── cloudwatch_reader.go
│   │   ├── cloudwatch_writer.go
│   │   ├── reader.go           # ローカル JSONL 読み取り
│   │   ├── writer.go           # ローカル JSONL 書き込み
│   │   └── aggregator.go
│   └── model/
│       └── otel_event.go
│
├── dashboard/                  # React SPA (TypeScript)
│   ├── src/
│   │   ├── App.tsx
│   │   ├── hooks/
│   │   │   └── useUsageData.ts     # S3 JSON fetch + クライアントサイド集計
│   │   └── components/
│   │       ├── RawEventsTable.tsx   # S3 JSON fetch + クライアントサイドフィルタ
│   │       ├── CostChart.tsx
│   │       ├── ModelBreakdown.tsx
│   │       ├── UserSummary.tsx
│   │       ├── SummaryCards.tsx
│   │       └── DateRangeFilter.tsx
│   ├── public/
│   │   └── data/                   # ローカル開発用ダミー JSON (.gitignore)
│   └── vite.config.ts
│
├── infra/terraform/
│   ├── modules/lambda/
│   │   ├── main.tf             # Collector Lambda, Generator Lambda, API Gateway, EventBridge
│   │   ├── frontend.tf         # S3, CloudFront, WAF
│   │   ├── iam.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── deployments/dev/
│       └── main.tf
│
└── .github/workflows/
    ├── deploy_lambda.yml       # Collector + Generator の ECR push + Lambda 更新
    └── deploy_frontend.yml     # dashboard ビルド + S3 sync + CloudFront invalidation
```

---

## S3 に配置される静的 JSON

Generator Lambda が 5 分ごとに生成:

```
s3://<bucket>/data/summary/per-day-per-model.json
s3://<bucket>/data/summary/per-day-per-user.json
s3://<bucket>/data/summary/per-day-per-terminal.json
s3://<bucket>/data/summary/per-day-per-version.json
s3://<bucket>/data/summary/per-day-per-speed.json
s3://<bucket>/data/events/recent.json
```

Cache-Control: `public, max-age=60` (CloudFront がオリジンの指定に従い 60 秒キャッシュ)

### サマリ JSON スキーマ

```json
{
  "data": [
    { "date": "2026-04-15", "key": "claude-sonnet-4-20250514", "total_cost_usd": 0.05, "input_tokens": 1000, "output_tokens": 500, "request_count": 1 }
  ],
  "meta": { "from": "2025-04-16", "to": "2026-04-16", "groupBy": "model" },
  "generated": "2026-04-16T10:30:45Z",
  "eventCount": 1234
}
```

### Raw Events JSON スキーマ

```json
{
  "data": [
    { "timestamp": "2026-04-16T10:30:00Z", "event_name": "claude_code.api_request", "user_email": "alice@example.com", "model": "claude-sonnet-4-20250514", "cost_usd": 0.005, "input_tokens": 1000, "output_tokens": 200, ... }
  ],
  "meta": { "from": "2025-04-16", "to": "2026-04-16", "count": 5000 },
  "generated": "2026-04-16T10:30:45Z"
}
```

`raw_attributes` は除外済み (ペイロード軽量化)。上限 10,000 件、新しい順。

---

## フロントエンドのデータフロー

```
useUsageData(groupBy='model')
    → fetch('/data/summary/per-day-per-model.json')  ← S3 (CloudFront経由)
    → クライアントサイドで日付範囲フィルタ
    → key → model フィールドにマッピング
    → groupBy=day の場合は日別に再集計

RawEventsTable
    → fetch('/data/events/recent.json')  ← S3 (CloudFront経由)
    → クライアントサイドで event_name / user_email フィルタ + ソート + limit
```

API Lambda へのリクエストは一切発生しない。

---

## ローカル開発

```bash
# 依存インストール
mise install

# Collector + LocalStack 起動
docker compose up collector

# Claude Code の OTel を向ける
# ~/.claude/settings.json: "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318"

# フロントエンド開発 (public/data/ にダミー JSON あり)
cd dashboard && npm install && npm run dev
# http://localhost:5173

# テスト
go test ./...
```

---

## デプロイ

### Lambda (Collector + Generator)

```
git push → GitHub Actions (deploy_lambda.yml)
    → Docker build (arm64) → ECR push → Lambda update-function-code
```

### フロントエンド

```
git push → GitHub Actions (deploy_frontend.yml)
    → npm ci → npm run build → S3 sync → CloudFront invalidation
```

### インフラ

```
cd infra/terraform/deployments/dev
terraform plan
terraform apply
```
