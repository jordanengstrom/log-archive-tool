#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Usage: ./log-archive.sh [-v] [-dest /path/to/dest] <log-directory>
verbose=0
dest_override=""
src=""

usage() {
  echo "Usage: $0 [-v] [-dest <dir>] <log-directory>"
  exit 2
}

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    -v) verbose=1; shift ;;
    -dest) shift; dest_override="${1-}"; shift ;;
    -h|--help) usage ;;
    --) shift; break ;;
    -*) echo "Unknown option: $1" >&2; usage ;;
    *)
      if [[ -z "$src" ]]; then src="$1"; shift
      else echo "Multiple input directories provided" >&2; usage
      fi
      ;;
  esac
done

if [[ -z "$src" ]]; then usage; fi
if [[ ! -d "$src" ]]; then echo "Input is not a directory: $src" >&2; exit 1; fi

# Normalize source to absolute path
src="$(cd "$src" && pwd -P)"
base="$(basename "$src")"
parent="$(dirname "$src")"

if [[ -n "$dest_override" ]]; then
  dest="$dest_override"
else
  dest="${parent}/${base}_archive"
fi

mkdir -p "$dest"

timestamp="$(date +%Y%m%d_%H%M%S)"
archive_name="${base}_archive_${timestamp}.tar.gz"
tmp_path="${dest}/${archive_name}.tmp"
final_path="${dest}/${archive_name}"

if [[ $verbose -eq 1 ]]; then
  echo "Source: $src"
  echo "Destination: $dest"
  echo "Archive (temp): $tmp_path"
fi

# Collect eligible files (non-recursive, regular files, not symlinks).
files=()
skipped=()
for f in "$src"/*; do
  [[ -e "$f" ]] || continue
  name="$(basename "$f")"
  if [[ -L "$f" ]]; then
    (( verbose )) && skipped+=("$name (symlink)")
    continue
  fi
  if [[ ! -f "$f" ]]; then
    (( verbose )) && skipped+=("$name (not regular file)")
    continue
  fi
  case "$name" in
    *.tar.gz|*.tgz|*.gz|archive_history.log)
      (( verbose )) && skipped+=("$name (skipped extension or history file)")
      continue
      ;;
  esac
  files+=("$name")
done

if [[ ${#files[@]} -eq 0 ]]; then
  echo "No files to archive" >&2
  exit 0
fi

# Ensure tmp_path is removed on failure; clear trap after successful mv
trap 'rm -f -- "$tmp_path" 2>/dev/null || true' EXIT

# Create archive with filenames only by changing to source dir via -C
if [[ $verbose -eq 1 ]]; then
  echo "Files to archive: ${#files[@]}"
  for ff in "${files[@]}"; do echo "  - $ff"; done
fi

tar -C "$src" -czf "$tmp_path" -- "${files[@]}"

# Move into final name atomically
mv "$tmp_path" "$final_path"
# disable cleanup trap now that file has been moved
trap - EXIT

# Compute totals
total_bytes=0
for ff in "${files[@]}"; do
  # Use portable wc -c to count bytes (works on macOS)
  sz=$(wc -c < "$src/$ff" | tr -d '[:space:]')
  total_bytes=$((total_bytes + sz))
done

timestamp_human="$(date '+%Y-%m-%d %H:%M:%S %Z')"
echo "${timestamp_human} archive=${archive_name} files=${#files[@]} total_bytes=${total_bytes}" >> "${dest}/archive_history.log"

echo "Archive complete: ${final_path}"
exit 0
