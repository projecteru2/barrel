.PHONY: deps build test binary

REPO_PATH := github.com/projecteru2/barrel
REVISION := $(shell git rev-parse HEAD || unknown)
BUILTAT := $(shell date +%Y-%m-%dT%H:%M:%S)
BUILD_PATH := target
VERSION := $(shell cat VERSION)
GO_LDFLAGS ?= -s -X $(REPO_PATH)/versioninfo.REVISION=$(REVISION) \
			  -X $(REPO_PATH)/versioninfo.BUILTAT=$(BUILTAT) \
			  -X $(REPO_PATH)/versioninfo.VERSION=$(VERSION)

clean:
	rm -rf target

deps:
	go mod download

test:

binary:
	go build -ldflags "$(GO_LDFLAGS)" -a -tags netgo -installsuffix netgo -o $(BUILD_PATH)/eru-barrel barrel.go

build: clean test binary