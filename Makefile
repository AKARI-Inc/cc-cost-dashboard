.PHONY: dev down test mock-seed mock-reset build push

dev:  ## Start all containers
	docker compose up --build

down:  ## Stop all containers
	docker compose down

test:  ## Run Go tests
	go test ./... -v

mock-seed:  ## Generate mock data
	go run scripts/generate_mock_data.go

mock-reset:  ## Reset data and regenerate mock
	rm -rf data/logs data/uploads
	$(MAKE) mock-seed

build:  ## Build Lambda container images
	docker build -f lambda/collector/Dockerfile -t cc-dashboard/collector .
	docker build -f lambda/api/Dockerfile -t cc-dashboard/api .
	docker build -f lambda/processor/Dockerfile -t cc-dashboard/processor .
	docker build -f lambda/generator/Dockerfile -t cc-dashboard/generator .

push:  ## Push to ECR (requires AWS_ACCOUNT_ID, AWS_REGION)
	$(foreach svc,collector api processor generator, \
		docker tag cc-dashboard/$(svc) $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/cc-dashboard/$(svc):latest && \
		docker push $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/cc-dashboard/$(svc):latest ;)
