#!/bin/bash
# LocalStack / MinIO の初期セットアップ
# - CloudWatch Log Groups を作成
# - MinIO バケットを作成
#
# docker compose から init サービスとして呼ばれる。冪等なので何度実行しても安全。

set -e

LS_ENDPOINT="http://localstack:4566"
MINIO_ENDPOINT="http://minio:9000"

echo "==> CloudWatch Log Groups を作成"
for LG in "/otel/claude-code" "/claude-ai/usage"; do
  if aws --endpoint-url="$LS_ENDPOINT" logs describe-log-groups \
       --log-group-name-prefix "$LG" \
       --query "logGroups[?logGroupName=='$LG']" \
       --output text 2>/dev/null | grep -q "$LG"; then
    echo "    既存: $LG"
  else
    aws --endpoint-url="$LS_ENDPOINT" logs create-log-group --log-group-name "$LG"
    echo "    作成: $LG"
  fi
done

echo "==> MinIO バケットを作成"
# AWS CLI で MinIO に対して S3 操作を実行
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
for BUCKET in "cc-dashboard-uploads" "cc-dashboard-static"; do
  if aws --endpoint-url="$MINIO_ENDPOINT" s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
    echo "    既存: $BUCKET"
  else
    aws --endpoint-url="$MINIO_ENDPOINT" s3api create-bucket --bucket "$BUCKET" 2>&1 | head -1 || true
    echo "    作成: $BUCKET"
  fi
done

echo "==> 初期化完了"
