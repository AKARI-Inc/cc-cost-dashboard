.PHONY: dev down test mock-seed mock-reset build push ecr-login tf-init tf-plan tf-apply tf-ecr-only

# ──── 設定 ────
AWS_PROFILE   ?= squad-ep-internal
AWS_REGION    ?= ap-northeast-1
AWS_ACCOUNT_ID ?= 050721760927
PROJECT_NAME  ?= cc-cost-dashboard-dev

ECR_BASE = $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
SERVICES = collector api

# ──── ローカル開発 ────

dev:  ## 全コンテナ起動
	docker compose up --build

down:  ## 全コンテナ停止
	docker compose down

test:  ## Go テスト実行
	go test ./... -v

mock-seed:  ## モックデータ生成
	go run scripts/generate_mock_data.go

mock-reset:  ## データリセット + モック再生成
	rm -rf data/logs data/uploads
	$(MAKE) mock-seed

# ──── Lambda コンテナイメージ ────

ecr-login:  ## ECR にログイン
	aws ecr get-login-password --region $(AWS_REGION) --profile $(AWS_PROFILE) | \
		docker login --username AWS --password-stdin $(ECR_BASE)

build:  ## Lambda 用コンテナイメージビルド (arm64)
	$(foreach svc,$(SERVICES), \
		docker build --platform linux/arm64 -f lambda/$(svc)/Dockerfile -t $(PROJECT_NAME)/$(svc):latest . ;)

push: ecr-login build  ## ECR push（ビルド込み）
	$(foreach svc,$(SERVICES), \
		docker tag $(PROJECT_NAME)/$(svc):latest $(ECR_BASE)/$(PROJECT_NAME)/$(svc):latest && \
		docker push $(ECR_BASE)/$(PROJECT_NAME)/$(svc):latest ;)

# ──── Terraform ────

tf-init:  ## Terraform 初期化
	cd infra/terraform/deployments/dev && terraform init

tf-plan:  ## Terraform plan
	cd infra/terraform/deployments/dev && terraform plan

tf-apply:  ## Terraform apply
	cd infra/terraform/deployments/dev && terraform apply

tf-ecr-only:  ## ECR リポジトリのみ作成（初回用）
	cd infra/terraform/deployments/dev && terraform apply \
		-target=module.lambda.aws_ecr_repository.collector \
		-target=module.lambda.aws_ecr_repository.api
