.PHONY: test vet check build-lambdas

test:
	go -C backend test -race ./...

vet:
	go -C backend vet ./...

check: test vet

build-lambdas:
	go -C backend run ./tools/build-lambdas
