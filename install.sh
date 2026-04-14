#!/usr/bin/env bash
set -euo pipefail

REPO="SSYCloud/batchjob-agent-kit"
VERSION="${VERSION:-latest}"
AGENT="codex"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SKILL_DIR="${SKILL_DIR:-}"
USE_HOMEBREW="auto"

usage() {
  cat <<'EOF'
Usage: install.sh [options]

Options:
  --agent <codex|claude>   Install the matching skill pack (default: codex)
  --install-dir <path>     Directory for batchjob-cli (default: ~/.local/bin)
  --skill-dir <path>       Override the destination directory for SKILL.md
  --version <tag|latest>   GitHub release tag to install (default: latest)
  --no-brew                Force GitHub Release install even if Homebrew is available
  --help                   Show this help text
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent)
      AGENT="${2:-codex}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-$HOME/.local/bin}"
      shift 2
      ;;
    --skill-dir)
      SKILL_DIR="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-latest}"
      shift 2
      ;;
    --no-brew)
      USE_HOMEBREW="never"
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

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64|amd64) ARCH="amd64" ;;
  *)
    echo "unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

mkdir -p "$INSTALL_DIR"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

resolve_skill_dir() {
  if [[ -n "$SKILL_DIR" ]]; then
    printf '%s\n' "$SKILL_DIR"
    return
  fi
  case "$AGENT" in
    codex)
      printf '%s\n' "$HOME/.codex/skills/batchjob"
      ;;
    claude)
      printf '%s\n' "$HOME/.claude/skills/batchjob"
      ;;
    *)
      echo "unsupported agent for automatic skill install: $AGENT" >&2
      exit 1
      ;;
  esac
}

resolve_tag() {
  if [[ "$VERSION" != "latest" ]]; then
    printf '%s\n' "$VERSION"
    return
  fi
  local api_url="https://api.github.com/repos/${REPO}/releases/latest"
  local tag
  tag="$(
    curl -fsSL "$api_url" \
      | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n1
  )"
  if [[ -z "$tag" ]]; then
    echo "failed to resolve latest release tag from $api_url" >&2
    exit 1
  fi
  printf '%s\n' "$tag"
}

can_use_homebrew() {
  [[ "$USE_HOMEBREW" != "never" ]] || return 1
  [[ "$VERSION" == "latest" ]] || return 1
  case "$OS" in
    darwin|linux) ;;
    *) return 1 ;;
  esac
  command -v brew >/dev/null 2>&1 || return 1
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

require_cmd curl
SKILL_ASSET="batchjob-skills.tar.gz"
CHECKSUM_ASSET="checksums.txt"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if can_use_homebrew; then
  TAG="homebrew-latest"
else
  require_cmd tar
  TAG="$(resolve_tag)"
  CLI_ASSET="batchjob-cli-${OS}-${ARCH}.tar.gz"
  BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
fi

echo "batchjob-agent-kit installer"
echo "repo: $REPO"
echo "version: $TAG"
echo "agent: $AGENT"
if can_use_homebrew; then
  echo "cli install: homebrew"
else
  echo "install dir: $INSTALL_DIR"
fi
echo "skill dir: $(resolve_skill_dir)"
echo

if can_use_homebrew; then
  if brew list --versions batchjob-cli >/dev/null 2>&1; then
    brew upgrade batchjob-cli || true
  else
    brew install ssycloud/tap/batchjob-cli
  fi
  local_cli_path="$INSTALL_DIR/batchjob-cli"
  if [[ -f "$local_cli_path" ]]; then
    rm -f "$local_cli_path"
    echo "removed shadowing local CLI: $local_cli_path"
  fi
  CLI_PATH="$(command -v batchjob-cli || true)"
  if [[ -z "$CLI_PATH" ]]; then
    echo "failed to resolve batchjob-cli after Homebrew install" >&2
    exit 1
  fi
else
  curl -fsSL -o "$TMP_DIR/$CLI_ASSET" "$BASE_URL/$CLI_ASSET"
  curl -fsSL -o "$TMP_DIR/$CHECKSUM_ASSET" "$BASE_URL/$CHECKSUM_ASSET"
  VERIFY_TOOL="$(checksum_tool)"
  verify_checksum "$VERIFY_TOOL" "$TMP_DIR/$CHECKSUM_ASSET" "$CLI_ASSET" "$TMP_DIR/$CLI_ASSET"

  mkdir -p "$TMP_DIR/cli"
  tar -xzf "$TMP_DIR/$CLI_ASSET" -C "$TMP_DIR/cli"
  install -m 0755 "$TMP_DIR/cli/batchjob-cli" "$INSTALL_DIR/batchjob-cli"
  CLI_PATH="$INSTALL_DIR/batchjob-cli"
fi

[[ -n "${BASE_URL:-}" ]] || BASE_URL="https://github.com/${REPO}/releases/download/$(resolve_tag)"
curl -fsSL -o "$TMP_DIR/$SKILL_ASSET" "$BASE_URL/$SKILL_ASSET"
if [[ ! -f "$TMP_DIR/$CHECKSUM_ASSET" ]]; then
  curl -fsSL -o "$TMP_DIR/$CHECKSUM_ASSET" "$BASE_URL/$CHECKSUM_ASSET"
fi
VERIFY_TOOL="${VERIFY_TOOL:-$(checksum_tool)}"
verify_checksum "$VERIFY_TOOL" "$TMP_DIR/$CHECKSUM_ASSET" "$SKILL_ASSET" "$TMP_DIR/$SKILL_ASSET"

mkdir -p "$TMP_DIR/skills"
tar -xzf "$TMP_DIR/$SKILL_ASSET" -C "$TMP_DIR/skills"
FINAL_SKILL_DIR="$(resolve_skill_dir)"
mkdir -p "$FINAL_SKILL_DIR"
install -m 0644 "$TMP_DIR/skills/skills/$AGENT/batchjob/SKILL.md" "$FINAL_SKILL_DIR/SKILL.md"

echo "installed:"
echo "  $CLI_PATH"
echo "  $(resolve_skill_dir)/SKILL.md"
echo
echo "next:"
echo "  export BATCHJOB_SERVER=https://batchjob-test.shengsuanyun.com/batch"
echo "  export BATCHJOB_TOKEN=your-token"
echo "  batchjob-cli doctor"
