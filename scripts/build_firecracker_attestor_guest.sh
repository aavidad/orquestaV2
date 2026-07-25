#!/usr/bin/env bash
set -euo pipefail

# Reproducible initramfs builder for the TestAttestor guest only. This does not
# build or run agents and does not discover/download host inputs.

PATH_SAFE="/usr/sbin:/usr/bin:/sbin:/bin"
MARKER_SCHEMA="orquesta_test_guest_build_tmp.v0"

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

usage() {
  cat <<'EOF'
Usage:
  build_firecracker_attestor_guest.sh --confirm-real \
    --source-root ABS_PATH --source-commit HEX \
    --toolchain-root ABS_PATH --busybox ABS_PATH \
    --output ABS_PATH --manifest ABS_PATH --tmp-parent ABS_PATH [--keep-temp]

Builds a deterministic TestAttestor initramfs. It compiles
./cmd/orquesta-test-guest with the supplied local Go toolchain, then packages
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
  [[ ! -e "$requested" ]] ||
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
  [[ -x "$root/bin/go" && -d "$root/pkg" && -d "$root/src" && -d "$root/lib" ]] ||
    fail "toolchain_incomplete" "provide_complete_local_go_toolchain" || return
  if [[ -n "$(find "$root" -xdev ! -type f ! -type d ! -type l -print -quit)" ]]; then
    fail "toolchain_special_file" "remove_special_files_from_toolchain"
    return
  fi
  env -i PATH="$PATH_SAFE" LC_ALL=C "$root/bin/go" version >/dev/null ||
    fail "toolchain_go_unusable" "repair_local_go_toolchain"
}

prepare_inputs() {
  SOURCE_ROOT="$(canonical_existing "$SOURCE_ROOT" source_root)" || return
  TOOLCHAIN_ROOT="$(canonical_existing "$TOOLCHAIN_ROOT" toolchain_root)" || return
  BUSYBOX_BIN="$(canonical_existing "$BUSYBOX_BIN" busybox)" || return
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
  [[ -d "$SOURCE_ROOT/cmd/orquesta-test-guest" ]] ||
    fail "guest_command_missing" "finish_cmd_orquesta-test-guest_before_real_build" || return
  validate_toolchain "$TOOLCHAIN_ROOT" || return
  validate_static_elf "$BUSYBOX_BIN" busybox
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
  rm -rf -- "$target"
}

cleanup() {
  local saved=$?
  trap - EXIT INT TERM
  if [[ -n "$RUN_DIR" && -d "$RUN_DIR" ]]; then
    if ((KEEP_TEMP == 1)); then
      printf 'ORQUESTA_TEST_GUEST_BUILD_TEMP_V0 retained=true path=%s\n' "$RUN_DIR" >&2
    else
      remove_run_dir "$RUN_DIR" || saved=1
    fi
  fi
  exit "$saved"
}

compile_runner() {
  local runner="$RUN_DIR/orquesta-test-guest"
  mkdir -m 700 "$RUN_DIR/tmp" "$RUN_DIR/go-cache" "$RUN_DIR/go-mod" "$RUN_DIR/go-path"
  (
    cd "$SOURCE_ROOT"
    env -i \
      PATH="$TOOLCHAIN_ROOT/bin:$PATH_SAFE" \
      LC_ALL=C TZ=UTC SOURCE_DATE_EPOCH=0 \
      TMPDIR="$RUN_DIR/tmp" GOCACHE="$RUN_DIR/go-cache" GOMODCACHE="$RUN_DIR/go-mod" \
      GOPATH="$RUN_DIR/go-path" GOENV=off GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      "$TOOLCHAIN_ROOT/bin/go" build -mod=vendor -trimpath -buildvcs=false \
      -o "$runner" ./cmd/orquesta-test-guest
  )
  chmod 500 "$runner"
  validate_static_elf "$runner" runner
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
  mkdir "$rootfs/bin" "$rootfs/dev" "$rootfs/proc" "$rootfs/sys" "$rootfs/tmp" "$rootfs/toolchain"
  cp -- "$runner" "$rootfs/orquesta-test-guest"
  cp -- "$BUSYBOX_BIN" "$rootfs/bin/busybox"
  cp -a -- "$TOOLCHAIN_ROOT/." "$rootfs/toolchain/"
  chmod 500 "$rootfs/orquesta-test-guest" "$rootfs/bin/busybox"
  chmod 700 "$rootfs/tmp"
  cat >"$rootfs/init" <<'EOF'
#!/bin/busybox sh
set -eu
umask 077
export PATH=/toolchain/bin:/bin
export GOROOT=/toolchain
export TMPDIR=/tmp
export GOCACHE=/tmp/go-cache
export GOPATH=/tmp/go-path
export GOENV=off
export GOPROXY=off
export GOSUMDB=off
export GOTOOLCHAIN=local
export CGO_ENABLED=0
/bin/busybox mkdir -p /tmp/go-cache /tmp/go-path
/bin/busybox mount -t proc proc /proc
/bin/busybox mount -t sysfs sysfs /sys
exec /orquesta-test-guest
EOF
  chmod 500 "$rootfs/init"
  normalize_mtimes "$rootfs"
  local source_tree packaged_tree
  source_tree="$(sha256_tree "$TOOLCHAIN_ROOT")"
  packaged_tree="$(sha256_tree "$rootfs/toolchain")"
  [[ "$source_tree" = "$packaged_tree" ]] ||
    fail "toolchain_copy_mismatch" "inspect_complete_toolchain_copy" || return
  (
    cd "$rootfs"
    env -i PATH="$PATH_SAFE" LC_ALL=C find . -print0 |
      env -i PATH="$PATH_SAFE" LC_ALL=C sort -z |
      env -i PATH="$PATH_SAFE" LC_ALL=C \
        cpio --null --quiet --reproducible -o --format=newc --owner=0:0
  ) | env -i PATH="$PATH_SAFE" LC_ALL=C gzip -n -9 >"$image"

  local runner_sha busybox_sha image_sha
  runner_sha="$(sha256_file "$runner")"
  busybox_sha="$(sha256_file "$BUSYBOX_BIN")"
  image_sha="$(sha256_file "$image")"
  cat >"$RUN_DIR/manifest.json" <<EOF
{"schema_version":"orquesta_test_attestor_guest.v0","platform":"linux/amd64","source_commit":"$SOURCE_COMMIT","runner_sha256":"sha256:$runner_sha","busybox_sha256":"sha256:$busybox_sha","toolchain_tree_sha256":"sha256:$source_tree","image_sha256":"sha256:$image_sha","build":{"cgo_enabled":false,"trimpath":true,"buildvcs":false,"archive":"newc","owner":"0:0","mtime_epoch":0,"gzip_name_time":false}}
EOF
  chmod 400 "$image" "$RUN_DIR/manifest.json"
}

publish_outputs() {
  [[ ! -e "$OUTPUT_IMAGE" && ! -e "$OUTPUT_MANIFEST" ]] ||
    fail "output_race_detected" "choose_new_output_paths" || return
  install -m 0444 "$RUN_DIR/orquesta-test-guest.cpio.gz" "$OUTPUT_IMAGE"
  if ! install -m 0444 "$RUN_DIR/manifest.json" "$OUTPUT_MANIFEST"; then
    rm -- "$OUTPUT_IMAGE"
    fail "manifest_publish_failed" "repair_manifest_destination_and_rebuild"
    return
  fi
}

main() {
  parse_args "$@"
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  for tool in file readelf find sort xargs touch tar cpio gzip sha256sum cut install; do
    command -v "$tool" >/dev/null ||
      fail "host_tool_missing" "install_local_${tool}" || exit 1
  done
  cpio --help | grep -q -- '--reproducible' ||
    fail "cpio_not_reproducible" "install_cpio_with_--reproducible" || exit 1
  prepare_inputs || exit 1
  prepare_run_dir
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
