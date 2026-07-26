#!/usr/bin/env bash
set -euo pipefail

# Reproducible initramfs builder for the TestAttestor guest only. This does not
# build or run agents and does not discover/download host inputs.

PATH_SAFE="/usr/sbin:/usr/bin:/sbin:/bin"
MARKER_SCHEMA="orquesta_test_guest_build_tmp.v0"
# This immutable packaging contract mirrors the guest's capacity reserves and
# tmpfs ceiling. Charging the unpacked image and both reserves inside 75% leaves
# the other 25% of guest RAM as kernel/runtime headroom.
readonly MEMORY_MIB_BYTES=1048576
readonly SCRATCH_FIXED_RESERVE_BYTES=67108864
readonly SCRATCH_CACHE_RESERVE_BYTES=268435456
readonly SCRATCH_TMPFS_PERCENT=75
readonly KERNEL_RUNTIME_HEADROOM_PERCENT=$((100 - SCRATCH_TMPFS_PERCENT))

CONFIRM_REAL=0
KEEP_TEMP=0
SOURCE_ROOT=""
SOURCE_COMMIT=""
TOOLCHAIN_ROOT=""
BUSYBOX_BIN=""
OUTPUT_IMAGE=""
OUTPUT_MANIFEST=""
TMP_PARENT=""

RUN_DIR=""
RUN_MARKER=""
SOURCE_EXPORT_ROOT=""
TOOLCHAIN_FD=""
TOOLCHAIN_PINNED_ROOT=""
TOOLCHAIN_SOURCE_IDENTITY=""
TOOLCHAIN_REQUIRE_ROOT_OWNERSHIP=0
TOOLCHAIN_VERSION=""
BUSYBOX_FD=""
BUSYBOX_PINNED_PATH=""
BUSYBOX_SOURCE_IDENTITY=""
BUSYBOX_SOURCE_SHA256=""
BUSYBOX_REQUIRE_ROOT_OWNERSHIP=0
BUSYBOX_VERSION=""
PUBLISH_IMAGE_STAGE=""
PUBLISH_IMAGE_STAGE_IDENTITY=""
PUBLISH_MANIFEST_STAGE=""
PUBLISH_MANIFEST_STAGE_IDENTITY=""
PUBLISHED_IMAGE_IDENTITY=""
PUBLISHED_IMAGE_PENDING=0
PUBLISHED_MANIFEST_IDENTITY=""
PUBLISHED_MANIFEST_PENDING=0
PUBLISHED_MANIFEST_DURABLE=0

usage() {
  cat <<'EOF'
Usage:
  build_firecracker_attestor_guest.sh --confirm-real \
    --source-root ABS_PATH --source-commit HEX \
    --toolchain-root ABS_PATH --busybox ABS_PATH \
    --output ABS_PATH --manifest ABS_PATH --tmp-parent ABS_PATH [--keep-temp]

Builds a deterministic TestAttestor initramfs. It compiles
./cmd/orquesta/test-guest with the supplied local Go toolchain, then packages
only the static runner, a static BusyBox and the complete toolchain under
/toolchain. It performs no download, installation, network setup, SSH,
cloud-init, secret capture or agent execution.
EOF
}

fail() {
  printf 'ORQUESTA_TEST_GUEST_BUILD_ERROR_V0 code=%s action=%s\n' "$1" "$2" >&2
  return 1
}

parse_args() {
  while (($#)); do
    case "$1" in
      --confirm-real) CONFIRM_REAL=1 ;;
      --keep-temp) KEEP_TEMP=1 ;;
      --source-root|--source-commit|--toolchain-root|--busybox|--output|--manifest|--tmp-parent)
        (($# >= 2)) || { usage >&2; exit 64; }
        case "$1" in
          --source-root) SOURCE_ROOT="$2" ;;
          --source-commit) SOURCE_COMMIT="$2" ;;
          --toolchain-root) TOOLCHAIN_ROOT="$2" ;;
          --busybox) BUSYBOX_BIN="$2" ;;
          --output) OUTPUT_IMAGE="$2" ;;
          --manifest) OUTPUT_MANIFEST="$2" ;;
          --tmp-parent) TMP_PARENT="$2" ;;
        esac
        shift
        ;;
      -h|--help) usage; exit 0 ;;
      *) usage >&2; exit 64 ;;
    esac
    shift
  done
  ((CONFIRM_REAL == 1)) ||
    { fail "real_confirmation_required" "rerun_with_--confirm-real"; exit 64; }
  for value in SOURCE_ROOT SOURCE_COMMIT TOOLCHAIN_ROOT BUSYBOX_BIN OUTPUT_IMAGE OUTPUT_MANIFEST TMP_PARENT; do
    [[ -n "${!value}" ]] ||
      { fail "required_input_missing" "provide_${value,,}"; exit 64; }
  done
}

canonical_existing() {
  local requested="$1"
  local kind="$2"
  [[ "$requested" = /* ]] ||
    fail "${kind}_path_not_absolute" "provide_absolute_${kind}_path" || return
  readlink -e -- "$requested" 2>/dev/null ||
    fail "${kind}_missing" "provide_existing_local_${kind}" || return
}

canonical_output() {
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
    fail "${kind}_basename_invalid" "choose_regular_${kind}_filename" || return
  printf '%s/%s\n' "$parent" "$base"
}

path_exists() {
  [[ -e "$1" || -L "$1" ]]
}

validate_output_pair_state() {
  local image="$1"
  local manifest="$2"
  local image_exists=0 manifest_exists=0
  path_exists "$image" && image_exists=1
  path_exists "$manifest" && manifest_exists=1
  if ((image_exists != manifest_exists)); then
    fail "partial_publication_detected" \
      "inspect_and_remove_only_the_uncommitted_image_or_manifest"
    return
  fi
}

path_within() {
  local parent="$1"
  local child="$2"
  [[ "$child" == "$parent" || "$child" == "$parent/"* ]]
}

validate_static_elf() {
  local binary="$1"
  local kind="$2"
  local description
  [[ -f "$binary" && -r "$binary" ]] ||
    fail "${kind}_unreadable" "provide_readable_${kind}" || return
  description="$(env -i PATH="$PATH_SAFE" LC_ALL=C file -L -- "$binary")" ||
    fail "${kind}_file_probe_failed" "inspect_${kind}" || return
  grep -Fq 'ELF 64-bit' <<<"$description" &&
    grep -Fq 'x86-64' <<<"$description" &&
    grep -Fq 'statically linked' <<<"$description" ||
    fail "${kind}_not_static_amd64" "provide_static_linux_amd64_${kind}" || return
  if env -i PATH="$PATH_SAFE" LC_ALL=C readelf -l -- "$binary" | grep -q INTERP; then
    fail "${kind}_has_dynamic_interpreter" "rebuild_${kind}_with_CGO_ENABLED_0"
    return
  fi
}

validate_toolchain() {
  local root="$1"
  reject_descendant_mounts "$root" || return
  [[ -x "$root/bin/go" && -d "$root/pkg" && -d "$root/src" && -d "$root/lib" ]] ||
    fail "toolchain_incomplete" "provide_complete_local_go_toolchain" || return
  if [[ -n "$(find "$root" -xdev ! -type f ! -type d ! -type l -print -quit)" ]]; then
    fail "toolchain_special_file" "remove_special_files_from_toolchain"
    return
  fi
  validate_relative_internal_symlinks "$root" toolchain || return
}

validate_relative_internal_symlinks() {
  local root="$1"
  local kind="$2"
  local link target resolved
  root="$(readlink -e -- "$root")" ||
    fail "${kind}_root_unresolvable" "repair_${kind}_root" || return
  while IFS= read -r -d '' link; do
    target="$(readlink -- "$link")" ||
      fail "${kind}_symlink_unreadable" "repair_${kind}_symlink" || return
    [[ -n "$target" && "$target" != /* ]] ||
      fail "${kind}_symlink_not_relative" "use_only_relative_${kind}_symlinks" || return
    resolved="$(readlink -e -- "$link" 2>/dev/null)" ||
      fail "${kind}_symlink_dangling" "repair_${kind}_symlink" || return
    path_within "$root" "$resolved" ||
      fail "${kind}_symlink_escape" "keep_${kind}_symlinks_inside_root" || return
  done < <(find "$root" -xdev -type l -print0)
}

reject_descendant_mounts() {
  local root="$1"
  local mountpoint
  root="$(readlink -e -- "$root")" ||
    fail "toolchain_root_unresolvable" "repair_toolchain_root" || return
  while IFS= read -r mountpoint; do
    if [[ "$mountpoint" == "$root/"* ]]; then
      fail "toolchain_submount" "remove_all_descendant_toolchain_mounts"
      return
    fi
  done < <(findmnt -rn -c -o TARGET)
}

validate_stable_toolchain_ownership() {
  local root="$1"
  if [[ -n "$(find "$root" -xdev \( ! -user root -o ! -group root \) -print -quit)" ]]; then
    fail "toolchain_not_root_owned" "provide_root_owned_toolchain_tree"
    return
  fi
  if [[ -n "$(find "$root" -xdev ! -type l -perm /022 -print -quit)" ]]; then
    fail "toolchain_group_or_world_writable" "remove_group_and_world_write_bits"
    return
  fi
}

release_toolchain_pin() {
  if [[ -n "$TOOLCHAIN_FD" ]]; then
    exec {TOOLCHAIN_FD}<&-
  fi
  TOOLCHAIN_FD=""
  TOOLCHAIN_PINNED_ROOT=""
  TOOLCHAIN_SOURCE_IDENTITY=""
  TOOLCHAIN_REQUIRE_ROOT_OWNERSHIP=0
  TOOLCHAIN_VERSION=""
}

release_busybox_pin() {
  if [[ -n "$BUSYBOX_FD" ]]; then
    exec {BUSYBOX_FD}<&-
  fi
  BUSYBOX_FD=""
  BUSYBOX_PINNED_PATH=""
  BUSYBOX_SOURCE_IDENTITY=""
  BUSYBOX_SOURCE_SHA256=""
  BUSYBOX_REQUIRE_ROOT_OWNERSHIP=0
  BUSYBOX_VERSION=""
}

validate_busybox_ownership() {
  local binary="$1"
  local uid gid mode permissions
  read -r uid gid mode < <(stat -Lc '%u %g %a' -- "$binary") ||
    fail "busybox_metadata_probe_failed" "inspect_busybox" || return
  [[ "$uid" = "0" && "$gid" = "0" ]] ||
    fail "busybox_not_root_owned" "provide_root_owned_busybox" || return
  permissions=$((8#$mode))
  (( (permissions & 0022) == 0 )) ||
    fail "busybox_group_or_world_writable" "remove_group_and_world_write_bits"
}

pin_busybox_source() {
  local binary="$1"
  local require_root_ownership="$2"
  release_busybox_pin
  [[ -f "$binary" && ! -L "$binary" ]] ||
    fail "busybox_not_regular" "provide_regular_static_busybox" || return
  if ((require_root_ownership == 1)); then
    validate_busybox_ownership "$binary" || return
  fi
  exec {BUSYBOX_FD}<"$binary" ||
    fail "busybox_pin_failed" "repair_busybox_permissions" || return
  BUSYBOX_PINNED_PATH="/proc/$$/fd/$BUSYBOX_FD"
  BUSYBOX_REQUIRE_ROOT_OWNERSHIP="$require_root_ownership"
  [[ -f "$BUSYBOX_PINNED_PATH" ]] ||
    fail "busybox_pin_not_regular" "provide_regular_static_busybox" || return
  if ((BUSYBOX_REQUIRE_ROOT_OWNERSHIP == 1)); then
    validate_busybox_ownership "$BUSYBOX_PINNED_PATH" || return
  fi
  BUSYBOX_SOURCE_IDENTITY="$(stat -Lc '%d:%i:%s:%Y:%Z:%u:%g:%a' -- "$BUSYBOX_PINNED_PATH")" ||
    fail "busybox_pin_probe_failed" "inspect_busybox" || return
  BUSYBOX_SOURCE_SHA256="$(sha256_file "$BUSYBOX_PINNED_PATH")" ||
    fail "busybox_hash_failed" "inspect_busybox" || return
  verify_busybox_pin
}

verify_busybox_pin() {
  local pinned_identity path_identity pinned_sha
  [[ -n "$BUSYBOX_PINNED_PATH" && -n "$BUSYBOX_SOURCE_IDENTITY" &&
    -n "$BUSYBOX_SOURCE_SHA256" ]] ||
    fail "busybox_not_pinned" "pin_busybox_before_copy" || return
  [[ -f "$BUSYBOX_PINNED_PATH" && ! -L "$BUSYBOX_BIN" ]] ||
    fail "busybox_path_changed" "retry_with_stable_busybox" || return
  if ((BUSYBOX_REQUIRE_ROOT_OWNERSHIP == 1)); then
    validate_busybox_ownership "$BUSYBOX_PINNED_PATH" || return
    validate_busybox_ownership "$BUSYBOX_BIN" || return
  fi
  pinned_identity="$(stat -Lc '%d:%i:%s:%Y:%Z:%u:%g:%a' -- "$BUSYBOX_PINNED_PATH")" ||
    fail "busybox_pin_lost" "retry_with_stable_busybox" || return
  [[ "$pinned_identity" = "$BUSYBOX_SOURCE_IDENTITY" ]] ||
    fail "busybox_metadata_changed" "retry_with_stable_busybox" || return
  path_identity="$(stat -Lc '%d:%i:%s:%Y:%Z:%u:%g:%a' -- "$BUSYBOX_BIN")" ||
    fail "busybox_path_lost" "retry_with_stable_busybox" || return
  [[ "$path_identity" = "$BUSYBOX_SOURCE_IDENTITY" ]] ||
    fail "busybox_path_changed" "retry_with_stable_busybox" || return
  pinned_sha="$(sha256_file "$BUSYBOX_PINNED_PATH")" ||
    fail "busybox_hash_failed" "inspect_busybox" || return
  [[ "$pinned_sha" = "$BUSYBOX_SOURCE_SHA256" ]] ||
    fail "busybox_content_changed" "retry_with_stable_busybox"
}

pin_toolchain_source() {
  local root="$1"
  local require_root_ownership="$2"
  local pinned_resolved
  release_toolchain_pin
  reject_descendant_mounts "$root" || return
  if ((require_root_ownership == 1)); then
    validate_stable_toolchain_ownership "$root" || return
  fi
  exec {TOOLCHAIN_FD}<"$root" ||
    fail "toolchain_pin_failed" "repair_toolchain_permissions" || return
  TOOLCHAIN_PINNED_ROOT="/proc/$$/fd/$TOOLCHAIN_FD"
  TOOLCHAIN_REQUIRE_ROOT_OWNERSHIP="$require_root_ownership"
  pinned_resolved="$(readlink -e -- "$TOOLCHAIN_PINNED_ROOT")" ||
    fail "toolchain_pin_unresolvable" "repair_toolchain_root" || return
  [[ "$pinned_resolved" = "$root" ]] ||
    fail "toolchain_pin_mismatch" "retry_with_stable_toolchain_root" || return
  TOOLCHAIN_SOURCE_IDENTITY="$(stat -Lc '%d:%i:%u:%g' -- "$TOOLCHAIN_PINNED_ROOT")" ||
    fail "toolchain_pin_probe_failed" "repair_toolchain_root" || return
  if ((TOOLCHAIN_REQUIRE_ROOT_OWNERSHIP == 1)); then
    # /proc/<pid>/fd/<n> is a process-owned symlink even when its pinned
    # directory is entirely root-owned. Validate the resolved directory, then
    # keep using the descriptor identity for every subsequent operation.
    validate_stable_toolchain_ownership "$pinned_resolved" || return
  fi
  verify_toolchain_pin
}

verify_toolchain_pin() {
  local identity path_identity pinned_resolved
  [[ -n "$TOOLCHAIN_PINNED_ROOT" && -n "$TOOLCHAIN_SOURCE_IDENTITY" ]] ||
    fail "toolchain_not_pinned" "pin_toolchain_before_copy" || return
  identity="$(stat -Lc '%d:%i:%u:%g' -- "$TOOLCHAIN_PINNED_ROOT")" ||
    fail "toolchain_pin_lost" "retry_with_stable_toolchain_root" || return
  [[ "$identity" = "$TOOLCHAIN_SOURCE_IDENTITY" ]] ||
    fail "toolchain_identity_changed" "retry_with_stable_toolchain_root" || return
  path_identity="$(stat -Lc '%d:%i:%u:%g' -- "$TOOLCHAIN_ROOT")" ||
    fail "toolchain_path_lost" "retry_with_stable_toolchain_root" || return
  [[ "$path_identity" = "$TOOLCHAIN_SOURCE_IDENTITY" ]] ||
    fail "toolchain_path_changed" "retry_with_stable_toolchain_root" || return
  if ((TOOLCHAIN_REQUIRE_ROOT_OWNERSHIP == 1)); then
    pinned_resolved="$(readlink -e -- "$TOOLCHAIN_PINNED_ROOT")" ||
      fail "toolchain_pin_unresolvable" "repair_toolchain_root" || return
    [[ "$pinned_resolved" = "$TOOLCHAIN_ROOT" ]] ||
      fail "toolchain_pin_mismatch" "retry_with_stable_toolchain_root" || return
    validate_stable_toolchain_ownership "$pinned_resolved" || return
  fi
  reject_descendant_mounts "$TOOLCHAIN_ROOT" || return
}

probe_pinned_toolchain() {
  local output
  verify_toolchain_pin || return
  validate_toolchain "$TOOLCHAIN_PINNED_ROOT" || return
  output="$(env -i PATH="$TOOLCHAIN_PINNED_ROOT/bin" LC_ALL=C \
    GOROOT="$TOOLCHAIN_PINNED_ROOT" GOENV=off GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local \
    "$TOOLCHAIN_PINNED_ROOT/bin/go" version)" ||
    fail "toolchain_go_unusable" "repair_local_go_toolchain" || return
  verify_toolchain_pin || return
  [[ "$output" =~ ^go\ version\ ([0-9A-Za-z._+-]+)\ linux/amd64$ ]] ||
    fail "toolchain_version_unstable" "provide_standard_linux_amd64_go_toolchain" || return
  TOOLCHAIN_VERSION="${BASH_REMATCH[1]}"
}

probe_pinned_busybox() {
  local output first_line
  verify_busybox_pin || return
  output="$(exec -c -a busybox "$BUSYBOX_PINNED_PATH" 2>&1)" ||
    fail "busybox_version_probe_failed" "provide_standard_static_busybox" || return
  verify_busybox_pin || return
  first_line="${output%%$'\n'*}"
  [[ "$first_line" =~ ^BusyBox[[:space:]](v[0-9A-Za-z._+-]+)[[:space:]] ]] ||
    fail "busybox_version_unstable" "provide_standard_static_busybox" || return
  BUSYBOX_VERSION="${BASH_REMATCH[1]}"
}

validate_source_provenance() {
  local root="$1"
  local expected_commit="$2"
  local repository_root head status
  repository_root="$(env -i PATH="$PATH_SAFE" LC_ALL=C \
    git -C "$root" rev-parse --show-toplevel 2>/dev/null)" ||
    fail "source_not_git_repository" "provide_local_git_checkout" || return
  repository_root="$(readlink -e -- "$repository_root")" ||
    fail "source_repository_unresolvable" "repair_local_git_checkout" || return
  [[ "$repository_root" = "$root" ]] ||
    fail "source_root_not_repository_root" "provide_exact_git_repository_root" || return
  head="$(env -i PATH="$PATH_SAFE" LC_ALL=C git -C "$root" rev-parse --verify HEAD 2>/dev/null)" ||
    fail "source_head_unavailable" "checkout_exact_source_commit" || return
  [[ "$head" = "$expected_commit" ]] ||
    fail "source_head_mismatch" "use_exact_checked_out_source_commit" || return
  status="$(env -i PATH="$PATH_SAFE" LC_ALL=C \
    git -C "$root" status --porcelain=v1 --untracked-files=all 2>/dev/null)" ||
    fail "source_status_unavailable" "repair_local_git_checkout" || return
  [[ -z "$status" ]] ||
    fail "source_tree_dirty" "commit_or_remove_all_tracked_index_and_untracked_changes"
}

export_source_commit() {
  local root="$1"
  local commit="$2"
  local destination="$3"
  local resolved record
  resolved="$(env -i PATH="$PATH_SAFE" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" rev-parse --verify "$commit^{commit}" 2>/dev/null)" ||
    fail "source_commit_unavailable" "fetch_or_checkout_exact_source_commit" || return
  [[ "$resolved" = "$commit" ]] ||
    fail "source_commit_mismatch" "provide_exact_commit_object_id" || return
  while IFS= read -r -d '' record; do
    [[ "$record" != 160000\ * ]] ||
      fail "source_gitlink_unsupported" "vendor_or_remove_source_submodules" || return
  done < <(env -i PATH="$PATH_SAFE" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" ls-tree -r -z "$commit")
  mkdir -m 0700 "$destination"
  env -i PATH="$PATH_SAFE" LC_ALL=C GIT_NO_REPLACE_OBJECTS=1 \
    git -C "$root" archive --format=tar "$commit" |
    env -i PATH="$PATH_SAFE" LC_ALL=C tar \
      --extract --directory "$destination" --no-same-owner --no-same-permissions
  if [[ -n "$(find "$destination" -xdev ! -type f ! -type d ! -type l -print -quit)" ]]; then
    fail "source_export_special_file" "inspect_exact_commit_export"
    return
  fi
  validate_relative_internal_symlinks "$destination" source_export || return
  normalize_read_only_tree_modes "$destination"
}

prepare_inputs() {
  SOURCE_ROOT="$(canonical_existing "$SOURCE_ROOT" source_root)" || return
  TOOLCHAIN_ROOT="$(canonical_existing "$TOOLCHAIN_ROOT" toolchain_root)" || return
  BUSYBOX_BIN="$(canonical_existing "$BUSYBOX_BIN" busybox)" || return
  validate_output_pair_state "$OUTPUT_IMAGE" "$OUTPUT_MANIFEST" || return
  OUTPUT_IMAGE="$(canonical_output "$OUTPUT_IMAGE" output)" || return
  OUTPUT_MANIFEST="$(canonical_output "$OUTPUT_MANIFEST" manifest)" || return
  TMP_PARENT="$(canonical_existing "$TMP_PARENT" tmp_parent)" || return
  [[ -d "$SOURCE_ROOT" && -d "$TOOLCHAIN_ROOT" && -d "$TMP_PARENT" && -w "$TMP_PARENT" ]] ||
    fail "input_directory_invalid" "provide_readable_source_toolchain_and_writable_tmp_parent" || return
  [[ "$TMP_PARENT" != "/" ]] ||
    fail "tmp_parent_too_broad" "provide_dedicated_tmp_parent" || return
  [[ "$OUTPUT_IMAGE" != "$OUTPUT_MANIFEST" ]] ||
    fail "output_paths_collide" "choose_distinct_image_and_manifest_paths" || return
  if path_within "$SOURCE_ROOT" "$OUTPUT_IMAGE" || path_within "$SOURCE_ROOT" "$OUTPUT_MANIFEST"; then
    fail "output_inside_source" "write_image_and_manifest_outside_source_tree"
    return
  fi
  [[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$|^[0-9a-f]{64}$ ]] ||
    fail "source_commit_invalid" "provide_lowercase_40_or_64_hex_commit" || return
  [[ -d "$SOURCE_ROOT/cmd/orquesta/test-guest" ]] ||
    fail "guest_command_missing" "finish_cmd_orquesta_test_guest_before_real_build" || return
  validate_source_provenance "$SOURCE_ROOT" "$SOURCE_COMMIT" || return
  pin_toolchain_source "$TOOLCHAIN_ROOT" 1 || return
  probe_pinned_toolchain || return
  pin_busybox_source "$BUSYBOX_BIN" 1 || return
  validate_static_elf "$BUSYBOX_PINNED_PATH" busybox || return
  probe_pinned_busybox
}

prepare_run_dir() {
  RUN_DIR="$(mktemp -d "$TMP_PARENT/orquesta-test-guest-build.XXXXXX")"
  chmod 700 "$RUN_DIR"
  RUN_MARKER="$RUN_DIR/.orquesta-test-guest-build-owned"
  printf 'schema=%s\nuid=%s\n' "$MARKER_SCHEMA" "$(id -u)" >"$RUN_MARKER"
}

remove_run_dir() {
  local target="$1"
  local marker="$target/.orquesta-test-guest-build-owned"
  [[ "$target" = /* && "$target" != "/" &&
    "$(basename -- "$target")" = orquesta-test-guest-build.* &&
    -f "$marker" ]] ||
    fail "cleanup_target_unowned" "inspect_only_marked_guest_build_tmp" || return
  grep -Fqx "schema=$MARKER_SCHEMA" "$marker" ||
    fail "cleanup_marker_invalid" "inspect_marked_guest_build_tmp" || return
  find "$target" -xdev -type d -exec chmod 0700 -- {} +
  rm -rf -- "$target"
}

cleanup() {
  local saved=$?
  trap - EXIT INT TERM
  cleanup_partial_publication || saved=1
  release_toolchain_pin
  release_busybox_pin
  if [[ -n "$RUN_DIR" && -d "$RUN_DIR" ]]; then
    if ((KEEP_TEMP == 1)); then
      printf 'ORQUESTA_TEST_GUEST_BUILD_TEMP_V0 retained=true path=%s\n' "$RUN_DIR" >&2
    else
      remove_run_dir "$RUN_DIR" || saved=1
    fi
  fi
  exit "$saved"
}

path_identity() {
  stat -Lc '%d:%i' -- "$1"
}

remove_owned_publication_path() {
  local path="$1"
  local expected_identity="$2"
  local kind="$3"
  local current_identity
  if ! path_exists "$path"; then
    return
  fi
  current_identity="$(path_identity "$path" 2>/dev/null)" || {
    printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=manual_recovery_required kind=%s path=%s reason=identity_unreadable\n' \
      "$kind" "$path" >&2
    return 1
  }
  if [[ "$current_identity" != "$expected_identity" ]]; then
    printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=manual_recovery_required kind=%s path=%s reason=identity_changed\n' \
      "$kind" "$path" >&2
    return 1
  fi
  rm -- "$path" || {
    printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=manual_recovery_required kind=%s path=%s reason=unlink_failed\n' \
      "$kind" "$path" >&2
    return 1
  }
  sync -f "$(dirname -- "$path")" || {
    printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=manual_recovery_required kind=%s path=%s reason=unlink_sync_failed\n' \
      "$kind" "$path" >&2
    return 1
  }
  printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=recovered kind=%s path=%s\n' \
    "$kind" "$path" >&2
}

cleanup_partial_publication() {
  local cleanup_status=0
  if ((PUBLISHED_MANIFEST_PENDING == 1 && PUBLISHED_MANIFEST_DURABLE == 1)) &&
    path_exists "$OUTPUT_MANIFEST" &&
    [[ "$(path_identity "$OUTPUT_MANIFEST" 2>/dev/null || true)" = "$PUBLISHED_MANIFEST_IDENTITY" ]]; then
    PUBLISHED_MANIFEST_PENDING=0
    PUBLISHED_MANIFEST_DURABLE=0
    PUBLISHED_IMAGE_PENDING=0
    printf 'ORQUESTA_TEST_GUEST_BUILD_PARTIAL_V0 state=committed manifest=%s\n' \
      "$OUTPUT_MANIFEST" >&2
  else
    if ((PUBLISHED_MANIFEST_PENDING == 1)); then
      remove_owned_publication_path \
        "$OUTPUT_MANIFEST" "$PUBLISHED_MANIFEST_IDENTITY" manifest || cleanup_status=1
      PUBLISHED_MANIFEST_PENDING=0
      PUBLISHED_MANIFEST_DURABLE=0
    fi
    if ((PUBLISHED_IMAGE_PENDING == 1)); then
      remove_owned_publication_path \
        "$OUTPUT_IMAGE" "$PUBLISHED_IMAGE_IDENTITY" image || cleanup_status=1
      PUBLISHED_IMAGE_PENDING=0
    fi
  fi
  if [[ -n "$PUBLISH_MANIFEST_STAGE" ]] && path_exists "$PUBLISH_MANIFEST_STAGE"; then
    remove_owned_publication_path \
      "$PUBLISH_MANIFEST_STAGE" "$PUBLISH_MANIFEST_STAGE_IDENTITY" manifest_stage ||
      cleanup_status=1
  fi
  if [[ -n "$PUBLISH_IMAGE_STAGE" ]] && path_exists "$PUBLISH_IMAGE_STAGE"; then
    remove_owned_publication_path \
      "$PUBLISH_IMAGE_STAGE" "$PUBLISH_IMAGE_STAGE_IDENTITY" image_stage ||
      cleanup_status=1
  fi
  PUBLISH_MANIFEST_STAGE=""
  PUBLISH_MANIFEST_STAGE_IDENTITY=""
  PUBLISH_IMAGE_STAGE=""
  PUBLISH_IMAGE_STAGE_IDENTITY=""
  return "$cleanup_status"
}

compile_runner_once() {
  local label="$1"
  local runner="$2"
  local build_root="$RUN_DIR/compile-$label"
  local build_status=0
  mkdir -m 0700 "$build_root" "$build_root/tmp" "$build_root/go-cache" \
    "$build_root/go-mod" "$build_root/go-path"
  (
    cd "$SOURCE_EXPORT_ROOT"
    verify_toolchain_pin || exit 1
    env -i \
      PATH="$TOOLCHAIN_PINNED_ROOT/bin:$PATH_SAFE" GOROOT="$TOOLCHAIN_PINNED_ROOT" \
      LC_ALL=C TZ=UTC SOURCE_DATE_EPOCH=0 \
      TMPDIR="$build_root/tmp" GOCACHE="$build_root/go-cache" GOMODCACHE="$build_root/go-mod" \
      GOPATH="$build_root/go-path" GOENV=off GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      "$TOOLCHAIN_PINNED_ROOT/bin/go" build -mod=vendor -trimpath -buildvcs=false \
      -o "$runner" ./cmd/orquesta/test-guest || build_status=$?
    verify_toolchain_pin || exit 1
    ((build_status == 0)) ||
      fail "runner_build_failed" "inspect_pinned_toolchain_and_exact_source" || exit 1
  )
  chmod 500 "$runner"
  validate_static_elf "$runner" runner
}

compile_runner() {
  local first="$RUN_DIR/orquesta-test-guest.first"
  local second="$RUN_DIR/orquesta-test-guest.second"
  compile_runner_once first "$first"
  compile_runner_once second "$second"
  cmp -s -- "$first" "$second" ||
    fail "runner_build_not_reproducible" "inspect_isolated_double_compilation" || return
  cp -- "$first" "$RUN_DIR/orquesta-test-guest"
  chmod 500 "$RUN_DIR/orquesta-test-guest"
}

sha256_file() {
  env -i PATH="$PATH_SAFE" LC_ALL=C sha256sum -- "$1" | cut -d' ' -f1
}

sha256_tree() {
  local root="$1"
  env -i PATH="$PATH_SAFE" LC_ALL=C \
    tar --sort=name --format=gnu --mtime=@0 --owner=0 --group=0 --numeric-owner \
    -cf - -C "$root" . |
    env -i PATH="$PATH_SAFE" LC_ALL=C sha256sum | cut -d' ' -f1
}

normalize_read_only_tree_modes() {
  local root="$1"
  local file
  while IFS= read -r -d '' file; do
    if [[ -x "$file" ]]; then
      chmod 0555 -- "$file"
    else
      chmod 0444 -- "$file"
    fi
  done < <(find "$root" -xdev -type f -print0)
  find "$root" -xdev -depth -type d -exec chmod 0555 -- {} +
}

verify_packaged_toolchain() {
  local root="$1"
  local entry mode
  validate_toolchain "$root" || return
  while IFS= read -r -d '' entry; do
    mode="$(stat -c '%a' -- "$entry")" ||
      fail "toolchain_mode_probe_failed" "inspect_packaged_toolchain" || return
    if [[ -d "$entry" ]]; then
      [[ "$mode" = "555" ]] ||
        fail "toolchain_directory_mode_invalid" "normalize_toolchain_directories_0555" || return
    elif [[ "$mode" != "444" && "$mode" != "555" ]]; then
      fail "toolchain_file_mode_invalid" "normalize_toolchain_files_0444_or_0555"
      return
    fi
  done < <(find "$root" -xdev \( -type d -o -type f \) -print0)
  if [[ -n "$(find "$root" -xdev \( -type d -o -type f \) -perm /0222 -print -quit)" ]]; then
    fail "toolchain_writable_entry" "remove_write_bits_from_packaged_toolchain"
    return
  fi
  run_nobody_go_test_probe "$root" "$RUN_DIR/toolchain-nobody-go-test"
}

run_nobody_go_test_probe() {
  local root="$1"
  local probe="$2"
  local source scratch
  source="$probe/source"
  scratch="$probe/scratch"
  mkdir -p "$source" "$scratch/gocache" "$scratch/gopath" \
    "$scratch/gotmp" "$scratch/home" "$scratch/tmp"
  chmod 0700 "$probe" "$scratch" "$scratch/gocache" "$scratch/gopath" \
    "$scratch/gotmp" "$scratch/home" "$scratch/tmp"
  cat >"$source/go.mod" <<'EOF'
module example.invalid/orquesta-toolchain-probe

go 1.23
EOF
  cat >"$source/probe_test.go" <<'EOF'
package probe

import "testing"

func TestPackagedToolchain(t *testing.T) {}
EOF
  chmod 0444 "$source/go.mod" "$source/probe_test.go"
  chmod 0555 "$source"
  env -i PATH="$PATH_SAFE" LC_ALL=C bwrap \
    --unshare-user --unshare-net --uid 65534 --gid 65534 --die-with-parent \
    --clearenv --setenv GOROOT /toolchain --setenv PATH /toolchain/bin \
    --setenv CGO_ENABLED 0 --setenv GOENV off --setenv GONOSUMDB '*' \
    --setenv GOPROXY off --setenv GOSUMDB off --setenv GOTOOLCHAIN local --setenv GOVCS off \
    --setenv GOCACHE /scratch/gocache --setenv GOPATH /scratch/gopath \
    --setenv GOTMPDIR /scratch/gotmp --setenv HOME /scratch/home --setenv TMPDIR /scratch/tmp \
    --ro-bind "$root" /toolchain --ro-bind "$source" /src --bind "$scratch" /scratch \
    --proc /proc --dev /dev --chdir /src \
    /toolchain/bin/go test -count=1 ./... >/dev/null ||
    fail "toolchain_nobody_go_test_failed" "repair_packaged_toolchain_or_scratch_permissions"
}

normalize_mtimes() {
  local root="$1"
  (
    cd "$root"
    env -i PATH="$PATH_SAFE" LC_ALL=C find . -print0 |
      env -i PATH="$PATH_SAFE" LC_ALL=C xargs -0 touch -h -d @0 --
  )
}

assemble_image() {
  local runner="$RUN_DIR/orquesta-test-guest"
  local rootfs="$RUN_DIR/rootfs"
  local image="$RUN_DIR/orquesta-test-guest.cpio.gz"
  mkdir -m 700 "$rootfs"
  mkdir "$rootfs/bin" "$rootfs/dev" "$rootfs/proc" "$rootfs/run" "$rootfs/sys" "$rootfs/toolchain"
  cp -- "$runner" "$rootfs/orquesta-test-guest"
  verify_busybox_pin || return
  cp -- "$BUSYBOX_PINNED_PATH" "$rootfs/bin/busybox"
  verify_busybox_pin || return
  verify_toolchain_pin || return
  cp -a -x -- "$TOOLCHAIN_PINNED_ROOT/." "$rootfs/toolchain/"
  verify_toolchain_pin || return
  normalize_read_only_tree_modes "$rootfs/toolchain"
  env -i PATH="$PATH_SAFE" LC_ALL=C diff --recursive --brief --no-dereference \
    "$TOOLCHAIN_PINNED_ROOT/." "$rootfs/toolchain" >/dev/null ||
    fail "toolchain_copy_mismatch" "inspect_complete_toolchain_copy" || return
  verify_toolchain_pin || return
  verify_packaged_toolchain "$rootfs/toolchain" || return
  chmod 0555 "$rootfs/orquesta-test-guest" "$rootfs/bin/busybox"
  cat >"$rootfs/init" <<'EOF'
#!/bin/busybox sh
set -eu
umask 077
/bin/busybox mount -t devtmpfs -o mode=0755,nosuid devtmpfs /dev
/bin/busybox mount -t proc proc /proc
/bin/busybox mount -t sysfs sysfs /sys
exec /orquesta-test-guest
EOF
  chmod 0555 "$rootfs/init"
  find "$rootfs" -xdev -depth -type d -exec chmod 0555 -- {} +
  normalize_mtimes "$rootfs"
  local packaged_tree
  packaged_tree="$(sha256_tree "$rootfs/toolchain")"
  if [[ -n "$(find "$rootfs" -xdev ! -type f ! -type d ! -type l -print -quit)" ||
    -n "$(find "$rootfs" -xdev \( -type d -o -type f \) -perm /0222 -print -quit)" ]]; then
    fail "rootfs_mutable_or_special" "inspect_immutable_guest_rootfs"
    return
  fi
  (
    cd "$rootfs"
    env -i PATH="$PATH_SAFE" LC_ALL=C find . -print0 |
      env -i PATH="$PATH_SAFE" LC_ALL=C sort -z |
      env -i PATH="$PATH_SAFE" LC_ALL=C \
        cpio --null --quiet --reproducible -o --format=newc --owner=0:0
  ) | env -i PATH="$PATH_SAFE" LC_ALL=C gzip -n -9 >"$image"

  local runner_sha busybox_sha image_sha unpacked_bytes
  local minimum_required_bytes minimum_required_mib minimum_guest_memory_mib
  verify_busybox_pin || return
  runner_sha="$(sha256_file "$runner")"
  busybox_sha="$(sha256_file "$rootfs/bin/busybox")"
  image_sha="$(sha256_file "$image")"
  unpacked_bytes="$(gzip -dc "$image" | wc -c)"
  [[ "$unpacked_bytes" =~ ^[1-9][0-9]*$ && "$unpacked_bytes" -le 9223372036518182912 ]] ||
    fail "unpacked_size_invalid" "inspect_generated_initramfs" || return
  minimum_required_bytes=$((unpacked_bytes +
    SCRATCH_FIXED_RESERVE_BYTES + SCRATCH_CACHE_RESERVE_BYTES))
  minimum_required_mib=$(((minimum_required_bytes + MEMORY_MIB_BYTES - 1) /
    MEMORY_MIB_BYTES))
  # ceil(ceil(required bytes / MiB) / 0.75), avoiding fractional arithmetic.
  minimum_guest_memory_mib=$(((minimum_required_mib * 100 +
    SCRATCH_TMPFS_PERCENT - 1) / SCRATCH_TMPFS_PERCENT))
  ((minimum_guest_memory_mib > 0)) ||
    fail "minimum_guest_memory_invalid" "inspect_generated_initramfs" || return
  cat >"$RUN_DIR/manifest.json" <<EOF
{"schema_version":"orquesta_test_attestor_guest.v0","platform":"linux/amd64","source_commit":"$SOURCE_COMMIT","runner_sha256":"sha256:$runner_sha","busybox_sha256":"sha256:$busybox_sha","busybox_version":"$BUSYBOX_VERSION","toolchain_tree_sha256":"sha256:$packaged_tree","toolchain_version":"$TOOLCHAIN_VERSION","image_sha256":"sha256:$image_sha","unpacked_bytes":$unpacked_bytes,"minimum_guest_memory_mib":$minimum_guest_memory_mib,"memory_contract":{"scratch_fixed_reserve_bytes":$SCRATCH_FIXED_RESERVE_BYTES,"scratch_cache_reserve_bytes":$SCRATCH_CACHE_RESERVE_BYTES,"tmpfs_percent":$SCRATCH_TMPFS_PERCENT,"kernel_runtime_headroom_percent":$KERNEL_RUNTIME_HEADROOM_PERCENT,"formula":"ceil(ceil((unpacked_bytes+scratch_fixed_reserve_bytes+scratch_cache_reserve_bytes)/MiB)*100/tmpfs_percent)"},"build":{"cgo_enabled":false,"trimpath":true,"buildvcs":false,"runner_double_build":true,"source":"exact_commit_private_export","archive":"newc","owner":"0:0","mtime_epoch":0,"gzip_name_time":false,"toolchain_directories":"0555","toolchain_executables":"0555","toolchain_data":"0444","toolchain_symlinks":"relative_internal","toolchain_nobody_go_test":true}}
EOF
  chmod 400 "$image" "$RUN_DIR/manifest.json"
}

publish_outputs() {
  local image_source="$RUN_DIR/orquesta-test-guest.cpio.gz"
  local manifest_source="$RUN_DIR/manifest.json"
  local image_parent manifest_parent
  image_parent="$(dirname -- "$OUTPUT_IMAGE")"
  manifest_parent="$(dirname -- "$OUTPUT_MANIFEST")"
  validate_output_pair_state "$OUTPUT_IMAGE" "$OUTPUT_MANIFEST" || return

  PUBLISH_IMAGE_STAGE="$(mktemp \
    "$image_parent/.$(basename -- "$OUTPUT_IMAGE").orquesta-stage.XXXXXX")" ||
    fail "image_stage_create_failed" "repair_image_destination" || return
  PUBLISH_IMAGE_STAGE_IDENTITY="$(path_identity "$PUBLISH_IMAGE_STAGE")" ||
    fail "image_stage_identity_failed" "repair_image_destination" || return
  cat -- "$image_source" >"$PUBLISH_IMAGE_STAGE" ||
    fail "image_stage_write_failed" "repair_image_destination" || return
  chmod 0444 "$PUBLISH_IMAGE_STAGE" ||
    fail "image_stage_mode_failed" "repair_image_destination" || return
  sync -f "$PUBLISH_IMAGE_STAGE" ||
    fail "image_stage_sync_failed" "repair_image_destination" || return
  [[ "$(path_identity "$PUBLISH_IMAGE_STAGE")" = "$PUBLISH_IMAGE_STAGE_IDENTITY" ]] ||
    fail "image_stage_identity_changed" "inspect_image_destination_race" || return
  PUBLISHED_IMAGE_IDENTITY="$PUBLISH_IMAGE_STAGE_IDENTITY"
  PUBLISHED_IMAGE_PENDING=1
  if ! ln -- "$PUBLISH_IMAGE_STAGE" "$OUTPUT_IMAGE"; then
    PUBLISHED_IMAGE_PENDING=0
    fail "output_no_clobber" "preserve_existing_output_and_choose_new_paths"
    return
  fi
  sync -f "$image_parent" ||
    fail "image_publish_sync_failed" "inspect_partial_publication" || return
  rm -- "$PUBLISH_IMAGE_STAGE" ||
    fail "image_stage_cleanup_failed" "inspect_partial_publication" || return
  PUBLISH_IMAGE_STAGE=""
  PUBLISH_IMAGE_STAGE_IDENTITY=""
  sync -f "$image_parent" ||
    fail "image_stage_unlink_sync_failed" "inspect_partial_publication" || return

  PUBLISH_MANIFEST_STAGE="$(mktemp \
    "$manifest_parent/.$(basename -- "$OUTPUT_MANIFEST").orquesta-stage.XXXXXX")" ||
    fail "manifest_stage_create_failed" "repair_manifest_destination" || return
  PUBLISH_MANIFEST_STAGE_IDENTITY="$(path_identity "$PUBLISH_MANIFEST_STAGE")" ||
    fail "manifest_stage_identity_failed" "repair_manifest_destination" || return
  cat -- "$manifest_source" >"$PUBLISH_MANIFEST_STAGE" ||
    fail "manifest_stage_write_failed" "repair_manifest_destination" || return
  chmod 0444 "$PUBLISH_MANIFEST_STAGE" ||
    fail "manifest_stage_mode_failed" "repair_manifest_destination" || return
  sync -f "$PUBLISH_MANIFEST_STAGE" ||
    fail "manifest_stage_sync_failed" "repair_manifest_destination" || return
  [[ "$(path_identity "$PUBLISH_MANIFEST_STAGE")" = "$PUBLISH_MANIFEST_STAGE_IDENTITY" ]] ||
    fail "manifest_stage_identity_changed" "inspect_manifest_destination_race" || return
  PUBLISHED_MANIFEST_IDENTITY="$PUBLISH_MANIFEST_STAGE_IDENTITY"
  PUBLISHED_MANIFEST_PENDING=1
  if ! ln -- "$PUBLISH_MANIFEST_STAGE" "$OUTPUT_MANIFEST"; then
    PUBLISHED_MANIFEST_PENDING=0
    fail "manifest_no_clobber" "preserve_existing_manifest_and_inspect_partial_publication"
    return
  fi
  sync -f "$manifest_parent" ||
    fail "manifest_publish_sync_failed" "inspect_committed_publication" || return
  PUBLISHED_MANIFEST_DURABLE=1
  PUBLISHED_IMAGE_PENDING=0
  rm -- "$PUBLISH_MANIFEST_STAGE" ||
    fail "manifest_stage_cleanup_failed" "remove_only_the_owned_manifest_stage" || return
  PUBLISH_MANIFEST_STAGE=""
  PUBLISH_MANIFEST_STAGE_IDENTITY=""
  sync -f "$manifest_parent" ||
    fail "manifest_stage_unlink_sync_failed" "inspect_committed_publication" || return
  PUBLISHED_MANIFEST_PENDING=0
  PUBLISHED_MANIFEST_DURABLE=0
}

main() {
  parse_args "$@"
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  for tool in bwrap cmp cp diff file findmnt git readelf find sort stat wc xargs touch tar cpio gzip sha256sum cut ln mktemp sync; do
    command -v "$tool" >/dev/null ||
      fail "host_tool_missing" "install_local_${tool}" || exit 1
  done
  cpio --help | grep -q -- '--reproducible' ||
    fail "cpio_not_reproducible" "install_cpio_with_--reproducible" || exit 1
  prepare_inputs || exit 1
  prepare_run_dir
  SOURCE_EXPORT_ROOT="$RUN_DIR/source"
  export_source_commit "$SOURCE_ROOT" "$SOURCE_COMMIT" "$SOURCE_EXPORT_ROOT"
  compile_runner
  assemble_image
  publish_outputs
  local image_sha
  image_sha="$(sha256_file "$OUTPUT_IMAGE")"
  printf 'ORQUESTA_TEST_GUEST_BUILD_V0 status=pass image_sha256=sha256:%s source_commit=%s network=absent ssh=absent cloud_init=absent\n' \
    "$image_sha" "$SOURCE_COMMIT"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
