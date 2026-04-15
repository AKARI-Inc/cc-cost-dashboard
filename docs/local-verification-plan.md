# Claude Code/claude.ai 利用量ダッシュボード ローカル検証プラン（Go + Docker）

## Context

要件整理ドキュメントで定義された AWS サーバーレス構成を、ローカルで段階的に検証する。バックエンド（Lambda 4つ）は Go、フロントエンドは React/TypeScript。ローカルは Docker Compose、本番は Lambda コンテナイメージ（ECR push）で統一。

**ゴール**: OTel受信 → デコード → JSONL保存 → 集計 → 可視化 のパイプラインをコンテナ上で動かし、実際の Claude Code テレメトリで検証する。

**ストレージ方針**: JSONL ファイル（日付分割）でディスクに追記。本番では CloudWatch Logs PutLogEvents に差し替え。

**開発環境方針**: Docker Compose で全サービスを起動。mise（.mise.toml）で Go/Node バージョンをピン留め。チームメンバーは `mise install && docker compose up` で開発環境が揃う。

---

## ディレクトリ構成

```
cc-cost-dashboard/
├── go.mod
├── go.sum
├── Makefile                        # ビルド・テスト・ECR push
├── docker-compose.yml              # ローカル開発用
├── .mise.toml                      # Go/Node バージョン管理（uv相当）
├── .env.example
├── .gitignore
│
├── cmd/                            # ローカル用エントリポイント（net/http サーバー）
│   ├── collector/
│   │   ├── main.go                 # OTel Collector (:4318)
│   │   └── Dockerfile
│   ├── api/
│   │   ├── main.go                 # API サーバー (:3000)
│   │   └── Dockerfile
│   └── processor/
│       ├── main.go                 # ZIP アップロード (:3001)
│       └── Dockerfile
│
├── lambda/                         # Lambda ハンドラ（本番用、Phase 6）
│   ├── collector/
│   │   ├── main.go
│   │   └── Dockerfile              # ECR push 用（aws-lambda-go ベース）
│   ├── api/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── processor/
│   │   ├── main.go
│   │   └── Dockerfile
│   └── generator/
│       ├── main.go                 # Dashboard Generator (EventBridge)
│       └── Dockerfile
│
├── internal/                       # ビジネスロジック（cmd/ と lambda/ で共有）
│   ├── collector/
│   │   ├── decoder.go              # protobuf デコード
│   │   ├── decoder_test.go
│   │   ├── extractor.go            # OTelログ → 構造化イベント抽出
│   │   └── extractor_test.go
│   ├── processor/
│   │   ├── zip_parser.go           # ZIP展開 + 会話JSON解析
│   │   ├── zip_parser_test.go
│   │   ├── token_estimator.go      # 1文字 ≈ 1.5トークン推定
│   │   └── token_estimator_test.go
│   ├── api/
│   │   ├── handler.go              # HTTP ハンドラ
│   │   └── handler_test.go
│   ├── storage/
│   │   ├── writer.go               # JSONL 追記（ローカル: ファイル / 本番: CloudWatch Logs）
│   │   ├── writer_test.go
│   │   ├── reader.go               # JSONL 読み込み + 期間フィルタ（ローカル: ファイル / 本番: Logs Insights）
│   │   ├── reader_test.go
│   │   ├── aggregator.go           # 日別/モデル別/ユーザー別 集計
│   │   └── aggregator_test.go
│   └── model/
│       ├── otel_event.go           # OtelEvent 構造体
│       └── claude_ai_event.go      # ClaudeAiEvent 構造体
│
├── dashboard/                      # React フロントエンド（唯一の TypeScript）
│   ├── package.json
│   ├── package-lock.json
│   ├── Dockerfile
│   ├── vite.config.ts              # proxy: /api → api:3000（Docker network）
│   ├── index.html
│   └── src/
│       ├── App.tsx
│       ├── components/
│       │   ├── CostChart.tsx       # 日別コスト推移 (Recharts)
│       │   ├── ModelBreakdown.tsx   # モデル別利用比率
│       │   ├── UserSummary.tsx      # ユーザー別消費量テーブル
│       │   ├── ClaudeAiUpload.tsx   # D&D ZIPアップロード
│       │   └── DateRangeFilter.tsx  # 期間フィルタ (7/30/90/365日)
│       └── hooks/
│           └── useUsageData.ts
│
├── scripts/
│   ├── generate_mock_data.go       # モックデータ JSONL 生成
│   └── send_test_otel.go           # テスト用 protobuf 送信
│
├── testdata/                       # テスト用固定データ
│   ├── sample_otel_payload.bin     # キャプチャした protobuf バイナリ
│   └── sample_export.zip           # モック claude.ai エクスポート
│
├── data/                           # ローカルデータ (gitignore, Docker volume)
│   ├── logs/
│   │   ├── otel/                   # = CloudWatch Logs /otel/claude-code
│   │   └── claude-ai/              # = CloudWatch Logs /claude-ai/usage
│   └── uploads/                    # = S3 upload bucket
│
└── infra/                          # CDK (Phase 6)
    ├── package.json
    ├── bin/app.ts
    └── lib/dashboard-stack.ts
```

---

## Docker Compose 構成

```yaml
# docker-compose.yml
services:
  collector:
    build:
      context: .
      dockerfile: cmd/collector/Dockerfile
    ports:
      - "4318:4318"
    volumes:
      - ./data:/data
    environment:
      - DATA_DIR=/data

  api:
    build:
      context: .
      dockerfile: cmd/api/Dockerfile
    ports:
      - "3000:3000"
    volumes:
      - ./data:/data
    environment:
      - DATA_DIR=/data

  processor:
    build:
      context: .
      dockerfile: cmd/processor/Dockerfile
    ports:
      - "3001:3001"
    volumes:
      - ./data:/data
    environment:
      - DATA_DIR=/data

  dashboard:
    build:
      context: ./dashboard
      dockerfile: Dockerfile
    ports:
      - "5173:5173"
    depends_on:
      - api
      - processor
```

全サービスが `./data` ボリュームを共有し、JSONL ファイルで連携する。

---

## ローカル ↔ AWS の対応表

| ローカル (Docker Compose) | AWS 本番 (Lambda + ECR) |
|--------------------------|------------------------|
| `data/logs/otel/*.jsonl` (volume内ファイル) | CloudWatch Logs `/otel/claude-code` |
| `data/logs/claude-ai/*.jsonl` (volume内ファイル) | CloudWatch Logs `/claude-ai/usage` |
| `data/uploads/` (volume内FS) | S3 upload bucket |
| `internal/storage/reader.go` + `aggregator.go` | CloudWatch Logs Insights クエリ |
| `cmd/collector` コンテナ (:4318) | Lambda コンテナイメージ (ECR) + Function URL |
| `cmd/processor` コンテナ (:3001) | Lambda コンテナイメージ (ECR) + S3トリガー |
| `cmd/api` コンテナ (:3000) | Lambda コンテナイメージ (ECR) + Function URL |
| `dashboard` コンテナ (:5173) | S3 + CloudFront |
| なし | WAF IPセット + Basic認証 |
| `cmd/*/Dockerfile` | `lambda/*/Dockerfile` (ECR push) |

---

## Dockerfile テンプレート

### ローカル用（cmd/collector/Dockerfile）

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/collector

FROM alpine:3.21
COPY --from=builder /server /server
EXPOSE 4318
CMD ["/server"]
```

### 本番 Lambda 用（lambda/collector/Dockerfile）

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o /handler ./lambda/collector

FROM public.ecr.aws/lambda/provided:al2023-arm64
COPY --from=builder /handler /handler
ENTRYPOINT ["/handler"]
```

同じビルドステージ、ビジネスロジック (`internal/`) を共有。エントリポイント（`cmd/` vs `lambda/`）だけが異なる。

### ダッシュボード用（dashboard/Dockerfile）

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host"]
```

---

## mise.toml（チーム開発環境統一）

```toml
[tools]
go = "1.24"
node = "22"
```

Docker 外で直接ビルド・テストする場合に Go/Node バージョンを統一する。`mise install` で自動セットアップ。

---

## Phase 1: 基盤 — Go モジュール + ストレージ層 + Docker 初期構成

**目的**: プロジェクト初期化、型定義、JSONL ストレージ層、Docker Compose 雛形、モックデータ生成。

### 実装内容

1. `go mod init` + `.mise.toml` + `.gitignore` 更新
2. **型定義** (`internal/model/`):
   ```go
   // otel_event.go
   type OtelEvent struct {
       Timestamp           string  `json:"timestamp"`
       EventName           string  `json:"event_name"`
       SessionID           string  `json:"session_id,omitempty"`
       UserEmail           string  `json:"user_email,omitempty"`
       Model               string  `json:"model,omitempty"`
       InputTokens         int     `json:"input_tokens,omitempty"`
       OutputTokens        int     `json:"output_tokens,omitempty"`
       CacheReadTokens     int     `json:"cache_read_tokens,omitempty"`
       CacheCreationTokens int     `json:"cache_creation_tokens,omitempty"`
       CostUSD             float64 `json:"cost_usd,omitempty"`
       DurationMs          int     `json:"duration_ms,omitempty"`
       CharCount           int     `json:"char_count,omitempty"`
       ToolName            string  `json:"tool_name,omitempty"`
   }
   ```
3. **ストレージ層** (`internal/storage/`):
   - `writer.go`: `AppendEvent(logGroup, event)` → `data/logs/{logGroup}/YYYY-MM-DD.jsonl` に追記。`syscall.Flock` でファイルロック
   - `reader.go`: `ReadEvents[T](logGroup, from, to)` → 対象日付の JSONL を読み、`[]T` で返す
   - `aggregator.go`: `AggregateByDay`, `AggregateByModel`, `AggregateByUser`
4. `scripts/generate_mock_data.go` — 10ユーザー × 30日分
5. `docker-compose.yml` 雛形（collector だけ先に動かす）

### 主要依存

- Go 標準ライブラリのみ（`encoding/json`, `os`, `bufio`, `time`, `syscall`）

### 検証方法

```bash
go run scripts/generate_mock_data.go
ls data/logs/otel/
head -3 data/logs/otel/2026-04-13.jsonl
go test ./internal/storage/... -v
```

---

## Phase 2: OTel Collector — protobuf 受信・デコード（コンテナ化）

**目的**: Claude Code から protobuf で送られる OTel ログを受信し、構造化イベントに変換して JSONL 保存。コンテナで動作。**最もリスクの高いコンポーネント。**

### 実装内容

1. **protobuf デコーダー** (`internal/collector/decoder.go`):
   ```go
   import (
       collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
       "google.golang.org/protobuf/proto"
   )
   func DecodeLogs(body []byte) (*collogspb.ExportLogsServiceRequest, error) {
       req := &collogspb.ExportLogsServiceRequest{}
       err := proto.Unmarshal(body, req)
       return req, err
   }
   ```
2. **イベント抽出** (`internal/collector/extractor.go`):
   - `resourceLogs[].resource.attributes` → `user.email`
   - `scopeLogs[].logRecords[]` → attributes から `event.name` で分岐
   - `claude_code.api_request`: model, input_tokens, output_tokens, cost_usd, duration_ms
   - `claude_code.user_prompt`: char_count
   - `claude_code.session.count`, `claude_code.tool_decision`, `claude_code.tool_result`
3. **HTTP サーバー** (`cmd/collector/main.go`):
   - `POST /v1/logs` → ReadAll → DecodeLogs → ExtractEvents → storage.AppendEvent
   - `POST /v1/traces`, `POST /v1/metrics` → 200 OK（無視）
   - `:4318`
4. `cmd/collector/Dockerfile`

### 主要依存

```
go.opentelemetry.io/proto/otlp v1.5.0
google.golang.org/protobuf v1.36.5
```

### 検証方法

```bash
# コンテナ起動
docker compose up collector

# テスト protobuf 送信
go run scripts/send_test_otel.go

# JSONL 確認
cat data/logs/otel/$(date +%Y-%m-%d).jsonl

# 実際の Claude Code で検証
# ~/.claude/settings.json:
#   "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318"

# ユニットテスト
go test ./internal/collector/... -v
```

---

## Phase 3: claude.ai ZIP プロセッサ（コンテナ化）

**目的**: claude.ai エクスポート ZIP をアップロード・解析・JSONL 保存。

### 実装内容

1. **ZIP パーサー** (`internal/processor/zip_parser.go`):
   - `archive/zip` で展開、会話 JSON 走査
   - メッセージの role, content 文字数, model を抽出
2. **トークン推定** (`internal/processor/token_estimator.go`):
   - `utf8.RuneCountInString` で日本語文字数を正確にカウント × 1.5
3. **HTTP サーバー** (`cmd/processor/main.go`):
   - `POST /api/upload/claude-ai` → multipart ZIP 受信 → 解析 → JSONL 追記
   - `:3001`
4. `cmd/processor/Dockerfile`

### 主要依存

- Go 標準ライブラリのみ

### 検証方法

```bash
docker compose up processor
curl -F "file=@testdata/sample_export.zip" http://localhost:3001/api/upload/claude-ai
cat data/logs/claude-ai/$(date +%Y-%m-%d).jsonl
go test ./internal/processor/... -v
```

---

## Phase 4: API サーバー + React ダッシュボード（コンテナ化）

**目的**: Go 集計 API + React/Recharts ダッシュボード。全てコンテナで動作。

### API 設計

```
GET /api/claude-code/usage?from=YYYY-MM-DD&to=YYYY-MM-DD&groupBy=day|model|user
GET /api/claude-ai/usage?from=YYYY-MM-DD&to=YYYY-MM-DD&groupBy=day|user
```

処理: リクエスト → `storage.ReadEvents()` → `storage.AggregateByXxx()` → JSON レスポンス

### ダッシュボード機能

| 機能 | コンポーネント | 優先度 |
|-----|-------------|-------|
| 日別コスト推移 | `CostChart.tsx` (Recharts LineChart) | 必須 |
| ユーザー別トークン消費量 | `UserSummary.tsx` (テーブル) | 必須 |
| モデル別利用比率 | `ModelBreakdown.tsx` (Recharts PieChart) | 必須 |
| コスト表示（USD） | `CostChart.tsx` に統合 | 必須 |
| 期間フィルタ | `DateRangeFilter.tsx` (7/30/90/365日) | 必須 |
| ZIP アップロード | `ClaudeAiUpload.tsx` (D&D) | 必須 |

### Vite プロキシ（Docker network 内）

```typescript
// dashboard/vite.config.ts
server: {
  host: true,  // コンテナ外からアクセス可能に
  proxy: {
    '/api': 'http://api:3000'  // Docker Compose サービス名で解決
  }
}
```

### 検証方法

```bash
docker compose up api dashboard
# http://localhost:5173 でダッシュボード確認
```

---

## Phase 5: 統合 — 全コンテナ一括起動 + E2E 検証

**目的**: `docker compose up` で全サービス起動。E2E テスト。

### Makefile

```makefile
.PHONY: dev down test mock-seed mock-reset build push

dev:                ## 全コンテナ起動（ホットリロード）
	docker compose up --build

down:               ## 全コンテナ停止
	docker compose down

test:               ## Go テスト実行
	go test ./... -v

mock-seed:          ## モックデータ生成
	go run scripts/generate_mock_data.go

mock-reset:         ## データリセット + モック再生成
	rm -rf data/logs data/uploads
	$(MAKE) mock-seed

build:              ## Lambda 用コンテナイメージビルド
	docker build -f lambda/collector/Dockerfile -t cc-dashboard/collector .
	docker build -f lambda/api/Dockerfile -t cc-dashboard/api .
	docker build -f lambda/processor/Dockerfile -t cc-dashboard/processor .
	docker build -f lambda/generator/Dockerfile -t cc-dashboard/generator .

push:               ## ECR push（要: AWS_ACCOUNT_ID, AWS_REGION）
	$(foreach svc,collector api processor generator, \
		docker tag cc-dashboard/$(svc) $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/cc-dashboard/$(svc):latest && \
		docker push $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/cc-dashboard/$(svc):latest ;)
```

### E2E 検証シナリオ

1. `make mock-seed` でモックデータ生成
2. `make dev` で全コンテナ起動
3. `http://localhost:5173` → モックデータのチャート確認
4. Claude Code OTel を `http://localhost:4318` に向けて実操作 → データがリアルタイムで JSONL に追記 → ダッシュボードリロードで反映
5. テスト ZIP をダッシュボードの D&D からアップロード → claude.ai タブに反映
6. `make test` で全ユニットテスト通過

---

## Phase 6: Lambda ハンドラ + ECR push + CDK

**目的**: 本番用 Lambda コンテナイメージを作成し、ECR push → CDK デプロイ。

### Lambda ハンドラ（`lambda/*/main.go`）

`internal/` のビジネスロジックを共有し、I/O 層だけ差し替え:

```go
// lambda/collector/main.go
func handler(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
    body, _ := base64.StdEncoding.DecodeString(event.Body)
    req, _ := collector.DecodeLogs(body)
    otelEvents := collector.ExtractEvents(req)
    // ローカル: storage.AppendEvent → ファイル追記
    // 本番:   cwlClient.PutLogEvents → CloudWatch Logs
    for _, e := range otelEvents {
        cwlClient.PutLogEvents(ctx, ...)
    }
    return events.LambdaFunctionURLResponse{StatusCode: 200}, nil
}
```

### I/O 差し替え表

| ローカル (cmd/) | 本番 (lambda/) |
|----------------|---------------|
| `storage.AppendEvent` → ファイル追記 | CloudWatch Logs `PutLogEvents` |
| `storage.ReadEvents` + `Aggregate*` → JSONL読み込み | CloudWatch Logs Insights クエリ |
| `r.FormFile` → ローカルFS | S3 `GetObject` (S3トリガー) |
| `cmd/*/Dockerfile` | `lambda/*/Dockerfile` → ECR push |

### デプロイフロー

```
docker build → docker tag → docker push (ECR) → cdk deploy
```

CDK（TypeScript, `infra/`）は Lambda コンテナイメージの ECR URI を参照してデプロイ。WAF + CloudFront + Basic認証は CDK で構成。

---

## リスクと対策

| リスク | 影響 | 対策 |
|-------|------|------|
| protobuf スキーマ変更 | Collector デコード失敗 | `go.opentelemetry.io/proto/otlp` は公式で安定。生バッファ保存でデバッグ |
| JSONL 同時書き込み | コンテナ間でファイル競合 | `syscall.Flock` でファイルロック。reader は JSON parse 失敗行をスキップ |
| claude.ai ZIP フォーマット変更 | パーサー破損 | `recover` + スキップログ + 防御的パース |
| Docker build キャッシュ | Go mod download が毎回走る | `go.mod`, `go.sum` を先に COPY してキャッシュ活用 |
| ECR push の権限 | CI/CD で認証失敗 | GitHub Actions に OIDC → AssumeRole を設定 |

---

## 実装順序とタイムライン

| Phase | 内容 | 依存 | 所要時間 |
|-------|------|------|---------|
| 1 | Go module + storage + model + mock + Docker雛形 | なし | 0.5日 |
| 2 | OTel Collector (decoder + extractor + container) | Phase 1 | 1日 |
| 3 | claude.ai ZIP プロセッサ (container) | Phase 1 | 0.5日 |
| 4 | API + React ダッシュボード (container) | Phase 1 | 1-2日 |
| 5 | 統合 E2E (`docker compose up` で全検証) | Phase 2-4 | 0.5日 |
| 6 | Lambda handler + ECR push + CDK | Phase 5 | 将来 |

Phase 2, 3, 4 は Phase 1 完了後に並行可能。合計 **3-4日** でローカル検証環境が完成。
