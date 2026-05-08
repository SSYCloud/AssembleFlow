#!/usr/bin/env bash
set -euo pipefail

REPO="SSYCloud/loomloom"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SKILL_DIR="${SKILL_DIR:-$HOME/.openclaw/workspace/skills/loomloom}"
MIRROR_BASE_URL="${LOOMLOOM_MIRROR_BASE_URL:-https://install.shengsuanyun.com/loomloom/releases}"
USE_GITHUB_FALLBACK="${USE_GITHUB_FALLBACK:-true}"
FORCE_DOWNLOAD="false"

usage() {
  cat <<'EOF'
Usage: install-openclaw.sh [options]

Options:
  --install-dir <path>       Directory for loomloom (default: ~/.local/bin)
  --skill-dir <path>         Destination directory for OpenClaw SKILL.md
  --version <tag|latest>     Release tag to install (default: latest)
  --mirror-base-url <url>    Mirror release base URL
  --no-github-fallback       Do not fallback to GitHub Release downloads
  --force-download           Ignore files bundled next to this script
  --help                     Show this help text

Environment:
  LOOMLOOM_MIRROR_BASE_URL   Mirror release base URL
  LOOMLOOM_SERVER            Server URL used by loomloom doctor
  LOOMLOOM_TOKEN             API token used by loomloom doctor
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-dir)
      INSTALL_DIR="${2:-$HOME/.local/bin}"
      shift 2
      ;;
    --skill-dir)
      SKILL_DIR="${2:-$HOME/.openclaw/workspace/skills/loomloom}"
      shift 2
      ;;
    --version)
      VERSION="${2:-latest}"
      shift 2
      ;;
    --mirror-base-url)
      MIRROR_BASE_URL="${2:-$MIRROR_BASE_URL}"
      shift 2
      ;;
    --no-github-fallback)
      USE_GITHUB_FALLBACK="false"
      shift
      ;;
    --force-download)
      FORCE_DOWNLOAD="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

script_dir() {
  local source="${BASH_SOURCE[0]}"
  while [[ -L "$source" ]]; do
    local dir
    dir="$(cd -P "$(dirname "$source")" >/dev/null 2>&1 && pwd)"
    source="$(readlink "$source")"
    [[ "$source" != /* ]] && source="$dir/$source"
  done
  cd -P "$(dirname "$source")" >/dev/null 2>&1 && pwd
}

checksum_tool() {
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' "sha256sum"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    printf '%s\n' "shasum -a 256"
    return
  fi
  printf '%s\n' ""
}

verify_checksum() {
  local tool="$1"
  local checksums_file="$2"
  local asset_name="$3"
  local asset_path="$4"
  local expected
  expected="$(awk -v name="$asset_name" '$2 == name { print $1 }' "$checksums_file")"
  if [[ -z "$expected" || -z "$tool" ]]; then
    return
  fi
  local actual
  actual="$($tool "$asset_path" | awk '{print $1}')"
  if [[ "$expected" != "$actual" ]]; then
    echo "checksum mismatch for $asset_name" >&2
    exit 1
  fi
}

download_with_retry() {
  local url="$1"
  local output="$2"
  local label="$3"
  local attempt
  for attempt in 1 2 3; do
    echo "downloading $label (attempt $attempt): $url"
    if curl -fL --connect-timeout 10 --max-time 300 --retry 2 --retry-delay 2 -C - -o "$output" "$url"; then
      return 0
    fi
    rm -f "$output"
    sleep "$((attempt * 2))"
  done
  return 1
}

json_value() {
  local key="$1"
  local file="$2"
  sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n1
}

install_local_bundle() {
  local bundle_dir="$1"
  if [[ ! -x "$bundle_dir/bin/loomloom" || ! -f "$bundle_dir/skills/loomloom/SKILL.md" ]]; then
    return 1
  fi

  mkdir -p "$INSTALL_DIR" "$SKILL_DIR"
  install -m 0755 "$bundle_dir/bin/loomloom" "$INSTALL_DIR/loomloom"
  install -m 0644 "$bundle_dir/skills/loomloom/SKILL.md" "$SKILL_DIR/SKILL.md"
  return 0
}

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux) ;;
  *)
    echo "OpenClaw installer currently supports Linux only; detected: $OS" >&2
    exit 1
    ;;
esac
case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64|amd64) ARCH="amd64" ;;
  *)
    echo "unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

require_cmd curl
require_cmd tar

SCRIPT_DIR="$(script_dir)"
if [[ "$FORCE_DOWNLOAD" != "true" ]] && install_local_bundle "$SCRIPT_DIR"; then
  CLI_PATH="$INSTALL_DIR/loomloom"
  echo "installed LoomLoom from local OpenClaw bundle"
else
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "$TMP_DIR"' EXIT

  if [[ "$VERSION" == "latest" ]]; then
    if download_with_retry "${MIRROR_BASE_URL%/}/latest/manifest.json" "$TMP_DIR/manifest.json" "latest manifest"; then
      VERSION="$(json_value version "$TMP_DIR/manifest.json")"
    elif [[ "$USE_GITHUB_FALLBACK" == "true" ]]; then
      VERSION="$(
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
          | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
          | head -n1
      )"
    fi
    if [[ -z "$VERSION" || "$VERSION" == "latest" ]]; then
      echo "failed to resolve latest LoomLoom release. Please retry later or use an offline OpenClaw bundle." >&2
      exit 1
    fi
  fi

  CLI_ASSET="loomloom-linux-${ARCH}.tar.gz"
  SKILL_ASSET="loomloom-skills.tar.gz"
  CHECKSUM_ASSET="checksums.txt"
  BASE_URL="${MIRROR_BASE_URL%/}/${VERSION}"
  GITHUB_BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

  download_asset() {
    local asset="$1"
    local output="$2"
    if download_with_retry "$BASE_URL/$asset" "$output" "$asset"; then
      return
    fi
    if [[ "$USE_GITHUB_FALLBACK" == "true" ]]; then
      echo "mirror download failed; falling back to GitHub for $asset" >&2
      download_with_retry "$GITHUB_BASE_URL/$asset" "$output" "$asset"
      return
    fi
    echo "failed to download $asset from mirror. Please retry later or use an offline OpenClaw bundle." >&2
    exit 1
  }

  download_asset "$CLI_ASSET" "$TMP_DIR/$CLI_ASSET"
  download_asset "$SKILL_ASSET" "$TMP_DIR/$SKILL_ASSET"
  download_asset "$CHECKSUM_ASSET" "$TMP_DIR/$CHECKSUM_ASSET"

  VERIFY_TOOL="$(checksum_tool)"
  verify_checksum "$VERIFY_TOOL" "$TMP_DIR/$CHECKSUM_ASSET" "$CLI_ASSET" "$TMP_DIR/$CLI_ASSET"
  verify_checksum "$VERIFY_TOOL" "$TMP_DIR/$CHECKSUM_ASSET" "$SKILL_ASSET" "$TMP_DIR/$SKILL_ASSET"

  mkdir -p "$TMP_DIR/cli" "$TMP_DIR/skills" "$INSTALL_DIR" "$SKILL_DIR"
  tar -xzf "$TMP_DIR/$CLI_ASSET" -C "$TMP_DIR/cli"
  tar -xzf "$TMP_DIR/$SKILL_ASSET" -C "$TMP_DIR/skills"
  install -m 0755 "$TMP_DIR/cli/loomloom" "$INSTALL_DIR/loomloom"
  install -m 0644 "$TMP_DIR/skills/skills/openclaw/loomloom/SKILL.md" "$SKILL_DIR/SKILL.md"
  CLI_PATH="$INSTALL_DIR/loomloom"
fi

echo
echo "installed:"
echo "  $CLI_PATH"
echo "  $SKILL_DIR/SKILL.md"
echo
echo "next:"
echo "  export LOOMLOOM_SERVER=https://batchjob-test.shengsuanyun.com/batch"
echo "  export LOOMLOOM_TOKEN=your-token"
echo "  loomloom doctor"
echo
if [[ -n "${LOOMLOOM_SERVER:-}" && -n "${LOOMLOOM_TOKEN:-}" ]]; then
  echo "running loomloom doctor..."
  "$CLI_PATH" doctor || {
    echo "loomloom doctor failed. Please check LOOMLOOM_SERVER and LOOMLOOM_TOKEN." >&2
    exit 1
  }
fi
