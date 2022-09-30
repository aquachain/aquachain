GOCMD ?= go # the go program
GOOS ?= $(shell ${GOCMD} env GOOS)
GOARCH ?= $(shell ${GOCMD} env GOARCH)
PREFIX ?= /usr/local
GOPATH ?= $(shell go env GOPATH)
GOTAGS ?= netgo osusergo static

export GOPATH
define LOGO
                              _           _
  __ _  __ _ _   _  __ _  ___| |__   __ _(_)_ __
 / _ '|/ _' | | | |/ _' |/ __| '_ \ / _' | | '_ \ 
| (_| | (_| | |_| | (_| | (__| | | | (_| | | | | |
 \__,_|\__, |\__,_|\__,_|\___|_| |_|\__,_|_|_| |_|
          |_|
endef
$(info $(LOGO))
aquachain_cmd=./cmd/aquachain
version != cat VERSION
COMMITHASH != git rev-parse --short HEAD
winextension :=
ifeq (windows,$(GOOS))
winextension = .exe
endif
maincmd_name := aquachain-$(version)-$(COMMITHASH)$(winextension)
PWD != pwd
build_dir ?= ./bin
default_target=$(build_dir)/aquachain$(winextension)
INSTALL_DIR ?= $(PREFIX)/bin
release_dir=rel
hashfn := sha384sum
golangci_linter_version := v1.24.0
main_deps := $(filter %.go,$(wildcard *.go */*.go */*/*.go */*/*/*.go */*/*/*/*.g))
cgo=$(CGO_ENABLED)

# disable cgo by default
CGO_ENABLED ?= 0
ifeq (1,$(cgo))
CGO_ENABLED = 1
endif

# change ${GOCMD} build flags
GO_FLAGS ?= 

ifeq (1,$(verbose))
GO_FLAGS += -v 
endif



# use ${GOCMD} for "net" and "os/user" packages (cgo by default)
#GO_TAGS := static

TAGS64 := $(shell printf "$(GOTAGS) $(tags)"|base64 -w 0)
ifneq (1,$(cgo))
#GO_FLAGS += -tags 'netgo osusergo static $(GO_TAGS)'
else
GO_FLAGS += -installsuffix cgo
LD_FLAGS += -linkmode external -extldflags -static
endif

LD_FLAGS := -X main.gitCommit=${COMMITHASH} -X main.buildDate=${shell date -u +%s} -s -w 
LD_FLAGS += -X gitlab.com/aquachain/aquachain/params.buildTags=${TAGS64}
# if release=1, rebuild all sources
codename=dev
ifeq (1,$(release))
GO_FLAGS += -a
codename=release
endif
LD_FLAGS += -X gitlab.com/aquachain/aquachain/params.VersionMeta=${codename}
GO_FLAGS += -ldflags '$(LD_FLAGS)'

export GOFILES=$(shell find . -iname '*.go' -type f | grep -v /vendor/ | grep -v /build/)
# build default target, aquachain for host OS/ARCH
#bin/aquachain:
$(default_target): $(GOFILES)
	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(GOTAGS) $(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)
	@echo compiled: $(default_target)
	@sha256sum $(default_target) 2>/dev/null || true
	@file $(default_target) 2>/dev/null || true
default: $(build_dir)/$(maincmd_name)-$(GOOS)-$(GOARCH)$(winextension)

.PHONY += default hash
echoflags:
	@echo "CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build $(GO_FLAGS) -o $@ $(aquachain_cmd)"

.PHONY += install
install:
	install -v $(build_dir)/aquachain $(INSTALL_DIR)/

all:
	GOBIN=$(build_dir) CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} install $(GO_FLAGS)  ./cmd/...

cross:
	mkdir -p $(build_dir)
	cd $(build_dir) && mkdir -p linux freebsd osx windows
	cd $(build_dir)/linux && GOOS=linux CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../../cmd/...
	cd $(build_dir)/freebsd && GOOS=freebsd CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../../cmd/...
	cd $(build_dir)/osx && GOOS=darwin CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../../cmd/...
	cd $(build_dir)/windows && GOOS=windows CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../../cmd/...


help:
	@echo
	@echo default target is: "$(default_target)"
	@echo 'make install' target is: "$(INSTALL_DIR)/"
	@echo using go flags: "$(GO_FLAGS)"
	@echo
	@echo to compile quickly, run \'make\' and then \'$(default_target) help\'
	@echo to install system-wide, run something like \'sudo make install\'
	@echo
	@echo "to cross-compile, try 'make cross' or 'make GOOS=windows'"
	@echo "to add things left out by default, use tags: 'make cgo=1'"
	@echo
	@echo note: this help response is dynamic and reacts to environmental variables.

test:
	CGO_ENABLED=0 bash testing/test-short-only.bash $(args)
race:
	CGO_ENABLED=1 bash testing/test-short-only.bash -race


release: cross hash package
clean:
	rm -rf $(build_dir)
hash: bin/hashes.txt
bin/hashes.txt: $(wildcard $(build_dir)/aqua*)
	$(hashfn) $^ > $@
.PHONY += bin/hashes.txt

release_files := \
	$(maincmd_name)-linux-amd64 \
	$(maincmd_name)-linux-arm \
	$(maincmd_name)-windows-amd64.exe \
	$(maincmd_name)-freebsd-amd64 \
	$(maincmd_name)-openbsd-amd64 \
	$(maincmd_name)-netbsd-amd64 \
	$(maincmd_name)-osx-amd64

# cross compile for each target OS/ARCH
crossold:	$(addprefix $(build_dir)/, $(release_files))
.PHONY += cross
#$(build_dir)/aquachain.exe:
#	GOOS=windows \
#	GOARCH=amd64 \
#	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-amd64: $(main_deps)
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-arm: $(main_deps)
	GOOS=linux \
	GOARCH=arm \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-windows-amd64.exe: $(main_deps)
	GOOS=windows \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-osx-amd64: $(main_deps)
	GOOS=darwin \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-openbsd-amd64: $(main_deps)
	GOOS=openbsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-netbsd-amd64: $(main_deps)
	GOOS=netbsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-freebsd-amd64: $(main_deps)
	GOOS=freebsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)


package: $(addprefix bin/,$(release_files)) bin/hashes.txt \
	$(release_dir)/$(maincmd_name)-windows-amd64.zip \
	$(release_dir)/$(maincmd_name)-osx-amd64.zip \
	$(release_dir)/$(maincmd_name)-linux-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-linux-arm.tar.gz \
	$(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz \



releasetexts := README.md COPYING AUTHORS bin/hashes.txt
$(release_dir)/$(maincmd_name)-windows-amd64.zip: $(build_dir)/$(maincmd_name)-windows-amd64.exe
	mkdir -p $(release_dir)
	zip $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-osx-amd64.zip: $(build_dir)/$(maincmd_name)-osx-amd64
	mkdir -p $(release_dir)
	zip $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-linux-amd64.tar.gz: $(build_dir)/$(maincmd_name)-linux-amd64
	mkdir -p $(release_dir)
	tar czvf $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-linux-arm.tar.gz: $(build_dir)/$(maincmd_name)-linux-arm
	mkdir -p $(release_dir)
	tar czvf $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-freebsd-amd64
	mkdir -p $(release_dir)
	tar czvf $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-openbsd-amd64
	mkdir -p $(release_dir)
	tar czvf $@ $(releasetexts) $^
$(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-netbsd-amd64
	mkdir -p $(release_dir)
	tar czvf $@ $(releasetexts) $^

.PHONY += hash


devtools:
	env GOBIN= ${GOCMD} get golang.org/x/tools/cmd/stringer
	env GOBIN= ${GOCMD} get github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= ${GOCMD} get github.com/fjl/gencodec
	env GOBIN= ${GOCMD} get github.com/golang/protobuf/protoc-gen-go
	env GOBIN= ${GOCMD} install gitlab.com/aquachain/x/cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

generate: devtools
	${GOCMD} generate ./...

goget:
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} get -v -t -d ./...

linter: bin/golangci-lint
	CGO_ENABLED=0 ./bin/golangci-lint -v run \
	  --deadline 20m \
	  --config .golangci.yml \
	  --build-tags static,netgo,osusergo \
	  -v

bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s $(golangci_linter_version)
