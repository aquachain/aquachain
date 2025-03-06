# $(info loading release.mk ...)

releasetexts := README.md COPYING AUTHORS

defaultwhat: 
	@echo "release mk file :)"

# TODO remove this line after fixing release directory issue
.PRECIOUS: $(build_dir)/% $(build_dir)/%/aquachain $(build_dir)/%/aquachain.exe

## package above binaries (eg release/aquachain-0.0.1-windows-amd64/)
package-one: $(release_dir)/$(maincmd_name)-windows-amd64.zip
package: $(release_dir)/$(maincmd_name)-windows-amd64.zip \
	$(release_dir)/$(maincmd_name)-osx-amd64.zip \
	$(release_dir)/$(maincmd_name)-linux-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-linux-riscv64.tar.gz \
	$(release_dir)/$(maincmd_name)-linux-arm.tar.gz \
	$(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz \
	$(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz \
	debs

debs:
	bash contrib/makedeb.bash linux-amd64 linux-arm linux-riscv64
	mv *.deb $(release_dir)/

# for not cross-compile
aquachain_$(version)_$(GOOS)_$(GOARCH).deb:
	bash contrib/makedeb.bash $(GOOS)-$(GOARCH)

# # create release packages
$(release_dir)/$(maincmd_name)-windows-%.zip: $(build_dir)/windows-%/
	zip -r $@ $^/aquachain* ${releasetexts}
	
$(release_dir)/$(maincmd_name)-osx-%.zip: $(build_dir)/osx-%/
	zip -r $@ $^/aquachain* ${releasetexts}

# create release binaries
# eg: bin/windows-amd64/aquachain.exe
$(build_dir)/%: $(GOFILES)
	$(info starting cross-compile $* -> $@)
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@/*
$(build_dir)/%/aquachain.exe: $(build_dir)/%/aquachain

$(release_dir)/$(maincmd_name)-%.tar.gz: $(build_dir)/%
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$*/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*

# $(build_dir)/windows-%: $(build_dir)/windows-%.exe
$(build_dir)/windows-%/aquachain$(winextension): 
	$(info building windows-$* -> $@)
	env GOOS=windows GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@
$(build_dir)/%/aquachain:
	$(info building $* -> $@)
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@

# cross-%: $(build_dir)/%
# 	$(info cross-compiling $* ... $^)
# 	# rm -rf tmprelease/${maincmd_name}-$*
# 	mkdir -p $(release_dir)
# 	mkdir -p tmprelease/${maincmd_name}-$*
# 	if [ $* = "windows-amd64" ]; then \
# 		cp -nvi $^ tmprelease/${maincmd_name}-$*/aquachain.exe; \
# 	else \
# 		cp -nvi $^ tmprelease/${maincmd_name}-$*/aquachain; \
# 	fi
# 	cp -t tmprelease/${maincmd_name}-$* ${releasetexts}
# 	if [ $* = "windows-amd64" ] || [ $* = "osx-amd64" ]; then \
# 		cd tmprelease && zip -r ../$(release_dir)/${maincmd_name}-$*.zip ${maincmd_name}-$*; \
# 	else \
# 		cd tmprelease && tar czf ../$(release_dir)/${maincmd_name}-$*.tar.gz ${maincmd_name}-$*; \
# 	fi

$(release_dir)/$(maincmd_name)-windows-amd64.zip: $(build_dir)/windows-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-windows-amd64
	mkdir -p tmprelease/${maincmd_name}-windows-amd64
	cp -t tmprelease/${maincmd_name}-windows-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-windows-amd64
$(release_dir)/$(maincmd_name)-osx-amd64.zip: $(build_dir)/darwin-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-osx-amd64
	mkdir -p tmprelease/${maincmd_name}-osx-amd64
	cp -t tmprelease/${maincmd_name}-osx-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-osx-amd64
$(release_dir)/$(maincmd_name)-%.tar.gz: $(build_dir)/%
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$* $^/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*
# $(release_dir)/$(maincmd_name)-linux-amd64.tar.gz: $(build_dir)/linux-amd64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-linux
# 	mkdir -p tmprelease/${maincmd_name}-linux
# 	cp -t tmprelease/${maincmd_name}-linux $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux
# $(release_dir)/$(maincmd_name)-linux-arm.tar.gz: $(build_dir)/linux-arm
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-linux-arm
# 	mkdir -p tmprelease/${maincmd_name}-linux-arm
# 	cp -t tmprelease/${maincmd_name}-linux-arm $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-arm
# $(release_dir)/$(maincmd_name)-linux-arm64.tar.gz: $(build_dir)/linux-arm64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-linux-arm64
# 	mkdir -p tmprelease/${maincmd_name}-linux-arm64
# 	cp -t tmprelease/${maincmd_name}-linux-arm64 $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-arm64
# $(release_dir)/$(maincmd_name)-linux-riscv64.tar.gz: $(build_dir)/linux-riscv64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-linux-riscv64
# 	mkdir -p tmprelease/${maincmd_name}-linux-riscv64
# 	cp -t tmprelease/${maincmd_name}-linux-riscv64 $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-linux-riscv64
# $(release_dir)/$(maincmd_name)-freebsd-amd64.tar.gz: $(build_dir)/freebsd-amd64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-freebsd
# 	mkdir -p tmprelease/${maincmd_name}-freebsd
# 	cp -t tmprelease/${maincmd_name}-freebsd $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-freebsd
# $(release_dir)/$(maincmd_name)-openbsd-amd64.tar.gz: $(build_dir)/openbsd-amd64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-openbsd
# 	mkdir -p tmprelease/${maincmd_name}-openbsd
# 	cp -t tmprelease/${maincmd_name}-openbsd $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-openbsd
# $(release_dir)/$(maincmd_name)-netbsd-amd64.tar.gz: $(build_dir)/netbsd-amd64
# 	mkdir -p $(release_dir)
# 	rm -rf tmprelease/${maincmd_name}-netbsd
# 	mkdir -p tmprelease/${maincmd_name}-netbsd
# 	cp -t tmprelease/${maincmd_name}-netbsd $^ ${releasetexts}
# 	cd tmprelease && tar czf ../$@ ${maincmd_name}-netbsd
