VERSION = $(shell git describe --tags)
BUILD_LDFLAGS = "-X main.Version=${VERSION}"
GOFILES = $(shell find . -type f -name "*.go" | grep -vE '\./(.git|.wercker|vendor)' | xargs echo)

.PHONY: imports
imports:
	@goimports -d -e ${GOFILES}
	@if test "$(shell goimports -d -e ${GOFILES})" = ""; then echo pass; else echo failed; exit 1; fi

.PHONY: lint
lint:
	@golint -set_exit_status ./...

.PHONY: test
test:
	@go test ./...

.PHONY: build_cross
build_cross:
	goxz -pv ${VERSION} -os=linux,darwin,windows -arch=amd64 -build-ldflags=${BUILD_LDFLAGS} -d dist .
