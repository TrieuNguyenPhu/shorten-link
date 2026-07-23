# Root entrypoint for the CV baseline: Python Lambda, React/Vite and AWS SAM.

.DEFAULT_GOAL := help

PYTHON ?= python
NPM ?= npm
PNPM ?= pnpm
SAM ?= sam

.PHONY: help install backend-install frontend-install dev frontend-dev \
	test backend-test backend-unit-test lint frontend-lint build frontend-build \
	sam-validate sam-build preview-install preview-dev preview-lint preview-build verify

help:
	@echo NPT Shorten Link legacy commands
	@echo make install          Install Python and React dependencies
	@echo make dev              Run the React Vite development server
	@echo make test             Run Python Lambda tests
	@echo make lint             Lint the React frontend
	@echo make build            Build the React frontend and SAM application
	@echo make sam-validate     Validate the SAM CloudFormation template
	@echo make sam-build        Validate and build the SAM application
	@echo make preview-install  Install isolated Next.js preview dependencies
	@echo make preview-dev      Run the isolated preview on port 3001
	@echo make preview-lint     Lint the isolated preview
	@echo make preview-build    Build the isolated static preview
	@echo make verify           Run all legacy and preview checks

install: backend-install frontend-install

backend-install:
	$(PYTHON) -m pip install -r backend/requirements.txt -r backend/tests/requirements.txt

frontend-install:
	$(NPM) --prefix frontend ci

dev: frontend-dev

frontend-dev:
	$(NPM) --prefix frontend run dev

test: backend-test

backend-test:
	$(PYTHON) -m pytest backend/tests -ra

backend-unit-test:
	$(PYTHON) -m pytest backend/tests/unit -ra

lint: frontend-lint

frontend-lint:
	$(NPM) --prefix frontend run lint

build: frontend-build sam-build

frontend-build:
	$(NPM) --prefix frontend run build

sam-validate:
	cd backend && $(SAM) validate --lint

sam-build: sam-validate
	cd backend && $(SAM) build

preview-install:
	$(PNPM) install --frozen-lockfile

preview-dev:
	$(PNPM) dev:preview

preview-lint:
	$(PNPM) lint:preview

preview-build:
	$(PNPM) build:preview

verify: backend-test frontend-lint frontend-build sam-build preview-lint preview-build
