.DEFAULT_GOAL := help
.PHONY: help install api web preview fmt lint test test-e2e test-dev build build-lambdas test-packaging terraform-check check

help:
	@echo "install api web preview fmt lint test test-e2e test-dev build build-lambdas test-packaging terraform-check check"

install:
	pnpm install --frozen-lockfile
	go -C services/lambda mod download

api:
	go -C services/lambda run ./cmd/api

web:
	pnpm dev

preview:
	node local/serve-static.mjs

fmt:
	go -C services/lambda fmt ./...
	pnpm format

lint:
	go -C services/lambda vet ./...
	pnpm lint
	pnpm typecheck

test:
	go -C services/lambda test ./...
	node --test local/serve-static.test.mjs infra/terraform/dev/static-rewrite.test.mjs
	pnpm test:deploy

test-e2e:
	pnpm test:e2e

test-dev:
	pnpm test:dev

build-lambdas:
	node local/build-lambdas.mjs

test-packaging:
	node --test local/build-lambdas.test.mjs

terraform-check:
	terraform -chdir=infra/terraform/bootstrap fmt -check
	terraform -chdir=infra/terraform/bootstrap init -backend=false -input=false
	terraform -chdir=infra/terraform/bootstrap validate
	terraform -chdir=infra/terraform/bootstrap test
	terraform -chdir=infra/terraform/dev fmt -check
	terraform -chdir=infra/terraform/dev init -backend=false -input=false
	terraform -chdir=infra/terraform/dev validate
	terraform -chdir=infra/terraform/dev test

build:
	go -C services/lambda build ./...
	pnpm build

check: test lint build
	pnpm format:check
