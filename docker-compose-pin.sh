#!/usr/bin/env bash
# Pins all images in a docker-compose.yml or Dockerfile to their local SHA256
# digest, editing the file in-place.
#
# Usage: ./docker-compose-pin.sh <docker-compose.yml|Dockerfile>

set -euo pipefail

die() { echo "$*" >&2; exit 1; }

[[ -n "${1:-}" ]] || die "Usage: $0 <docker-compose.yml|Dockerfile>"

FILE="$1"
OUTPUT=$(cat "$FILE")

is_dockerfile() {
  local basename
  basename=$(basename "$FILE")
  [[ "$basename" == Dockerfile* || "$basename" == *.dockerfile ]]
}

dockerfile_images() {
  grep -iE '^FROM ' "$FILE" \
    | awk '{print $2}' \
    | grep -v scratch
}

compose_images() {
  docker compose -f "$FILE" config \
    | awk '/^\s*image:/ {print $2}'
}

resolve_digest() {
  local image="$1"
  local digest

  # Prefer local — no network required
  digest=$(docker image inspect "$image" --format '{{index .RepoDigests 0}}' 2>/dev/null || true)
  [[ -n "$digest" ]] && { echo "$digest"; return; }

  # Fall back to remote manifest
  echo "No local digest for '$image', trying remote..." >&2
  digest=$(docker manifest inspect "$image" 2>/dev/null \
    | awk -F'"' '/"digest"/ { print $4; exit }' || true)
  [[ -z "$digest" ]] && { echo ""; return; }

  # manifest inspect returns a bare sha256:..., build the full repo@digest form
  echo "${image%%:*}@${digest}"
}

pin_image() {
  local image="$1"
  local digest

  digest=$(resolve_digest "$image")
  if [[ -z "$digest" ]]; then
    echo "Warning: no digest found for '$image', skipping." >&2
    return
  fi

  # Anchor to trailing end-of-line to avoid matching inside already-pinned digests.
  OUTPUT=$(echo "$OUTPUT" | sed "s| ${image}$| ${digest}|g")
}

if is_dockerfile; then
  while IFS= read -r image; do pin_image "$image"; done < <(dockerfile_images)
else
  while IFS= read -r image; do pin_image "$image"; done < <(compose_images)
fi

echo "$OUTPUT" > "$FILE"