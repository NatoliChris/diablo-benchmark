GOBIN := go
BUILDFLAGS := -v

default: all

all: diablo

diablo:
	# mkdir -p $(PWD)/bin
	# $(GOBIN) build -v -o bin/diablo main/diablo.go
	$(GOBIN) build $(BUILDFLAGS) -o diablo main/diablo.go

clean:
	-rm -rf bin/*

.PHONY: default clean
