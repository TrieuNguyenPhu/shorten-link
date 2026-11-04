.PHONY: test vet check

test:
	go -C backend test -race ./...

vet:
	go -C backend vet ./...

check: test vet
