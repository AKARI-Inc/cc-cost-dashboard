terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # 初回は local backend。S3 backend に移行する場合は以下をアンコメント:
  # backend "s3" {
  #   bucket         = "cc-cost-dashboard-dev-tfstate"
  #   key            = "terraform.tfstate"
  #   region         = "ap-northeast-1"
  #   encrypt        = true
  #   dynamodb_table = "cc-cost-dashboard-dev-tfstate-lock"
  # }
}

provider "aws" {
  region  = "ap-northeast-1"
  profile = "squad-ep-internal"

  default_tags {
    tags = {
      Project     = "cc-cost-dashboard"
      Environment = "dev"
      ManagedBy   = "terraform"
    }
  }
}

# WAF for CloudFront は us-east-1 必須
provider "aws" {
  alias   = "us_east_1"
  region  = "us-east-1"
  profile = "squad-ep-internal"

  default_tags {
    tags = {
      Project     = "cc-cost-dashboard"
      Environment = "dev"
      ManagedBy   = "terraform"
    }
  }
}
