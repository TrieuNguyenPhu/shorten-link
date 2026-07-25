.PHONY: build-ShortenerFunction

ifeq ($(OS),Windows_NT)
BUILD_SHORTENER = powershell.exe -NoProfile -NonInteractive -Command "$$env:GOOS='linux'; $$env:GOARCH='arm64'; $$env:CGO_ENABLED='0'; go build -o '$(ARTIFACTS_DIR)/bootstrap' ./cmd/api"
else
BUILD_SHORTENER = GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o "$(ARTIFACTS_DIR)/bootstrap" ./cmd/api
endif

build-ShortenerFunction:
	$(BUILD_SHORTENER)
