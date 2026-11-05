.PHONY: test vet check build-lambdas test-integration terraform-check

test:
	go -C backend test -race ./...

test-integration:
	go -C backend test -race -tags=integration -count=1 ./internal/repository/... ./tools/local-http/...

vet:
	go -C backend vet ./...

check: test vet terraform-check frontend-check operations-check

build-lambdas:
	go -C backend run ./tools/build-lambdas

terraform-check: build-lambdas
	terraform fmt -check -recursive infrastructure/terraform
	@for root in infrastructure/terraform/bootstrap infrastructure/terraform/environments/dev; do \
		terraform -chdir="$$root" init -backend=false -input=false -lockfile=readonly && \
		terraform -chdir="$$root" validate -no-color && \
		terraform -chdir="$$root" test -no-color || exit $$?; \
	done

.PHONY: frontend-check operations-check
frontend-check:
	corepack pnpm@10.33.2 --dir frontend install --frozen-lockfile
	corepack pnpm@10.33.2 --dir frontend lint
	corepack pnpm@10.33.2 --dir frontend typecheck
	corepack pnpm@10.33.2 --dir frontend build
	corepack pnpm@10.33.2 --dir frontend test

operations-check:
	bash -n scripts/deploy-canary.sh scripts/deploy-app.sh scripts/publish-frontend.sh tests/e2e/canary.sh
	python3 -m unittest discover -s scripts/tests -v
	node --test tests/load/redirect.test.mjs
