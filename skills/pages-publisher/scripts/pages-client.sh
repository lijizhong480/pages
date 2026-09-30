#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  pages-client.sh health
  pages-client.sh workspaces
  pages-client.sh projects <workspace>
  pages-client.sh pages <workspace> <project>
  pages-client.sh versions <workspace> <project> <slug>
  pages-client.sh upload <workspace> <project> <slug> <title> <html-file>
  pages-client.sh publish <workspace> <project> <slug> <version>

Environment:
  PAGES_BASE_URL  Pages origin (default: http://localhost:8080)
  PAGES_TOKEN     Workspace or platform bearer token
EOF
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 2
}

require_args() {
  local expected="$1"
  shift
  [[ "$#" -eq "$expected" ]] || { usage >&2; exit 2; }
}

base_url="${PAGES_BASE_URL:-http://localhost:8080}"
base_url="${base_url%/}"
token="${PAGES_TOKEN:-}"
command_name="${1:-}"
[[ -n "$command_name" ]] || { usage; exit 2; }
shift

request() {
  [[ -n "$token" ]] || die "PAGES_TOKEN is required for authenticated API calls"
  curl --fail-with-body --silent --show-error \
    -H "Authorization: Bearer ${token}" "$@"
}

case "$command_name" in
  health)
    require_args 0 "$@"
    curl --fail-with-body --silent --show-error "${base_url}/healthz"
    ;;
  workspaces)
    require_args 0 "$@"
    request "${base_url}/api/workspaces"
    ;;
  projects)
    require_args 1 "$@"
    request "${base_url}/api/workspaces/$1/projects"
    ;;
  pages)
    require_args 2 "$@"
    request "${base_url}/api/workspaces/$1/projects/$2/pages"
    ;;
  versions)
    require_args 3 "$@"
    request "${base_url}/api/workspaces/$1/projects/$2/pages/$3/versions"
    ;;
  upload)
    require_args 5 "$@"
    [[ -f "$5" ]] || die "HTML file not found: $5"
    request -X POST \
      -F "slug=$3" \
      -F "title=$4" \
      -F "file=@$5;type=text/html" \
      "${base_url}/api/workspaces/$1/projects/$2/pages"
    ;;
  publish)
    require_args 4 "$@"
    request -X POST \
      -H "Content-Type: application/json" \
      --data "{\"version\":\"$4\"}" \
      "${base_url}/api/workspaces/$1/projects/$2/pages/$3/publish"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    die "unknown command: $command_name"
    ;;
esac

printf '\n'
