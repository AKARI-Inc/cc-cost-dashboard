# ──────────────────────────────────────────────
# AWS WAFv2 (CloudFront 用、us-east-1)
# 許可 IP 以外からのアクセスを 403 で拒否する
# waf_allowed_ips が空の場合は作らない
# ──────────────────────────────────────────────

locals {
  enable_waf = length(var.waf_allowed_ips) > 0
}

resource "aws_wafv2_ip_set" "allowed" {
  count    = local.enable_waf ? 1 : 0
  provider = aws.us_east_1

  name               = "${var.project_name}-allowed-ips"
  description        = "Dashboard allowed IP list"
  scope              = "CLOUDFRONT"
  ip_address_version = "IPV4"
  addresses          = var.waf_allowed_ips
}

resource "aws_wafv2_web_acl" "main" {
  count    = local.enable_waf ? 1 : 0
  provider = aws.us_east_1

  name        = "${var.project_name}-web-acl"
  description = "Dashboard WAF IP allowlist"
  scope       = "CLOUDFRONT"

  default_action {
    block {}
  }

  rule {
    name     = "AllowFromAllowlist"
    priority = 1

    action {
      allow {}
    }

    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.allowed[0].arn
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AllowFromAllowlist"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.project_name}-web-acl"
    sampled_requests_enabled   = true
  }
}
