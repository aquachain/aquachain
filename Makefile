# the go program
GOCMD ?= go

GOOS ?= $(shell ${GOCMD} env GOOS)
GOARCH ?= $(shell ${GOCMD} env GOARCH)
PREFIX ?= /usr/local
GOPATH ?= $(shell go env GOPATH)
tags ?= netgo osusergo static
LINKER_FLAGS ?= -s -w

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
COMMITHASH := ${GITHUB_SHA}
version  :=     $(shell git describe --tags --always --dirty)
ifeq (,$(version))
version := $(shell cat VERSION)
endif
ifeq (,$(COMMITHASH))
COMMITHASH := $(shell git rev-parse --short HEAD)
endif
winextension :=
ifeq (windows,$(GOOS))
winextension = .exe
endif
maincmd_name := aquachain-$(version)-$(COMMITHASH)$(winextension)
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

ifeq (1,$(verbose))
GO_FLAGS += -v 
endif

ifeq (all,$(cmds))
aquachain_cmd=./cmd/...
endif



# use ${GOCMD} for "net" and "os/user" packages (cgo by default)
#GO_TAGS := static

TAGS64 := $(shell printf "$(tags)"|base64 | tr -d '\r\n' | tr -d '\n')
ifneq (1,$(cgo))
#GO_FLAGS += -tags 'netgo osusergo static $(GO_TAGS)'
else
GO_FLAGS += -installsuffix cgo
LINKER_FLAGS += -linkmode external -extldflags -static
endif

LINKER_FLAGS = -X main.gitCommit=${COMMITHASH} -X main.gitTag=${version} -X main.buildDate=${shell date -u +%s} -s -w 
LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.buildtags=${TAGS64}

## if release=1, rebuild all sources
ifeq (1,$(release))
codename = $(shell echo "${version}" | grep "-" | cut -d- -f2)
GO_FLAGS += -a
ifeq (,$(codename))
codename=release
endif
endif

ifeq (,$(codename))
codename=dev
endif

LINKER_FLAGS += -X gitlab.com/aquachain/aquachain/params.VersionMeta=${codename}
GO_FLAGS += -ldflags '$(LINKER_FLAGS)'

# rebuild if any go file changes
export GOFILES=$(shell find . -name '*.go' -type f -not \( -path "./vendor/*" -o -path "./build/*" \))
# export GOFILES=$(shell find . -iname '*.go' -type f -not \( -path "./vendor/*" -o -path "./build/*" \) | grep -v /vendor/ | grep -v /build/)
# build shorttarget, aquachain for host OS/ARCH
# shorttarget = "bin/aquachain" or "bin/aquachain.exe"
shorttarget=$(build_dir)/aquachain$(winextension)
default_arch_target=$(build_dir)/$(maincmd_name)-$(GOOS)-$(GOARCH)$(winextension)
$(shorttarget): $(GOFILES)
	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)
	@echo compiled: $(shorttarget)
	@sha256sum $(shorttarget) 2>/dev/null || true
	@file $(shorttarget) 2>/dev/null || true
default: $(shorttarget)
echo: # useful lol
	@echo "GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)"
	@echo GOCMD $(GOCMD)
	@echo GOFILES $(GOFILES)
	@echo shorttarget $(shorttarget)
	@echo default_arch_target $(default_arch_target)
	@echo GO_FLAGS $(GO_FLAGS)
	@echo aquachain_cmd $(aquachain_cmd)
	@echo tags $(tags)
	@echo GOTAGS $(GOTAGS)
	@echo GOOS $(GOOS)
	@echo GOARCH $(GOARCH)
	@echo COMMITHASH $(COMMITHASH)
	@echo version $(version)
	@echo codename $(codename)
	@echo LINKER_FLAGS $(LINKER_FLAGS)
	@echo TAGS64 $(TAGS64)
	@echo cgo $(cgo)

# default: $(default_arch_target)
# 	@echo compiled: $<
# 	@sha256sum $< 2>/dev/null || true
# 	@file $< 2>/dev/null || true
bootnode: bin/aquabootnode
bin/aquabootnode: $(GOFILES)
	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o bin/aquabootnode ./cmd/aquabootnode

.PHONY += default bootnode hash
echoflags:
	@echo "CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build $(GO_FLAGS) -o $@ $(aquachain_cmd)"

.PHONY += install
install:
	install -v $(build_dir)/aquachain $(INSTALL_DIR)/
internal/jsre/deps/bindata.go: internal/jsre/deps/web3.js  internal/jsre/deps/bignumber.js
	@test -x "$(shell which go-bindata)" || echo 'go-bindata not found in PATH. run make devtools to install required development dependencies PATH=${PATH}'
	test ! -x "$(shell which go-bindata)" || go generate -v ./$(shell dirname $@)/...
all:
	mkdir -p $(build_dir)
	cd $(build_dir) && \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../cmd/...

cross:
	@echo to build a release, use "make clean release release=1"
	mkdir -p $(build_dir)
	cd $(build_dir) && mkdir -p linux freebsd osx windows
	cd $(build_dir)/linux && GOOS=linux \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../.${aquachain_cmd}
	cd $(build_dir)/freebsd && GOOS=freebsd \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../.${aquachain_cmd}
	cd $(build_dir)/osx && GOOS=darwin \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../.${aquachain_cmd}
	cd $(build_dir)/windows && GOOS=windows \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../.${aquachain_cmd}


help:
	@echo
	@echo without args, target is: "$(shorttarget)"
	@echo 'make build', target is: "$(default_arch_target)"
	@echo 'make install' target is: "$(INSTALL_DIR)/"
	@echo using go flags: "$(GO_FLAGS)"
	@echo
	@echo to compile quickly, run \'make\' and then \'$(shorttarget) help\'
	@echo to install system-wide, run something like \'sudo make install\'
	@echo
	@echo "to cross-compile, try 'make cross' or 'make GOOS=windows'"
	@echo "to add things left out by default, use tags: 'make tags=metrics'"
	@echo
	@echo "clean compile package release: 'make clean release release=1'"
	@echo
	@echo "cross-compile release: 'make clean cross release=1'"
	@echo "cross-compile all tools: 'make clean cross release=1 cmds=all'"
	@echo "compile with cgo and usb support: make cgo=1 tags=usb'"
	@echo
	@echo note: this help response is dynamic and reacts to environmental variables.

test:
	CGO_ENABLED=0 bash testing/test-short-only.bash $(args)
race:
	CGO_ENABLED=1 bash testing/test-short-only.bash -race


checkrelease:
ifneq (1,$(release))
	echo "use make release release=1"
	exit 1
endif
release: checkrelease package hash
clean:
	rm -rf bin release docs tmprelease
hash: release/SHA384.txt
release/SHA384.txt:
	$(hashfn) release/*.tar.gz release/*.zip | tee $@

release_files := \
	$(maincmd_name)-linux-amd64 \
	$(maincmd_name)-linux-arm \
	$(maincmd_name)-linux-riscv64 \
	$(maincmd_name)-windows-amd64.exe \
	$(maincmd_name)-freebsd-amd64 \
	$(maincmd_name)-openbsd-amd64 \
	$(maincmd_name)-netbsd-amd64 \
	$(maincmd_name)-osx-amd64

# cross compile for each target OS/ARCH
crossold:	$(addprefix $(build_dir)/, $(release_files))
.PHONY += cross


## build binaries for each OS
$(build_dir)/$(maincmd_name)-linux-amd64: $(main_deps)
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-arm: $(main_deps)
	GOOS=linux \
	GOARCH=arm \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-arm64: $(main_deps)
	GOOS=linux \
	GOARCH=arm64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-riscv64: $(main_deps)
	GOOS=linux \
	GOARCH=riscv64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-windows-amd64.exe: $(main_deps)
	GOOS=windows \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -buildvcs=false $(GO_FLAGS) -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-osx-amd64: $(main_deps)
	GOOS=darwin \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -buildvcs=false $(GO_FLAGS) -o $@ $(aquachain_cmd)
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


## package above binaries 
package: $(release_dir)/$(maincmd_name)-windows-amd64.zip \
	$(release_dir)/$(maincmd_name)-osx-amd64.zip \
	$(release_dir)/$(maincmd_name)-linux-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-linux-riscv64.tar.gz \
	$(release_dir)/$(maincmd_name)-linux-arm.tar.gz \
	$(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz

# broken: $(release_dir)/$(maincmd_name)-linux-arm64.tar.gz



releasetexts := README.md COPYING AUTHORS
$(release_dir)/$(maincmd_name)-windows-amd64.zip: $(build_dir)/$(maincmd_name)-windows-amd64.exe
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-windows
	mkdir -p tmprelease/${maincmd_name}-windows
	cp -t tmprelease/${maincmd_name}-windows $^ ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-windows
$(release_dir)/$(maincmd_name)-osx-amd64.zip: $(build_dir)/$(maincmd_name)-osx-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-osx
	mkdir -p tmprelease/${maincmd_name}-osx
	cp -t tmprelease/${maincmd_name}-osx $^ ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-osx
$(release_dir)/$(maincmd_name)-linux-amd64.tar.gz: $(build_dir)/$(maincmd_name)-linux-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-linux
	mkdir -p tmprelease/${maincmd_name}-linux
	cp -t tmprelease/${maincmd_name}-linux $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux
$(release_dir)/$(maincmd_name)-linux-arm.tar.gz: $(build_dir)/$(maincmd_name)-linux-arm
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-linux-arm
	mkdir -p tmprelease/${maincmd_name}-linux-arm
	cp -t tmprelease/${maincmd_name}-linux-arm $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-arm
$(release_dir)/$(maincmd_name)-linux-arm64.tar.gz: $(build_dir)/$(maincmd_name)-linux-arm64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-linux-arm64
	mkdir -p tmprelease/${maincmd_name}-linux-arm64
	cp -t tmprelease/${maincmd_name}-linux-arm64 $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-arm64
$(release_dir)/$(maincmd_name)-linux-riscv64.tar.gz: $(build_dir)/$(maincmd_name)-linux-riscv64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-linux-riscv64
	mkdir -p tmprelease/${maincmd_name}-linux-riscv64
	cp -t tmprelease/${maincmd_name}-linux-riscv64 $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-riscv64
$(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-freebsd-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-freebsd
	mkdir -p tmprelease/${maincmd_name}-freebsd
	cp -t tmprelease/${maincmd_name}-freebsd $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-freebsd
$(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-openbsd-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-openbsd
	mkdir -p tmprelease/${maincmd_name}-openbsd
	cp -t tmprelease/${maincmd_name}-openbsd $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-openbsd
$(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz: $(build_dir)/$(maincmd_name)-netbsd-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-netbsd
	mkdir -p tmprelease/${maincmd_name}-netbsd
	cp -t tmprelease/${maincmd_name}-netbsd $^ ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-netbsd

.PHONY += hash


devtools:
	${GOCMD} install golang.org/x/tools/cmd/stringer@latest
	${GOCMD} install github.com/kevinburke/go-bindata/v4/...@latest
	${GOCMD} install github.com/fjl/gencodec@latest
#	env GOBIN= ${GOCMD} get github.com/golang/protobuf/protoc-gen-go@latest
	${GOCMD} install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#	${GOCMD} install gitlab.com/aquachain/x/cmd/aqua-abigen@latest
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm (eg. https://github.com/nvm-sh/nvm)'
	@type "solc" 2> /dev/null || echo 'Please install solc (eg. https://github.com/ethereum/solidity/releases)'
	@type "protoc" 2> /dev/null || echo 'Please install protoc (eg. apt install protobuf-compiler)'

generate: devtools
	${GOCMD} generate ./...

goget:
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} get -v -t -d ./...

linter: bin/golangci-lint
	CGO_ENABLED=0 ./bin/golangci-lint -v run \
	  --config .golangci.yml \
	  --build-tags static,netgo,osusergo \
	  -v

bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s $(golangci_linter_version)

docs: mkdocs.yml Documentation/*/*
	@echo building docs
	mkdocs build

docker:
	docker build -t aquachain/aquachain .
