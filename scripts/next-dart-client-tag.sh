#!/usr/bin/env bash
# Print the next release tag for the beergargoyle-dart-client repository.
#
# Usage: next-dart-client-tag.sh [REPO_DIR] [BUMP] [EXPLICIT_TAG]
#   REPO_DIR      git checkout of the dart client with its tags fetched (default: .)
#   BUMP          patch | minor | major (default: patch)
#   EXPLICIT_TAG  vMAJOR.MINOR.PATCH; overrides BUMP when non-empty
#
# The newest existing vX.Y.Z tag is bumped. With no tags yet the first release
# is v1.0.0. Fails if the resulting tag already exists or is not vX.Y.Z.
set -euo pipefail

repo_dir=${1:-.}
bump=${2:-patch}
explicit=${3:-}

tag_pattern='^v[0-9]+\.[0-9]+\.[0-9]+$'

if [ -n "$explicit" ]; then
  next=$explicit
else
  latest=$(git -C "$repo_dir" tag --list 'v[0-9]*' --sort=-v:refname | grep -E "$tag_pattern" | head -n1 || true)

  if [ -z "$latest" ]; then
    next=v1.0.0
  else
    IFS=. read -r major minor patch <<<"${latest#v}"

    case "$bump" in
      major) major=$((major + 1)); minor=0; patch=0 ;;
      minor) minor=$((minor + 1)); patch=0 ;;
      patch) patch=$((patch + 1)) ;;
      *)
        echo "unknown bump '$bump' (want patch, minor or major)" >&2
        exit 2
        ;;
    esac

    next="v${major}.${minor}.${patch}"
  fi
fi

if ! [[ $next =~ $tag_pattern ]]; then
  echo "tag '$next' is not of the form vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

if git -C "$repo_dir" rev-parse --verify --quiet "refs/tags/$next" >/dev/null; then
  echo "tag $next already exists" >&2
  exit 1
fi

printf '%s\n' "$next"
