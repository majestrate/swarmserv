REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_VERSION ?= $(shell test -e .git && ( echo -n "-" && git rev-parse --short HEAD ) || true)

NS_ROOT := github.com/majestrate

ifdef GOROOT
	GO = $(GOROOT)/bin/go
else
	GO = $(shell which go)
endif

EXE = swarmserver

TAGS ?= release

all: clean build

build: $(EXE)

$(EXE): 
	$(GO) build -v -ldflags "-X $(NS_ROOT)/swarmserv/version.Git=$(GIT_VERSION)" -tags='$(TAGS)' -o $(EXE)

test:
	$(GO) test $(NS_ROOT)/swarmserv/...

go-clean:
	$(GO) clean

clean: go-clean
	$(RM) $(EXE)

