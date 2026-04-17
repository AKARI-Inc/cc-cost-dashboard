#!/bin/bash
# LocalStack の初期セットアップ (CloudWatch Log Groups を作成)
#
# docker compose から init サービスとして呼ばれる。冪等なので何度実行しても安全。

set -e

LS_ENDPOINT="http://localstack:4566"

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

echo "==> 初期化完了"
