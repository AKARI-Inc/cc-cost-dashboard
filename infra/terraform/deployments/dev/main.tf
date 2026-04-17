locals {
  project_name = "cc-cost-dashboard-dev"
  aws_region   = "ap-northeast-1"
  account_id   = "050721760927"
}

module "lambda" {
  source = "../../modules/lambda"

  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }

  project_name = local.project_name
  aws_region   = local.aws_region

  # WAF: ダッシュボードへのアクセスを許可する IP CIDR
  waf_allowed_ips = [
    "122.210.117.198/32",
  ]

  collector_image_uri = "${local.account_id}.dkr.ecr.${local.aws_region}.amazonaws.com/${local.project_name}/collector:latest"
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

output "custom_domain" {
  value = module.lambda.custom_domain
}

output "custom_domain_nameservers" {
  description = "親ドメイン (dx-akari.com) に委譲する NS レコード"
  value       = module.lambda.custom_domain_nameservers
}

output "dashboard_url" {
  description = "ダッシュボード URL (カスタムドメイン)"
  value       = "https://${module.lambda.custom_domain}"
}

