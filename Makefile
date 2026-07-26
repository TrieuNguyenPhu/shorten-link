# Root developer and operator entrypoint.

.DEFAULT_GOAL := help

PNPM ?= pnpm
GO ?= go

ifeq ($(OS),Windows_NT)
CODEGRAPH ?= "$(LOCALAPPDATA)\codegraph\current\bin\codegraph.cmd"
else
CODEGRAPH ?= codegraph
endif

OPENAPI_SPEC ?= openapi/openapi.yaml
REDOCLY_VERSION ?= 2.40.0

.PHONY: help install dev dev-all dev-backend dev-frontend dev-api dev-web \
	lint-web build build-web \
	format-api tidy-api vet-api test test-web test-api test-api-race \
	verify verify-all security audit-deps openapi-lint \
	codegraph-init codegraph-status codegraph-sync codegraph-index

help:
	@echo NPT ShortenLink commands
	@echo.
	@echo Development
	@echo   make install              Install pnpm dependencies from the lockfile
	@echo   make dev                  Run backend and frontend together
	@echo   make dev-backend          Run the Go API on localhost:8080
	@echo   make dev-frontend         Run Next.js on localhost:3000
	@echo   make dev-all              Alias for make dev
	@echo.
	@echo Quality
	@echo   make verify               Run frontend and backend verification
	@echo   make verify-all           Add dependency audit, vulnerability scan and OpenAPI lint
	@echo   make test-api-race        Run Go race tests; requires a CGO compiler
	@echo   make openapi-lint         Lint the OpenAPI contract with pinned Redocly
	@echo.
	@echo CodeGraph
	@echo   make codegraph-init       Initialize the local repository graph
	@echo   make codegraph-status     Show graph health and pending changes
	@echo   make codegraph-sync       Incrementally synchronize the graph
	@echo   make codegraph-index      Rebuild the graph from scratch

install:
	$(PNPM) install --frozen-lockfile

dev: dev-all

dev-all:
	+@$(MAKE) --no-print-directory -j2 dev-backend dev-frontend

dev-backend:
	$(PNPM) dev:api

dev-frontend:
	$(PNPM) dev:web

# Backward-compatible aliases.
dev-api: dev-backend

dev-web: dev-frontend

lint-web:
	$(PNPM) lint:web

build: build-web

build-web:
	$(PNPM) build:web

format-api:
	$(GO) fmt ./services/shortener-api/...

tidy-api:
	$(GO) -C services/shortener-api mod tidy

vet-api:
	$(PNPM) vet:api

test: test-web test-api

test-web:
	$(PNPM) test:web

test-api:
	$(PNPM) test:api

test-api-race:
	$(PNPM) test:api:race

verify:
	$(PNPM) verify

verify-all: verify security openapi-lint

security: audit-deps vuln-api

audit-deps:
	$(PNPM) audit:deps

vuln-api:
	$(PNPM) vuln:api

openapi-lint:
	$(PNPM) --package=@redocly/cli@$(REDOCLY_VERSION) dlx redocly lint $(OPENAPI_SPEC)

codegraph-init:
	$(CODEGRAPH) init .

codegraph-status:
	$(CODEGRAPH) status .

codegraph-sync:
	$(CODEGRAPH) sync .

codegraph-index:
	$(CODEGRAPH) index .
