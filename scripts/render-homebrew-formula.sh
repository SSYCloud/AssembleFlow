#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: render-homebrew-formula.sh --tag <vX.Y.Z> --checksums <path>
EOF
}

tag=""
checksums=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      tag="${2:-}"
      shift 2
      ;;
    --checksums)
      checksums="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$tag" || -z "$checksums" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -r "$checksums" ]]; then
  echo "checksums file is not readable: $checksums" >&2
  exit 1
fi

version="${tag#v}"

checksum_for() {
  local name="$1"
  local value
  value="$(awk -v target="$name" '$2 == target { print $1 }' "$checksums")"
  if [[ -z "$value" ]]; then
    echo "missing checksum for $name in $checksums" >&2
    exit 1
  fi
  printf '%s\n' "$value"
}

darwin_arm64_sha="$(checksum_for "batchjob-cli-darwin-arm64.tar.gz")"
darwin_amd64_sha="$(checksum_for "batchjob-cli-darwin-amd64.tar.gz")"
linux_arm64_sha="$(checksum_for "batchjob-cli-linux-arm64.tar.gz")"
linux_amd64_sha="$(checksum_for "batchjob-cli-linux-amd64.tar.gz")"

cat <<EOF
class BatchjobCli < Formula
  desc "Developer CLI for hosted BatchJob skills"
  homepage "https://github.com/SSYCloud/batchjob-agent-kit"
  version "$version"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/SSYCloud/batchjob-agent-kit/releases/download/$tag/batchjob-cli-darwin-arm64.tar.gz"
      sha256 "$darwin_arm64_sha"
    else
      url "https://github.com/SSYCloud/batchjob-agent-kit/releases/download/$tag/batchjob-cli-darwin-amd64.tar.gz"
      sha256 "$darwin_amd64_sha"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/SSYCloud/batchjob-agent-kit/releases/download/$tag/batchjob-cli-linux-arm64.tar.gz"
      sha256 "$linux_arm64_sha"
    else
      url "https://github.com/SSYCloud/batchjob-agent-kit/releases/download/$tag/batchjob-cli-linux-amd64.tar.gz"
      sha256 "$linux_amd64_sha"
    end
  end

  def install
    bin.install "batchjob-cli"
  end

  test do
    assert_match "Developer CLI for hosted BatchJob skills", shell_output("#{bin}/batchjob-cli --help")
  end
end
EOF
