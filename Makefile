GOBIN := go
BUILDFLAGS := -v

default: all

all: reqs diablo

reqs:
	$(GOBIN) mod download
	$(GOBIN) mod vendor

diablo:
	$(GOBIN) build $(BUILDFLAGS) -o diablo

clean:
	rm diablo

.PHONY: default clean reqs
