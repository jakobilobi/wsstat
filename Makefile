CMD=wsstat
PACKAGE_NAME=github.com/jakobilobi/${CMD}

build:
	go build -o bin/${CMD} main.go

.PHONY: build

# TODO: Set the default goal to explain make options
.DEFAULT_GOAL := build
