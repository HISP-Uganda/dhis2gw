#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
STAGE="$ROOT/linux-build"
VERSION=${DHIS2GW_VERSION:-1.0.0}

goreleaser release --snapshot --clean

rm -rf "$STAGE"
mkdir -p "$STAGE"

copy_release_binary() {
	arch=$1
	source_name=$2
	destination_name=$3
	tree=$4
	source_path=$(find "$ROOT/dist" -type f -path "*_linux_${arch}*/$source_name" -print -quit)

	if [ -z "$source_path" ]; then
		printf '%s\n' "Missing GoReleaser Linux/$arch binary: $source_name" >&2
		exit 1
	fi

	cp -a "$source_path" "$tree/build/$destination_name"
}

stage_architecture() {
	arch=$1
	tree="$STAGE/$arch"

	mkdir -p "$tree"

	for path in cmd db docs internal schemas; do
		cp -a "$ROOT/$path" "$tree/"
	done

	# Do not copy ignored local binaries that may exist inside command directories.
	rm -f "$tree/cmd/api/api" "$tree/cmd/ibp/ibp" \
		"$tree/cmd/pbs_sync/pbs_sync" "$tree/cmd/worker/worker"

	for path in go.mod go.sum README.md; do
		cp -a "$ROOT/$path" "$tree/"
	done

	mkdir -p "$tree/packaging"
	cp -a "$ROOT/packaging/dhis2gw.yml" "$tree/packaging/"
	cp -a "$ROOT/packaging/debian" "$tree/"
	cp -a "$ROOT/packaging/linux-build-README.md" "$tree/LINUX-BUILD.md"

	sed "s/@VERSION@/$VERSION/g" \
		"$tree/debian/changelog.in" > "$tree/debian/changelog"
	rm -f "$tree/debian/changelog.in"
	sed "s/^Architecture: .*/Architecture: $arch/" \
		"$tree/debian/control" > "$tree/debian/control.tmp"
	mv "$tree/debian/control.tmp" "$tree/debian/control"
	chmod 0755 "$tree/debian/rules"

	mkdir -p "$tree/build"
	copy_release_binary "$arch" dhis2gw dhis2gw "$tree"
	copy_release_binary "$arch" worker dhis2gw-worker "$tree"
	copy_release_binary "$arch" pbs-sync dhis2gw-pbs-sync "$tree"
	copy_release_binary "$arch" migrate dhis2gw-migrate "$tree"
}

stage_architecture amd64
stage_architecture arm64

printf '%s\n' "Linux Debian build trees created at $STAGE (version $VERSION)"
