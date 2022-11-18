#
# Borrowed from: https://github.com/gopasspw/gopass/blob/master/Makefile
#
FIRST_GOPATH           := $(firstword $(subst :, ,$(GOPATH)))
PKGS                   := $(shell go list ./...)
GOFILES_NOVENDOR       := $(shell find . -name vendor -prune -o -type f -name '*.go' -not -name '*.pb.go' -print)
GOFILES_BUILD          := $(shell find . -type f -name '*.go' -not -name '*_test.go')
GOTAS_VERSION 				 ?= $(shell git describe --tags --always --dirty)
GOTAS_OUTPUT           ?= gotas
GOTAS_REVISION         := $(shell git describe --tags --always --dirty)
DATE                   := $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" '+%FT%T%z' 2>/dev/null || date -u '+%FT%T%z')
BUILDFLAGS_NOPIE       := -trimpath -ldflags="-s -w -X main.version=$(GOTAS_VERSION) -X main.commit=$(GOTAS_REVISION) -X main.date=$(DATE)" -gcflags="-trimpath=$(GOPATH)" -asmflags="-trimpath=$(GOPATH)"
BUILDFLAGS             ?= $(BUILDFLAGS_NOPIE) -buildmode=pie
TESTFLAGS              ?=
PWD                    := $(shell pwd)
PREFIX                 ?= $(GOPATH)
BINDIR                 ?= $(PREFIX)/bin
GO                     := GO111MODULE=on go
GOOS                   ?= $(shell go version | cut -d' ' -f4 | cut -d'/' -f1)
GOARCH                 ?= $(shell go version | cut -d' ' -f4 | cut -d'/' -f2)
TAGS                   ?= 

export GO111MODULE=on

OK := $(shell tput setaf 6; echo ' [OK]'; tput sgr0;)

all: sysinfo build
build: $(GOTAS_OUTPUT)

sysinfo:
	@echo ">> SYSTEM INFORMATION"
	@echo -n "     PLATFORM   : $(shell uname -a)"
	@printf '%s\n' '$(OK)'
	@echo -n "     PWD:       : $(shell pwd)"
	@printf '%s\n' '$(OK)'
	@echo -n "     GO         : $(shell go version)"
	@printf '%s\n' '$(OK)'
	@echo -n "     BUILDFLAGS : $(BUILDFLAGS)"
	@printf '%s\n' '$(OK)'
	@echo -n "     GIT        : $(shell git version)"
	@printf '%s\n' '$(OK)'
	@echo -n "     GPG        : $(shell which gpg) $(shell gpg --version | head -1)"
	@printf '%s\n' '$(OK)'
	@echo -n "     GPGAgent   : $(shell which gpg-agent) $(shell gpg-agent --version | head -1)"
	@printf '%s\n' '$(OK)'

clean:
	@echo -n ">> CLEAN"
	@$(GO) clean -i ./...
	@rm -f ./coverage-all.html
	@rm -f ./coverage-all.out
	@rm -f ./coverage.out
	@find . -type f -name "coverage.out" -delete
	@rm -f gotas-*-*
	@rm -f tests/tests
	@rm -f *.test
	@rm -rf dist/*
	@printf '%s\n' '$(OK)'

$(GOTAS_OUTPUT): $(GOFILES_BUILD)
	@echo -n ">> BUILD, version = $(GOTAS_VERSION)/$(GOTAS_REVISION), output = $@"
	@$(GO) build -o $@ $(BUILDFLAGS)
	@printf '%s\n' '$(OK)'

fulltest: $(GOTAS_OUTPUT)
	@echo ">> TEST, \"full-mode\": race detector off"
	@echo "mode: atomic" > coverage-all.out
	@$(foreach pkg, $(PKGS),\
	    echo -n "     ";\
		go test -run '(Test|Example)' $(BUILDFLAGS) $(TESTFLAGS) -coverprofile=coverage.out -covermode=atomic $(pkg) || exit 1;\
		tail -n +2 coverage.out >> coverage-all.out;)
	@$(GO) tool cover -html=coverage-all.out -o coverage-all.html

test: $(GOTAS_OUTPUT)
	@echo ">> TEST, \"fast-mode\": race detector off"
	@$(foreach pkg, $(PKGS),\
	    echo -n "     ";\
		$(GO) test -test.short -run '(Test|Example)' $(BUILDFLAGS) $(TESTFLAGS) $(pkg) || exit 1;)

test-win: $(GOTAS_OUTPUT)
	@echo ">> TEST, \"fast-mode-win\": race detector off"
	@$(foreach pkg, $(PKGS),\
		$(GO) test -test.short -run '(Test|Example)' $(pkg) || exit 1;)

codequality:
	@echo ">> CODE QUALITY"

	@echo -n "     REVIVE    "
	@which revive > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mgechev/revive; \
	fi
	@revive -formatter friendly -exclude vendor/... ./...
	@printf '%s\n' '$(OK)'

	@echo -n "     FMT       "
	@$(foreach gofile, $(GOFILES_NOVENDOR),\
			out=$$(gofmt -s -l -d -e $(gofile) | tee /dev/stderr); if [ -n "$$out" ]; then exit 1; fi;)
	@printf '%s\n' '$(OK)'

	@echo -n "     VET       "
	@$(GO) vet ./...
	@printf '%s\n' '$(OK)'

	@echo -n "     CYCLO     "
	@which gocyclo > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@$(foreach gofile, $(GOFILES_NOVENDOR),\
			gocyclo -over 22 $(gofile) || exit 1;)
	@printf '%s\n' '$(OK)'

	@echo -n "     INEFF     "
	@which ineffassign > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) install github.com/gordonklaus/ineffassign@latest; \
	fi
	@ineffassign . || exit 1
	@printf '%s\n' '$(OK)'

	@echo -n "     SPELL     "
	@which misspell > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) install github.com/client9/misspell/cmd/misspell@latest; \
	fi
	@$(foreach gofile, $(GOFILES_NOVENDOR),\
			misspell --error $(gofile) || exit 1;)
	@printf '%s\n' '$(OK)'

	@echo -n "     STATICCHECK "
	@which staticcheck > /dev/null; if [ $$? -ne 0  ]; then \
		$(GO) install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@staticcheck $(PKGS) || exit 1
	@printf '%s\n' '$(OK)'

	@echo -n "     UNPARAM "
	@which unparam > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) install mvdan.cc/unparam@latest; \
	fi
	@unparam -exported=false $(PKGS)
	@printf '%s\n' '$(OK)'

gen:
	@go generate ./...

fmt:
	@gofmt -s -l -w $(GOFILES_NOVENDOR)
	@goimports -l -w $(GOFILES_NOVENDOR)
	@go mod tidy

deps:
	@go build -v ./...

upgrade: gen fmt
	@go get -u ./...

.PHONY: clean build install sysinfo test codequality 

