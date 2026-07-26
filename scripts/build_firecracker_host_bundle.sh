#!/usr/bin/env bash
set -euo pipefail

# Builds the two privileged-host Firecracker executables from an exact Git
# object and publishes a receipt last. The receipt is the commit marker for the
# bundle: binaries without it are an incomplete build, never an activation
# candidate.

readonly SAFE_PATH="/usr/sbin:/usr/bin:/sbin:/bin"
readonly RECEIPT_SCHEMA="orquesta_firecracker_host_bundle_build.v1"
readonly RUN_MARKER_SCHEMA="orquesta_firecracker_host_bundle_tmp.v1"

CONFIRM_REAL=0
SOURCE_ROOT=""
SOURCE_COMMIT=""
TOOLCHAIN_ROOT=""
OUTPUT_ROOT=""
TMP_PARENT=""
RUN_ROOT=""
BUILDER_PATH=""
BUILDER_SHA256=""

usage() {
  cat <<'EOF'
Usage:
  build_firecracker_host_bundle.sh --confirm-real \
    --source-root ABS_PATH --source-commit HEX \
    --toolchain-root ABS_PATH --output-root ABS_PATH \
    --tmp-parent ABS_PATH

Builds launcher and physical-E2E supervisor twice from an exact private
`git archive` export. It publishes:

  orquesta-firecracker-launcher
  orquesta-firecracker-attestor-e2e
  orquesta-firecracker-host-build.receipt.json

The output root must not exist. No network access or download is performed.
EOF
}

fail() {
  printf 'ORQUESTA_FIRECRACKER_HOST_BUILD_ERROR code=%s action=%s\n' \
    "$1" "$2" >&2
  return 1
}

parse_args() {
  while (($#)); do
    case "$1" in
      --confirm-real)
        CONFIRM_REAL=1
        ;;
      --source-root|--source-commit|--toolchain-root|--output-root|--tmp-parent)
        (($# >= 2)) || { usage >&2; exit 64; }
        case "$1" in
          --source-root) SOURCE_ROOT="$2" ;;
          --source-commit) SOURCE_COMMIT="$2" ;;
          --toolchain-root) TOOLCHAIN_ROOT="$2" ;;
          --output-root) OUTPUT_ROOT="$2" ;;
          --tmp-parent) TMP_PARENT="$2" ;;
        esac
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        usage >&2
        exit 64
        ;;
    esac
    shift
  done
  ((CONFIRM_REAL == 1)) ||
    { fail real_confirmation_required rerun_with_--confirm-real; exit 64; }
  for variable in SOURCE_ROOT SOURCE_COMMIT TOOLCHAIN_ROOT OUTPUT_ROOT TMP_PARENT; do
    [[ -n "${!variable}" ]] ||
      { fail required_input_missing "provide_${variable,,}"; exit 64; }
  done
}

canonical_existing_directory() {
  local requested="$1"
  local kind="$2"
  local resolved
  [[ "$requested" = /* ]] ||
    fail "${kind}_path_not_absolute" "provide_absolute_${kind}_path" || return
  resolved="$(readlink -e -- "$requested" 2>/dev/null)" ||
    fail "${kind}_missing" "provide_existing_${kind}" || return
  [[ -d "$resolved" ]] ||
    fail "${kind}_not_directory" "provide_directory_${kind}" || return
  printf '%s\n' "$resolved"
}

canonical_new_directory() {
  local requested="$1"
  local kind="$2"
  local parent base
  [[ "$requested" = /* && "$requested" != "/" ]] ||
    fail "${kind}_path_not_absolute" "provide_absolute_${kind}_path" || return
  [[ ! -e "$requested" && ! -L "$requested" ]] ||
    fail "${kind}_already_exists" "choose_new_${kind}_path" || return
  parent="$(readlink -e -- "$(dirname -- "$requested")" 2>/dev/null)" ||
    fail "${kind}_parent_missing" "create_dedicated_${kind}_parent" || return
  [[ -d "$parent" && -w "$parent" ]] ||
    fail "${kind}_parent_unwritable" "choose_writable_${kind}_parent" || return
  base="$(basename -- "$requested")"
  [[ "$base" != "." && "$base" != ".." ]] ||
    fail "${kind}_basename_invalid" "choose_regular_${kind}_name" || return
  printf '%s/%s\n' "$parent" "$base"
}

path_within() {
  local parent="$1"
  local child="$2"
  [[ "$child" == "$parent" || "$child" == "$parent/"* ]]
}

sha256_file() {
  env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum -- "$1" | cut -d' ' -f1
}

sha256_tree() {
  local root="$1"
  env -i PATH="$SAFE_PATH" LC_ALL=C \
    tar --sort=name --format=gnu --mtime=@0 --owner=0 --group=0 --numeric-owner \
      -cf - -C "$root" . |
    env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1
}

validate_internal_symlinks() {
  local root="$1"
  local kind="$2"
  local link target resolved
  while IFS= read -r -d '' link; do
    target="$(readlink -- "$link")" ||
      fail "${kind}_symlink_unreadable" "repair_${kind}_symlink" || return
    [[ -n "$target" && "$target" != /* ]] ||
      fail "${kind}_symlink_not_relative" "use_relative_${kind}_symlinks" || return
    resolved="$(readlink -e -- "$link" 2>/dev/null)" ||
      fail "${kind}_symlink_dangling" "repair_${kind}_symlink" || return
    path_within "$root" "$resolved" ||
      fail "${kind}_symlink_escape" "keep_${kind}_symlinks_inside_root" || return
  done < <(find "$root" -xdev -type l -print0)
}

validate_toolchain() {
  local root="$1"
  [[ -x "$root/bin/go" && -d "$root/pkg" && -d "$root/src" ]] ||
    fail toolchain_incomplete provide_complete_local_go_toolchain || return
  [[ -z "$(find "$root" -xdev \
    \( ! -user root -o ! -group root -o -perm /022 \) -print -quit)" ]] ||
    fail toolchain_metadata_unsafe provide_root_owned_nonwritable_toolchain || return
  [[ -z "$(find "$root" -xdev ! -type f ! -type d ! -type l -print -quit)" ]] ||
    fail toolchain_special_file remove_special_files_from_toolchain || return
  validate_internal_symlinks "$root" toolchain
}

resolve_source_identity() {
  local root="$1"
  local commit="$2"
  local resolved tree
  [[ "$commit" =~ ^[0-9a-f]{40}$|^[0-9a-f]{64}$ ]] ||
    fail source_commit_invalid provide_lowercase_full_commit || return
  resolved="$(env -i PATH="$SAFE_PATH" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" rev-parse --verify "$commit^{commit}" 2>/dev/null)" ||
    fail source_commit_unavailable fetch_exact_source_commit || return
  [[ "$resolved" == "$commit" ]] ||
    fail source_commit_mismatch provide_exact_commit_object_id || return
  tree="$(env -i PATH="$SAFE_PATH" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" rev-parse --verify "$commit^{tree}" 2>/dev/null)" ||
    fail source_tree_unavailable repair_source_repository || return
  [[ "$tree" =~ ^[0-9a-f]{40}$|^[0-9a-f]{64}$ ]] ||
    fail source_tree_invalid repair_source_repository || return
  if env -i PATH="$SAFE_PATH" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" ls-tree -r "$commit" | awk '$1 == "160000" { found=1 } END { exit !found }'; then
    fail source_gitlink_unsupported vendor_or_remove_source_submodules
    return
  fi
  printf '%s\n' "$tree"
}

prepare_inputs() {
  SOURCE_ROOT="$(canonical_existing_directory "$SOURCE_ROOT" source_root)" || return
  TOOLCHAIN_ROOT="$(canonical_existing_directory "$TOOLCHAIN_ROOT" toolchain_root)" ||
    return
  TMP_PARENT="$(canonical_existing_directory "$TMP_PARENT" tmp_parent)" || return
  OUTPUT_ROOT="$(canonical_new_directory "$OUTPUT_ROOT" output_root)" || return
  [[ -w "$TMP_PARENT" ]] ||
    fail tmp_parent_unwritable choose_writable_private_tmp_parent || return
  [[ "$TMP_PARENT" != "/" ]] ||
    fail tmp_parent_too_broad choose_dedicated_tmp_parent || return
  if path_within "$SOURCE_ROOT" "$OUTPUT_ROOT"; then
    fail output_inside_source choose_output_outside_source_repository
    return
  fi
  validate_toolchain "$TOOLCHAIN_ROOT"
}

prepare_run_root() {
  RUN_ROOT="$(mktemp -d "$TMP_PARENT/orquesta-firecracker-host-build.XXXXXX")"
  chmod 0700 "$RUN_ROOT"
  printf 'schema=%s\nuid=%s\n' "$RUN_MARKER_SCHEMA" "$(id -u)" \
    >"$RUN_ROOT/.orquesta-firecracker-host-build-owned"
}

cleanup() {
  local marker
  [[ -n "$RUN_ROOT" ]] || return 0
  marker="$RUN_ROOT/.orquesta-firecracker-host-build-owned"
  if [[ "$RUN_ROOT" != "$TMP_PARENT"/orquesta-firecracker-host-build.* ||
    ! -f "$marker" ||
    "$(<"$marker")" != $'schema='"$RUN_MARKER_SCHEMA"$'\nuid='"$(id -u)" ]]; then
    printf 'ORQUESTA_FIRECRACKER_HOST_BUILD_PARTIAL state=retained path=%s\n' \
      "$RUN_ROOT" >&2
    return 1
  fi
  find "$RUN_ROOT" -xdev -type d -exec chmod 0700 -- {} + 2>/dev/null || true
  rm -rf -- "$RUN_ROOT"
  RUN_ROOT=""
}

export_source() {
  local archive="$RUN_ROOT/source.tar"
  local export_root="$RUN_ROOT/source"
  mkdir -m 0700 "$export_root"
  env -i PATH="$SAFE_PATH" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$SOURCE_ROOT" archive --format=tar --output="$archive" "$SOURCE_COMMIT"
  env -i PATH="$SAFE_PATH" LC_ALL=C \
    tar -xf "$archive" -C "$export_root" --no-same-owner --no-same-permissions
  [[ -z "$(find "$export_root" -xdev ! -type f ! -type d ! -type l -print -quit)" ]] ||
    fail source_export_special_file inspect_exact_commit_export || return
  validate_internal_symlinks "$export_root" source_export
}

validate_static_elf() {
  local path="$1"
  local kind="$2"
  local description
  description="$(env -i PATH="$SAFE_PATH" LC_ALL=C file -L -- "$path")" ||
    fail "${kind}_file_probe_failed" inspect_compiled_binary || return
  [[ "$description" == *"ELF 64-bit"* && "$description" == *"x86-64"* &&
    "$description" == *"statically linked"* ]] ||
    fail "${kind}_not_static_linux_amd64" inspect_build_recipe || return
  if env -i PATH="$SAFE_PATH" LC_ALL=C readelf -l -- "$path" | grep -q INTERP; then
    fail "${kind}_dynamic_interpreter" keep_cgo_disabled
    return
  fi
}

build_once() {
  local label="$1"
  local package="$2"
  local output="$3"
  local build_root="$RUN_ROOT/build-$label"
  mkdir -m 0700 "$build_root" "$build_root/tmp" "$build_root/cache" \
    "$build_root/mod" "$build_root/path"
  (
    cd "$RUN_ROOT/source"
    env -i \
      PATH="$TOOLCHAIN_ROOT/bin:$SAFE_PATH" GOROOT="$TOOLCHAIN_ROOT" \
      LC_ALL=C TZ=UTC SOURCE_DATE_EPOCH=0 GOENV=off GOTOOLCHAIN=local \
      GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      TMPDIR="$build_root/tmp" GOCACHE="$build_root/cache" \
      GOMODCACHE="$build_root/mod" GOPATH="$build_root/path" \
      "$TOOLCHAIN_ROOT/bin/go" build -mod=vendor -trimpath -buildvcs=false \
        -o "$output" "$package"
  ) || fail "${label}_build_failed" inspect_exact_source_and_toolchain || return
  chmod 0755 "$output"
  validate_static_elf "$output" "$label"
}

build_pair() {
  local name="$1"
  local package="$2"
  build_once "$name-a" "$package" "$RUN_ROOT/$name-a"
  build_once "$name-b" "$package" "$RUN_ROOT/$name-b"
  cmp -s -- "$RUN_ROOT/$name-a" "$RUN_ROOT/$name-b" ||
    fail "${name}_not_reproducible" inspect_isolated_double_build || return
}

write_receipt() {
  local source_tree="$1"
  local source_archive_sha="$2"
  local source_commit_object_sha="$3"
  local toolchain_version="$4"
  local toolchain_tree_sha="$5"
  local toolchain_go_sha="$6"
  local launcher_sha launcher_size supervisor_sha supervisor_size
  launcher_sha="$(sha256_file "$RUN_ROOT/launcher-a")"
  launcher_size="$(stat -c '%s' "$RUN_ROOT/launcher-a")"
  supervisor_sha="$(sha256_file "$RUN_ROOT/supervisor-a")"
  supervisor_size="$(stat -c '%s' "$RUN_ROOT/supervisor-a")"
  env -i PATH="$SAFE_PATH" LC_ALL=C python3 - \
    "$RUN_ROOT/receipt.json" "$RECEIPT_SCHEMA" \
    "$SOURCE_COMMIT" "$source_tree" "$source_archive_sha" \
    "$source_commit_object_sha" "$toolchain_version" "$toolchain_tree_sha" \
    "$toolchain_go_sha" "$BUILDER_SHA256" "$launcher_sha" "$launcher_size" \
    "$supervisor_sha" "$supervisor_size" <<'PY'
import json
import pathlib
import sys

(
    output, schema, commit, tree, archive_sha, commit_object_sha,
    toolchain_version, toolchain_tree_sha, toolchain_go_sha,
    builder_sha, launcher_sha, launcher_size, supervisor_sha, supervisor_size,
) = sys.argv[1:]

document = {
    "artifacts": {
        "launcher": {
            "file": "orquesta-firecracker-launcher",
            "mode": "0755",
            "package": "./cmd/orquesta-firecracker-launcher",
            "sha256": f"sha256:{launcher_sha}",
            "size_bytes": int(launcher_size),
        },
        "supervisor": {
            "file": "orquesta-firecracker-attestor-e2e",
            "mode": "0755",
            "package": "./cmd/orquesta-firecracker-attestor-e2e",
            "sha256": f"sha256:{supervisor_sha}",
            "size_bytes": int(supervisor_size),
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
        "sha256": f"sha256:{builder_sha}",
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
    "schema_version": schema,
    "source": {
        "archive_sha256": f"sha256:{archive_sha}",
        "commit": commit,
        "commit_object_sha256": f"sha256:{commit_object_sha}",
        "export": "git_archive_exact_commit",
        "gitlinks": False,
        "tree_oid": tree,
    },
    "toolchain": {
        "go_binary_sha256": f"sha256:{toolchain_go_sha}",
        "platform": "linux/amd64",
        "tree_sha256": f"sha256:{toolchain_tree_sha}",
        "version": toolchain_version,
    },
}
pathlib.Path(output).write_text(
    json.dumps(document, separators=(",", ":"), sort_keys=True) + "\n",
    encoding="utf-8",
)
PY
  chmod 0444 "$RUN_ROOT/receipt.json"
}

publish_bundle() {
  mkdir -m 0700 "$OUTPUT_ROOT"
  install -m 0755 "$RUN_ROOT/launcher-a" \
    "$OUTPUT_ROOT/orquesta-firecracker-launcher"
  install -m 0755 "$RUN_ROOT/supervisor-a" \
    "$OUTPUT_ROOT/orquesta-firecracker-attestor-e2e"
  sync -f "$OUTPUT_ROOT/orquesta-firecracker-launcher"
  sync -f "$OUTPUT_ROOT/orquesta-firecracker-attestor-e2e"
  install -m 0444 "$RUN_ROOT/receipt.json" \
    "$OUTPUT_ROOT/orquesta-firecracker-host-build.receipt.json"
  sync -f "$OUTPUT_ROOT/orquesta-firecracker-host-build.receipt.json"
  sync -f "$OUTPUT_ROOT"
  sync -f "$(dirname -- "$OUTPUT_ROOT")"
}

main() {
  umask 077
  parse_args "$@"
  for tool in awk cmp cut file find git grep install mktemp python3 readelf \
    readlink sha256sum stat sync tar; do
    command -v "$tool" >/dev/null ||
      { fail host_tool_missing "install_local_$tool"; exit 1; }
  done
  prepare_inputs
  BUILDER_PATH="$(readlink -e -- "${BASH_SOURCE[0]}")" ||
    { fail builder_path_unavailable inspect_builder_source; exit 1; }
  [[ -f "$BUILDER_PATH" && ! -L "$BUILDER_PATH" &&
    "$(stat -c '%h' "$BUILDER_PATH")" == "1" &&
    "$((8#$(stat -c '%a' "$BUILDER_PATH") & 8#022))" == "0" ]] ||
    { fail builder_metadata_unsafe use_single_nonwritable_builder_file; exit 1; }
  BUILDER_SHA256="$(sha256_file "$BUILDER_PATH")"
  local source_tree
  source_tree="$(resolve_source_identity "$SOURCE_ROOT" "$SOURCE_COMMIT")"
  prepare_run_root
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  local toolchain_tree_before toolchain_tree_after toolchain_version
  local toolchain_go_sha source_archive_sha source_commit_object_sha
  toolchain_tree_before="$(sha256_tree "$TOOLCHAIN_ROOT")"
  toolchain_go_sha="$(sha256_file "$TOOLCHAIN_ROOT/bin/go")"
  toolchain_version="$(env -i PATH="$TOOLCHAIN_ROOT/bin" LC_ALL=C \
    GOROOT="$TOOLCHAIN_ROOT" GOENV=off GOTOOLCHAIN=local \
    "$TOOLCHAIN_ROOT/bin/go" version)"
  [[ "$toolchain_version" =~ ^go\ version\ ([0-9A-Za-z._+-]+)\ linux/amd64$ ]] ||
    fail toolchain_version_invalid provide_standard_linux_amd64_go_toolchain
  toolchain_version="${BASH_REMATCH[1]}"

  export_source
  source_archive_sha="$(sha256_file "$RUN_ROOT/source.tar")"
  source_commit_object_sha="$(env -i PATH="$SAFE_PATH" LC_ALL=C \
    GIT_NO_REPLACE_OBJECTS=1 git -C "$SOURCE_ROOT" cat-file commit "$SOURCE_COMMIT" |
    env -i PATH="$SAFE_PATH" LC_ALL=C sha256sum | cut -d' ' -f1)"

  build_pair launcher ./cmd/orquesta-firecracker-launcher
  build_pair supervisor ./cmd/orquesta-firecracker-attestor-e2e

  toolchain_tree_after="$(sha256_tree "$TOOLCHAIN_ROOT")"
  [[ "$toolchain_tree_after" == "$toolchain_tree_before" ]] ||
    fail toolchain_changed_during_build inspect_root_toolchain
  [[ "$(sha256_file "$TOOLCHAIN_ROOT/bin/go")" == "$toolchain_go_sha" ]] ||
    fail toolchain_go_changed_during_build inspect_root_toolchain
  [[ "$(sha256_file "$BUILDER_PATH")" == "$BUILDER_SHA256" ]] ||
    fail builder_changed_during_build inspect_builder_source

  write_receipt "$source_tree" "$source_archive_sha" "$source_commit_object_sha" \
    "$toolchain_version" "$toolchain_tree_before" "$toolchain_go_sha"
  publish_bundle

  printf '%s\n' \
    "status=passed" \
    "schema=$RECEIPT_SCHEMA" \
    "source_commit=$SOURCE_COMMIT" \
    "source_tree=$source_tree" \
    "launcher_sha256=$(sha256_file "$OUTPUT_ROOT/orquesta-firecracker-launcher")" \
    "supervisor_sha256=$(sha256_file "$OUTPUT_ROOT/orquesta-firecracker-attestor-e2e")" \
    "receipt_sha256=$(sha256_file "$OUTPUT_ROOT/orquesta-firecracker-host-build.receipt.json")" \
    "output_root=$OUTPUT_ROOT"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
