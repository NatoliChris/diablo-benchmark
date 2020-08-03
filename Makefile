GOBIN := go

default: all

all: diablo

diablo:
	mkdir -p $(PWD)/bin
	$(GOBIN) build -v -o bin/diablo main/diablo.go

clean:
	-rm -rf bin/*

.PHONY: default clean
