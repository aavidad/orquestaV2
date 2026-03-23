#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DSN="${ORQUESTA_TEST_POSTGRES_DSN:-}"
GOCACHE_DIR="${GOCACHE:-/tmp/go-build-orquesta}"

if [[ -z "${DSN}" ]]; then
  echo "ORQUESTA_TEST_POSTGRES_DSN no definido; se omite TestOpenPostgresBootstrapLimpio"
  exit 0
fi

cd "${ROOT_DIR}"

exec env GOCACHE="${GOCACHE_DIR}" go test -run '^TestOpenPostgresBootstrapLimpio$' ./db
