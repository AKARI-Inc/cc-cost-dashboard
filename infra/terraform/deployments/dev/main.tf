locals {
  project_name = "cc-cost-dashboard-dev"
  aws_region   = "ap-northeast-1"
  account_id   = "050721760927"
}

module "lambda" {
  source = "../../modules/lambda"

  project_name = local.project_name
  aws_region   = local.aws_region

  collector_image_uri = "${local.account_id}.dkr.ecr.${local.aws_region}.amazonaws.com/${local.project_name}/collector:latest"
  api_image_uri       = "${local.account_id}.dkr.ecr.${local.aws_region}.amazonaws.com/${local.project_name}/api:latest"
  generator_image_uri = "${local.account_id}.dkr.ecr.${local.aws_region}.amazonaws.com/${local.project_name}/generator:latest"

  lambda_memory_size = 256
  lambda_timeout     = 30

  generator_schedule = "rate(5 minutes)" # TODO: 検証後に rate(1 hour) に変更

  github_repo                 = "AKARI-Inc/cc-cost-dashboard"
  create_github_oidc_provider = true
}

# ──── Outputs ────

output "api_gateway_url" {
  value = module.lambda.api_gateway_url
}

output "collector_endpoint" {
  description = "OTEL_EXPORTER_OTLP_ENDPOINT に設定する URL"
  value       = module.lambda.collector_endpoint
}

output "api_endpoint" {
  value = module.lambda.api_endpoint
}

output "cloudfront_url" {
  description = "ダッシュボード URL"
  value       = "https://${module.lambda.cloudfront_domain_name}"
}

output "cloudfront_distribution_id" {
  value = module.lambda.cloudfront_distribution_id
}

output "frontend_bucket_name" {
  value = module.lambda.frontend_bucket_name
}

output "github_actions_role_arn" {
  value = module.lambda.github_actions_role_arn
}

output "collector_ecr_repository_url" {
  value = module.lambda.collector_ecr_repository_url
}

output "api_ecr_repository_url" {
  value = module.lambda.api_ecr_repository_url
}
