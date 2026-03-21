#!/bin/bash
set -euo pipefail
export PATH="/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin:${PATH:-}"

fixture_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plaincode_bin="${PLAINCODE_BIN:-}"
curl_bin="${CURL_BIN:-}"
port="${PORT:-8080}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not available; skipping Docker smoke"
  exit 0
fi

if [[ -z "$plaincode_bin" ]]; then
  plaincode_bin="$(command -v plaincode 2>/dev/null || true)"
fi
if [[ -z "$plaincode_bin" ]]; then
  echo "plaincode binary not found; set PLAINCODE_BIN" >&2
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

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/codex-health-go-docker.XXXXXX")"
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

image_tag="plaincode-codex-health-go:local"
docker build -t "$image_tag" .

container_id=""
cleanup_container() {
  if [[ -n "${container_id:-}" ]]; then
    docker rm -f "$container_id" >/dev/null 2>&1 || true
  fi
}
trap 'cleanup_container; cleanup' EXIT

container_id="$(docker run -d -p "${port}:8080" -e PORT=8080 "$image_tag")"

for _ in $(seq 1 60); do
  if response="$("$curl_bin" -fsS "http://127.0.0.1:${port}/health" 2>/dev/null)"; then
    normalized="$(printf '%s' "$response" | tr -d '[:space:]')"
    if [[ "$normalized" == '{"status":"good"}' ]]; then
      exit 0
    fi
  fi
  sleep 0.2
done

echo "docker container did not become healthy"
docker logs "$container_id" || true
exit 1
