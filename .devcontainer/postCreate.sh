#!/usr/bin/env bash
# Codespaces / devcontainer 初回セットアップ
# - mise.toml に従って go / node を導入
# - dashboard の npm 依存をインストール
# - .env をデフォルトでコピー（既存があれば触らない）
set -euo pipefail

echo "==> mise trust"
mise trust

echo "==> mise install (go / node)"
mise install

echo "==> dashboard npm install"
mise exec -- bash -c 'cd dashboard && npm install'

if [ ! -f .env ]; then
  echo "==> .env を .env.example からコピー"
  cp .env.example .env
else
  echo "==> 既存の .env を維持"
fi

echo "==> セットアップ完了"
