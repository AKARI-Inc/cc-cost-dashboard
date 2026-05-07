# cc-cost-dashboard

Claude Code の OTel テレメトリを受信し、コスト・使用量を可視化するダッシュボード。

```
Claude Code (OTel)
    │  POST /v1/logs
    ▼
Collector (Go, :4318)
    │
    ├─ STORAGE=file        → data/logs/otel/YYYY-MM-DD.jsonl
    └─ STORAGE=cloudwatch  → CloudWatch Logs (LocalStack または AWS)
                                       │
                                       ▼
                              Generator Lambda (本番のみ)
                                       │
                                       ▼
                             S3 静的 JSON (data/summary/*.json)
                                       │
                                       ▼
                              Dashboard (React SPA, :5173)
```

本番構成の詳細は [docs/local-verification-plan.md](docs/local-verification-plan.md) と
[docs/generator-redesign.md](docs/generator-redesign.md) を参照。

---

## リポジトリ構成

| ディレクトリ | 内容 |
|---|---|
| [cmd/collector/](cmd/collector/) | ローカル開発用 Collector (`:4318` で OTLP/HTTP 受信) |
| [lambda/collector/](lambda/collector/) | 本番 Collector Lambda |
| [lambda/generator/](lambda/generator/) | 本番 Generator Lambda (CloudWatch Logs → S3 集計 JSON) |
| [internal/](internal/) | `collector` (protobuf デコード) / `storage` (file・CloudWatch) / `model` |
| [dashboard/](dashboard/) | React 19 + Vite 6 の SPA |
| [scripts/](scripts/) | LocalStack/MinIO 初期化 (`init.sh`)、モックデータ生成 (`generate_mock_data.go`) |
| [infra/terraform/](infra/terraform/) | 本番デプロイ用 Terraform |
| [docs/](docs/) | アーキテクチャと再設計ドキュメント |

---

## 前提ツール

| ツール | バージョン | 用途 |
|---|---|---|
| [mise](https://mise.jdx.dev/) | 任意 | [mise.toml](mise.toml) で `go=1.24` / `node=22` を一発で揃える |
| Go | 1.24 | Collector / Generator / モック生成 |
| Node.js | 22 | Dashboard (Vite) |
| Docker / Docker Compose | 最新 | LocalStack / MinIO / Collector コンテナ |
| AWS CLI | 任意 | LocalStack を直接叩いて確認したい場合 |

mise を使う場合：

```bash
mise install
```

mise を使わない場合は Homebrew 等で `go@1.24` と `node@22` を別途用意する。

---

## クイックスタート（最短で UI を見る）

[dashboard/public/data/](dashboard/public/data/) に集計済みのサンプル JSON が同梱されているので、
バックエンドを起動しなくてもダッシュボードは動く。

```bash
mise install
cd dashboard
npm install
npm run dev
```

→ http://localhost:5173 で開く。`mise install` は初回のみ。

### GitHub Codespaces で開く場合

リポジトリに [.devcontainer/](.devcontainer/) を同梱しているので、Codespaces で開けば
Go / Node / Docker / AWS CLI / GitHub CLI と `npm install` まで自動で揃う。

1. GitHub のリポジトリページ右上 **Code → Codespaces → Create codespace on main**
2. 初回ビルドが終わると VS Code Web が開く
3. ターミナルで以下を実行（クイックスタートと同じ）

```bash
cd dashboard && npm run dev
```

`5173` が自動で forward され、ポップアップから直接プレビューを開ける。
Collector を一緒に動かしたい場合は、別ターミナルで `make dev`。

> Codespaces のスペックは **4 core / 8 GB** を推奨（`hostRequirements` で指定済み）。
> LocalStack + MinIO + Collector + Vite を同時に立てるとメモリ余裕は少なめ。
> 重ければ `STORAGE=file` モードのまま `docker compose up collector` だけにする。

Claude Code の利用ログを実際に流したい場合は、次節「ローカルの動作モード」のモード B / C へ。

---

## ローカルの動作モード

| モード | 用途 | 必要なもの |
|---|---|---|
| **A. Dashboard 単独** | UI のレイアウト確認・コンポーネント開発 | Node.js のみ |
| **B. Collector + Dashboard (file)** | 自分の Claude Code 利用ログを流して JSONL に書き出す | Docker + Node.js |
| **C. フルスタック (CloudWatch)** | 本番に近い構成（LocalStack の CloudWatch Logs に書く） | Docker + Node.js |

### モード A: Dashboard 単独

クイックスタートと同じ。`npm run dev` のみ。

### モード B: Collector + Dashboard (file ストレージ)

`.env` は `STORAGE=file` のままでよい。

```bash
cp .env.example .env
make dev
```

`make dev` で `localstack` / `minio` / `init` / `collector` が起動する。

別シェルで Dashboard：

```bash
cd dashboard && npm install && npm run dev
```

Claude Code 側で OTel エンドポイントを向ける（[Claude Code 連携](#claude-code-連携) 参照）。
受信ログは [data/logs/otel/](data/) 以下に JSONL として追記される（既存のサンプルファイルにも追記される形になる）。

> モード B でも `make dev` は LocalStack と MinIO も同時に立ち上げる。
> Collector 自身は `STORAGE=file` なら LocalStack を使わないので、リソースを抑えたければ
> `docker compose up collector` のように個別起動してもよい。

### モード C: フルスタック (CloudWatch ストレージ)

```bash
cp .env.example .env
```

`.env` を開いて `STORAGE=file` を `STORAGE=cloudwatch` に書き換えてから起動。

```bash
make dev
```

`init` サービスが CloudWatch Log Group (`/otel/claude-code`, `/claude-ai/usage`) と
MinIO バケット (`cc-dashboard-uploads`, `cc-dashboard-static`) を冪等に作成する
([scripts/init.sh](scripts/init.sh))。

ローカルでは Generator Lambda は動かないため、Dashboard が読む集計 JSON は
[dashboard/public/data/](dashboard/public/data/) のサンプルのまま。
CloudWatch 上の生イベントは AWS CLI で確認する：

```bash
aws --endpoint-url=http://localhost:4567 logs tail /otel/claude-code --follow
```

---

## 各コンポーネントの起動・停止

### LocalStack / MinIO / init

`make dev` で同時に起動するが、個別操作したい場合：

```bash
docker compose up localstack minio init
docker compose stop localstack minio
docker compose down
```

上から順に「起動」「停止（データは残る）」「コンテナ削除（永続データは `data/` に残る）」。

ヘルスチェック：

```bash
curl http://localhost:4567/_localstack/health
curl http://localhost:9002/minio/health/live
```

### Collector

Docker で起動（推奨）：

```bash
docker compose up --build collector
```

ホストで直接起動（Go ツールチェイン経由）：

```bash
DATA_DIR=./data STORAGE=file go run ./cmd/collector
```

ヘルスチェック：

```bash
curl http://localhost:4318/health
```

`{"status":"ok"}` が返れば OK。

### Dashboard

ホストで起動（HMR を効かせる、推奨）：

```bash
cd dashboard
npm install
npm run dev
```

→ http://localhost:5173

利用可能なスクリプト：

| コマンド | 内容 |
|---|---|
| `npm run dev` | Vite dev server |
| `npm run build` | TypeScript ビルド + Vite ビルド |
| `npm run preview` | ビルド成果物のローカルプレビュー |
| `npm run typecheck` | 型チェックのみ |
| `npm run format` | Prettier 適用 |

---

## Claude Code 連携

Claude Code から OTLP/HTTP で本ローカル Collector に向ける。
`~/.claude/settings.json` に以下を追記：

```jsonc
{
  "env": {
    "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
    "CLAUDE_CODE_ENABLE_TELEMETRY": "1"
  }
}
```

設定後、新しいシェルで Claude Code を起動すると Collector の標準出力にイベント受信ログが流れる。
受信先は `STORAGE` の値で切り替わる（次節）。

---

## ストレージモード切替

[.env.example](.env.example) の `STORAGE` で 2 系統を切り替える。

| 値 | 書き込み先 | 用途 |
|---|---|---|
| `file` (デフォルト) | `data/logs/otel/YYYY-MM-DD.jsonl` | ローカル単独動作 |
| `cloudwatch` | LocalStack の CloudWatch Logs (`/otel/claude-code`) | 本番に近い構成 |

`cloudwatch` モードでは `.env` に以下も併せて設定する（LocalStack は test/test で通る）。
`AWS_ENDPOINT_URL` は docker compose 内のサービス名 `localstack:4566` を指す。

```dotenv
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_DEFAULT_REGION=ap-northeast-1
AWS_ENDPOINT_URL=http://localstack:4566
```

ホストで `go run ./cmd/collector` する場合は `AWS_ENDPOINT_URL=http://localhost:4567` に読み替える。

---

## サンプルデータ

リポジトリには動作確認用のサンプルが最初から同梱されている。

| 場所 | 内容 |
|---|---|
| [dashboard/public/data/summary/](dashboard/public/data/summary/) `dashboard/public/data/events/recent.json` | Dashboard が fetch する集計済み JSON。Vite dev で即時表示される |
| [data/logs/otel/](data/) (`*.jsonl`) | Collector が file モードで書き出す JSONL の実例。Generator のローカル検証等に使える |

クローン直後でも UI が見えるので、自分の Claude Code を流すまで何もできない、という状態にはならない。

---

## テスト

```bash
make test
```

(`go test ./... -v` を実行する)

Dashboard の型チェック：

```bash
cd dashboard && npm run typecheck
```

---

## ポート / エンドポイント一覧

| サービス | URL | 認証 / 備考 |
|---|---|---|
| Collector (OTLP) | http://localhost:4318/v1/logs | OTel HTTP/protobuf |
| Collector (Health) | http://localhost:4318/health | `{"status":"ok"}` |
| Dashboard (host) | http://localhost:5173 | `npm run dev` |
| LocalStack | http://localhost:4567 | コンテナ内は `4566`。CLI では `--endpoint-url=http://localhost:4567` |
| MinIO API | http://localhost:9002 | `minioadmin` / `minioadmin` |
| MinIO Console | http://localhost:9003 | 同上 |

---

## トラブルシューティング

### `--profile full` を使うと `processor` のビルドに失敗する

[docker-compose.yml](docker-compose.yml) には `processor` サービスが定義されているが、
対応する `cmd/processor/Dockerfile` は現在リポジトリに存在しない。
`make dev` (= `docker compose up --build`) は `full` プロファイルを有効化しないので通常運用では踏まないが、
明示的に `--profile full` を指定するとビルドエラーになる。**`processor` は使わない**。

### ポート 4318 が他のプロセスに取られている

ホストで他の OTel コレクタが動いていると衝突する。Compose 側を変えたい場合は
[docker-compose.yml](docker-compose.yml) の `collector` の `ports` を編集する
（`"4318:4318"` の左側がホストポート）。
ホストで `go run` する場合は `PORT=4319 go run ./cmd/collector` で変更可能。

### Dashboard を Compose 経由で起動したいとき

[docker-compose.yml](docker-compose.yml) の `dashboard` サービスは `profiles: ["full"]` 配下で、
ホストポートは `5174` にマップされる（コンテナ内 `5173`）。
HMR の安定性のためホスト直 `npm run dev` を推奨。

### CloudWatch モードで起動直後にエラーが出る

[scripts/init.sh](scripts/init.sh) が走り終わる前に Collector が書き込みを試みると
Log Group 不在で失敗することがある。`make dev` を一度止めてから再度起動すれば解消する。
`init` の完了は次のコマンドで確認できる：

```bash
docker compose logs init | tail -20
```

### `data/` 以下の所有者が root になる

Linux 環境で `docker compose up` 後に `data/` 以下が root 所有になり手元から触れなくなる場合がある：

```bash
sudo chown -R "$USER":"$USER" data
```

### 完全クリーンアップ

```bash
docker compose down -v
rm -rf data/localstack data/minio data/uploads data/logs/otel/raw
git checkout -- data/logs/otel
```

最後の `git checkout` で、自分が流したログを捨ててリポジトリ同梱のサンプルに戻す。

`data/logs/otel/*.jsonl` はサンプルとしてリポジトリに含まれているので、
丸ごと `rm -rf data/logs` するとサンプルも消える点に注意。

---

## 参考リンク

- [docs/local-verification-plan.md](docs/local-verification-plan.md) — 全体アーキテクチャ・S3 JSON スキーマ
- [docs/generator-redesign.md](docs/generator-redesign.md) — Generator Lambda の Step Functions 移行設計
- [Makefile](Makefile) — 利用可能なターゲット一覧
- [.env.example](.env.example) — 環境変数の説明
