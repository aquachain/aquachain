#$(info loading mkconfig.mk ...)
# the actual go command
GOCMD ?= $(shell which go)
ifeq (x,x$(GOCMD))
$(error "go command not found in PATH")
exit 1
endif

PWD != pwd
# might be set by user or recursive make
GOOS != ${GOCMD} env GOOS
GOARCH != ${GOCMD} env GOARCH
GOPATH != ${GOCMD} env GOPATH
# rebuild target if *any* go file changes
GOFILES ?= $(shell find . -name '*.go' -type f -not \( -path "./vendor/*" -o -path "./build/*" \))
COMMITHASH ?= $(shell git rev-parse --short=6 HEAD)


version ?= $(shell git describe --tags --always --dirty)
# ci: fix version if not tagged ('checkout' actions module?)
ifeq (,$(findstring v,$(version)))
version := $(shell echo v0.0.0-$(version) | cut -c2-)
endif

# build flags and tags (TODO: use a separator instead of base64)
tags ?= netgo osusergo static
TAGS64 ?= $(shell printf "$(tags)"|base64 | tr -d '\r\n' | tr -d '\n' || true)
#$(info tags = "$(tags)" b64: $(TAGS64))

# export env used by recursive make
export GOOS GOARCH GOPATH GOCMD GOFLAGS TAGS64

# aquachain command for building each target version
aquachain_cmd := ./cmd/aquachain

# windows extension maybe
winextension :=
ifeq (windows,$(GOOS))
winextension = .exe
endif

# maincmd_name will be the name of the binary
maincmd_name := aquachain-$(version)
#$(info maincmd_name = $(maincmd_name))

# output release tarballs here
release_dir ?= release
# the hash for release files
hashfn ?= sha384sum
# linter version
golangci_linter_version ?= v1.64.5

# disable cgo by default, but allowed for now
CGO_ENABLED ?= 0
cgo ?= ${CGO_ENABLED}
ifeq (1,$(cgo))
CGO_ENABLED = 1
endif
export CGO_ENABLED
#$(info CGO_ENABLED = $(cgo))

# change ${GOCMD} build flags
GO_FLAGS ?= 
GO_FLAGS += -trimpath -v

# verbose build (super verbose)
ifeq (1,$(verbose))
GO_FLAGS += -x
endif

ifeq (1,$(release))
GO_FLAGS += -a
endif

# cgo specific flags
ifeq (1,$(cgo))
GO_FLAGS += -installsuffix cgo
LINKER_FLAGS += -linkmode external -extldflags -static
endif

# linker flags
LINKER_FLAGS ?= -s -w
LINKER_FLAGS += -buildid= -X main.gitCommit=${COMMITHASH} -X main.gitTag=${version} -X main.buildDate=${shell date -u +%s}
LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.buildtags=${TAGS64}

# codename to be used in version string
ifeq (,$(codename))
ifeq (1,$(release))
codename=release
endif
endif
ifneq (,$(codename))
LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.VersionMeta=${codename}
endif

# go ldflags escaping aaaaaahhhhh
GO_FLAGS += -ldflags '$(LINKER_FLAGS)'
#$(info GO_FLAGS = $(GO_FLAGS))


