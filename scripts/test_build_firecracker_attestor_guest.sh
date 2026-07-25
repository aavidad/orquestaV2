#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILDER="$ROOT/scripts/build_firecracker_attestor_guest.sh"
TEST_ROOT="$(mktemp -d /tmp/orquesta-test-guest-builder-test.XXXXXX)"
trap 'rm -rf -- "$TEST_ROOT"' EXIT

bash -n "$BUILDER"
bash -n "$0"
# shellcheck source=scripts/build_firecracker_attestor_guest.sh
source "$BUILDER"

assert_failure_contains() {
  local expected="$1"
  shift
  local output status=0
  output="$("$@" 2>&1)" || status=$?
  [[ "$status" != "0" && "$output" = *"$expected"* ]] || {
    printf 'guest_builder_test_failed expected=%s status=%s output=%s\n' \
      "$expected" "$status" "$output" >&2
    exit 1
  }
}

make_fake_toolchain() {
  local root="$1"
  mkdir -p "$root/bin" "$root/pkg/tool/linux_amd64" "$root/src/runtime" "$root/lib/time"
  cat >"$root/bin/go" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" = "version" ]]; then
  printf 'go version go-fake linux/amd64\n'
  exit 0
fi
exit 99
EOF
  chmod 500 "$root/bin/go"
  printf 'fake-tool\n' >"$root/pkg/tool/linux_amd64/compile"
  printf 'package runtime\n' >"$root/src/runtime/runtime.go"
  printf 'UTC\n' >"$root/lib/time/zoneinfo.zip"
  ln -s runtime.go "$root/src/runtime/alias.go"
}

fake_build() {
  local destination="$1"
  TMP_PARENT="$TEST_ROOT"
  KEEP_TEMP=0
  BUSYBOX_BIN="/usr/bin/busybox"
  TOOLCHAIN_ROOT="$TEST_ROOT/toolchain"
  SOURCE_COMMIT="0123456789abcdef0123456789abcdef01234567"
  OUTPUT_IMAGE="$destination/guest.cpio.gz"
  OUTPUT_MANIFEST="$destination/manifest.json"
  mkdir "$destination"
  prepare_run_dir
  cp -- /usr/bin/busybox "$RUN_DIR/orquesta-test-guest"
  chmod 500 "$RUN_DIR/orquesta-test-guest"
  validate_static_elf "$RUN_DIR/orquesta-test-guest" runner
  assemble_image
  publish_outputs
  remove_run_dir "$RUN_DIR"
  RUN_DIR=""
  RUN_MARKER=""
}

test_guards() {
  assert_failure_contains 'code=real_confirmation_required' "$BUILDER"
  assert_failure_contains 'code=runner_not_static_amd64' validate_static_elf /bin/sh runner
  local unmarked="$TEST_ROOT/orquesta-test-guest-build.unmarked"
  mkdir "$unmarked"
  assert_failure_contains 'code=cleanup_target_unowned' remove_run_dir "$unmarked"
  [[ -d "$unmarked" ]] || {
    printf 'guest_builder_test_failed unmarked_tmp_removed\n' >&2
    exit 1
  }
}

test_reproducible_fake_build() {
  make_fake_toolchain "$TEST_ROOT/toolchain"
  validate_toolchain "$TEST_ROOT/toolchain"
  validate_static_elf /usr/bin/busybox busybox
  fake_build "$TEST_ROOT/first"
  fake_build "$TEST_ROOT/second"
  cmp "$TEST_ROOT/first/guest.cpio.gz" "$TEST_ROOT/second/guest.cpio.gz"
  cmp "$TEST_ROOT/first/manifest.json" "$TEST_ROOT/second/manifest.json"

  listing="$(gzip -dc "$TEST_ROOT/first/guest.cpio.gz" | cpio --quiet -it)"
  for expected in init orquesta-test-guest bin/busybox toolchain/bin/go toolchain/src/runtime/runtime.go; do
    grep -Fqx "$expected" <<<"$listing" || {
      printf 'guest_builder_test_failed missing_archive_entry=%s\n' "$expected" >&2
      exit 1
    }
  done
  init="$(gzip -dc "$TEST_ROOT/first/guest.cpio.gz" | cpio --quiet -i --to-stdout init)"
  grep -Fqx 'exec /orquesta-test-guest' <<<"$init"
  grep -Fqx 'export GOROOT=/toolchain' <<<"$init"
  grep -Fqx 'export GOPROXY=off' <<<"$init"
  if grep -Eqi 'ssh|cloud-init|wget|curl' <<<"$init"; then
    printf 'guest_builder_test_failed forbidden_init_surface\n' >&2
    exit 1
  fi

  image_sha="$(sha256sum "$TEST_ROOT/first/guest.cpio.gz" | cut -d' ' -f1)"
  grep -Fq "\"image_sha256\":\"sha256:$image_sha\"" "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"source_commit\":\"$SOURCE_COMMIT\"" "$TEST_ROOT/first/manifest.json"
}

test_guards
test_reproducible_fake_build
printf 'ORQUESTA_TEST_GUEST_BUILDER_TEST_V0 status=pass mode=fake\n'
