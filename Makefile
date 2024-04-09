CMD=wsstat
PACKAGE_NAME=github.com/jakobilobi/${CMD}
OS_ARCH_PAIRS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

build:
	go build -o bin/${CMD} main.go

build-all: $(OS_ARCH_PAIRS)

$(OS_ARCH_PAIRS):
	GOOS=$(firstword $(subst /, ,$@)) \
	GOARCH=$(lastword $(subst /, ,$@)) \
	go build -o 'bin/$(CMD)-$(firstword $(subst /, ,$@))-$(lastword $(subst /, ,$@))' main.go

explain:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the binary for the host OS/Arch."
	@echo "  build-all - Build binaries for all target OS/Arch pairs."
	@echo "  explain   - Display this help message"

.PHONY: build build-all $(OS_ARCH_PAIRS) explain

.DEFAULT_GOAL := explain
