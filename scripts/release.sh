#!/usr/bin/env bash
set -euo pipefail

version="${VERSION#v}"
mkdir -p dist

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os="${target%/*}"
  arch="${target#*/}"
  name="promptq_${version}_${os}_${arch}"
  binary="promptq"
  [[ "$os" == windows ]] && binary="promptq.exe"

  mkdir -p "dist/$name"
  GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/ast-lw/promptq/internal/promptq.Version=$version" \
    -o "dist/$name/$binary" ./cmd/promptq

  if [[ "$os" == windows ]]; then
    (cd dist && zip -qr "$name.zip" "$name")
  else
    tar -C dist -czf "dist/$name.tar.gz" "$name"
  fi
  rm -rf "dist/$name"
done

(cd dist && shasum -a 256 ./*.tar.gz ./*.zip > checksums.txt)
