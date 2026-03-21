#!/bin/bash
set -euo pipefail
export PATH="/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin:${PATH:-}"

fixture_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plaincode_bin="${PLAINCODE_BIN:-}"
go_bin="${GO_BIN:-}"
curl_bin="${CURL_BIN:-}"
port="${PORT:-8080}"

if [[ -z "$plaincode_bin" ]]; then
  plaincode_bin="$(command -v plaincode 2>/dev/null || true)"
fi
if [[ -z "$plaincode_bin" ]]; then
  echo "plaincode binary not found; set PLAINCODE_BIN" >&2
  exit 1
fi

if [[ -z "$go_bin" ]]; then
  go_bin="$(command -v go 2>/dev/null || true)"
fi
if [[ -z "$go_bin" && -x /usr/local/go/bin/go ]]; then
  go_bin="/usr/local/go/bin/go"
fi
if [[ -z "$go_bin" ]]; then
  echo "go binary not found; set GO_BIN" >&2
  exit 1
fi

if [[ -z "$curl_bin" ]]; then
  curl_bin="$(command -v curl 2>/dev/null || true)"
fi
if [[ -z "$curl_bin" && -x /usr/bin/curl ]]; then
  curl_bin="/usr/bin/curl"
fi
if [[ -z "$curl_bin" ]]; then
  echo "curl binary not found; set CURL_BIN" >&2
  exit 1
fi

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/codex-health-go.XXXXXX")"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

cp -R "$fixture_dir"/. "$tmpdir"/

cd "$tmpdir"

build_log="$tmpdir/build.log"
"$plaincode_bin" build --spec health/server >"$build_log" 2>&1

for required in go.mod go.sum main.go main_test.go Dockerfile .dockerignore; do
  if [[ ! -f "$required" ]]; then
    echo "expected generated file missing: $required" >&2
    cat "$build_log"
    exit 1
  fi
done

"$go_bin" test ./...

log_file="$tmpdir/server.log"
server_pid=""
cleanup_server() {
  if [[ -n "${server_pid:-}" ]] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
}
trap 'cleanup_server; cleanup' EXIT

PORT="$port" "$go_bin" run . >"$log_file" 2>&1 &
server_pid=$!

for _ in $(seq 1 50); do
  if response="$("$curl_bin" -fsS "http://127.0.0.1:${port}/health" 2>/dev/null)"; then
    normalized="$(printf '%s' "$response" | tr -d '[:space:]')"
    if [[ "$normalized" == '{"status":"good"}' ]]; then
      exit 0
    fi
  fi
  sleep 0.2
done

cleanup_server
echo "server did not become healthy"
cat "$log_file"
exit 1
