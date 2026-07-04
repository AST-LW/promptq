#!/usr/bin/env bash
set -euo pipefail

: "${HOMEBREW_TAP_TOKEN:?HOMEBREW_TAP_TOKEN is required}"
: "${VERSION:?VERSION is required}"

version="${VERSION#v}"
tag="v${version}"

tap_owner="ast-lw"
tap_repo_name="homebrew-tap"
tap_name="${tap_owner}/tap"
formula_name="promptq"

source_url="https://github.com/ast-lw/promptq/archive/refs/tags/${tag}.tar.gz"
source_archive="$(mktemp)"

homebrew_repo="$(brew --repository)"
tap_parent="${homebrew_repo}/Library/Taps/${tap_owner}"
tap_dir="${tap_parent}/${tap_repo_name}"

cleanup() {
  rm -f "$source_archive"
}
trap cleanup EXIT

echo "Preparing Homebrew formula for ${formula_name} ${tag}"

curl -fsSL "$source_url" -o "$source_archive"
sha256="$(shasum -a 256 "$source_archive" | awk '{print $1}')"

rm -rf "$tap_dir"
mkdir -p "$tap_parent"

git clone "https://x-access-token:${HOMEBREW_TAP_TOKEN}@github.com/${tap_owner}/${tap_repo_name}.git" "$tap_dir"

mkdir -p "$tap_dir/Formula"

awk -v url="$source_url" -v sha="$sha256" '
  /url "/ { next }
  /sha256 "/ { next }

  /license "MIT"/ {
    print "  url \"" url "\""
    print "  sha256 \"" sha "\""
    print ""
    print
    next
  }

  { print }
' packaging/homebrew/promptq.rb > "$tap_dir/Formula/${formula_name}.rb"

echo "Generated formula:"
cat "$tap_dir/Formula/${formula_name}.rb"

brew update --quiet

HOMEBREW_NO_AUTO_UPDATE=1 brew audit --strict --new --formula "${tap_name}/${formula_name}"

HOMEBREW_NO_AUTO_UPDATE=1 brew install --build-from-source "${tap_name}/${formula_name}"
HOMEBREW_NO_AUTO_UPDATE=1 brew test "${tap_name}/${formula_name}"

git -C "$tap_dir" config user.name "github-actions[bot]"
git -C "$tap_dir" config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git -C "$tap_dir" add "Formula/${formula_name}.rb"

if git -C "$tap_dir" diff --cached --quiet; then
  echo "Formula already up to date for ${tag}"
  exit 0
fi

git -C "$tap_dir" commit -m "${formula_name} ${version}"
git -C "$tap_dir" push