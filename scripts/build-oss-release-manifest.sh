#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <release-dir> <version> <public-base-url> <output-file>" >&2
  exit 1
fi

RELEASE_DIR="$1"
VERSION="$2"
PUBLIC_BASE_URL="${3%/}"
OUTPUT_FILE="$4"
CHECKSUMS_FILE="$RELEASE_DIR/checksums.txt"

if [[ ! -f "$CHECKSUMS_FILE" ]]; then
  echo "missing checksums file: $CHECKSUMS_FILE" >&2
  exit 1
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

{
  printf '{\n'
  printf '  "version": "%s",\n' "$VERSION"
  printf '  "baseUrl": "%s/%s",\n' "$PUBLIC_BASE_URL" "$VERSION"
  printf '  "files": [\n'

  first="true"
  while read -r sha name; do
    [[ -n "$sha" && -n "$name" ]] || continue
    [[ -f "$RELEASE_DIR/$name" ]] || continue
    size="$(wc -c < "$RELEASE_DIR/$name" | tr -d '[:space:]')"
    if [[ "$first" == "true" ]]; then
      first="false"
    else
      printf ',\n'
    fi
    printf '    {"name": "%s", "url": "%s/%s/%s", "githubFallbackUrl": "https://github.com/SSYCloud/loomloom/releases/download/%s/%s", "sha256": "%s", "size": %s}' \
      "$name" "$PUBLIC_BASE_URL" "$VERSION" "$name" "$VERSION" "$name" "$sha" "$size"
  done < "$CHECKSUMS_FILE"

  printf '\n'
  printf '  ]\n'
  printf '}\n'
} > "$tmp_file"

mv "$tmp_file" "$OUTPUT_FILE"
