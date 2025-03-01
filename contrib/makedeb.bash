#!/bin/bash
# makedeb.bash - to make a debian package from release binary
# called as.. eg bash contrib/makedeb.bash linux-amd64 linux-arm linux-riscv64
# requires: make, dpkg-deb, gzip, du, mktemp, cut, getent, adduser, addgroup, systemctl
set -e

if [ ! -f contrib/makedeb.bash ]; then
    echo 'fatal: run this script from the root of the source tree'
    exit 1
fi

service_file=contrib/aquachain.service
k01file=contrib/K01aquachain
default_aqua_homedir=/var/lib/aquachain

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

    echo found: $binfile
    echo "building debian package for $goos-$goarch"

    # use -s to avoid 'make' output
    version=$(make -s print-version)

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
    chmod -R 755 $tmpdir

    # copy the binary to the package directory
    cp $binfile $tmpdir/usr/local/bin/aquachain
    chmod 755 $tmpdir/usr/local/bin/aquachain

    # copy the service file to the package directory
    cp $service_file $tmpdir/etc/systemd/system/aquachain.service
    chmod 644 $tmpdir/etc/systemd/system/aquachain.service

    # add man page if exists in contrib/ dir when we make one
    manfile=contrib/aquachain.1
    if [ -f $manfile ]; then
        mkdir -p $tmpdir/usr/share/man/man1
        cp $manfile $tmpdir/usr/share/man/man1
        gzip -9 $tmpdir/usr/share/man/man1/aquachain.1
    else
        echo "warn: missing $manfile"
    fi

    # this helps graceful shutdown when power-button is pressed
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
Depends: adduser
Optional: systemd
Section: net
Priority: optional
Keywords: aquachain, blockchain, coin, evm, smart contracts
Homepage: https://aquachain.github.io
Description: daemon and client for the aquachain peer-to-peer network
 Aquachain is a blockchain platform with a native coin, AQUA.
 It is a fork of Ethereum with a few changes.
 .
 This package contains the aquachain daemon and client.
EOF

    # create the preinst file
    cat >$tmpdir/DEBIAN/preinst <<EOF
#!/bin/sh
set -e
if [ "\$1" = "install" ]; then
    if ! getent group aquachain >/dev/null; then
        addgroup --system aqua
    fi
    if ! getent passwd aquachain >/dev/null; then
        adduser --system --no-create-home --ingroup aqua --home $default_aqua_homedir aqua
    fi
    mkdir -p $default_aqua_homedir
    chown -R aquachain:aquachain $default_aqua_homedir

fi

if [ "\$1" = "remove" ]; then
    if getent passwd aquachain >/dev/null; then
        userdel aquachain
    fi
    if getent group aquachain >/dev/null; then
        groupdel aquachain
    fi
fi
EOF
    chmod 755 $tmpdir/DEBIAN/preinst

    # copy the postinst and prerm files
    cp -v contrib/debpkg/postinst $tmpdir/DEBIAN/postinst
    cp -v contrib/debpkg/prerm $tmpdir/DEBIAN/prerm
    cp -v contrib/debpkg/templates $tmpdir/DEBIAN/templates
    cp -v contrib/debpkg/config $tmpdir/DEBIAN/config
    test ! -f contrib/debpkg/conffiles || cp -v contrib/debpkg/conffiles $tmpdir/DEBIAN/conffiles
    chmod 755 $tmpdir/DEBIAN/postinst
    chmod 755 $tmpdir/DEBIAN/prerm
    chmod 644 $tmpdir/DEBIAN/templates
    chmod 644 $tmpdir/DEBIAN/config
    test ! -f contrib/debpkg/conffiles || chmod 644 $tmpdir/DEBIAN/conffiles
    


    # create the postrm file
    cat >$tmpdir/DEBIAN/postrm <<EOF
#!/bin/sh
set -e
if ! which systemctl >/dev/null; then
    echo "warn: systemctl not found, skipping aquachain.service installation"
    exit 0
fi
systemctl daemon-reload
EOF
    chmod 755 $tmpdir/DEBIAN/postrm

    # build the debian package
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
