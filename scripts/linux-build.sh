#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
STAGE="$ROOT/linux-build"
VERSION=${DHIS2GW_VERSION:-1.0.0}

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

printf '%s\n' "Linux Debian build tree created at $STAGE (version $VERSION)"
