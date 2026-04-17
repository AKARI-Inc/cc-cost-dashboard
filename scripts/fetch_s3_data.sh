#!/usr/bin/env bash
# 本番 S3 の summary/events JSON をローカル (dashboard/public/data/) に取り込む。
# ダッシュボードをローカル起動して本番データで動作確認するためのスクリプト。
#
# Usage:
#   ./scripts/fetch_s3_data.sh                    # 既定のプロファイル/バケット
#   AWS_PROFILE=foo ./scripts/fetch_s3_data.sh    # プロファイル差し替え
#   S3_BUCKET=xxx ./scripts/fetch_s3_data.sh      # バケット差し替え

set -euo pipefail

AWS_PROFILE="${AWS_PROFILE:-squad-ep-internal}"
AWS_REGION="${AWS_REGION:-ap-northeast-1}"
BUCKET="${S3_BUCKET:-cc-cost-dashboard-dev-front-bucket}"

DEST="dashboard/public/data"

echo "==> s3://${BUCKET}/data/ → ${DEST}/"
echo "    profile=${AWS_PROFILE} region=${AWS_REGION}"

mkdir -p "${DEST}/summary" "${DEST}/events"

aws s3 sync "s3://${BUCKET}/data/summary/" "${DEST}/summary/" \
  --profile "${AWS_PROFILE}" --region "${AWS_REGION}" \
  --exclude "*" --include "*.json"

aws s3 sync "s3://${BUCKET}/data/events/" "${DEST}/events/" \
  --profile "${AWS_PROFILE}" --region "${AWS_REGION}" \
  --exclude "*" --include "*.json"

echo "==> 完了"
ls -la "${DEST}/summary" "${DEST}/events"
