#!/usr/bin/env bash
set -Eeuo pipefail

export LC_ALL=C
export TZ=UTC

readonly RECEIPT_SCHEMA="orquesta.runtime-v41-rebuild-receipt.v1"
readonly VENDOR_PACKAGE_REL="vendor/github.com/aavidad/agente_microvm/conectores/orquesta"
readonly VENDOR_MODULE_PATH="github.com/aavidad/agente_microvm/conectores/orquesta"
readonly VENDOR_MODULE_VERSION="v0.0.0-20260902163835-f17173c661e8"
readonly VENDOR_MODULE_REVISION="f17173c661e80789b731b11a112e7411f48ba83b"
readonly VENDOR_MODULE_SUM="h1:vuFER9h1mN83LoZiofxSnwNpSpWoHQiE/xMdo0MNoZg="
readonly VENDOR_MODULE_GO_MOD_SUM="h1:Q/wNodoAhLQTJweffqspI2QT4XZ/4OJiGnyqhNwbT+s="
readonly VENDOR_PACKAGE_TREE_SHA256="a9bfe50a4f272f3a48aacc5aa97e44cacc2f1604003a9d1f75f646cad7831db8"
readonly REQUIRED_GO_VERSION="go version go1.25.11 linux/amd64"
readonly SOURCE_DATE_EPOCH_FIXED="0"
readonly SAFE_PATH="/usr/bin:/bin"
export PATH="$SAFE_PATH"
readonly -a REQUIRED_V41_RUNTIME_PATHS=(
  config/registry.json
  internal/commands/registry.json
  internal/adapters/state/sqlite/migrations/040_terminal_agent_launch_reconciliation.sql
  internal/adapters/state/sqlite/migrations/041_expired_agent_launch_continuation_authority.sql
  internal/i18n/catalogs/en.json
  internal/i18n/catalogs/es.json
  internal/i18n/manifest.json
  "$VENDOR_PACKAGE_REL/README.md"
)
readonly -a VENDOR_PACKAGE_FILES=(
  README.md
  cliente.go
  concesion.go
  concesion_egreso_codec.go
  contrato.go
  credencial_codex.go
  errores.go
  intermediacion.go
  json_estricto.go
  perfil.go
  validacion_contenido.go
)

usage() {
  printf '%s\n' \
    "Uso: ${0##*/} --source-root RUTA --go RUTA --expected-go-sha256 SHA256 \\" \
    "  --expected-toolchain-tree-sha256 SHA256 --expected-builder-sha256 SHA256 \\" \
    "  --output RUTA --receipt RUTA --tmp-parent RUTA" >&2
}

fail() {
  printf 'orquesta_v41_runtime_build=error reason=%s\n' "$1" >&2
  exit 1
}

is_sha256() {
  [[ "$1" =~ ^[0-9a-f]{64}$ ]]
}

SOURCE_ROOT=""
GO_BIN=""
EXPECTED_GO_SHA256=""
EXPECTED_TOOLCHAIN_TREE_SHA256=""
EXPECTED_BUILDER_SHA256=""
OUTPUT=""
RECEIPT=""
TMP_PARENT=""

while (($# > 0)); do
  case "$1" in
    --source-root|--go|--expected-go-sha256|--expected-toolchain-tree-sha256|--expected-builder-sha256|--output|--receipt|--tmp-parent)
      (($# >= 2)) || fail "missing_value_${1#--}"
      case "$1" in
        --source-root) SOURCE_ROOT="$2" ;;
        --go) GO_BIN="$2" ;;
        --expected-go-sha256) EXPECTED_GO_SHA256="$2" ;;
        --expected-toolchain-tree-sha256) EXPECTED_TOOLCHAIN_TREE_SHA256="$2" ;;
        --expected-builder-sha256) EXPECTED_BUILDER_SHA256="$2" ;;
        --output) OUTPUT="$2" ;;
        --receipt) RECEIPT="$2" ;;
        --tmp-parent) TMP_PARENT="$2" ;;
      esac
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unsupported_argument"
      ;;
  esac
done

# This check deliberately precedes every go invocation, mkdir/mktemp and output
# operation. The expected digest is supplied by the caller, so the builder does
# not try to make its own digest authoritative.
is_sha256 "$EXPECTED_BUILDER_SHA256" || fail "expected_builder_sha256_invalid"
SELF_SOURCE="${BASH_SOURCE[0]}"
[[ -f "$SELF_SOURCE" && ! -L "$SELF_SOURCE" ]] || fail "builder_path_invalid"
SELF_DIR="$(cd -- "$(dirname -- "$SELF_SOURCE")" && pwd -P)"
SELF_PATH="$SELF_DIR/$(basename -- "$SELF_SOURCE")"
[[ "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$SELF_PATH" | cut -d' ' -f1)" == \
  "$EXPECTED_BUILDER_SHA256" ]] || fail "builder_sha256_mismatch"

umask 077

for value in "$SOURCE_ROOT" "$GO_BIN" "$OUTPUT" "$RECEIPT" "$TMP_PARENT"; do
  [[ "$value" == /* ]] || fail "path_not_absolute"
done
is_sha256 "$EXPECTED_GO_SHA256" || fail "expected_go_sha256_invalid"
is_sha256 "$EXPECTED_TOOLCHAIN_TREE_SHA256" || fail "expected_toolchain_tree_sha256_invalid"

for tool in awk basename cat chmod cmp cp cut dirname file find grep id install jq \
  mktemp mv readelf readlink rmdir sha256sum sort stat tar tr unlink wc; do
  command -v "$tool" >/dev/null || fail "required_tool_missing_$tool"
done

canonical_directory() {
  local path="$1"
  local label="$2"
  local canonical
  [[ -d "$path" && ! -L "$path" ]] || fail "${label}_invalid"
  canonical="$(readlink -e -- "$path")" || fail "${label}_unresolvable"
  [[ "$canonical" == "$path" ]] || fail "${label}_not_canonical"
  printf '%s\n' "$canonical"
}

SOURCE_ROOT="$(canonical_directory "$SOURCE_ROOT" source_root)"
TMP_PARENT="$(canonical_directory "$TMP_PARENT" tmp_parent)"
[[ "$(stat -c '%u' -- "$TMP_PARENT")" == "$(id -u)" ]] || fail "tmp_parent_owner_invalid"
(( (8#$(stat -c '%a' -- "$TMP_PARENT") & 8#077) == 0 )) || fail "tmp_parent_mode_insecure"

[[ -f "$GO_BIN" && -x "$GO_BIN" && ! -L "$GO_BIN" ]] || fail "go_binary_invalid"
[[ "$(readlink -e -- "$GO_BIN")" == "$GO_BIN" ]] || fail "go_binary_not_canonical"
[[ "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$GO_BIN" | cut -d' ' -f1)" == \
  "$EXPECTED_GO_SHA256" ]] || fail "go_binary_sha256_mismatch"
[[ "$(env -i HOME=/nonexistent GOENV=off GOTOOLCHAIN=local PATH=/nonexistent \
  "$GO_BIN" version)" == "$REQUIRED_GO_VERSION" ]] || fail "go_version_mismatch"
GO_ROOT="$(env -i HOME=/nonexistent GOENV=off GOTOOLCHAIN=local PATH=/nonexistent \
  "$GO_BIN" env GOROOT)"
[[ -d "$GO_ROOT" && ! -L "$GO_ROOT" && "$GO_BIN" == "$GO_ROOT"/* ]] ||
  fail "go_toolchain_not_self_contained"
[[ -z "$(find "$GO_ROOT" -xdev \( -type l -o ! -user root -o -perm /022 \) -print -quit)" ]] ||
  fail "go_toolchain_metadata_unsafe"
TOOLCHAIN_TREE_SHA256="$(
  env -i PATH="$SAFE_PATH" LC_ALL=C tar --sort=name --format=gnu --mtime=@0 \
    --owner=0 --group=0 --numeric-owner -cf - -C "$GO_ROOT" . |
    env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1
)"
[[ "$TOOLCHAIN_TREE_SHA256" == "$EXPECTED_TOOLCHAIN_TREE_SHA256" ]] ||
  fail "go_toolchain_tree_sha256_mismatch"

OUTPUT_PARENT="$(canonical_directory "$(dirname -- "$OUTPUT")" output_parent)"
RECEIPT_PARENT="$(canonical_directory "$(dirname -- "$RECEIPT")" receipt_parent)"
[[ "$OUTPUT_PARENT" == "$RECEIPT_PARENT" ]] || fail "output_and_receipt_parent_differ"
[[ "$(stat -c '%u' -- "$OUTPUT_PARENT")" == "$(id -u)" ]] || fail "output_parent_owner_invalid"
(( (8#$(stat -c '%a' -- "$OUTPUT_PARENT") & 8#077) == 0 )) || fail "output_parent_mode_insecure"
[[ "$OUTPUT" != "$RECEIPT" ]] || fail "output_and_receipt_same_path"
[[ ! -e "$OUTPUT" && ! -L "$OUTPUT" ]] || fail "output_already_exists"
[[ ! -e "$RECEIPT" && ! -L "$RECEIPT" ]] || fail "receipt_already_exists"

RUN_ROOT=""
RUN_IDENTITY=""
OUTPUT_STAGE=""
RECEIPT_STAGE=""
OUTPUT_PUBLISHED=0
OUTPUT_IDENTITY=""
FINALIZED=0

cleanup() {
  local status=$?
  trap - EXIT INT TERM
  set +e
  for stage in "$OUTPUT_STAGE" "$RECEIPT_STAGE"; do
    if [[ -n "$stage" && -f "$stage" && ! -L "$stage" ]]; then
      unlink -- "$stage"
    fi
  done
  if ((status != 0 && OUTPUT_PUBLISHED == 1)) &&
    [[ -f "$OUTPUT" && ! -L "$OUTPUT" && "$(stat -c '%d:%i' -- "$OUTPUT" 2>/dev/null)" == "$OUTPUT_IDENTITY" &&
      ! -e "$RECEIPT" && ! -L "$RECEIPT" ]]; then
    unlink -- "$OUTPUT"
  fi
  if ((status == 0 && FINALIZED == 1)) &&
    [[ -d "$RUN_ROOT" && ! -L "$RUN_ROOT" && "$(stat -c '%d:%i' -- "$RUN_ROOT" 2>/dev/null)" == "$RUN_IDENTITY" &&
      "$RUN_ROOT" == "$TMP_PARENT"/orquesta-v41-runtime.* ]]; then
    chmod -R u+w -- "$RUN_ROOT"
    find "$RUN_ROOT" -xdev -depth -mindepth 1 -delete
    rmdir -- "$RUN_ROOT"
  elif ((status != 0)) && [[ -n "$RUN_ROOT" ]]; then
    printf 'orquesta_v41_runtime_build=preserved run_root=%s\n' "$RUN_ROOT" >&2
  fi
  return "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

RUN_ROOT="$(mktemp -d "$TMP_PARENT/orquesta-v41-runtime.XXXXXX")"
RUN_IDENTITY="$(stat -c '%d:%i' -- "$RUN_ROOT")"

make_isolated_environment() {
  local root="$1"
  install -d -m 0700 -- "$root/home" "$root/gocache" "$root/gomodcache" \
    "$root/gopath" "$root/gotmp" "$root/tmp"
}

run_go_list() {
  local source="$1"
  local environment="$2"
  local output="$3"
  make_isolated_environment "$environment"
  (
    cd -- "$source"
    env -i CGO_ENABLED=0 GOAMD64=v1 GOARCH=amd64 GOENV=off \
      GOFLAGS=-mod=vendor GOCACHE="$environment/gocache" \
      GOMODCACHE="$environment/gomodcache" GONOSUMDB='*' GOPATH="$environment/gopath" \
      GOPROXY=off GOSUMDB=off GOOS=linux GOTOOLCHAIN=local \
      GOTMPDIR="$environment/gotmp" HOME="$environment/home" LANG=C LC_ALL=C \
      PATH="$GO_ROOT/bin:/usr/bin:/bin" SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH_FIXED" \
      TMPDIR="$environment/tmp" TZ=UTC \
      "$GO_BIN" list -mod=vendor -buildvcs=false -deps -json ./cmd/orquesta > "$output"
  ) || fail "go_list_failed"
}

run_generation_checks() {
  local config_environment="$RUN_ROOT/generation-config-env"
  local commands_environment="$RUN_ROOT/generation-commands-env"
  local status=0
  make_isolated_environment "$config_environment"
  make_isolated_environment "$commands_environment"

  (
    cd -- "$SOURCE_ROOT"
    env -i CGO_ENABLED=0 GOAMD64=v1 GOARCH=amd64 GOENV=off \
      GOFLAGS=-mod=vendor GOCACHE="$config_environment/gocache" \
      GOMODCACHE="$config_environment/gomodcache" GONOSUMDB='*' GOPATH="$config_environment/gopath" \
      GOPROXY=off GOSUMDB=off GOOS=linux GOTOOLCHAIN=local \
      GOTMPDIR="$config_environment/gotmp" HOME="$config_environment/home" LANG=C LC_ALL=C \
      PATH="$GO_ROOT/bin:/usr/bin:/bin" SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH_FIXED" \
      TMPDIR="$config_environment/tmp" TZ=UTC \
      "$GO_BIN" test -mod=vendor -count=1 \
        -run '^TestCanonicalRegistryAndEveryGeneratedArtifactStaySynchronized$' ./internal/config
  ) > "$RUN_ROOT/generation-config.stdout" 2> "$RUN_ROOT/generation-config.stderr" || status=$?
  [[ "$status" == 0 ]] || fail "config_generation_check_failed"

  status=0
  (
    cd -- "$SOURCE_ROOT"
    env -i CGO_ENABLED=0 GOAMD64=v1 GOARCH=amd64 GOENV=off \
      GOFLAGS=-mod=vendor GOCACHE="$commands_environment/gocache" \
      GOMODCACHE="$commands_environment/gomodcache" GONOSUMDB='*' GOPATH="$commands_environment/gopath" \
      GOPROXY=off GOSUMDB=off GOOS=linux GOTOOLCHAIN=local \
      GOTMPDIR="$commands_environment/gotmp" HOME="$commands_environment/home" LANG=C LC_ALL=C \
      PATH="$GO_ROOT/bin:/usr/bin:/bin" SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH_FIXED" \
      TMPDIR="$commands_environment/tmp" TZ=UTC \
      "$GO_BIN" run -mod=vendor ./internal/commands/cmd/commandgen \
        --root "$SOURCE_ROOT" --registry internal/commands/registry.json --check
  ) > "$RUN_ROOT/generation-commands.stdout" 2> "$RUN_ROOT/generation-commands.stderr" || status=$?
  [[ "$status" == 0 ]] || fail "command_generation_check_failed"
}

derive_allowlist() {
  local deps_json="$1"
  local source="$2"
  local output="$3"
  local generated="$output.generated"

  jq -se --arg source "$source" '
    ([.[] | select(.ImportPath == "orquesta/cmd/orquesta" and .Name == "main" and .Dir == ($source + "/cmd/orquesta"))] | length) == 1
    and ([.[] | select(.Standard != true and ((.Dir | startswith($source + "/")) | not))] | length) == 0
  ' "$deps_json" >/dev/null || fail "go_list_scope_invalid"

  jq -sr --arg source "$source" '
    [ .[]
      | select(.Standard != true)
      | .Dir as $dir
      | ([.GoFiles[]?, .CgoFiles[]?, .CFiles[]?, .CXXFiles[]?, .MFiles[]?,
          .HFiles[]?, .FFiles[]?, .SFiles[]?, .SwigFiles[]?, .SwigCXXFiles[]?,
          .SysoFiles[]?, .EmbedFiles[]?] | unique[])
      | ($dir + "/" + .)
      | select(startswith($source + "/"))
      | ltrimstr($source + "/")
    ] | unique[]
  ' "$deps_json" > "$generated" || fail "allowlist_derivation_failed"
  {
    cat -- "$generated"
    printf '%s\n' go.mod go.sum vendor/modules.txt "${REQUIRED_V41_RUNTIME_PATHS[@]}"
  } | sort -u > "$output"
  unlink -- "$generated"
  [[ -s "$output" ]] || fail "allowlist_empty"
  local required
  for required in "${REQUIRED_V41_RUNTIME_PATHS[@]}"; do
    grep -Fxq -- "$required" "$output" || fail "required_v41_runtime_source_missing"
  done
}

validate_relative_path() {
  local path="$1"
  [[ "$path" =~ ^[A-Za-z0-9._@+/-]+$ ]] || fail "allowlist_path_charset_invalid"
  [[ -n "$path" && "$path" != /* && "$path" != */../* && "$path" != ../* &&
    "$path" != */./* && "$path" != ./* && "$path" != *//* ]] || fail "allowlist_path_invalid"
  [[ "$path" != .git && "$path" != .git/* && "$path" != */.git/* &&
    "$path" != .tmp && "$path" != .tmp/* && "$path" != */.tmp/* ]] ||
    fail "allowlist_forbidden_path"
}

write_source_manifest() {
  local source="$1"
  local allowlist="$2"
  local output="$3"
  local relative path canonical size mode digest
  : > "$output"
  while IFS= read -r relative; do
    validate_relative_path "$relative"
    path="$source/$relative"
    [[ -f "$path" && ! -L "$path" ]] || fail "source_file_not_regular"
    canonical="$(readlink -e -- "$path")" || fail "source_file_unresolvable"
    [[ "$canonical" == "$path" ]] || fail "source_file_symlinked"
    [[ "$(stat -c '%u' -- "$path")" == "$(id -u)" ]] || fail "source_file_owner_invalid"
    [[ "$(stat -c '%h' -- "$path")" == 1 ]] || fail "source_file_link_count_invalid"
    mode="$(stat -c '%a' -- "$path")"
    (( (8#$mode & 8#7000) == 0 )) || fail "source_file_special_mode"
    size="$(stat -c '%s' -- "$path")"
    digest="$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$path" | cut -d' ' -f1)"
    printf '%s\t%s\t%s\t%s\n' "$relative" "$size" "$mode" "$digest" >> "$output"
  done < "$allowlist"
}

copy_snapshot() {
  local source="$1"
  local manifest="$2"
  local destination="$3"
  local relative size _source_mode digest parent target
  install -d -m 0700 -- "$destination"
  while IFS=$'\t' read -r relative size _source_mode digest; do
    parent="$(dirname -- "$destination/$relative")"
    install -d -m 0700 -- "$parent"
    target="$destination/$relative"
    install -m 0444 -- "$source/$relative" "$target"
    [[ "$(stat -c '%s' -- "$target")" == "$size" &&
      "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$target" | cut -d' ' -f1)" == "$digest" ]] ||
      fail "snapshot_copy_divergent"
  done < "$manifest"
  find "$destination" -xdev -type d -exec chmod 0555 -- {} +
}

verify_snapshot() {
  local snapshot="$1"
  local manifest="$2"
  local relative size _source_mode digest target
  [[ -z "$(find "$snapshot" -xdev \( -type l -o ! -type d ! -type f \) -print -quit)" ]] ||
    fail "snapshot_special_file"
  while IFS=$'\t' read -r relative size _source_mode digest; do
    target="$snapshot/$relative"
    [[ -f "$target" && ! -L "$target" && "$(stat -c '%a' -- "$target")" == 444 &&
      "$(stat -c '%s' -- "$target")" == "$size" &&
      "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$target" | cut -d' ' -f1)" == "$digest" ]] ||
      fail "snapshot_manifest_mismatch"
  done < "$manifest"
  [[ "$(find "$snapshot" -xdev -type f | wc -l)" == "$(wc -l < "$manifest")" ]] ||
    fail "snapshot_file_count_mismatch"
}

validate_published_vendor_module() {
  local source="$1"
  local package="$source/$VENDOR_PACKAGE_REL"
  local actual_tree observed_names expected_names file path
  awk -v module="$VENDOR_MODULE_PATH" -v version="$VENDOR_MODULE_VERSION" '
    $1 == "require" && $2 == "(" { in_require = 1; next }
    $1 == "replace" && $2 == "(" { in_replace = 1; next }
    $1 == ")" { in_require = 0; in_replace = 0; next }
    $1 == "require" && $2 == module {
      matches++
      if ($3 != version) invalid = 1
      next
    }
    in_require && $1 == module {
      matches++
      if ($2 != version) invalid = 1
      next
    }
    ($1 == "replace" && $2 == module) || (in_replace && $1 == module) { invalid = 1 }
    END { exit !(matches == 1 && invalid == 0) }
  ' "$source/go.mod" || fail "published_module_go_mod_invalid"
  awk -v module="$VENDOR_MODULE_PATH" -v version="$VENDOR_MODULE_VERSION" \
    -v module_sum="$VENDOR_MODULE_SUM" -v go_mod_sum="$VENDOR_MODULE_GO_MOD_SUM" '
    $1 == module && $2 == version && $3 == module_sum { module_matches++ }
    $1 == module && $2 == version "/go.mod" && $3 == go_mod_sum { go_mod_matches++ }
    $1 == module && ($2 == version || $2 == version "/go.mod") { selected++ }
    END { exit !(module_matches == 1 && go_mod_matches == 1 && selected == 2) }
  ' "$source/go.sum" || fail "published_module_go_sum_invalid"
  awk -v module="$VENDOR_MODULE_PATH" -v version="$VENDOR_MODULE_VERSION" '
    BEGIN { prefix = "# " module " "; expected = prefix version }
    index($0, prefix) == 1 { headers++; if ($0 == expected) exact++ }
    $0 == module { packages++ }
    END { exit !(headers == 1 && exact == 1 && packages == 1) }
  ' "$source/vendor/modules.txt" || fail "published_module_modules_invalid"
  [[ -d "$package" && ! -L "$package" ]] || fail "published_module_package_invalid"
  observed_names="$(find "$package" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)"
  expected_names="$(printf '%s\n' "${VENDOR_PACKAGE_FILES[@]}" | sort)"
  [[ "$observed_names" == "$expected_names" ]] || fail "published_module_inventory_invalid"
  for file in "${VENDOR_PACKAGE_FILES[@]}"; do
    path="$package/$file"
    [[ -f "$path" && ! -L "$path" && "$(stat -c '%h' -- "$path")" == 1 ]] ||
      fail "published_module_file_invalid"
  done
  actual_tree="$({
    for file in "${VENDOR_PACKAGE_FILES[@]}"; do
      path="$package/$file"
      printf '%s\t%s\t%s\n' "$file" "$(stat -c '%s' -- "$path")" \
        "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$path" | cut -d' ' -f1)"
    done
  } | sort | env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1)"
  [[ "$actual_tree" == "$VENDOR_PACKAGE_TREE_SHA256" ]] ||
    fail "published_module_tree_divergent"
}

run_go_list "$SOURCE_ROOT" "$RUN_ROOT/list-before-env" "$RUN_ROOT/deps-before.json"
derive_allowlist "$RUN_ROOT/deps-before.json" "$SOURCE_ROOT" "$RUN_ROOT/allowlist-before.txt"
write_source_manifest "$SOURCE_ROOT" "$RUN_ROOT/allowlist-before.txt" "$RUN_ROOT/source-before.tsv"
run_generation_checks
copy_snapshot "$SOURCE_ROOT" "$RUN_ROOT/source-before.tsv" "$RUN_ROOT/snapshot"
verify_snapshot "$RUN_ROOT/snapshot" "$RUN_ROOT/source-before.tsv"
validate_published_vendor_module "$RUN_ROOT/snapshot"

run_go_list "$SOURCE_ROOT" "$RUN_ROOT/list-after-env" "$RUN_ROOT/deps-after.json"
derive_allowlist "$RUN_ROOT/deps-after.json" "$SOURCE_ROOT" "$RUN_ROOT/allowlist-after.txt"
write_source_manifest "$SOURCE_ROOT" "$RUN_ROOT/allowlist-after.txt" "$RUN_ROOT/source-after.tsv"
cmp -s -- "$RUN_ROOT/allowlist-before.txt" "$RUN_ROOT/allowlist-after.txt" ||
  fail "source_allowlist_changed_during_snapshot"
cmp -s -- "$RUN_ROOT/source-before.tsv" "$RUN_ROOT/source-after.tsv" ||
  fail "source_bytes_or_modes_changed_during_snapshot"

prepare_replica() {
  local label="$1"
  local root="$RUN_ROOT/replica-$label"
  install -d -m 0700 -- "$root/source" "$root/bin"
  cp -a -- "$RUN_ROOT/snapshot/." "$root/source/"
  find "$root/source" -xdev -type d -exec chmod 0555 -- {} +
  verify_snapshot "$root/source" "$RUN_ROOT/source-before.tsv"
  run_go_list "$root/source" "$root/list-env" "$root/deps.json"
  derive_allowlist "$root/deps.json" "$root/source" "$root/allowlist.txt"
  cmp -s -- "$RUN_ROOT/allowlist-before.txt" "$root/allowlist.txt" ||
    fail "replica_${label}_allowlist_divergent"
  make_isolated_environment "$root/build-env"
}

build_replica() {
  local label="$1"
  local root="$RUN_ROOT/replica-$label"
  local status=0
  (
    cd -- "$root/source"
    env -i CGO_ENABLED=0 GOAMD64=v1 GOARCH=amd64 GOENV=off \
      GOFLAGS=-mod=vendor GOCACHE="$root/build-env/gocache" \
      GOMODCACHE="$root/build-env/gomodcache" GONOSUMDB='*' GOPATH="$root/build-env/gopath" \
      GOPROXY=off GOSUMDB=off GOOS=linux GOTOOLCHAIN=local \
      GOTMPDIR="$root/build-env/gotmp" HOME="$root/build-env/home" LANG=C LC_ALL=C \
      PATH="$GO_ROOT/bin:/usr/bin:/bin" SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH_FIXED" \
      TMPDIR="$root/build-env/tmp" TZ=UTC \
      "$GO_BIN" build -mod=vendor -trimpath -buildvcs=false \
        -ldflags='-s -w -buildid=' -o "$root/bin/orquesta" ./cmd/orquesta
  ) > "$root/build.stdout" 2> "$root/build.stderr" || status=$?
  [[ "$status" == 0 ]] || fail "replica_${label}_build_failed"
}

prepare_replica a
prepare_replica b
build_replica a
build_replica b

BINARY_A="$RUN_ROOT/replica-a/bin/orquesta"
BINARY_B="$RUN_ROOT/replica-b/bin/orquesta"
cmp -s -- "$BINARY_A" "$BINARY_B" || fail "replica_binary_mismatch"
file "$BINARY_A" | grep -Fq 'ELF 64-bit LSB executable, x86-64' || fail "output_not_elf_x86_64"
if readelf -lW "$BINARY_A" | grep -F 'INTERP' >/dev/null ||
  readelf -dW "$BINARY_A" 2>/dev/null | grep -F '(NEEDED)' >/dev/null; then
  fail "output_not_static"
fi

[[ "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$SELF_PATH" | cut -d' ' -f1)" == \
  "$EXPECTED_BUILDER_SHA256" ]] || fail "builder_changed_during_build"
[[ "$(env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$GO_BIN" | cut -d' ' -f1)" == \
  "$EXPECTED_GO_SHA256" ]] || fail "go_binary_changed_during_build"
TOOLCHAIN_TREE_SHA256_AFTER="$(
  env -i PATH="$SAFE_PATH" LC_ALL=C tar --sort=name --format=gnu --mtime=@0 \
    --owner=0 --group=0 --numeric-owner -cf - -C "$GO_ROOT" . |
    env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1
)"
[[ "$TOOLCHAIN_TREE_SHA256_AFTER" == "$EXPECTED_TOOLCHAIN_TREE_SHA256" ]] ||
  fail "go_toolchain_changed_during_build"

ALLOWLIST_SHA256="$(sha256sum "$RUN_ROOT/allowlist-before.txt" | cut -d' ' -f1)"
SOURCE_MANIFEST_SHA256="$(sha256sum "$RUN_ROOT/source-before.tsv" | cut -d' ' -f1)"
OUTPUT_SHA256="$(sha256sum "$BINARY_A" | cut -d' ' -f1)"
OUTPUT_BYTES="$(stat -c '%s' -- "$BINARY_A")"
FILE_COUNT="$(wc -l < "$RUN_ROOT/source-before.tsv")"
SOURCE_BYTES="$(awk -F $'\t' '{ total += $2 } END { print total + 0 }' "$RUN_ROOT/source-before.tsv")"
PACKAGE_COUNT="$(jq -s 'length' "$RUN_ROOT/deps-before.json")"
STANDARD_PACKAGE_COUNT="$(jq -s '[.[] | select(.Standard == true)] | length' "$RUN_ROOT/deps-before.json")"
MODULE_PACKAGE_COUNT="$(jq -s --arg source "$SOURCE_ROOT" \
  '[.[] | select(.Standard != true and ((.Dir | startswith($source + "/vendor/")) | not))] | length' \
  "$RUN_ROOT/deps-before.json")"
VENDOR_PACKAGE_COUNT="$(jq -s --arg source "$SOURCE_ROOT" \
  '[.[] | select(.Standard != true and (.Dir | startswith($source + "/vendor/")))] | length' \
  "$RUN_ROOT/deps-before.json")"

jq -Rn '
  [inputs | split("\t") | {
    path: .[0], bytes: (.[1] | tonumber), source_mode: .[2],
    snapshot_mode: "0444", sha256: .[3]
  }]
' < "$RUN_ROOT/source-before.tsv" > "$RUN_ROOT/source-files.json"

RECEIPT_CONTENT="$RUN_ROOT/receipt.json"
jq -n \
  --arg schema "$RECEIPT_SCHEMA" \
  --arg builder_path "$SELF_PATH" \
  --arg builder_sha256 "$EXPECTED_BUILDER_SHA256" \
  --arg source_root "$SOURCE_ROOT" \
  --arg allowlist_sha256 "$ALLOWLIST_SHA256" \
  --arg source_manifest_sha256 "$SOURCE_MANIFEST_SHA256" \
  --arg go_path "$GO_BIN" \
  --arg go_sha256 "$EXPECTED_GO_SHA256" \
  --arg toolchain_tree_sha256 "$EXPECTED_TOOLCHAIN_TREE_SHA256" \
  --arg output_sha256 "$OUTPUT_SHA256" \
  --arg stdout_a_sha256 "$(sha256sum "$RUN_ROOT/replica-a/build.stdout" | cut -d' ' -f1)" \
  --arg stderr_a_sha256 "$(sha256sum "$RUN_ROOT/replica-a/build.stderr" | cut -d' ' -f1)" \
  --arg stdout_b_sha256 "$(sha256sum "$RUN_ROOT/replica-b/build.stdout" | cut -d' ' -f1)" \
  --arg stderr_b_sha256 "$(sha256sum "$RUN_ROOT/replica-b/build.stderr" | cut -d' ' -f1)" \
  --arg vendor_module_path "$VENDOR_MODULE_PATH" \
  --arg vendor_module_version "$VENDOR_MODULE_VERSION" \
  --arg vendor_module_revision "$VENDOR_MODULE_REVISION" \
  --arg vendor_module_sum "$VENDOR_MODULE_SUM" \
  --arg vendor_module_go_mod_sum "$VENDOR_MODULE_GO_MOD_SUM" \
  --arg vendor_package_tree_sha256 "$VENDOR_PACKAGE_TREE_SHA256" \
  --arg config_check_stdout_sha256 "$(sha256sum "$RUN_ROOT/generation-config.stdout" | cut -d' ' -f1)" \
  --arg config_check_stderr_sha256 "$(sha256sum "$RUN_ROOT/generation-config.stderr" | cut -d' ' -f1)" \
  --arg command_check_stdout_sha256 "$(sha256sum "$RUN_ROOT/generation-commands.stdout" | cut -d' ' -f1)" \
  --arg command_check_stderr_sha256 "$(sha256sum "$RUN_ROOT/generation-commands.stderr" | cut -d' ' -f1)" \
  --argjson output_bytes "$OUTPUT_BYTES" \
  --argjson file_count "$FILE_COUNT" \
  --argjson source_bytes "$SOURCE_BYTES" \
  --argjson package_count "$PACKAGE_COUNT" \
  --argjson standard_package_count "$STANDARD_PACKAGE_COUNT" \
  --argjson module_package_count "$MODULE_PACKAGE_COUNT" \
  --argjson vendor_package_count "$VENDOR_PACKAGE_COUNT" \
  --slurpfile files "$RUN_ROOT/source-files.json" \
  '{
    schema: $schema,
    status: "passed",
    provenance: {
      authority: "byte_manifest_not_git",
      commit_required: false,
      branch_required: false,
      builder: {path: $builder_path, expected_sha256: $builder_sha256, verified_before_side_effects: true}
    },
    source: {
      root: $source_root,
      allowlist: {algorithm: "go_list_deps_files_embeds_plus_fixed_anchors", sha256: $allowlist_sha256},
      manifest_sha256: $source_manifest_sha256,
      file_count: $file_count,
      bytes: $source_bytes,
      directory_mode: "0555",
      live_go_list_passes: 2,
      stable_between_passes: true,
      files: $files[0],
      forbidden_prefixes: [".git/", ".tmp/"]
    },
    dependency_closure: {
      packages: $package_count,
      standard_packages: $standard_package_count,
      module_packages: $module_package_count,
      vendor_packages: $vendor_package_count,
      vendor_mode: true,
      published_module: {
        path: $vendor_module_path,
        version: $vendor_module_version,
        revision: $vendor_module_revision,
        module_sum: $vendor_module_sum,
        go_mod_sum: $vendor_module_go_mod_sum,
        vendor_tree_sha256: $vendor_package_tree_sha256,
        local_overlay: false
      }
    },
    generation_checks: {
      config: {
        command: "go test -mod=vendor -count=1 -run ^TestCanonicalRegistryAndEveryGeneratedArtifactStaySynchronized$ ./internal/config",
        status: "passed",
        stdout_sha256: $config_check_stdout_sha256,
        stderr_sha256: $config_check_stderr_sha256
      },
      commands: {
        command: "go run -mod=vendor ./internal/commands/cmd/commandgen --root SOURCE --registry internal/commands/registry.json --check",
        status: "passed",
        stdout_sha256: $command_check_stdout_sha256,
        stderr_sha256: $command_check_stderr_sha256
      },
      semantic_validation_duplicated_by_builder: false
    },
    toolchain: {
      go_path: $go_path,
      version: "go1.25.11 linux/amd64",
      go_sha256: $go_sha256,
      tree_sha256: $toolchain_tree_sha256
    },
    build: {
      target: "./cmd/orquesta",
      platform: "linux/amd64",
      cgo_enabled: false,
      goamd64: "v1",
      source_date_epoch: 0,
      flags: ["-mod=vendor", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid="],
      network: {goproxy: "off", gosumdb: "off"},
      replicas: {
        count: 2,
        go_list_passes: 2,
        isolated_home_cache_gomodcache_gopath_gotmp_tmp: true,
        byte_equal: true,
        a: {stdout_sha256: $stdout_a_sha256, stderr_sha256: $stderr_a_sha256},
        b: {stdout_sha256: $stdout_b_sha256, stderr_sha256: $stderr_b_sha256}
      }
    },
    output: {
      sha256: $output_sha256,
      bytes: $output_bytes,
      mode: "0500",
      elf_x86_64: true,
      static: true
    }
  }' > "$RECEIPT_CONTENT"
chmod 0400 -- "$RECEIPT_CONTENT"

OUTPUT_STAGE="$(mktemp "$OUTPUT_PARENT/.$(basename -- "$OUTPUT").orquesta-stage.XXXXXX")"
RECEIPT_STAGE="$(mktemp "$RECEIPT_PARENT/.$(basename -- "$RECEIPT").orquesta-stage.XXXXXX")"
install -m 0500 -- "$BINARY_A" "$OUTPUT_STAGE"
cat -- "$RECEIPT_CONTENT" > "$RECEIPT_STAGE"
chmod 0400 -- "$RECEIPT_STAGE"
[[ "$(sha256sum "$OUTPUT_STAGE" | cut -d' ' -f1)" == "$OUTPUT_SHA256" ]] ||
  fail "output_stage_divergent"

mv -Tn -- "$OUTPUT_STAGE" "$OUTPUT"
[[ -f "$OUTPUT" && ! -L "$OUTPUT" && ! -e "$OUTPUT_STAGE" ]] || fail "output_publish_race"
OUTPUT_STAGE=""
OUTPUT_PUBLISHED=1
OUTPUT_IDENTITY="$(stat -c '%d:%i' -- "$OUTPUT")"
mv -Tn -- "$RECEIPT_STAGE" "$RECEIPT"
[[ -f "$RECEIPT" && ! -L "$RECEIPT" && ! -e "$RECEIPT_STAGE" ]] || fail "receipt_publish_race"
RECEIPT_STAGE=""
FINALIZED=1

printf 'orquesta_v41_runtime_build=ok output_sha256=%s output_bytes=%s receipt_sha256=%s files=%s\n' \
  "$OUTPUT_SHA256" "$OUTPUT_BYTES" "$(sha256sum "$RECEIPT" | cut -d' ' -f1)" "$FILE_COUNT"
