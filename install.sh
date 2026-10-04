#!/bin/sh
set -eu

releases=https://github.com/bidirekt/cli/releases
usage="usage: install.sh [-b <dir>] [vX.Y.Z]"

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

bin_dir=$HOME/.local/bin
while getopts b: flag; do
  case $flag in
    b) bin_dir=$OPTARG ;;
    *) fail "$usage" ;;
  esac
done
shift $((OPTIND - 1))
[ $# -le 1 ] || fail "$usage"
version=${1:-}

os=$(uname -s)
case $os in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "no build for OS $os: install.sh supports Linux and macOS, the other builds are at $releases" ;;
esac
arch=$(uname -m)
case $arch in
  x86_64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) fail "no build for architecture $arch: bidirekt is built for x86_64 and arm64" ;;
esac

if [ -z "$version" ]; then
  latest=$(curl -sSfL -o /dev/null -w '%{url_effective}' "$releases/latest") ||
    fail "could not resolve the latest release from $releases/latest"
  case $latest in
    "$releases"/tag/*) version=${latest#"$releases"/tag/} ;;
    *) fail "no release published at $releases" ;;
  esac
fi
case $version in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) fail "invalid version \"$version\": expected vX.Y.Z" ;;
esac

archive=bidirekt_${version#v}_${os}_${arch}.tar.gz
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
# dash and busybox ash skip the EXIT trap when a signal kills the shell.
trap 'exit 1' HUP INT TERM

for file in "$archive" checksums.txt; do
  curl -sSfL -o "$tmp/$file" "$releases/download/$version/$file" ||
    fail "could not download $releases/download/$version/$file"
done

expected=$(awk -v name="$archive" '$2 == name { print $1 }' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$archive")
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp/$archive")
else
  fail "sha256sum or shasum is required to verify the download"
fi
[ "${actual%% *}" = "$expected" ] ||
  fail "checksum of $archive does not match checksums.txt, nothing was installed"

tar -xzf "$tmp/$archive" -C "$tmp" bidirekt
mkdir -p "$bin_dir"
mv "$tmp/bidirekt" "$bin_dir/bidirekt"
printf 'bidirekt %s installed to %s\n' "${version#v}" "$bin_dir/bidirekt"

case :$PATH: in
  *:"$bin_dir":*) ;;
  *) printf '%s\n' "$bin_dir is not on your PATH. Add it in your shell profile:" "  export PATH=\"$bin_dir:\$PATH\"" ;;
esac
