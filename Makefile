GOBIN := go
BUILDFLAGS := -v
PKG := "diablo-benchmark"
PKG_LIST := $(shell go list ${PKG}/... | grep -v vendor/)

default: diablo

all: lint diablo

reqs:
	GO111MODULE=off GO111MODULE=off go get -v golang.org/x/lint/golint
	$(GOBIN) mod download
	# $(GOBIN) mod vendor

lint:
	@golint -set_exit_status ${PKG_LIST}

diablo:
	$(GOBIN) build $(BUILDFLAGS) -o $@

clean:
	rm diablo

.PHONY: default clean reqs diablo
