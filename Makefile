PLUGIN_ID := tokenrhythm-balance

ifeq ($(OS),Windows_NT)
PLUGIN_EXT := dll
else
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
PLUGIN_EXT := dylib
else
PLUGIN_EXT := so
endif
endif

OUTPUT ?= bin/$(PLUGIN_ID).$(PLUGIN_EXT)

.PHONY: build test fmt vet clean

build:
	go build -buildmode=c-shared -o $(OUTPUT) .
	rm -f bin/$(PLUGIN_ID).h

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin
