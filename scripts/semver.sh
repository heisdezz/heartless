#!/usr/bin/env bash
# Compute the next semantic version from conventional commits.
#
# The baseline is the latest vX.Y.Z git tag, or the version in mygo.json when
# no tag exists. Commits since the baseline are scanned for conventional
# commit types; the highest bump wins:
#
#   BREAKING CHANGE / type!  -> major
#   feat:                    -> minor
#   fix:, perf:, revert:     -> patch
#   anything else            -> no release
#
# Writes version, tag, should_release and bump to GITHUB_OUTPUT when running
# in Actions, and prints them for local use otherwise.

set -euo pipefail

# --- baseline ---------------------------------------------------------------
file_version="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' mygo.json | head -n1)"
file_version="${file_version#v}"
if [ -z "$file_version" ]; then
  echo "could not read version from mygo.json" >&2
  exit 1
fi

# Only stable vX.Y.Z tags are release baselines; prerelease/build tags are ignored.
latest_tag="$(git tag --list 'v[0-9]*' --sort=-v:refname \
  | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1 || true)"
if [ -n "$latest_tag" ]; then
  current="${latest_tag#v}"
  range="${latest_tag}..HEAD"
  have_tag=1
else
  current="$file_version"
  range="HEAD"
  have_tag=0
fi

# --- scan commits -----------------------------------------------------------
commits="$(git log --format='%s%n%b' "$range" 2>/dev/null || true)"

major=0
minor=0
patch=0
while IFS= read -r line; do
  [ -z "$line" ] && continue
  if printf '%s\n' "$line" | grep -q '^BREAKING CHANGE:' ||
     printf '%s\n' "$line" | grep -Eq '^[a-zA-Z]+(\([^)]*\))?!:'; then
    major=1
  elif printf '%s\n' "$line" | grep -Eq '^feat(\([^)]*\))?:'; then
    minor=1
  elif printf '%s\n' "$line" | grep -Eq '^(fix|perf|revert)(\([^)]*\))?:'; then
    patch=1
  fi
done <<< "$commits"

# --- next version -----------------------------------------------------------
IFS='.' read -r M m p <<< "$current"
M="${M:-0}"
m="${m:-0}"
p="${p:-0}"
p="${p%%-*}"

if [ "$have_tag" -eq 0 ]; then
  # No tag yet: publish the version declared in mygo.json as the first release.
  next="$file_version"
  bump="initial"
  should_release="true"
elif [ "$major" -eq 1 ]; then
  M=$((M + 1)); m=0; p=0
  next="$M.$m.$p"; bump="major"; should_release="true"
elif [ "$minor" -eq 1 ]; then
  m=$((m + 1)); p=0
  next="$M.$m.$p"; bump="minor"; should_release="true"
elif [ "$patch" -eq 1 ]; then
  p=$((p + 1))
  next="$M.$m.$p"; bump="patch"; should_release="true"
else
  next="$current"
  bump="none"
  should_release="false"
fi

tag="v$next"

echo "current=$current"
echo "version=$next"
echo "tag=$tag"
echo "bump=$bump"
echo "should_release=$should_release"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "version=$next"
    echo "tag=$tag"
    echo "bump=$bump"
    echo "should_release=$should_release"
  } >> "$GITHUB_OUTPUT"
fi
