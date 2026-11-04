.PHONY: test vet check build-lambdas test-integration

test:
	go -C backend test -race ./...

test-integration:
	go -C backend test -race -tags=integration -count=1 ./internal/repository/... ./tools/local-http/...

vet:
	go -C backend vet ./...

check: test vet

build-lambdas:
	go -C backend run ./tools/build-lambdas
