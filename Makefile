# Root developer entrypoint while the backend and infrastructure are rebuilt
# through independently reviewable pull requests.

.DEFAULT_GOAL := help

PNPM ?= pnpm

ifeq ($(OS),Windows_NT)
CODEGRAPH ?= "$(LOCALAPPDATA)\codegraph\current\bin\codegraph.cmd"
else
CODEGRAPH ?= codegraph
endif

OPENAPI_SPEC ?= openapi/openapi.yaml
REDOCLY_VERSION ?= 2.40.0

.PHONY: help install dev dev-web lint-web build build-web test test-web \
	verify verify-all security audit-deps openapi-lint \
	codegraph-init codegraph-status codegraph-sync codegraph-index

help:
	@echo NPT ShortenLink commands
	@echo.
	@echo Development
	@echo   make install              Install pnpm dependencies from the lockfile
	@echo   make dev                  Run the Next.js application
	@echo   make dev-web              Run Next.js on localhost:3000
	@echo.
	@echo Quality
	@echo   make verify               Run frontend tests, lint and production build
	@echo   make verify-all           Add dependency audit and OpenAPI lint
	@echo   make openapi-lint         Lint the OpenAPI contract with pinned Redocly
	@echo.
	@echo CodeGraph
	@echo   make codegraph-init       Initialize the local repository graph
	@echo   make codegraph-status     Show graph health and pending changes
	@echo   make codegraph-sync       Incrementally synchronize the graph
	@echo   make codegraph-index      Rebuild the graph from scratch

install:
	$(PNPM) install --frozen-lockfile

dev: dev-web

dev-web:
	$(PNPM) dev:web

lint-web:
	$(PNPM) lint:web

build: build-web

build-web:
	$(PNPM) build:web

test: test-web

test-web:
	$(PNPM) test:web

verify:
	$(PNPM) verify

verify-all: verify security openapi-lint

security: audit-deps

audit-deps:
	$(PNPM) audit:deps

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
