#!/bin/bash

set -e

VERSION="1.0.3"
PROTECTED_MODE="no"

export GO15VENDOREXPERIMENT=1

cd $(dirname "${BASH_SOURCE[0]}")
OD="$(pwd)"
WD=$OD

package() {
	echo Packaging $1 Binary
	bdir=triangle-${VERSION}-$2-$3
	rm -rf packages/$bdir && mkdir -p packages/$bdir
	GOOS=$2 GOARCH=$3 ./build.sh
	if [ "$2" == "windows" ]; then
		mv triangle packages/$bdir/triangle.exe
	else
		mv triangle packages/$bdir
	fi
	cp README.md packages/$bdir
	cd packages
	if [ "$2" == "linux" ]; then
		tar -zcf $bdir.tar.gz $bdir
	else
		zip -r -q $bdir.zip $bdir
	fi
	rm -rf $bdir
	cd ..
}

if [ "$1" == "package" ]; then
	rm -rf packages/
	package "Windows" "windows" "amd64"
	package "Mac" "darwin" "amd64"
	package "Linux" "linux" "amd64"
	package "FreeBSD" "freebsd" "amd64"
	exit
fi

# temp directory for storing isolated environment.
TMP="$(mktemp -d -t sdb.XXXX)"
rmtemp() {
	rm -rf "$TMP"
}
trap rmtemp EXIT

if [ "$NOCOPY" != "1" ]; then
	# copy all files to an isloated directory.
	WD="$TMP/src/github.com/esimov/triangle"
	export GOPATH="$TMP"
	for file in `find . -type f`; do
		# TODO: use .gitignore to ignore, or possibly just use git to determine the file list.
		if [[ "$file" != "." && "$file" != ./.git* && "$file" != ./triangle ]]; then
			mkdir -p "$WD/$(dirname "${file}")"
			cp -P "$file" "$WD/$(dirname "${file}")"
		fi
	done
	cd $WD
fi

# build and store objects into original directory.
go build -ldflags "-X main.version=$VERSION" -o "$OD/triangle" cmd/triangle/*.go