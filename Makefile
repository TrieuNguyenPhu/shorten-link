# Root developer entrypoint. AWS SAM's custom Lambda builder remains in
# services/shortener-api/Makefile and is invoked automatically by `sam build`.

.DEFAULT_GOAL := help

PNPM ?= pnpm
GO ?= go
SAM ?= sam

ifeq ($(OS),Windows_NT)
CODEGRAPH ?= "$(LOCALAPPDATA)\codegraph\current\bin\codegraph.cmd"
else
CODEGRAPH ?= codegraph
endif

SAM_TEMPLATE ?= infra/aws/template.yaml
OPENAPI_SPEC ?= openapi/openapi.yaml
REDOCLY_VERSION ?= 2.40.0

.PHONY: help install dev dev-api dev-web \
	lint-web build build-web format-api check-format-api tidy-api vet-api test test-api test-api-race \
	verify verify-all security audit-deps vuln-api openapi-lint \
	sam-validate sam-build sam-deploy-guided \
	codegraph-init codegraph-status codegraph-sync codegraph-index

help:
	@echo NPT ShortenLink commands
	@echo.
	@echo Development
	@echo   make install              Install pnpm dependencies from the lockfile
	@echo   make dev                  Run API and web together
	@echo   make dev-api              Run the Go API on localhost:8080
	@echo   make dev-web              Run Next.js on localhost:3000
	@echo.
	@echo Quality
	@echo   make verify               Run frontend lint/build and backend vet/tests
	@echo   make verify-all           Add security, OpenAPI and SAM validation/build
	@echo   make test                 Run backend tests
	@echo   make test-api-race        Run Go race tests; requires a CGO compiler
	@echo   make check-format-api     Fail if committed Go files need gofmt
	@echo   make security             Audit pnpm and reachable Go vulnerabilities
	@echo   make openapi-lint         Lint the OpenAPI contract with pinned Redocly
	@echo.
	@echo Build and deploy
	@echo   make build                Build static web and the SAM Lambda artifact
	@echo   make sam-validate         Validate the SAM/CloudFormation template
	@echo   make sam-build            Validate and build the SAM application
	@echo   make sam-deploy-guided    Build, then start guided AWS deployment
	@echo.
	@echo CodeGraph
	@echo   make codegraph-init       Initialize the local repository graph
	@echo   make codegraph-status     Show graph health and pending changes
	@echo   make codegraph-sync       Incrementally synchronize the graph
	@echo   make codegraph-index      Rebuild the graph from scratch

install:
	$(PNPM) install --frozen-lockfile

dev:
	+@$(MAKE) --no-print-directory -j2 dev-api dev-web

dev-api:
	$(PNPM) dev:api

dev-web:
	$(PNPM) dev:web

lint-web:
	$(PNPM) lint:web

build: build-web sam-build

build-web:
	$(PNPM) build:web

format-api:
	$(GO) fmt ./services/shortener-api/...

check-format-api:
	$(PNPM) format:check:api

tidy-api:
	$(GO) -C services/shortener-api mod tidy

vet-api:
	$(PNPM) vet:api

test: test-api

test-api:
	$(PNPM) test:api

test-api-race:
	$(PNPM) test:api:race

verify:
	$(PNPM) verify

verify-all: verify security openapi-lint sam-build

security: audit-deps vuln-api

audit-deps:
	$(PNPM) audit:deps

vuln-api:
	$(PNPM) vuln:api

openapi-lint:
	$(PNPM) --package=@redocly/cli@$(REDOCLY_VERSION) dlx redocly lint $(OPENAPI_SPEC)

sam-validate:
	$(SAM) validate --lint --template-file $(SAM_TEMPLATE)

sam-build: sam-validate
	$(SAM) build --parallel --template-file $(SAM_TEMPLATE)

sam-deploy-guided: sam-build
	$(SAM) deploy --guided --template-file .aws-sam/build/template.yaml

codegraph-init:
	$(CODEGRAPH) init .

codegraph-status:
	$(CODEGRAPH) status .

codegraph-sync:
	$(CODEGRAPH) sync .

codegraph-index:
	$(CODEGRAPH) index .
