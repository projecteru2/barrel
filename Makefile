.PHONY: deps build test binary mocks

REPO_PATH := github.com/projecteru2/barrel
REVISION := $(shell git rev-parse HEAD || unknown)
BUILTAT := $(shell date +%Y-%m-%dT%H:%M:%S)
BUILD_PATH := target
VERSION := $(shell git tag --contains $(shell git rev-list --tags --max-count=1) | sed '/-/!{s/$$/_/}' | sort -V -r | sed 's/_$$//' | head -1)
GO_LDFLAGS ?= -X $(REPO_PATH)/versioninfo.REVISION=$(REVISION) \
			  -X $(REPO_PATH)/versioninfo.BUILTAT=$(BUILTAT) \
			  -X $(REPO_PATH)/versioninfo.VERSION=$(VERSION)

binary:
	go build -ldflags "$(GO_LDFLAGS)" -a -tags "netgo osusergo" -installsuffix netgo -o eru-barrel
	go build -ldflags "-s -w $(GO_LDFLAGS)" -a -tags "netgo osusergo" -installsuffix netgo -o eru-barrel-utils cmd/ctr/ctr.go

strip: binary
	objcopy --only-keep-debug eru-barrel eru-barrel.debug
	objcopy --strip-debug --add-gnu-debuglink=eru-barrel.debug eru-barrel eru-barrel.release

test:
	go vet `go list ./... | grep -v '/vendor/' | grep -v '/tools'`
	go test -timeout 30s -count=1 -cover \
		./app/... \
		./proxy/... \
		./vessel/... \
		./utils/... \
		./resources/... 

cloc:
	cloc --exclude-dir=vendor,3rdmocks,mocks,tools --not-match-f=test .

lint:
	golangci-lint run

build: test binary

deps:
	env GO111MODULE=on go mod download
	env GO111MODULE=on go mod vendor

mocks:
	cd vessel; \
	mockery --name FixedIPAllocator && \
	mockery --name CalicoIPAllocator && \
	mockery --name ContainerVessel && \
	mockery --name DockerNetworkManager	