CMD=wsstat
PACKAGE_NAME=github.com/jakobilobi/${CMD}
OS_ARCH_PAIRS = linux/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/386 windows/amd64 windows/arm64
VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

build:
	go build ${LDFLAGS} -o bin/${CMD} $(PACKAGE_NAME)

build-all: $(OS_ARCH_PAIRS)

$(OS_ARCH_PAIRS):
	GOOS=$(firstword $(subst /, ,$@)) \
	GOARCH=$(lastword $(subst /, ,$@)) \
	go build ${LDFLAGS} -o 'bin/$(CMD)-$(firstword $(subst /, ,$@))-$(lastword $(subst /, ,$@))' $(PACKAGE_NAME)

explain:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the binary for the host OS/Arch."
	@echo "  build-all - Build binaries for all target OS/Arch pairs."
	@echo "  explain   - Display this help message"

.PHONY: build build-all $(OS_ARCH_PAIRS) explain

.DEFAULT_GOAL := explain
