#!/usr/bin/env bash
set -Eeuo pipefail

export LC_ALL=C
export TZ=UTC
umask 077

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
BUILDER="$SCRIPT_DIR/build_orquesta_v41_runtime.sh"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd -P)"
VENDOR_SOURCE="$REPO_ROOT/vendor/github.com/aavidad/agente_microvm/conectores/orquesta"
GO_BIN="/srv/orquesta-self/toolchains/go1.25.11/bin/go"
SAFE_PATH="/usr/bin:/bin"
export PATH="$SAFE_PATH"
TEST_ROOT="$(mktemp -d /tmp/orquesta-v41-runtime-builder-test.XXXXXX)"
TEST_IDENTITY="$(stat -c '%d:%i' -- "$TEST_ROOT")"

cleanup() {
  local status=$?
  trap - EXIT INT TERM
  set +e
  if [[ -d "$TEST_ROOT" && ! -L "$TEST_ROOT" &&
    "$(stat -c '%d:%i' -- "$TEST_ROOT" 2>/dev/null)" == "$TEST_IDENTITY" &&
    "$TEST_ROOT" == /tmp/orquesta-v41-runtime-builder-test.* ]]; then
    chmod -R u+w -- "$TEST_ROOT"
    find "$TEST_ROOT" -xdev -depth -mindepth 1 -delete
    rmdir -- "$TEST_ROOT"
  fi
  return "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fail_test() {
  printf 'test_build_orquesta_v41_runtime=error reason=%s\n' "$1" >&2
  exit 1
}

sha256_file() {
  env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$1" | cut -d' ' -f1
}

BUILDER_SHA256="$(sha256_file "$BUILDER")"
GO_SHA256="$(sha256_file "$GO_BIN")"
GO_ROOT="$(env -i HOME=/nonexistent GOENV=off GOTOOLCHAIN=local PATH=/nonexistent "$GO_BIN" env GOROOT)"
TOOLCHAIN_TREE_SHA256="$(
  env -i PATH="$SAFE_PATH" LC_ALL=C tar --sort=name --format=gnu --mtime=@0 \
    --owner=0 --group=0 --numeric-owner -cf - -C "$GO_ROOT" . |
    env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1
)"

SOURCE="$TEST_ROOT/source"
PUBLISH="$TEST_ROOT/publish"
TEMPORARY="$TEST_ROOT/temporary"
mkdir -m 0700 -- "$SOURCE" "$PUBLISH" "$TEMPORARY"
mkdir -m 0755 -- "$SOURCE/cmd" "$SOURCE/cmd/orquesta" "$SOURCE/internal" \
  "$SOURCE/config" "$SOURCE/internal/config" "$SOURCE/internal/commands" \
  "$SOURCE/internal/commands/cmd" "$SOURCE/internal/commands/cmd/commandgen" \
  "$SOURCE/internal/adapters" "$SOURCE/internal/adapters/state" \
  "$SOURCE/internal/adapters/state/sqlite" "$SOURCE/internal/adapters/state/sqlite/migrations" \
  "$SOURCE/internal/i18n" "$SOURCE/internal/i18n/catalogs" \
  "$SOURCE/vendor" "$SOURCE/vendor/github.com" "$SOURCE/vendor/github.com/aavidad" \
  "$SOURCE/vendor/github.com/aavidad/agente_microvm" \
  "$SOURCE/vendor/github.com/aavidad/agente_microvm/conectores" \
  "$SOURCE/vendor/github.com/aavidad/agente_microvm/conectores/orquesta" \
  "$SOURCE/.git" "$SOURCE/.tmp" "$SOURCE/internal/unused"

cat > "$SOURCE/go.mod" <<'EOF'
module orquesta

go 1.25.0

toolchain go1.25.11

require github.com/aavidad/agente_microvm/conectores/orquesta v0.0.0-20260902163835-f17173c661e8
EOF
cat > "$SOURCE/go.sum" <<'EOF'
github.com/aavidad/agente_microvm/conectores/orquesta v0.0.0-20260902163835-f17173c661e8 h1:vuFER9h1mN83LoZiofxSnwNpSpWoHQiE/xMdo0MNoZg=
github.com/aavidad/agente_microvm/conectores/orquesta v0.0.0-20260902163835-f17173c661e8/go.mod h1:Q/wNodoAhLQTJweffqspI2QT4XZ/4OJiGnyqhNwbT+s=
EOF
printf '{"schema":"config-fixture"}\n' > "$SOURCE/config/registry.json"
printf '{"schema":"commands-fixture"}\n' > "$SOURCE/internal/commands/registry.json"
cat > "$SOURCE/internal/config/keys_generated.go" <<'EOF'
package config

const Projection = "config-v41"
EOF
cat > "$SOURCE/internal/config/config_test.go" <<'EOF'
package config

import (
	"os"
	"testing"
)

func TestCanonicalRegistryAndEveryGeneratedArtifactStaySynchronized(t *testing.T) {
	body, err := os.ReadFile("../../config/registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{\"schema\":\"config-fixture\"}\n" {
		t.Fatalf("config registry differs from generated projection: %q", body)
	}
	if Projection != "config-v41" {
		t.Fatalf("unexpected generated config projection: %q", Projection)
	}
}
EOF
cat > "$SOURCE/internal/commands/definitions_generated.go" <<'EOF'
package commands

const Projection = "commands-v41"
EOF
cat > "$SOURCE/internal/commands/cmd/commandgen/main.go" <<'EOF'
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := flag.String("root", "", "source root")
	registry := flag.String("registry", "", "registry path")
	check := flag.Bool("check", false, "verify projections")
	write := flag.Bool("write", false, "write projections")
	flag.Parse()
	if *root == "" || *registry == "" || !*check || *write || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "invalid commandgen invocation")
		os.Exit(2)
	}
	body, err := os.ReadFile(filepath.Join(*root, filepath.FromSlash(*registry)))
	if err != nil || string(body) != "{\"schema\":\"commands-fixture\"}\n" {
		fmt.Fprintln(os.Stderr, "command registry differs from fixture authority")
		os.Exit(1)
	}
	projection, err := os.ReadFile(filepath.Join(*root, "internal", "commands", "definitions_generated.go"))
	if err != nil || !strings.Contains(string(projection), `const Projection = "commands-v41"`) {
		fmt.Fprintln(os.Stderr, "command projection differs from registry")
		os.Exit(1)
	}
}
EOF
cat > "$SOURCE/vendor/modules.txt" <<'EOF'
# github.com/aavidad/agente_microvm/conectores/orquesta v0.0.0-20260902163835-f17173c661e8
## explicit; go 1.25
github.com/aavidad/agente_microvm/conectores/orquesta
EOF
cp -a -- "$VENDOR_SOURCE/." \
  "$SOURCE/vendor/github.com/aavidad/agente_microvm/conectores/orquesta/"
cat > "$SOURCE/internal/adapters/state/sqlite/migrations.go" <<'EOF'
package sqlite

import "embed"

//go:embed migrations/*.sql
var Files embed.FS

func Read(path string) string {
	content, err := Files.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
EOF
printf 'CREATE TABLE v40_fixture(id INTEGER);\n' > \
  "$SOURCE/internal/adapters/state/sqlite/migrations/040_terminal_agent_launch_reconciliation.sql"
printf 'CREATE TABLE v41_fixture(id INTEGER);\n' > \
  "$SOURCE/internal/adapters/state/sqlite/migrations/041_expired_agent_launch_continuation_authority.sql"
cat > "$SOURCE/internal/i18n/catalog.go" <<'EOF'
package i18n

import "embed"

//go:embed catalogs/*.json manifest.json
var Files embed.FS

func Read(path string) string {
	content, err := Files.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
EOF
printf '{"locale":"es"}\n' > "$SOURCE/internal/i18n/catalogs/es.json"
printf '{"locale":"en"}\n' > "$SOURCE/internal/i18n/catalogs/en.json"
printf '{"schema":"fixture"}\n' > "$SOURCE/internal/i18n/manifest.json"
cat > "$SOURCE/cmd/orquesta/main.go" <<'EOF'
package main

import (
	"fmt"

	_ "github.com/aavidad/agente_microvm/conectores/orquesta"
	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/commands"
	"orquesta/internal/config"
	"orquesta/internal/i18n"
)

func main() {
	fmt.Print(config.Projection + ":" + commands.Projection + ":" +
		i18n.Read("manifest.json") + ":" +
		sqlite.Read("migrations/041_expired_agent_launch_continuation_authority.sql"))
}
EOF
printf 'package unused\n' > "$SOURCE/internal/unused/unused.go"
printf 'package poison\n' > "$SOURCE/.git/poison.go"
printf 'package poison\n' > "$SOURCE/.tmp/poison.go"
printf 'package main\n' > "$SOURCE/cmd/orquesta/main_test.go"

BASE_ARGS=(
  --source-root "$SOURCE"
  --go "$GO_BIN"
  --expected-go-sha256 "$GO_SHA256"
  --expected-toolchain-tree-sha256 "$TOOLCHAIN_TREE_SHA256"
  --tmp-parent "$TEMPORARY"
)

before_count="$(find "$TEMPORARY" -mindepth 1 -maxdepth 1 | wc -l)"
if "$BUILDER" --expected-builder-sha256 "$(printf '0%.0s' {1..64})" "${BASE_ARGS[@]}" \
  --output "$PUBLISH/orquesta" --receipt "$PUBLISH/orquesta.receipt.json" \
  > "$TEST_ROOT/bad-builder.stdout" 2> "$TEST_ROOT/bad-builder.stderr"; then
  fail_test "bad_builder_digest_accepted"
fi
grep -Fq 'reason=builder_sha256_mismatch' "$TEST_ROOT/bad-builder.stderr" ||
  fail_test "bad_builder_digest_wrong_error"
[[ "$(find "$TEMPORARY" -mindepth 1 -maxdepth 1 | wc -l)" == "$before_count" &&
  ! -e "$PUBLISH/orquesta" && ! -e "$PUBLISH/orquesta.receipt.json" ]] ||
  fail_test "bad_builder_digest_had_side_effects"

PUBLISHED_README="$SOURCE/vendor/github.com/aavidad/agente_microvm/conectores/orquesta/README.md"
mv -- "$PUBLISHED_README" "$PUBLISHED_README.real"
ln -s -- "${PUBLISHED_README##*/}.real" "$PUBLISHED_README"
if "$BUILDER" --expected-builder-sha256 "$BUILDER_SHA256" "${BASE_ARGS[@]}" \
  --output "$PUBLISH/symlink-output" --receipt "$PUBLISH/symlink-receipt.json" \
  > "$TEST_ROOT/symlink.stdout" 2> "$TEST_ROOT/symlink.stderr"; then
  fail_test "selected_symlink_accepted"
fi
grep -Fq 'reason=source_file_not_regular' "$TEST_ROOT/symlink.stderr" || {
  cat "$TEST_ROOT/symlink.stderr" >&2
  fail_test "selected_symlink_wrong_error"
}
[[ ! -e "$PUBLISH/symlink-output" && ! -e "$PUBLISH/symlink-receipt.json" ]] ||
  fail_test "selected_symlink_published_output"
unlink -- "$PUBLISHED_README"
mv -- "$PUBLISHED_README.real" "$PUBLISHED_README"
while IFS= read -r preserved; do
  [[ "$preserved" == "$TEMPORARY"/orquesta-v41-runtime.* && -d "$preserved" && ! -L "$preserved" ]] ||
    fail_test "unexpected_preserved_path"
  chmod -R u+w -- "$preserved"
  find "$preserved" -xdev -depth -mindepth 1 -delete
  rmdir -- "$preserved"
done < <(find "$TEMPORARY" -mindepth 1 -maxdepth 1 -type d -name 'orquesta-v41-runtime.*' -print)

"$BUILDER" --expected-builder-sha256 "$BUILDER_SHA256" "${BASE_ARGS[@]}" \
  --output "$PUBLISH/orquesta" --receipt "$PUBLISH/orquesta.receipt.json" \
  > "$TEST_ROOT/build.stdout" 2> "$TEST_ROOT/build.stderr" || {
    cat "$TEST_ROOT/build.stderr" >&2
    fail_test "fixture_build_failed"
  }

[[ -f "$PUBLISH/orquesta" && ! -L "$PUBLISH/orquesta" &&
  "$(stat -c '%a' -- "$PUBLISH/orquesta")" == 500 ]] || fail_test "output_invalid"
[[ -f "$PUBLISH/orquesta.receipt.json" && ! -L "$PUBLISH/orquesta.receipt.json" &&
  "$(stat -c '%a' -- "$PUBLISH/orquesta.receipt.json")" == 400 ]] || fail_test "receipt_invalid"

EXPECTED_OUTPUT=$'config-v41:commands-v41:{"schema":"fixture"}\n:CREATE TABLE v41_fixture(id INTEGER);'
[[ "$("$PUBLISH/orquesta")" == "$EXPECTED_OUTPUT" ]] || fail_test "binary_behavior_invalid"

jq -e --arg builder "$BUILDER_SHA256" '
  .schema == "orquesta.runtime-v41-rebuild-receipt.v1"
  and .status == "passed"
  and .provenance.authority == "byte_manifest_not_git"
  and .provenance.commit_required == false
  and .provenance.branch_required == false
  and .provenance.builder.expected_sha256 == $builder
  and .provenance.builder.verified_before_side_effects == true
  and .dependency_closure.vendor_mode == true
  and .dependency_closure.published_module.path == "github.com/aavidad/agente_microvm/conectores/orquesta"
  and .dependency_closure.published_module.version == "v0.0.0-20260902163835-f17173c661e8"
  and .dependency_closure.published_module.revision == "f17173c661e80789b731b11a112e7411f48ba83b"
  and .dependency_closure.published_module.local_overlay == false
  and .source.live_go_list_passes == 2
  and .source.stable_between_passes == true
  and .generation_checks.config.status == "passed"
  and .generation_checks.commands.status == "passed"
  and .generation_checks.semantic_validation_duplicated_by_builder == false
  and .build.network.goproxy == "off"
  and .build.network.gosumdb == "off"
  and .build.replicas.count == 2
  and .build.replicas.go_list_passes == 2
  and .build.replicas.byte_equal == true
  and .output.static == true
  and .output.mode == "0500"
' "$PUBLISH/orquesta.receipt.json" >/dev/null || fail_test "receipt_contract_invalid"

for required in \
  config/registry.json \
  internal/config/keys_generated.go \
  internal/commands/registry.json \
  internal/commands/definitions_generated.go \
  internal/adapters/state/sqlite/migrations/040_terminal_agent_launch_reconciliation.sql \
  internal/adapters/state/sqlite/migrations/041_expired_agent_launch_continuation_authority.sql \
  internal/i18n/catalogs/es.json \
  internal/i18n/catalogs/en.json \
  internal/i18n/manifest.json \
  vendor/github.com/aavidad/agente_microvm/conectores/orquesta/cliente.go \
  vendor/github.com/aavidad/agente_microvm/conectores/orquesta/contrato.go \
  vendor/github.com/aavidad/agente_microvm/conectores/orquesta/README.md; do
  jq -e --arg path "$required" '.source.files | any(.path == $path)' \
    "$PUBLISH/orquesta.receipt.json" >/dev/null || fail_test "required_source_missing"
done

for forbidden in .git/poison.go .tmp/poison.go internal/unused/unused.go cmd/orquesta/main_test.go; do
  if jq -e --arg path "$forbidden" '.source.files | any(.path == $path)' \
    "$PUBLISH/orquesta.receipt.json" >/dev/null; then
    fail_test "forbidden_or_unused_source_included"
  fi
done

[[ -z "$(find "$TEMPORARY" -mindepth 1 -print -quit)" ]] || fail_test "successful_run_not_cleaned"
printf 'test_build_orquesta_v41_runtime=ok output_sha256=%s receipt_sha256=%s\n' \
  "$(sha256_file "$PUBLISH/orquesta")" "$(sha256_file "$PUBLISH/orquesta.receipt.json")"
