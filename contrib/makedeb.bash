#!/bin/bash
# makedeb.bash - to make a debian package from release binary
# called as.. eg bash contrib/makedeb.bash linux-amd64 linux-arm linux-riscv64
# requires: make, dpkg-deb, gzip, du, mktemp, cut, getent, adduser, addgroup, systemctl
set -e
echo "makedeb.bash packaging args=$@"

if [ ! -f contrib/makedeb.bash ]; then
    echo 'fatal: run this script from the root of the source tree'
    exit 1
fi

service_file=contrib/aquachain.service
k01file=contrib/K01aquachain
default_aqua_homedir=/var/lib/aquachain
manfile=contrib/aquachain.1

# use -s to avoid 'make' output
version=$(make -s print-version)
echo $version
if [ -z "$version" ]; then
    echo "fatal: missing version"
    exit 1
fi
# no spaces
if [[ $version == *" "* ]]; then
    echo "fatal: version has spaces"
    exit 1
fi

for file in $service_file $k01file; do
    if [ ! -f $file ]; then
        echo "fatal: missing $file"
        exit 1
    fi
done

if [ -z "$1" ]; then
    echo "usage: $0 goos-goarch"
    echo "example: $0 linux-amd64"
    exit 1
fi

build_deb() {
    goos=$(echo $1 | cut -d- -f1)
    goarch=$(echo $1 | cut -d- -f2)

    if [ -z "$goos" ] || [ -z "$goarch" ]; then
        echo "fatal: invalid goos-goarch: $1"
        exit 1
    fi

    if [ "$goos" = "windows" ]; then
        echo "fatal: windows not supported"
        exit 1
    fi

    bindir=bin/$goos-$goarch
    if [ ! -d $bindir ]; then
        echo "fatal: missing $bindir"
        echo "run 'make cross GOOS=$goos GOARCH=$goarch' first"
        exit 1
    fi

    binfile=$(ls -1 $bindir/aquachain 2>/dev/null | tail -n1)
    if [ -z "$binfile" ] || [ ! -f $binfile ]; then
        echo "fatal: missing $bindir/aquachain"
        exit 1
    fi

    echo found binary: $binfile
    ls -ln $binfile
    sha256sum $binfile
    echo "makedeb.bash packaging $goos-$goarch"

    # create a temporary directory
    tmpdir=$(mktemp -d)
    echo "created: $tmpdir"

    # fix umask
    umask 022

    # create the debian package directory structure
    mkdir -p $tmpdir/DEBIAN
    mkdir -p $tmpdir/usr/local/bin
    mkdir -p $tmpdir/etc/systemd/system
    mkdir -p $tmpdir/etc/init.d
    mkdir -p $tmpdir/etc/aquachain
    chmod -R 755 $tmpdir
    chmod 750 $tmpdir/etc/aquachain

    # copy the binary to the package directory
    cp $binfile $tmpdir/usr/local/bin/aquachain
    chmod 755 $tmpdir/usr/local/bin/aquachain

    # copy the service file to the package directory
    cp $service_file $tmpdir/etc/systemd/system/aquachain.service
    chmod 644 $tmpdir/etc/systemd/system/aquachain.service

    # add man page
    if [ -f $manfile ]; then
        mkdir -p $tmpdir/usr/share/man/man1
        cp $manfile $tmpdir/usr/share/man/man1
        gzip -9 $tmpdir/usr/share/man/man1/aquachain.1
    else
        echo "warn: missing $manfile"
    fi

    if [ -f contrib/debpkg/aquachain.conf ]; then
        cp contrib/debpkg/aquachain.conf $tmpdir/etc/aquachain/aquachain.conf
        chmod 600 $tmpdir/etc/aquachain/aquachain.conf
    else
        echo "warn: missing contrib/aquachain.conf"
    fi

    if [ -f contrib/start-aquachain.sh ]; then
        cp contrib/start-aquachain.sh $tmpdir/usr/local/bin/start-aquachain.sh
        chmod 755 $tmpdir/usr/local/bin/start-aquachain.sh
    else
        echo "warn: missing contrib/start-aquachain.sh"
    fi


    # this creates warnings, but helps graceful shutdown when power-button is pressed
    cp $k01file $tmpdir/etc/init.d/K01aquachain
    chmod 755 $tmpdir/etc/init.d/K01aquachain

    debianarch=$goarch
    echo "version: $version"
    # create the control file
    cat >$tmpdir/DEBIAN/control <<EOF
Package: aquachain
Version: ${version#v}
Architecture: $goarch
Maintainer: Aquachain Core Developers <aquachain@riseup.net>
Installed-Size: $(du -s $tmpdir | cut -f1)
Section: net
Priority: optional
Keywords: aquachain, blockchain, coin, EVM, smart contracts
Homepage: https://aquachain.github.io
Description: daemon and client for the aquachain peer-to-peer network
EOF
    # copy the postinst and prerm files
    cp -v contrib/debpkg/aquachain.preinst $tmpdir/DEBIAN/preinst
    cp -v contrib/debpkg/aquachain.postinst $tmpdir/DEBIAN/postinst
    cp -v contrib/debpkg/aquachain.prerm $tmpdir/DEBIAN/prerm
    cp -v contrib/debpkg/aquachain.templates $tmpdir/DEBIAN/templates
    cp -v contrib/debpkg/aquachain.config $tmpdir/DEBIAN/config
    cp -v contrib/debpkg/aquachain.postrm $tmpdir/DEBIAN/postrm
    chmod 755 $tmpdir/DEBIAN/preinst
    chmod 755 $tmpdir/DEBIAN/postinst
    chmod 755 $tmpdir/DEBIAN/prerm
    chmod 644 $tmpdir/DEBIAN/templates
    chmod 755 $tmpdir/DEBIAN/config
    chmod 755 $tmpdir/DEBIAN/postrm
    test ! -f contrib/debpkg/conffiles || cp -v contrib/debpkg/conffiles $tmpdir/DEBIAN/conffiles
    test ! -f contrib/debpkg/conffiles || chmod 644 $tmpdir/DEBIAN/conffiles
    chown -R root:root $tmpdir
    # build the debian package
    if [ -f "aquachain-$version-$goos-$goarch.deb" ]; then
        echo "warn: removing existing aquachain-$version-$goos-$goarch.deb"
        rm -f "aquachain-$version-$goos-$goarch.deb"
    fi
    echo "building: aquachain-$version-$goos-$goarch.deb"
    dpkg-deb --build $tmpdir "aquachain-$version-$goos-$goarch.deb"
    echo "created: aquachain-$version-$goos-$goarch.deb"
    sha256sum "aquachain-$version-$goos-$goarch.deb"

    rm -rf $tmpdir
    echo "removed: $tmpdir"
}
# end build_deb

for goos_goarch in $@; do
    if [[ $goos_goarch != linux-* ]]; then
        echo "fatal: only linux duh"
        exit 1
    fi
    build_deb $goos_goarch
done

if [ -z "$1" ]; then
    echo "usage: $0 goos-goarch"
    echo "example: $0 linux-amd64"
    exit 1
fi
