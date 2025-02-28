#!/bin/bash
# makedeb.bash - to make a debian package from release binary
set -e

if [ ! -f contrib/makedeb.bash ]; then
    echo 'fatal: run this script from the root of the source tree'
    exit 1
fi

service_file=contrib/aquachain.service
if [ ! -f $service_file ]; then
    echo "fatal: missing $service_file"
    exit 1
fi

k01file=contrib/K01aquachain
if [ ! -f $k01file ]; then
    echo "fatal: missing $k01file"
    exit 1
fi

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
    if [ -z "$bindir" ]; then
        echo "fatal: empty bindir"
        exit 1
    fi
    if [ ! -d $bindir ]; then
        echo "fatal: missing $bindir"
        echo "run 'make cross GOOS=$goos GOARCH=$goarch' first"
        exit 1
    fi
    binfile=$(ls -1 $bindir/aquachain 2>/dev/null | tail -n1)
    if [ -z "$binfile" ]; then
        echo "fatal: missing $bindir/aquachain"
        exit 1
    fi
    if [ ! -f $binfile ]; then
        echo "fatal: missing $binfile"
        exit 1
    fi
    echo found: $binfile

    echo "building debian package for $goos-$goarch"

    version=$(make print-version)

    # create a temporary directory
    tmpdir=$(mktemp -d)
    echo "created: $tmpdir"

    # fix umask
    umask 022

    # files: K01aquachain, aquachain.service

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

    # this helps graceful shutdown when power-button is pressed
    cp $k01file $tmpdir/etc/init.d/K01aquachain
    chmod 755 $tmpdir/etc/init.d/K01aquachain

    # create the control file
    cat >$tmpdir/DEBIAN/control <<EOF
Package: aquachain
Version: ${version#v}
Architecture: $goarch
Maintainer: Aquachain Core Developers <aquachain@riseup.net>
Installed-Size: $(du -s $tmpdir | cut -f1)
Depends: systemd
Section: net
Priority: optional
Homepage: https://aquachain.github.io
Description: Aquachain
 Aquachain is a decentralized cryptocurrency.
EOF

    # create the postinst file
    cat >$tmpdir/DEBIAN/postinst <<EOF
#!/bin/sh
set -e
# add user and group

# enable and start the service
systemctl daemon-reload || true
systemctl enable --now aquachain || true
EOF
    chmod 755 $tmpdir/DEBIAN/postinst

    # create the prerm file
    cat >$tmpdir/DEBIAN/prerm <<EOF
#!/bin/sh
set -e
systemctl disable --now aquachain || true
EOF
    chmod 755 $tmpdir/DEBIAN/prerm

    # create the postrm file
    cat >$tmpdir/DEBIAN/postrm <<EOF
#!/bin/sh
set -e
systemctl daemon-reload || true
EOF
    chmod 755 $tmpdir/DEBIAN/postrm

    echo "building: aquachain-$version-$goos-$goarch.deb"
    # build the debian package
    dpkg-deb --build $tmpdir "aquachain-$version-$goos-$goarch.deb"

    echo "created: aquachain-$version-$goos-$goarch.deb"

    sha256sum "aquachain-$version-$goos-$goarch.deb"

}

for goos_goarch in $@; do
    echo "building debian package for $goos_goarch"
    build_deb $goos_goarch
done