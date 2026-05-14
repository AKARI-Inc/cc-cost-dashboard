variable "project_name" {
  description = "プロジェクト名（リソース命名に使用）"
  type        = string
}

variable "aws_region" {
  description = "AWS リージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "collector_image_uri" {
  description = "Collector Lambda のコンテナイメージ URI"
  type        = string
}

variable "lambda_memory_size" {
  description = "Lambda 関数のメモリサイズ (MB)"
  type        = number
  default     = 256
}

variable "lambda_timeout" {
  description = "Lambda 関数のタイムアウト (秒)"
  type        = number
  default     = 30
}

variable "generator_image_uri" {
  description = "Generator Lambda のコンテナイメージ URI"
  type        = string
}

variable "generator_schedule" {
  description = "Generator EventBridge スケジュール式"
  type        = string
  # 5 分間隔だと前ジョブ完了前に次が起動し重複するため 15 分を基準値とする
  # (各環境の deployments/*/main.tf で必要に応じて上書き可能)
  default = "rate(15 minutes)"
}

variable "github_repo" {
  description = "GitHub リポジトリ (org/repo 形式)"
  type        = string
  default     = "AKARI-Inc/cc-cost-dashboard"
}

variable "create_github_oidc_provider" {
  description = "GitHub OIDC プロバイダーを新規作成するか（アカウントに既存の場合は false）"
  type        = bool
  default     = false
}

variable "waf_allowed_ips" {
  description = "ダッシュボードへのアクセスを許可する IP CIDR リスト (例: [\"122.210.117.198/32\"])"
  type        = list(string)
  default     = []
}
