#!/usr/bin/env bash
set -euo pipefail

image="ghcr.io/${GITHUB_REPOSITORY,,}"

workdir="$(mktemp -d)"
git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git" "$workdir"
cd "$workdir"

echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_ACTOR" --password-stdin

docker build -f Containerfile -t "$image:${GITHUB_SHA::7}" -t "$image:latest" .
docker push "$image" --all-tags
