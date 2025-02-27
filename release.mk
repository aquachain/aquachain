$(info loading release.mk ...)

# release files (old, TODO remove)
release_files := \
	$(maincmd_name)-linux-amd64 \
	$(maincmd_name)-linux-arm \
	$(maincmd_name)-linux-riscv64 \
	$(maincmd_name)-windows-amd64.exe \
	$(maincmd_name)-freebsd-amd64 \
	$(maincmd_name)-openbsd-amd64 \
	$(maincmd_name)-netbsd-amd64 \
	$(maincmd_name)-osx-amd64
releasetexts := README.md COPYING AUTHORS

defaultwhat: 
	@echo "release mk file :)"

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


# create release packages

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
