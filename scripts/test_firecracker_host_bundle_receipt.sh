#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOT
readonly BUILDER="$ROOT/scripts/build_firecracker_host_bundle.sh"
readonly VERIFIER="$ROOT/scripts/verify_firecracker_host_bundle_receipt.py"
TEST_ROOT="$(mktemp -d /tmp/orquesta-firecracker-host-receipt-test.XXXXXX)"
readonly TEST_ROOT
trap 'rm -rf -- "$TEST_ROOT"' EXIT

bash -n "$BUILDER"
bash -n "$0"
python3 -m py_compile "$VERIFIER"

printf 'launcher\n' >"$TEST_ROOT/launcher"
printf 'supervisor\n' >"$TEST_ROOT/supervisor"
printf 'builder\n' >"$TEST_ROOT/builder"
chmod 0755 "$TEST_ROOT/launcher" "$TEST_ROOT/supervisor"
chmod 0555 "$TEST_ROOT/builder"

readonly COMMIT="1111111111111111111111111111111111111111"
readonly TREE="2222222222222222222222222222222222222222"
readonly ARCHIVE_SHA="3333333333333333333333333333333333333333333333333333333333333333"
readonly COMMIT_OBJECT_SHA="4444444444444444444444444444444444444444444444444444444444444444"
readonly TOOLCHAIN_TREE_SHA="5555555555555555555555555555555555555555555555555555555555555555"
readonly TOOLCHAIN_GO_SHA="6666666666666666666666666666666666666666666666666666666666666666"
BUILDER_SHA="$(sha256sum "$TEST_ROOT/builder" | cut -d' ' -f1)"
readonly BUILDER_SHA
LAUNCHER_SHA="$(sha256sum "$TEST_ROOT/launcher" | cut -d' ' -f1)"
readonly LAUNCHER_SHA
SUPERVISOR_SHA="$(sha256sum "$TEST_ROOT/supervisor" | cut -d' ' -f1)"
readonly SUPERVISOR_SHA
LAUNCHER_SIZE="$(stat -c '%s' "$TEST_ROOT/launcher")"
readonly LAUNCHER_SIZE
SUPERVISOR_SIZE="$(stat -c '%s' "$TEST_ROOT/supervisor")"
readonly SUPERVISOR_SIZE

write_receipt() {
  local output="$1"
  python3 - "$output" <<PY
import json
import pathlib

document = {
    "artifacts": {
        "launcher": {
            "file": "orquesta-firecracker-launcher",
            "mode": "0755",
            "package": "./cmd/orquesta-firecracker-launcher",
            "sha256": "sha256:$LAUNCHER_SHA",
            "size_bytes": $LAUNCHER_SIZE,
        },
        "supervisor": {
            "file": "orquesta-firecracker-attestor-e2e",
            "mode": "0755",
            "package": "./cmd/orquesta-firecracker-attestor-e2e",
            "sha256": "sha256:$SUPERVISOR_SHA",
            "size_bytes": $SUPERVISOR_SIZE,
        },
    },
    "build": {
        "buildvcs": False,
        "cgo_enabled": False,
        "double_build": True,
        "goarch": "amd64",
        "goos": "linux",
        "isolated_caches": True,
        "mod": "vendor",
        "recipe": "go build -mod=vendor -trimpath -buildvcs=false",
        "source_date_epoch": 0,
        "trimpath": True,
    },
    "builder": {
        "file": "build_firecracker_host_bundle.sh",
        "sha256": "sha256:$BUILDER_SHA",
    },
    "environment": {
        "CGO_ENABLED": "0",
        "GOCACHE": "isolated_private",
        "GOENV": "off",
        "GOMODCACHE": "isolated_private",
        "GOOS": "linux",
        "GOARCH": "amd64",
        "GOPATH": "isolated_private",
        "GOPROXY": "off",
        "GOSUMDB": "off",
        "GOTOOLCHAIN": "local",
        "LC_ALL": "C",
        "SOURCE_DATE_EPOCH": "0",
        "TMPDIR": "isolated_private",
        "TZ": "UTC",
    },
    "schema_version": "orquesta_firecracker_host_bundle_build.v1",
    "source": {
        "archive_sha256": "sha256:$ARCHIVE_SHA",
        "commit": "$COMMIT",
        "commit_object_sha256": "sha256:$COMMIT_OBJECT_SHA",
        "export": "git_archive_exact_commit",
        "gitlinks": False,
        "tree_oid": "$TREE",
    },
    "toolchain": {
        "go_binary_sha256": "sha256:$TOOLCHAIN_GO_SHA",
        "platform": "linux/amd64",
        "tree_sha256": "sha256:$TOOLCHAIN_TREE_SHA",
        "version": "go1.25.11",
    },
}
pathlib.Path("$output").write_text(
    json.dumps(document, separators=(",", ":"), sort_keys=True) + "\n",
    encoding="utf-8",
)
PY
  chmod 0444 "$output"
}

verify() {
  local receipt="$1"
  local receipt_sha
  receipt_sha="$(sha256sum "$receipt" | cut -d' ' -f1)"
  "$VERIFIER" \
    --receipt "$receipt" \
    --expected-receipt-sha256 "$receipt_sha" \
    --expected-source-commit "$COMMIT" \
    --expected-source-tree "$TREE" \
    --expected-source-archive-sha256 "$ARCHIVE_SHA" \
    --expected-source-commit-object-sha256 "$COMMIT_OBJECT_SHA" \
    --expected-toolchain-version go1.25.11 \
    --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA" \
    --expected-toolchain-go-sha256 "$TOOLCHAIN_GO_SHA" \
    --expected-builder-sha256 "$BUILDER_SHA" \
    --builder "$TEST_ROOT/builder" \
    --launcher "$TEST_ROOT/launcher" \
    --expected-launcher-sha256 "$LAUNCHER_SHA" \
    --supervisor "$TEST_ROOT/supervisor" \
    --expected-supervisor-sha256 "$SUPERVISOR_SHA"
}

expect_failure() {
  local expected="$1"
  shift
  local output status=0
  output="$("$@" 2>&1)" || status=$?
  [[ "$status" != 0 && "$output" == *"code=$expected"* ]] || {
    printf 'firecracker_host_receipt_test_failed expected=%s status=%s output=%s\n' \
      "$expected" "$status" "$output" >&2
    exit 1
  }
}

write_receipt "$TEST_ROOT/receipt.json"
verify "$TEST_ROOT/receipt.json" >/dev/null

cp "$TEST_ROOT/receipt.json" "$TEST_ROOT/receipt-substituted.json"
chmod 0644 "$TEST_ROOT/receipt-substituted.json"
python3 - "$TEST_ROOT/receipt-substituted.json" <<'PY'
import pathlib
path = pathlib.Path(__import__("sys").argv[1])
path.write_text(path.read_text().replace("go1.25.11", "go1.25.12"), encoding="utf-8")
PY
expect_failure receipt_hash "$VERIFIER" \
  --receipt "$TEST_ROOT/receipt-substituted.json" \
  --expected-receipt-sha256 "$(sha256sum "$TEST_ROOT/receipt.json" | cut -d' ' -f1)" \
  --expected-source-commit "$COMMIT" --expected-source-tree "$TREE" \
  --expected-source-archive-sha256 "$ARCHIVE_SHA" \
  --expected-source-commit-object-sha256 "$COMMIT_OBJECT_SHA" \
  --expected-toolchain-version go1.25.11 \
  --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA" \
  --expected-toolchain-go-sha256 "$TOOLCHAIN_GO_SHA" \
  --expected-builder-sha256 "$BUILDER_SHA" \
  --builder "$TEST_ROOT/builder" \
  --launcher "$TEST_ROOT/launcher" --expected-launcher-sha256 "$LAUNCHER_SHA" \
  --supervisor "$TEST_ROOT/supervisor" --expected-supervisor-sha256 "$SUPERVISOR_SHA"

cp "$TEST_ROOT/launcher" "$TEST_ROOT/launcher-tampered"
printf 'tamper\n' >>"$TEST_ROOT/launcher-tampered"
chmod 0755 "$TEST_ROOT/launcher-tampered"
expect_failure launcher_identity "$VERIFIER" \
  --receipt "$TEST_ROOT/receipt.json" \
  --expected-receipt-sha256 "$(sha256sum "$TEST_ROOT/receipt.json" | cut -d' ' -f1)" \
  --expected-source-commit "$COMMIT" --expected-source-tree "$TREE" \
  --expected-source-archive-sha256 "$ARCHIVE_SHA" \
  --expected-source-commit-object-sha256 "$COMMIT_OBJECT_SHA" \
  --expected-toolchain-version go1.25.11 \
  --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA" \
  --expected-toolchain-go-sha256 "$TOOLCHAIN_GO_SHA" \
  --expected-builder-sha256 "$BUILDER_SHA" \
  --builder "$TEST_ROOT/builder" \
  --launcher "$TEST_ROOT/launcher-tampered" --expected-launcher-sha256 "$LAUNCHER_SHA" \
  --supervisor "$TEST_ROOT/supervisor" --expected-supervisor-sha256 "$SUPERVISOR_SHA"

cp "$TEST_ROOT/builder" "$TEST_ROOT/builder-tampered"
chmod 0755 "$TEST_ROOT/builder-tampered"
printf 'tamper\n' >>"$TEST_ROOT/builder-tampered"
chmod 0555 "$TEST_ROOT/builder-tampered"
expect_failure builder_identity "$VERIFIER" \
  --receipt "$TEST_ROOT/receipt.json" \
  --expected-receipt-sha256 "$(sha256sum "$TEST_ROOT/receipt.json" | cut -d' ' -f1)" \
  --expected-source-commit "$COMMIT" --expected-source-tree "$TREE" \
  --expected-source-archive-sha256 "$ARCHIVE_SHA" \
  --expected-source-commit-object-sha256 "$COMMIT_OBJECT_SHA" \
  --expected-toolchain-version go1.25.11 \
  --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA" \
  --expected-toolchain-go-sha256 "$TOOLCHAIN_GO_SHA" \
  --expected-builder-sha256 "$BUILDER_SHA" \
  --builder "$TEST_ROOT/builder-tampered" \
  --launcher "$TEST_ROOT/launcher" --expected-launcher-sha256 "$LAUNCHER_SHA" \
  --supervisor "$TEST_ROOT/supervisor" --expected-supervisor-sha256 "$SUPERVISOR_SHA"

python3 - "$TEST_ROOT/receipt-duplicate.json" <<'PY'
import pathlib
import sys
source = pathlib.Path(sys.argv[1])
source.write_text(
    '{"schema_version":"orquesta_firecracker_host_bundle_build.v1",'
    '"schema_version":"orquesta_firecracker_host_bundle_build.v1"}\n',
    encoding="utf-8",
)
PY
chmod 0444 "$TEST_ROOT/receipt-duplicate.json"
expect_failure receipt_duplicate_key "$VERIFIER" \
  --receipt "$TEST_ROOT/receipt-duplicate.json" \
  --expected-receipt-sha256 "$(sha256sum "$TEST_ROOT/receipt-duplicate.json" | cut -d' ' -f1)" \
  --expected-source-commit "$COMMIT" --expected-source-tree "$TREE" \
  --expected-source-archive-sha256 "$ARCHIVE_SHA" \
  --expected-source-commit-object-sha256 "$COMMIT_OBJECT_SHA" \
  --expected-toolchain-version go1.25.11 \
  --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA" \
  --expected-toolchain-go-sha256 "$TOOLCHAIN_GO_SHA" \
  --expected-builder-sha256 "$BUILDER_SHA" \
  --builder "$TEST_ROOT/builder" \
  --launcher "$TEST_ROOT/launcher" --expected-launcher-sha256 "$LAUNCHER_SHA" \
  --supervisor "$TEST_ROOT/supervisor" --expected-supervisor-sha256 "$SUPERVISOR_SHA"

printf 'firecracker_host_bundle_receipt_test=ok\n'
