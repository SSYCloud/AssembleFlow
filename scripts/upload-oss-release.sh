#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <release-dir> <version>" >&2
  exit 1
fi

RELEASE_DIR="$1"
VERSION="$2"

: "${OSS_ACCESS_KEY_ID:?OSS_ACCESS_KEY_ID is required}"
: "${OSS_ACCESS_KEY_SECRET:?OSS_ACCESS_KEY_SECRET is required}"
: "${OSS_ENDPOINT:?OSS_ENDPOINT is required}"
: "${OSS_BUCKET:?OSS_BUCKET is required}"
: "${OSS_PUBLIC_BASE_URL:?OSS_PUBLIC_BASE_URL is required}"

OSS_PREFIX="${OSS_PREFIX:-loomloom/releases}"
OSSUTIL_BIN="${OSSUTIL_BIN:-ossutil}"
OSS_TARGET="oss://${OSS_BUCKET}/${OSS_PREFIX%/}/${VERSION}"
OSS_LATEST_TARGET="oss://${OSS_BUCKET}/${OSS_PREFIX%/}/latest"
PUBLIC_BASE_URL="${OSS_PUBLIC_BASE_URL%/}/${OSS_PREFIX%/}"

if ! command -v "$OSSUTIL_BIN" >/dev/null 2>&1; then
  echo "missing ossutil command: $OSSUTIL_BIN" >&2
  echo "Install ossutil before running this script, or set OSSUTIL_BIN to its path." >&2
  exit 1
fi

if [[ ! -d "$RELEASE_DIR" ]]; then
  echo "release directory not found: $RELEASE_DIR" >&2
  exit 1
fi

scripts_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
"$scripts_dir/build-oss-release-manifest.sh" "$RELEASE_DIR" "$VERSION" "$PUBLIC_BASE_URL" "$RELEASE_DIR/manifest.oss.json"

ossutil() {
  "$OSSUTIL_BIN" \
    -e "$OSS_ENDPOINT" \
    -i "$OSS_ACCESS_KEY_ID" \
    -k "$OSS_ACCESS_KEY_SECRET" \
    "$@"
}

echo "uploading LoomLoom release assets to $OSS_TARGET"
ossutil cp -r "$RELEASE_DIR/" "$OSS_TARGET/" --update

echo "updating latest OpenClaw installer and manifest at $OSS_LATEST_TARGET"
ossutil cp "$RELEASE_DIR/manifest.oss.json" "$OSS_LATEST_TARGET/manifest.json" --update
ossutil cp "$RELEASE_DIR/install-openclaw.sh" "$OSS_LATEST_TARGET/install-openclaw.sh" --update

echo "OSS mirror publish complete:"
echo "  ${PUBLIC_BASE_URL}/${VERSION}/"
echo "  ${PUBLIC_BASE_URL}/latest/install-openclaw.sh"
