#!/bin/bash
# makedeb.bash - to make a debian package from release binary
set -e

if [ ! -f contrib/makedeb.bash ]; then
    echo 'fatal: run this script from the root of the source tree'
    exit 1
fi

service_file=contrib/aquachain.service
k01file=contrib/K01aquachain

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
Depends: adduser, systemd
Section: net
Priority: optional
Keywords: aquachain, blockchain, coin
Homepage: https://aquachain.github.io
Description: Aquachain
 Aquachain RPC server
EOF

    # create the postinst file
    cat >$tmpdir/DEBIAN/postinst <<EOF
#!/bin/sh
set -e
if ! which systemctl >/dev/null; then
    echo "warn: systemd not found, skipping aquachain.service installation"
    exit 0
fi
# add user and group
if ! getent group aqua >/dev/null; then
    addgroup --system aqua
fi
if ! getent passwd aqua >/dev/null; then
    adduser --system --no-create-home --ingroup aqua --home /var/lib/aquachain --shell /usr/sbin/nologin aqua
fi
# enable and start the service
systemctl daemon-reload
systemctl enable --now aquachain
EOF
    chmod 755 $tmpdir/DEBIAN/postinst

    # create the prerm file
    cat >$tmpdir/DEBIAN/prerm <<EOF
#!/bin/sh
set -e
if ! which systemctl >/dev/null; then
    echo "warn: systemd not found, skipping aquachain.service installation"
    exit 0
fi
systemctl disable --now aquachain
EOF
    chmod 755 $tmpdir/DEBIAN/prerm

    # create the postrm file
    cat >$tmpdir/DEBIAN/postrm <<EOF
#!/bin/sh
set -e
if ! which systemctl >/dev/null; then
    echo "warn: systemd not found, skipping aquachain.service installation"
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
}

# called as.. eg bash contrib/makedeb.bash linux-amd64 linux-arm linux-riscv64
for goos_goarch in $@; do
    echo "building debian package for $goos_goarch"
    build_deb $goos_goarch
done
