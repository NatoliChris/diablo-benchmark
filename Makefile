GOBIN := go
BUILDFLAGS := -v
PKG := "diablo-benchmark"
PKGFOLDERS := blockchains/... communication/... core/...

GOPATH=$(PWD)/.go
export GOPATH

default: diablo
	./diablo

all: lint diablo

reqs:
	GO111MODULE=off GO111MODULE=off go get -v golang.org/x/lint/golint
	$(GOBIN) mod download
	$(GOBIN) mod vendor

lint:
	@golint -set_exit_status $(PKGFOLDERS)

diablo:
	$(GOBIN) build $(BUILDFLAGS) -o $@

clean:
	rm diablo

.PHONY: default clean reqs diablo
