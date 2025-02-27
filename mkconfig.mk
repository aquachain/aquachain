# the go program
GOCMD ?= $(shell which go)
GOOS ?= $(shell ${GOCMD} env GOOS)
GOARCH ?= $(shell ${GOCMD} env GOARCH)
GOPATH ?= $(shell go env GOPATH)
# for installation
PREFIX ?= /usr/local
export GOFILES=$(shell find . -name '*.go' -type f -not \( -path "./vendor/*" -o -path "./build/*" \))
export GOOS GOARCH GOPATH
# build flags and tags
tags ?= netgo osusergo static
TAGS64 := $(shell printf "$(tags)"|base64 | tr -d '\r\n' | tr -d '\n')
LINKER_FLAGS ?= -s -w
# rebuild target if *any* go file changes


aquachain_cmd=./cmd/aquachain
COMMITHASH := ${GITHUB_SHA}
version  :=  $(shell git describe --tags --dirty --always 2>/dev/null || cat VERSION 2>/dev/null || echo "0.0.0")
ifeq (,$(COMMITHASH))
COMMITHASH := $(shell git rev-parse --short=6 HEAD)
endif
winextension :=
ifeq (windows,$(GOOS))
winextension = .exe
endif
maincmd_name := aquachain-$(version)$(winextension)
PWD != pwd
build_dir ?= ./bin
INSTALL_DIR ?= $(PREFIX)/bin
release_dir=release
hashfn := sha384sum
golangci_linter_version := v1.64.5
main_deps := $(filter %.go,$(wildcard *.go */*.go */*/*.go */*/*/*.go */*/*/*/*.g))
cgo=$(CGO_ENABLED)

# disable cgo by default
CGO_ENABLED ?= 0
ifeq (1,$(cgo))
CGO_ENABLED = 1
endif

# change ${GOCMD} build flags
GO_FLAGS ?= 
GO_FLAGS += -trimpath -v

# verbose build (super verbose)
ifeq (1,$(verbose))
GO_FLAGS += -x
endif

# build all commands
ifeq (all,$(cmds))
aquachain_cmd=./cmd/...
endif


ifneq (1,$(cgo))
#GO_FLAGS += -tags 'netgo osusergo static $(GO_TAGS)'
else
GO_FLAGS += -installsuffix cgo
LINKER_FLAGS += -linkmode external -extldflags -static
endif

LINKER_FLAGS = -w -s -buildid= -X main.gitCommit=${COMMITHASH} -X main.gitTag=${version} -X main.buildDate=${shell date -u +%s}
LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.buildtags=${TAGS64}
## if release=1, rebuild all sources
ifeq (1,$(release))
codename = $(shell echo "${version}" | grep "-" | cut -d- -f2)
GO_FLAGS += -a
ifeq (,$(codename))
codename=release
endif
endif

# codename to be used in version string
ifneq (,$(codename))
LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.VersionMeta=${codename}
endif

# go ldflags escaping aaaaaahhhhh
GO_FLAGS += -ldflags '$(LINKER_FLAGS)'
# export GOFILES=$(shell find . -iname '*.go' -type f -not \( -path "./vendor/*" -o -path "./build/*" \) | grep -v /vendor/ | grep -v /build/)
# build shorttarget, aquachain for host OS/ARCH
# shorttarget = "bin/aquachain" or "bin/aquachain.exe"
