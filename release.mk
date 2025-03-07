# $(info loading release.mk ...)

releasetexts := README.md COPYING AUTHORS

defaultwhat: 
	@echo "release mk file :)"

# TODO remove this line after fixing release directory issue
.PRECIOUS: bin/% tmprelease/bin/%/aquachain tmprelease/bin/%/aquachain.exe

## package above binaries (eg release/aquachain-0.0.1-windows-amd64/)
.PHONY: debs package package-win deb
package-win: $(release_dir)/$(maincmd_name)-windows-amd64.zip
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
$(release_dir)/$(maincmd_name)-windows-%.zip: tmprelease/bin/windows-%/
	zip -vr $@ $^/aquachain* ${releasetexts}
	
$(release_dir)/$(maincmd_name)-osx-%.zip: tmprelease/bin/osx-%/
	zip -vr $@ $^/aquachain* ${releasetexts}

# create release binaries
# eg: windows-amd64/aquachain.exe
tmprelease/bin/%: $(GOFILES)
	$(info starting cross-compile $* -> $@)
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@/*
tmprelease/bin/%/aquachain.exe: tmprelease/bin/%/aquachain

$(release_dir)/$(maincmd_name)-%.tar.gz: tmprelease/bin/release/%
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$*/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*

# bin/windows-%: bin/windows-%.exe
tmprelease/bin/windows-%/aquachain$(winextension): 
	$(info building windows-$* -> $@)
	env GOOS=windows GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@
tmprelease/bin/%/aquachain:
	$(info building $* -> $@)
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@

# cross-%: bin/%
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

$(release_dir)/$(maincmd_name)-windows-amd64.zip: tmprelease/bin/windows-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-windows-amd64
	mkdir -p tmprelease/${maincmd_name}-windows-amd64
	cp -t tmprelease/${maincmd_name}-windows-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-windows-amd64
$(release_dir)/$(maincmd_name)-osx-amd64.zip: tmprelease/bin/darwin-amd64
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-osx-amd64
	mkdir -p tmprelease/${maincmd_name}-osx-amd64
	cp -t tmprelease/${maincmd_name}-osx-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-osx-amd64
$(release_dir)/$(maincmd_name)-%.tar.gz: tmprelease/bin/%
	mkdir -p $(release_dir)
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$* $^/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*
