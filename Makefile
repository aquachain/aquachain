# edit mkconfig.mk if necessary
include mkconfig.mk
gobindatacmd ?= $(shell which go-bindata)
# for install target
build_dir ?= bin
PREFIX ?= /usr/local
INSTALL_DIR ?= $(PREFIX)/bin

maybeext := 
ifeq (windows,$(GOOS))
maybeext := .exe
endif

# the main target is bin/aquachain or bin/aquachain.exe
shorttarget := $(build_dir)/aquachain$(maybeext)
# $(info shorttarget = $(shorttarget))

define LOGO
                              _           _
  __ _  __ _ _   _  __ _  ___| |__   __ _(_)_ __
 / _ '|/ _' | | | |/ _' |/ __| '_ \ / _' | | '_ \ 
| (_| | (_| | |_| | (_| | (__| | | | (_| | | | | |
 \__,_|\__, |\__,_|\__,_|\___|_| |_|\__,_|_|_| |_|
          |_|
	Latest Source: https://gitlab.com/aquachain/aquachain
	Website: https://aquachain.github.io

Target architecture: $(GOOS)/$(GOARCH)
Version: $(version) (commit=$(COMMITHASH)) $(codename)
endef

# apt install file 

# targets
$(shorttarget): $(GOFILES)
	$(info $(LOGO))
	$(info Building... $@)
	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)
	@echo Compiled: $(shorttarget)
	@sha256sum $(shorttarget) 2>/dev/null || echo "warn: 'sha256sum' command not found"
	@file $(shorttarget) 2>/dev/null || echo "warn: 'file' command not found"
# if on windows, this would become .exe.exe but whatever
$(shorttarget).exe: $(GOFILES)
	$(info Building... $@)
	GOOS=windows CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)
	@echo compiled: $(shorttarget)
	@sha256sum $(shorttarget) 2>/dev/null || true
	@file $(shorttarget) 2>/dev/null || true
.PHONY += install
install:
	install -v $(build_dir)/aquachain $(INSTALL_DIR)/
default: $(shorttarget)
print-version:
	@echo $(version)
echoflags:
	@echo "CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build $(GO_FLAGS) -o $@ $(aquachain_cmd)"
echo:
	$(info  )
	$(info Variables:)
	$(info  )
	@$(foreach V,$(.VARIABLES), \
		$(if $(filter-out environment% default automatic, $(origin $V)), \
			$(if $(filter-out LOGO GOFILES,$V), \
				$(info $V=$($V)) )))
	$(info  )
clean:
	rm -rf bin release docs tmprelease
# echo: # useful lol
# 	@echo "GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)"
# 	@echo GOCMD=$(GOCMD)
# 	@echo shorttarget=$(shorttarget)
# 	@echo GO_FLAGS=$(GO_FLAGS)
# 	@echo aquachain_cmd=$(aquachain_cmd)
# 	@echo tags=$(tags)
# 	@echo GOTAGS=$(GOTAGS)
# 	@echo GOOS=$(GOOS)
# 	@echo GOARCH=$(GOARCH)
# 	@echo COMMITHASH=$(COMMITHASH)
# 	@echo version=$(version)
# 	@echo codename=$(codename)
# 	@echo LINKER_FLAGS=$(LINKER_FLAGS)
# 	@echo TAGS64=$(TAGS64)
# 	@echo cgo=$(cgo)
# 	@echo build_dir=$(build_dir)
# 	@echo INSTALL_DIR=$(INSTALL_DIR)
# 	@echo release_dir=$(release_dir)
# 	@echo hashfn=$(hashfn)
# 	@echo golangci_linter_version=$(golangci_linter_version)
# 	@echo PWD=$(PWD)
bootnode: bin/aquabootnode
# bin/%: $(GOFILES)
# 	$(info Building... ./cmd/$*)
# 	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o bin/$* ./cmd/$*

.PHONY += default bootnode hash

internal/jsre/deps/bindata.go: internal/jsre/deps/web3.js  internal/jsre/deps/bignumber.js
	@test -x "$(gobindatacmd)" || echo 'warn: go-bindata not found in PATH. run make devtools to install required development dependencies'
	@test -x "$(gobindatacmd)" || exit 0
	@echo "regenerating embedded javascript assets"
	@test ! -x "$(gobindatacmd)" || go generate -v ./$(shell dirname $@)/...
all:
	mkdir -p $(build_dir)
	cd $(build_dir) && \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../cmd/...

main_command_dir := ${aquachain_cmd}

# cross compilation wizard target for testing different architectures
cross:
	test -n "$(GOOS)"
	test -n "$(GOARCH)" 
	@echo cross-compiling for $(GOOS)/$(GOARCH)
	@echo warn: to build a real release, use "make clean release release=1"
	@mkdir -p $(build_dir)/${GOOS}-${GOARCH}
	$(info Building to directory: $(build_dir)/${GOOS}-${GOARCH})
	cd $(build_dir)/${GOOS}-${GOARCH} && GOOS=${GOOS} GOARCH=${GOARCH} \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . $(GO_FLAGS) ../.${main_command_dir}

help:
	@echo
	@echo without args, target is: "$(shorttarget)"
	@echo 'make install' target is: "$(INSTALL_DIR)/"
	@echo using go flags: "$(GO_FLAGS)"
	@echo
	@echo to compile quickly, run \'make\' and then \'$(shorttarget) version\'
	@echo then, to install system-wide, run something like \'sudo make install\'
	@echo
	@echo adding 'release=1' to any target will rebuild the dependency libraries.
	@echo
	@echo -n "Building a new release:\n\tmake clean release release=1"
	@echo
	@echo
	@echo "to cross-compile a single OS, try 'make cross' or 'make GOOS=windows'"
	@echo "to add things left out by default, use tags: 'make tags=metrics'"
	@echo
	@echo
	@#echo "cross-compile release: 'make clean cross release=1'"
	@#echo "cross-compile all tools: 'make clean cross release=1 cmds=all'"
	@#echo "compile with CGO tracer support: make cgo=1'"
	@echo required shell commands: git, go, sha256sum, file, date, base64, tr, printf, which, pwd, echo, find
test:
	CGO_ENABLED=0 bash testing/test-short-only.bash $(args)
race:
	CGO_ENABLED=1 bash testing/test-short-only.bash -race

ifeq (1,$(release))
include release.mk
endif

.PHONY += release
checkrelease:
ifneq (1,$(release))
	echo "use make release release=1"
	exit 1
endif
release: checkrelease package hash
release/SHA384.txt:
	$(hashfn) release/*.tar.gz release/*.zip | tee $@
hash: release/SHA384.txt
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
