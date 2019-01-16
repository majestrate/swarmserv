REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_VERSION ?= $(shell test -e .git && ( echo -n "-" && git rev-parse --short HEAD ) || true)

ifdef GOROOT
	GO = $(GOROOT)/bin/go
else
	GO = $(shell which go)
endif

EXE = swarmserver

TAGS ?=

all: clean build

build: $(EXE)


$(EXE): 
	GOPATH=$(REPO) $(GO) build -a -ldflags "-X swarmserv/version.Git=$(GIT_VERSION)" -tags='$(TAGS)' -o $(EXE)

test:
	GOPATH=$(REPO) $(GO) test swarmserv/...

go-clean:
	GOPATH=$(REPO) $(GO) clean

clean: go-clean
	$(RM) $(EXE)

