data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# ──────────────────────────────────────────────
# ECR Repositories
# ──────────────────────────────────────────────

resource "aws_ecr_repository" "collector" {
  name                 = "${var.project_name}/collector"
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  image_scanning_configuration {
    scan_on_push = false
  }
}

resource "aws_ecr_repository" "generator" {
  name                 = "${var.project_name}/generator"
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  image_scanning_configuration {
    scan_on_push = false
  }
}

# ──────────────────────────────────────────────
# CloudWatch Log Groups（アプリケーションデータ用）
# ──────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "otel_logs" {
  name              = "/otel/claude-code"
  retention_in_days = 90
}

# Lambda 実行ログ
resource "aws_cloudwatch_log_group" "collector_lambda_logs" {
  name              = "/aws/lambda/${var.project_name}-collector"
  retention_in_days = 14
}

# ──────────────────────────────────────────────
# Lambda Functions
# ──────────────────────────────────────────────

resource "aws_lambda_function" "collector" {
  function_name = "${var.project_name}-collector"
  role          = aws_iam_role.lambda_role.arn

  package_type  = "Image"
  image_uri     = var.collector_image_uri
  architectures = ["arm64"]

  timeout     = var.lambda_timeout
  memory_size = var.lambda_memory_size

  environment {
    variables = {
      STORAGE = "cloudwatch"
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.collector_lambda_logs,
  ]
}

# ──────────────────────────────────────────────
# API Gateway HTTP API
# SCP で Lambda Function URL のパブリックアクセスがブロックされるため、
# API Gateway HTTP API 経由で Lambda を公開する。
# ──────────────────────────────────────────────

resource "aws_apigatewayv2_api" "main" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["GET", "POST", "OPTIONS"]
    allow_headers = ["content-type"]
  }
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.main.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.apigw_logs.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      method         = "$context.httpMethod"
      path           = "$context.path"
      status         = "$context.status"
      responseLength = "$context.responseLength"
      latency        = "$context.integrationLatency"
    })
  }
}

resource "aws_cloudwatch_log_group" "apigw_logs" {
  name              = "/aws/apigateway/${var.project_name}"
  retention_in_days = 14
}

# --- Collector integration ---

resource "aws_apigatewayv2_integration" "collector" {
  api_id                 = aws_apigatewayv2_api.main.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.collector.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "collector_logs" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "POST /v1/logs"
  target    = "integrations/${aws_apigatewayv2_integration.collector.id}"
}

resource "aws_apigatewayv2_route" "collector_traces" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "POST /v1/traces"
  target    = "integrations/${aws_apigatewayv2_integration.collector.id}"
}

resource "aws_apigatewayv2_route" "collector_metrics" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "POST /v1/metrics"
  target    = "integrations/${aws_apigatewayv2_integration.collector.id}"
}

resource "aws_lambda_permission" "apigw_collector" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.collector.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.main.execution_arn}/*/*"
}

# ──────────────────────────────────────────────
# Generator Lambda + EventBridge Schedule
# CloudWatch Logs → 集計 → S3 に書き出し
# ──────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "generator_lambda_logs" {
  name              = "/aws/lambda/${var.project_name}-generator"
  retention_in_days = 14
}

resource "aws_lambda_function" "generator" {
  function_name = "${var.project_name}-generator"
  role          = aws_iam_role.lambda_role.arn

  package_type  = "Image"
  image_uri     = var.generator_image_uri
  architectures = ["arm64"]

  # 利用者数増による CloudWatch Logs 量増加で OOM/timeout が頻発したため、
  # AWS Console 上で 8192MB/900s に手動引き上げ済み。本ファイルは drift していたため
  # 整合性を取る目的で更新。並行して lookbackDays = 90 (lambda/generator/main.go)
  # により実際の処理は数分で済むが、安全マージンを取って上限値のまま据え置く。
  timeout     = 900
  memory_size = 8192

  environment {
    variables = {
      S3_BUCKET = aws_s3_bucket.frontend.id
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.generator_lambda_logs,
  ]
}

# EventBridge スケジュール (一時的に 5 分間隔、本番では rate(1 hour) に変更)
resource "aws_cloudwatch_event_rule" "generator_schedule" {
  name                = "${var.project_name}-generator-schedule"
  schedule_expression = var.generator_schedule
}

resource "aws_cloudwatch_event_target" "generator" {
  rule      = aws_cloudwatch_event_rule.generator_schedule.name
  target_id = "GeneratorLambda"
  arn       = aws_lambda_function.generator.arn
}

resource "aws_lambda_permission" "eventbridge_generator" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.generator.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.generator_schedule.arn
}
