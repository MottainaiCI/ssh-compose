GOLANG_VERSION=$(shell go env GOVERSION)

# go tool nm ./luet | grep Commit
override LDFLAGS += -X "github.com/MottainaiCI/ssh-compose/cmd.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %Z')"
override LDFLAGS += -X "github.com/MottainaiCI/ssh-compose/cmd.BuildCommit=$(shell git rev-parse HEAD)"
override LDFLAGS += -X "github.com/MottainaiCI/ssh-compose/cmd.BuildGoVersion=$(GOLANG_VERSION)"

NAME ?= ssh-compose
PACKAGE_NAME ?= $(NAME)
BUILD_PLATFORMS ?= -osarch="linux/amd64" -osarch="linux/386" -osarch="linux/arm"
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	GO111MODULE=off go get github.com/onsi/ginkgo/v2/ginkgo
	GO111MODULE=off go get github.com/onsi/gomega/...
	ginkgo -r -flake-attempts 3 ./...

.PHONY: coverage
coverage:
	go test ./... -coverprofile=coverage.txt -race -covermode=atomic

.PHONY: test-coverage
test-coverage:
	scripts/ginkgo.coverage.sh --codecov

.PHONY: clean
clean:
	rm -rf release/

.PHONY: deps
deps:
	go env
	# Installing dependencies...
	GO111MODULE=off go get golang.org/x/lint/golint
	GO111MODULE=off go get github.com/mitchellh/gox
	GO111MODULE=on go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
	GO111MODULE=off go get github.com/onsi/gomega/...
	ginkgo version

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)'

.PHONY: build-small
build-small:
	@$(MAKE) LDFLAGS+="-s -w" build
	upx --brute -1 $(NAME)

.PHONY: lint
lint:
	golint ./... | grep -v "be unexported"

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: goreleaser-snapshot
goreleaser-snapshot:
	rm -rf dist/ || true
	goreleaser release --debug --skip-publish  --skip-validate --snapshot
