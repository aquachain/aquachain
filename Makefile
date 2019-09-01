GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
PREFIX ?= /usr/local
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
maincmd_name := aquachain-$(version)-$(COMMITHASH)
build_dir=$(PWD)/bin
INSTALL_DIR ?= $(PREFIX)/bin/
release_dir=rel
hashfn := sha384sum
golangci_linter_version := v1.17.1
main_deps := $(filter %.go,$(wildcard *.go */*.go */*/*.go */*/*/*.go */*/*/*/*.g))

# disable cgo by default
CGO_ENABLED ?= 0
ifeq (1,$(cgo))
CGO_ENABLED = 1
endif

# change go build flags
GO_FLAGS ?= 

ifeq (1,$(verbose))
GO_FLAGS += -v 
endif

# if release=1, rebuild all sources
ifeq (1,$(release))
GO_FLAGS += -a
endif

# use go for "net" and "os/user" packages (cgo by default)
GO_TAGS := static
ifneq (1,$(CGO_ENABLED))
GO_TAGS += netgo osusergo
else
GO_FLAGS += -installsuffix static -tags netgo osusergo static
LD_FLAGS += -linkmode external -extldflags -static
endif

LD_FLAGS := -ldflags '-X main.gitCommit=${COMMITHASH} -X main.buildDate=${shell date -u +%s} -s -w' -v
GO_FLAGS += $(LD_FLAGS)

# build default target, aquachain for host OS/ARCH
$(build_dir)/aquachain:
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
default: $(build_dir)/$(maincmd_name)-$(GOOS)-$(GOARCH)
.PHONY += default hash

release_files := \
	$(maincmd_name)-linux-amd64 \
	$(maincmd_name)-linux-arm \
	$(maincmd_name)-windows-amd64.exe \
	$(maincmd_name)-freebsd-amd64 \
	$(maincmd_name)-openbsd-amd64 \
	$(maincmd_name)-netbsd-amd64
# cross compile for each target OS/ARCH
cross:	$(addprefix $(build_dir)/, $(release_files))
.PHONY += cross
$(build_dir)/aquachain.exe:
	GOOS=windows \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-amd64: $(main_deps)
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-linux-arm: $(main_deps)
	GOOS=linux \
	GOARCH=arm \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-windows-amd64.exe: $(main_deps)
	GOOS=windows \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-osx-amd64: $(main_deps)
	GOOS=darwin \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-openbsd-amd64: $(main_deps)
	GOOS=openbsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-netbsd-amd64: $(main_deps)
	GOOS=netbsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)
$(build_dir)/$(maincmd_name)-freebsd-amd64: $(main_deps)
	GOOS=freebsd \
	GOARCH=amd64 \
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) -tags '$(GO_TAGS)' -o $@ $(aquachain_cmd)

.PHONY += install
install:
	install -v $(build_dir)/aquachain $(INSTALL_DIR)/

release: cross hash package
clean:
	rm -rf $(build_dir)
hash: bin/hashes.txt
bin/hashes.txt: $(wildcard $(build_dir)/aqua*)
	$(hashfn) $^ > $@
.PHONY += bin/hashes.txt
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
	env GOBIN= go get golang.org/x/tools/cmd/stringer
	env GOBIN= go get github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get github.com/fjl/gencodec
	env GOBIN= go get github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install gitlab.com/aquachain/x/cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

generate: devtools
	go generate ./...

test: all
	build/env.sh go run build/ci.go test

test-verbose: all
	build/env.sh go run build/ci.go test -v

test-race: all
	build/env.sh go run build/ci.go test -race

test-musl: musl
	build/env.sh go run build/ci.go test -musl

lint:
	build/env.sh go run build/ci.go lint

linter: bin/golangci-lint
	./bin/golangci-lint -v run --color never --deadline 10m ./...

bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s $(golangci_linter_version)

race:
	build/env.sh go run build/ci.go install -- -race ./cmd/aquachain/

all:
	GOBIN=$(build_dir) CGO_ENABLED=$(CGO_ENABLED) go install $(GO_FLAGS) -tags '$(GO_TAGS)' ./cmd/...

