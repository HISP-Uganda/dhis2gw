#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
STAGE="$ROOT/linux-build"
VERSION=${DHIS2GW_VERSION:-1.0.0}

goreleaser release --snapshot --rm-dist

rm -rf "$STAGE"
mkdir -p "$STAGE"

for path in cmd db docs internal schemas; do
	cp -a "$ROOT/$path" "$STAGE/"
done

# Do not copy ignored local binaries that may exist inside command directories.
rm -f "$STAGE/cmd/api/api" "$STAGE/cmd/ibp/ibp" \
	"$STAGE/cmd/pbs_sync/pbs_sync" "$STAGE/cmd/worker/worker"

for path in go.mod go.sum README.md; do
	cp -a "$ROOT/$path" "$STAGE/"
done

mkdir -p "$STAGE/packaging"
cp -a "$ROOT/packaging/dhis2gw.yml" "$STAGE/packaging/"
cp -a "$ROOT/packaging/debian" "$STAGE/"
cp -a "$ROOT/packaging/linux-build-README.md" "$STAGE/LINUX-BUILD.md"

sed "s/@VERSION@/$VERSION/g" \
	"$STAGE/debian/changelog.in" > "$STAGE/debian/changelog"
rm -f "$STAGE/debian/changelog.in"
chmod 0755 "$STAGE/debian/rules"

mkdir -p "$STAGE/build"

copy_release_binary() {
	source_name=$1
	destination_name=$2
	source_path=$(find "$ROOT/dist" -type f -path "*_linux_amd64/$source_name" -print -quit)

	if [ -z "$source_path" ]; then
		printf '%s\n' "Missing GoReleaser Linux/amd64 binary: $source_name" >&2
		exit 1
	fi

	cp -a "$source_path" "$STAGE/build/$destination_name"
}

copy_release_binary dhis2gw dhis2gw
copy_release_binary worker dhis2gw-worker
copy_release_binary pbs-sync dhis2gw-pbs-sync
copy_release_binary migrate dhis2gw-migrate

printf '%s\n' "Linux Debian build tree created at $STAGE (version $VERSION)"
