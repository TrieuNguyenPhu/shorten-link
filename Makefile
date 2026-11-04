.PHONY: test vet check build-lambdas test-integration terraform-check

test:
	go -C backend test -race ./...

test-integration:
	go -C backend test -race -tags=integration -count=1 ./internal/repository/... ./tools/local-http/...

vet:
	go -C backend vet ./...

check: test vet terraform-check

build-lambdas:
	go -C backend run ./tools/build-lambdas

terraform-check: build-lambdas
	terraform fmt -check -recursive infrastructure/terraform
	@for root in infrastructure/terraform/bootstrap infrastructure/terraform/environments/dev; do \
		terraform -chdir="$$root" init -backend=false -input=false -lockfile=readonly && \
		terraform -chdir="$$root" validate -no-color && \
		terraform -chdir="$$root" test -no-color || exit $$?; \
	done
