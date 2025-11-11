#!/usr/bin/make -f

PACKAGES=$(shell go list ./... | grep -v '/simulation')
VERSION := $(shell git describe --abbrev=6 --dirty --always --tags)
COMMIT := $(shell git log -1 --format='%H')
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
BUILDDIR ?= $(CURDIR)/build
LEDGER_ENABLED ?= true

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(GAIA_BUILD_OPTIONS)))
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=althea \
	-X github.com/cosmos/cosmos-sdk/version.AppName=althea \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

# Shared security hardening flags for all builds
SHARED_SECURITY_CGO := -D_FORTIFY_SOURCE=2 -fstack-protector-strong

# Platform-specific linker flags
# Linux uses ELF-specific hardening flags, macOS uses different flags
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	# macOS: clang linker doesn't support -z flags
	SHARED_SECURITY_LD := -fstack-protector-strong
else
	# Linux: use full ELF hardening
	SHARED_SECURITY_LD := -Wl,-z,relro,-z,now -Wl,-z,noexecstack -fstack-protector-strong
endif

# Security hardening flags for dynamic builds (adds PIE)
SECURITY_FLAGS := GOFLAGS='-buildmode=pie' CGO_CPPFLAGS="$(SHARED_SECURITY_CGO)" CGO_LDFLAGS="$(SHARED_SECURITY_LD)"

# Static build flags with maximum security hardening including PIE via static-pie
# Using musl-gcc with -static-pie provides PIE protection for static binaries
# -buildmode=pie is required to prevent -no-pie from being added by the linker
# Note: Static builds are primarily for Linux (musl-gcc)
ifeq ($(UNAME_S),Darwin)
	STATIC_LDFLAGS := $(ldflags) -linkmode external -extldflags "-static $(SHARED_SECURITY_LD)"
else
	STATIC_LDFLAGS := $(ldflags) -linkmode external -extldflags "-static-pie $(SHARED_SECURITY_LD)"
endif
STATIC_BUILD_FLAGS := -buildmode=pie -tags "netgo osusergo static_build" -ldflags '$(STATIC_LDFLAGS)' -trimpath

all: install

# run go mod verify + go install
install: go.sum install-core

# does not run go mod verify
install-core:
		$(SECURITY_FLAGS) go install $(BUILD_FLAGS) ./cmd/althea

# build static binary with maximum security hardening (ledger support disabled for static builds)
static: go.sum
		mkdir -p $(BUILDDIR)
		CGO_ENABLED=1 CC=musl-gcc CGO_CPPFLAGS="$(SHARED_SECURITY_CGO)" CGO_LDFLAGS="$(SHARED_SECURITY_LD)" go build -a $(STATIC_BUILD_FLAGS) -o $(BUILDDIR)/althea ./cmd/althea

# install static binary to GOPATH/bin
static-install: static
		mkdir -p $(shell go env GOPATH)/bin
		cp $(BUILDDIR)/althea $(shell go env GOPATH)/bin/althea

# macOS build targets for release binaries
ifeq ($(UNAME_S),Darwin)
MACOS_TARGETS := x86_64-apple-darwin aarch64-apple-darwin

.PHONY: macos-build
macos-build: $(MACOS_TARGETS)

.PHONY: $(MACOS_TARGETS)
$(MACOS_TARGETS): go.sum
	@echo "Building for $@"
	@mkdir -p $(BUILDDIR)
	@if [ "$@" = "aarch64-apple-darwin" ]; then \
		export GOARCH=arm64; \
	else \
		export GOARCH=amd64; \
	fi; \
	export GOOS=darwin; \
	export CGO_ENABLED=1; \
	export GOFLAGS='-buildmode=pie'; \
	export CGO_CPPFLAGS="$(SHARED_SECURITY_CGO)"; \
	export CGO_LDFLAGS="$(SHARED_SECURITY_LD)"; \
	go build -tags "$(build_tags)" -ldflags '$(ldflags)' -o $(BUILDDIR)/althea-$@ ./cmd/althea
endif

# Windows build target for release binaries
.PHONY: windows-build
windows-build: go.sum
	@echo "Building for Windows x64"
	@mkdir -p $(BUILDDIR)
	@CGO_ENABLED=1 go build -tags "$(build_tags)" -ldflags '$(ldflags)' -o $(BUILDDIR)/althea-windows-amd64.exe ./cmd/althea

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

test:
	@go test -mod=readonly $(PACKAGES)

# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

###############################################################################
###                           Protobuf                                    ###
###############################################################################

proto-gen:
	./contrib/local/protocgen.sh

proto-lint:
	@$(DOCKER_BUF) lint --error-format=json

proto-check-breaking:
	@$(DOCKER_BUF) breaking --against "https://github.com/AltheaFoundation/althea-L1.git#branch=main"

TM_URL           = https://raw.githubusercontent.com/tendermint/tendermint/v0.34.0-rc3/proto/tendermint
GOGO_PROTO_URL   = https://raw.githubusercontent.com/regen-network/protobuf/cosmos
COSMOS_PROTO_URL = https://raw.githubusercontent.com/regen-network/cosmos-proto/master
COSMOS_SDK_PROTO_URL = https://raw.githubusercontent.com/cosmos/cosmos-sdk/master/proto/cosmos/base

TM_CRYPTO_TYPES     = third_party/proto/tendermint/crypto
TM_ABCI_TYPES       = third_party/proto/tendermint/abci
TM_TYPES     	    = third_party/proto/tendermint/types
TM_VERSION 			= third_party/proto/tendermint/version
TM_LIBS				= third_party/proto/tendermint/libs/bits

GOGO_PROTO_TYPES    = third_party/proto/gogoproto
COSMOS_PROTO_TYPES  = third_party/proto/cosmos_proto

SDK_ABCI_TYPES  	= third_party/proto/cosmos/base/abci/v1beta1
SDK_QUERY_TYPES  	= third_party/proto/cosmos/base/query/v1beta1
SDK_COIN_TYPES  	= third_party/proto/cosmos/base/v1beta1

proto-update-deps:
	# TODO: also download
	# - google/api/annotations.proto
	# - google/api/http.proto
	# - google/api/httpbody.proto
	# - google/protobuf/any.proto
	mkdir -p $(GOGO_PROTO_TYPES)
	curl -sSL $(GOGO_PROTO_URL)/gogoproto/gogo.proto > $(GOGO_PROTO_TYPES)/gogo.proto

	mkdir -p $(COSMOS_PROTO_TYPES)
	curl -sSL $(COSMOS_PROTO_URL)/cosmos.proto > $(COSMOS_PROTO_TYPES)/cosmos.proto

	mkdir -p $(TM_ABCI_TYPES)
	curl -sSL $(TM_URL)/abci/types.proto > $(TM_ABCI_TYPES)/types.proto

	mkdir -p $(TM_VERSION)
	curl -sSL $(TM_URL)/version/types.proto > $(TM_VERSION)/types.proto

	mkdir -p $(TM_TYPES)
	curl -sSL $(TM_URL)/types/types.proto > $(TM_TYPES)/types.proto
	curl -sSL $(TM_URL)/types/evidence.proto > $(TM_TYPES)/evidence.proto
	curl -sSL $(TM_URL)/types/params.proto > $(TM_TYPES)/params.proto

	mkdir -p $(TM_CRYPTO_TYPES)
	curl -sSL $(TM_URL)/crypto/proof.proto > $(TM_CRYPTO_TYPES)/proof.proto
	curl -sSL $(TM_URL)/crypto/keys.proto > $(TM_CRYPTO_TYPES)/keys.proto

	mkdir -p $(TM_LIBS)
	curl -sSL $(TM_URL)/libs/bits/types.proto > $(TM_LIBS)/types.proto

	mkdir -p $(SDK_ABCI_TYPES)
	curl -sSL $(COSMOS_SDK_PROTO_URL)/abci/v1beta1/abci.proto > $(SDK_ABCI_TYPES)/abci.proto

	mkdir -p $(SDK_QUERY_TYPES)
	curl -sSL $(COSMOS_SDK_PROTO_URL)/query/v1beta1/pagination.proto > $(SDK_QUERY_TYPES)/pagination.proto

	mkdir -p $(SDK_COIN_TYPES)
	curl -sSL $(COSMOS_SDK_PROTO_URL)/v1beta1/coin.proto > $(SDK_COIN_TYPES)/coin.proto

PREFIX ?= /usr/local
BIN ?= $(PREFIX)/bin
UNAME_S ?= $(shell uname -s)
UNAME_M ?= $(shell uname -m)

BUF_VERSION ?= 0.41.0

PROTOC_VERSION ?= 3.16.0
ifeq ($(UNAME_S),Linux)
  PROTOC_ZIP ?= protoc-${PROTOC_VERSION}-linux-x86_64.zip
endif
ifeq ($(UNAME_S),Darwin)
  PROTOC_ZIP ?= protoc-${PROTOC_VERSION}-osx-x86_64.zip
endif

proto-tools: proto-tools-stamp buf

proto-tools-stamp:
	echo "Installing protoc compiler..."
	(cd /tmp; \
	curl -OL "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}"; \
	unzip -o ${PROTOC_ZIP} -d $(PREFIX) bin/protoc; \
	unzip -o ${PROTOC_ZIP} -d $(PREFIX) 'include/*'; \
	rm -f ${PROTOC_ZIP})

	echo "Installing protoc-gen-gocosmos..."
	go install github.com/regen-network/cosmos-proto/protoc-gen-gocosmos

	echo "Installing protoc-gen-grpc-gateway..."
	go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0

	# Create dummy file to satisfy dependency and avoid
	# rebuilding when this Makefile target is hit twice
	# in a row
	touch $@

buf: buf-stamp

buf-stamp:
	@echo "Installing buf..."
	curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-${UNAME_S}-${UNAME_M}" \
    -o "${BIN}/buf" && \
	chmod +x "${BIN}/buf"

	touch $@

proto-tools-clean:
	rm -f proto-tools-stamp buf-stamp

BUILD_TARGETS := build

build: BUILD_ARGS=-o $(BUILDDIR)/

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

###############################################################################
###                           MISC DIRECTIVES                               ###
###############################################################################

tools: proto-tools buf

clean: proto-tools-clean
	rm -rf $(BUILDDIR)/ artifacts/
