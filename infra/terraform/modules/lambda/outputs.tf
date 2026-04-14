output "api_gateway_url" {
  description = "API Gateway HTTP API のベース URL"
  value       = aws_apigatewayv2_stage.default.invoke_url
}

output "collector_endpoint" {
  description = "OTel Collector エンドポイント (OTEL_EXPORTER_OTLP_ENDPOINT に設定)"
  value       = aws_apigatewayv2_stage.default.invoke_url
}

output "api_endpoint" {
  description = "API エンドポイント (/api/... にアクセス)"
  value       = "${aws_apigatewayv2_stage.default.invoke_url}/api"
}

output "collector_ecr_repository_url" {
  description = "Collector ECR リポジトリ URL"
  value       = aws_ecr_repository.collector.repository_url
}

output "api_ecr_repository_url" {
  description = "API ECR リポジトリ URL"
  value       = aws_ecr_repository.api.repository_url
}

output "collector_function_name" {
  value = aws_lambda_function.collector.function_name
}

output "api_function_name" {
  value = aws_lambda_function.api.function_name
}

output "cloudfront_domain_name" {
  description = "CloudFront ディストリビューションのドメイン名"
  value       = aws_cloudfront_distribution.frontend.domain_name
}

output "cloudfront_distribution_id" {
  description = "CloudFront ディストリビューション ID"
  value       = aws_cloudfront_distribution.frontend.id
}

output "frontend_bucket_name" {
  description = "フロントエンド S3 バケット名"
  value       = aws_s3_bucket.frontend.id
}

output "github_actions_role_arn" {
  description = "GitHub Actions OIDC ロール ARN"
  value       = aws_iam_role.github_actions.arn
}
