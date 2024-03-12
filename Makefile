.PHONY: help install build serve generate dev-up dev-migrate dev-down dev-teardown stage-up test run docs bot bench k6 coverage install-k6 inject

BLUE = \033[34m
NC = \033[0m

help:  ## Show help message.
	@printf "Usage:\n"
	@printf "  make $(BLUE)<target>$(NC)\n\n"
	@printf "Targets:\n"
	@perl -nle'print $& if m{^[a-zA-Z0-9_-]+:.*?## .*$$}' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
		{printf "$(BLUE)  %-18s$(NC) %s\n", $$1, $$2}'


install: ## Install the dependencies
	./script/run.sh install
	@go install github.com/swaggo/swag/cmd/swag@latest

build:
	go build -v ./...

serve: generate ## Serve the application with hot reload in dev mode
	./script/run.sh dev serve

generate: ## Generate the gorm queries
	./script/run.sh generate

dev-up: ## Start the container for development
	./script/run.sh dev start


dev-migrate: ## Run the migrations
	./script/run.sh dev migrate


dev-down: ## Down the container for development
	./script/run.sh dev stop


dev-teardown: ## Down the container and release all resources
	./script/run.sh dev teardown

stage-up: ## Start the container for stage environment
	./script/run.sh stage start --build

test: ## Run the tests
	./script/run.sh dev test

coverage:
	go test -coverprofile=coverage.out -v ./...
	go tool cover -html=coverage.out
	go tool cover -html=coverage.out -o coverage.html

run: ## Run the application
	./script/run.sh dev run

docs: install ## Generate the swagger docs
	swag init --parseDependency --parseInternal -g ./cmd/backend/main.go

bot: ## Run the bot
	./script/run.sh dev bot

bench: ## Run the benchmarks
	go test -run='^$' -bench=. -benchtime=20s -benchmem ./...

install-k6: ## Install k6
	go install go.k6.io/xk6/cmd/xk6@latest
	cd k6 && xk6 build --with github.com/grafana/xk6-dashboard@latest

k6: ## Run the k6 tests
	cd k6 && XK6_BROWSER_LOG=error ./k6 run --out web-dashboard=export=test-report.html script.js

inject: ## Inject the data
	./script/run.sh dev inject
