MODULE:=powerjob-client-go
PKG:=./...

GO:=go
MOCKGEN:=$(shell which mockgen 2>/dev/null)

.PHONY: tidy build test cover run mock lint fmt tools

tools:
	@if [ -z "$(MOCKGEN)" ]; then \
		GOBIN=$$(pwd)/bin $(GO) install github.com/golang/mock/mockgen@v1.6.0; \
		printf "installed mockgen to bin/\n"; \
	fi

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt $(PKG)

lint:
	@echo "(可选) 这里可集成 golangci-lint"

build:
	$(GO) build $(PKG)

test:
	$(GO) test -race -cover -coverprofile=coverage.out $(PKG)

cover:
	$(GO) tool cover -func=coverage.out

mock: tools
	@MOCK=$$(which mockgen 2>/dev/null || echo bin/mockgen); \
	GOMODCACHE=$$(pwd)/.gomodcache GOPATH=$$(pwd)/.gopath $$MOCK -destination=mocks/mock_serverapi.go -package=mocks \
		powerjob-client-go/client ServerAPI
