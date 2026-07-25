#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILDER="$ROOT/scripts/build_firecracker_attestor_guest.sh"
TEST_ROOT="$(mktemp -d /tmp/orquesta-test-guest-builder-test.XXXXXX)"
EXACT_TEST_COMMIT=""
cleanup_test_root() {
  find "$TEST_ROOT" -xdev -type d -exec chmod 0700 -- {} + 2>/dev/null || true
  rm -rf -- "$TEST_ROOT"
}
trap cleanup_test_root EXIT

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
  local source="$TEST_ROOT/fake-go.go"
  mkdir -p "$root/bin" "$root/pkg/tool/linux_amd64" "$root/src/runtime" "$root/lib/time"
  find "$root" -type d -exec chmod 0777 -- {} +
  cat >"$source" <<'EOF'
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Println("go version go-fake linux/amd64")
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "test" {
		if os.Getenv("GOROOT") != "/toolchain" || os.Getenv("GOPROXY") != "off" {
			os.Exit(97)
		}
		for _, key := range []string{"GOCACHE", "GOPATH", "GOTMPDIR", "HOME", "TMPDIR"} {
			file, err := os.CreateTemp(os.Getenv(key), "write-probe-")
			if err != nil {
				os.Exit(98)
			}
			_ = file.Close()
		}
		info, err := os.Stat("/src/probe_test.go")
		if err != nil || info.Mode().Perm()&0222 != 0 {
			os.Exit(96)
		}
		return
	}
	os.Exit(99)
}
EOF
  env CGO_ENABLED=0 go build -trimpath -buildvcs=false -o "$root/bin/go" "$source"
  chmod 0777 "$root/bin/go"
  printf 'fake-tool\n' >"$root/pkg/tool/linux_amd64/compile"
  chmod 0755 "$root/pkg/tool/linux_amd64/compile"
  printf 'package runtime\n' >"$root/src/runtime/runtime.go"
  chmod 0666 "$root/src/runtime/runtime.go"
  printf 'UTC\n' >"$root/lib/time/zoneinfo.zip"
  ln -s runtime.go "$root/src/runtime/alias.go"
}

fake_build() {
  local destination="$1"
  TMP_PARENT="$TEST_ROOT"
  KEEP_TEMP=0
  BUSYBOX_BIN="/usr/bin/busybox"
  TOOLCHAIN_ROOT="$TEST_ROOT/toolchain"
  SOURCE_COMMIT="$EXACT_TEST_COMMIT"
  OUTPUT_IMAGE="$destination/guest.cpio.gz"
  OUTPUT_MANIFEST="$destination/manifest.json"
  mkdir "$destination"
  pin_toolchain_source "$TOOLCHAIN_ROOT" 0
  probe_pinned_toolchain
  pin_busybox_source "$BUSYBOX_BIN" 1
  probe_pinned_busybox
  prepare_run_dir
  cp -- "$TEST_ROOT/double-compiled-runner" "$RUN_DIR/orquesta-test-guest"
  chmod 500 "$RUN_DIR/orquesta-test-guest"
  validate_static_elf "$RUN_DIR/orquesta-test-guest" runner
  assemble_image
  publish_outputs
  remove_run_dir "$RUN_DIR"
  release_toolchain_pin
  release_busybox_pin
  cleanup_partial_publication
  RUN_DIR=""
  RUN_MARKER=""
}

test_guards() {
  assert_failure_contains 'code=real_confirmation_required' "$BUILDER"
  assert_failure_contains 'code=runner_not_static_amd64' validate_static_elf /bin/sh runner
  assert_failure_contains 'code=toolchain_not_root_owned' \
    pin_toolchain_source "$TEST_ROOT/toolchain" 1
  local untrusted="$TEST_ROOT/toolchain-untrusted" execution_marker="$TEST_ROOT/untrusted-go-ran"
  cp -a "$TEST_ROOT/toolchain" "$untrusted"
  cat >"$untrusted/bin/go" <<EOF
#!/bin/sh
: >"$execution_marker"
printf 'go version malicious linux/amd64\\n'
EOF
  chmod 0755 "$untrusted/bin/go"
  assert_failure_contains 'code=toolchain_not_root_owned' \
    pin_toolchain_source "$untrusted" 1
  [[ ! -e "$execution_marker" ]] || {
    printf 'guest_builder_test_failed untrusted_go_executed_before_pin\n' >&2
    exit 1
  }
  local unmarked="$TEST_ROOT/orquesta-test-guest-build.unmarked"
  mkdir "$unmarked"
  assert_failure_contains 'code=cleanup_target_unowned' remove_run_dir "$unmarked"
  [[ -d "$unmarked" ]] || {
    printf 'guest_builder_test_failed unmarked_tmp_removed\n' >&2
    exit 1
  }

  local absolute="$TEST_ROOT/toolchain-absolute"
  cp -a "$TEST_ROOT/toolchain" "$absolute"
  ln -s /etc/passwd "$absolute/src/runtime/absolute"
  assert_failure_contains 'code=toolchain_symlink_not_relative' validate_toolchain "$absolute"

  local outside="$TEST_ROOT/outside" escaping="$TEST_ROOT/toolchain-escaping"
  printf 'outside\n' >"$outside"
  cp -a "$TEST_ROOT/toolchain" "$escaping"
  ln -s ../../../outside "$escaping/src/runtime/escaping"
  assert_failure_contains 'code=toolchain_symlink_escape' validate_toolchain "$escaping"

  local special="$TEST_ROOT/toolchain-special"
  cp -a "$TEST_ROOT/toolchain" "$special"
  mkfifo "$special/src/runtime/fifo"
  assert_failure_contains 'code=toolchain_special_file' validate_toolchain "$special"

  local repository="$TEST_ROOT/source-repository" head
  mkdir "$repository"
  git -C "$repository" init --quiet
  git -C "$repository" config user.name guest-builder-test
  git -C "$repository" config user.email guest-builder-test@example.invalid
  printf 'tracked\n' >"$repository/tracked"
  git -C "$repository" add tracked
  git -C "$repository" commit --quiet -m initial
  head="$(git -C "$repository" rev-parse HEAD)"
  EXACT_TEST_COMMIT="$head"
  validate_source_provenance "$repository" "$head"
  assert_failure_contains 'code=source_head_mismatch' \
    validate_source_provenance "$repository" 0000000000000000000000000000000000000000
  printf 'dirty\n' >>"$repository/tracked"
  assert_failure_contains 'code=source_tree_dirty' validate_source_provenance "$repository" "$head"
}

test_exact_commit_export_and_double_compile() {
  local repository="$TEST_ROOT/compile-source"
  local head output mutator_pid
  mkdir -p "$repository/cmd/orquesta-test-guest"
  cat >"$repository/go.mod" <<'EOF'
module example.invalid/orquesta-guest-build-test

go 1.23
EOF
  cat >"$repository/.gitignore" <<'EOF'
ignored.go
EOF
  cat >"$repository/cmd/orquesta-test-guest/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("committed-source")
}
EOF
  git -C "$repository" init --quiet
  git -C "$repository" config user.name guest-builder-test
  git -C "$repository" config user.email guest-builder-test@example.invalid
  git -C "$repository" add .
  git -C "$repository" commit --quiet -m source
  head="$(git -C "$repository" rev-parse HEAD)"
  printf 'package main\nfunc ignored syntax\n' >"$repository/cmd/orquesta-test-guest/ignored.go"
  validate_source_provenance "$repository" "$head"
  cat >"$repository/cmd/orquesta-test-guest/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("moved-ref")
}
EOF
  git -C "$repository" add cmd/orquesta-test-guest/main.go
  git -C "$repository" commit --quiet -m moved-ref
  assert_failure_contains 'code=source_head_mismatch' \
    validate_source_provenance "$repository" "$head"

  TMP_PARENT="$TEST_ROOT"
  SOURCE_ROOT="$repository"
  SOURCE_COMMIT="$head"
  TOOLCHAIN_ROOT="/usr/local/go"
  pin_toolchain_source "$TOOLCHAIN_ROOT" 0
  prepare_run_dir
  SOURCE_EXPORT_ROOT="$RUN_DIR/source"
  export_source_commit "$SOURCE_ROOT" "$SOURCE_COMMIT" "$SOURCE_EXPORT_ROOT"
  [[ ! -e "$SOURCE_EXPORT_ROOT/cmd/orquesta-test-guest/ignored.go" ]] || {
    printf 'guest_builder_test_failed ignored_go_exported\n' >&2
    exit 1
  }
  (
    for _ in {1..100}; do
      printf 'package main\nfunc concurrent mutation\n' \
        >"$repository/cmd/orquesta-test-guest/main.go"
      sleep 0.005
    done
  ) &
  mutator_pid=$!
  compile_runner
  wait "$mutator_pid"
  output="$("$RUN_DIR/orquesta-test-guest")"
  [[ "$output" = "committed-source" ]] || {
    printf 'guest_builder_test_failed worktree_influenced_build output=%s\n' "$output" >&2
    exit 1
  }
  cmp "$RUN_DIR/orquesta-test-guest.first" "$RUN_DIR/orquesta-test-guest.second"
  cp "$RUN_DIR/orquesta-test-guest" "$TEST_ROOT/double-compiled-runner"
  remove_run_dir "$RUN_DIR"
  release_toolchain_pin
  RUN_DIR=""
  RUN_MARKER=""

  local unsafe="$TEST_ROOT/source-unsafe-symlink"
  mkdir "$unsafe"
  git -C "$unsafe" init --quiet
  git -C "$unsafe" config user.name guest-builder-test
  git -C "$unsafe" config user.email guest-builder-test@example.invalid
  ln -s /etc/passwd "$unsafe/absolute"
  git -C "$unsafe" add absolute
  git -C "$unsafe" commit --quiet -m unsafe
  head="$(git -C "$unsafe" rev-parse HEAD)"
  assert_failure_contains 'code=source_export_symlink_not_relative' \
    export_source_commit "$unsafe" "$head" "$TEST_ROOT/unsafe-export"
}

test_submount_guard() {
  local probe="$TEST_ROOT/submount-probe"
  local mounted="$TEST_ROOT/submount-content"
  local output status=0
  mkdir -p "$probe/child" "$mounted"
  # shellcheck disable=SC2016
  output="$(bwrap \
    --unshare-user --unshare-net --uid 0 --gid 0 --die-with-parent \
    --ro-bind / / \
    --bind "$probe" "$probe" \
    --bind "$mounted" "$probe/child" \
    /bin/bash -c 'source "$1"; reject_descendant_mounts "$2"' \
    bash "$BUILDER" "$probe" 2>&1)" || status=$?
  [[ "$status" != "0" && "$output" = *"code=toolchain_submount"* ]] || {
    printf 'guest_builder_test_failed submount_guard status=%s output=%s\n' "$status" "$output" >&2
    exit 1
  }
}

test_busybox_pin_race() {
  local source="$TEST_ROOT/busybox-race"
  local original="$TEST_ROOT/busybox-race.original"
  local copied="$TEST_ROOT/busybox-race.copied"
  local replacer_pid
  cp /usr/bin/busybox "$source"
  BUSYBOX_BIN="$source"
  pin_busybox_source "$BUSYBOX_BIN" 0
  (
    mv "$source" "$original"
    cp /bin/true "$source"
  ) &
  replacer_pid=$!
  wait "$replacer_pid"
  assert_failure_contains 'code=busybox_' verify_busybox_pin
  cp "$BUSYBOX_PINNED_PATH" "$copied"
  cmp "$original" "$copied"
  release_busybox_pin
}

test_toolchain_pin_race() {
  local source="$TEST_ROOT/toolchain-race"
  local original="$TEST_ROOT/toolchain-race.original"
  local output
  cp -a "$TEST_ROOT/toolchain" "$source"
  TOOLCHAIN_ROOT="$source"
  pin_toolchain_source "$TOOLCHAIN_ROOT" 0
  mv "$source" "$original"
  cp -a "$TEST_ROOT/toolchain" "$source"
  assert_failure_contains 'code=toolchain_path_changed' verify_toolchain_pin
  output="$(env -i PATH="$TOOLCHAIN_PINNED_ROOT/bin" LC_ALL=C \
    GOROOT="$TOOLCHAIN_PINNED_ROOT" "$TOOLCHAIN_PINNED_ROOT/bin/go" version)"
  [[ "$output" = "go version go-fake linux/amd64" ]] || {
    printf 'guest_builder_test_failed toolchain_fd_not_original output=%s\n' "$output" >&2
    exit 1
  }
  release_toolchain_pin
}

test_root_ownership_validation_uses_resolved_toolchain() {
  local source="$TEST_ROOT/toolchain-resolved-ownership"
  local original_validator validation_calls=0
  cp -a "$TEST_ROOT/toolchain" "$source"
  original_validator="$(declare -f validate_stable_toolchain_ownership)"
  validate_stable_toolchain_ownership() {
    [[ "$1" == "$source" && "$1" != /proc/*/fd/* ]] || {
      printf 'guest_builder_test_failed ownership_validated_process_symlink path=%s\n' \
        "$1" >&2
      return 1
    }
    validation_calls=$((validation_calls + 1))
  }
  TOOLCHAIN_ROOT="$source"
  pin_toolchain_source "$TOOLCHAIN_ROOT" 1
  [[ "$validation_calls" -ge 2 ]] || {
    printf 'guest_builder_test_failed resolved_ownership_validation_calls=%s\n' \
      "$validation_calls" >&2
    return 1
  }
  release_toolchain_pin
  eval "$original_validator"
}

test_real_go_nobody_probe() {
  run_nobody_go_test_probe /usr/local/go "$TEST_ROOT/real-go-nobody-probe"
}

test_publish_no_clobber_and_partial_diagnostic() {
  local destination="$TEST_ROOT/publish-no-clobber"
  local output status=0
  mkdir -p "$destination/run"
  printf 'new-image\n' >"$destination/run/orquesta-test-guest.cpio.gz"
  printf 'new-manifest\n' >"$destination/run/manifest.json"
  printf 'existing-image\n' >"$destination/image"
  printf 'existing-manifest\n' >"$destination/manifest"
  output="$(bash -c '
    source "$1"
    RUN_DIR="$2"
    OUTPUT_IMAGE="$3"
    OUTPUT_MANIFEST="$4"
    trap cleanup_partial_publication EXIT
    publish_outputs
  ' bash "$BUILDER" "$destination/run" "$destination/image" "$destination/manifest" \
    2>&1)" || status=$?
  [[ "$status" != "0" && "$output" = *"code=output_no_clobber"* ]] || {
    printf 'guest_builder_test_failed publication_no_clobber status=%s output=%s\n' \
      "$status" "$output" >&2
    exit 1
  }
  [[ "$(cat "$destination/image")" = "existing-image" &&
    "$(cat "$destination/manifest")" = "existing-manifest" ]] || {
    printf 'guest_builder_test_failed publication_clobbered_existing_pair\n' >&2
    exit 1
  }
  rm -- "$destination/manifest"
  assert_failure_contains 'code=partial_publication_detected' \
    validate_output_pair_state "$destination/image" "$destination/manifest"
}

test_publish_race_has_single_matching_winner() {
  local destination="$TEST_ROOT/publish-race"
  local status_a=0 status_b=0 winner
  mkdir -p "$destination/a" "$destination/b"
  printf 'image-a\n' >"$destination/a/orquesta-test-guest.cpio.gz"
  printf 'candidate-a\n' >"$destination/a/manifest.json"
  printf 'image-b\n' >"$destination/b/orquesta-test-guest.cpio.gz"
  printf 'candidate-b\n' >"$destination/b/manifest.json"
  bash -c '
    source "$1"
    RUN_DIR="$2"
    OUTPUT_IMAGE="$3"
    OUTPUT_MANIFEST="$4"
    trap cleanup_partial_publication EXIT
    publish_outputs
  ' bash "$BUILDER" "$destination/a" "$destination/image" "$destination/manifest" \
    >"$destination/a.log" 2>&1 &
  local pid_a=$!
  bash -c '
    source "$1"
    RUN_DIR="$2"
    OUTPUT_IMAGE="$3"
    OUTPUT_MANIFEST="$4"
    trap cleanup_partial_publication EXIT
    publish_outputs
  ' bash "$BUILDER" "$destination/b" "$destination/image" "$destination/manifest" \
    >"$destination/b.log" 2>&1 &
  local pid_b=$!
  wait "$pid_a" || status_a=$?
  wait "$pid_b" || status_b=$?
  if (( (status_a == 0) == (status_b == 0) )); then
    printf 'guest_builder_test_failed publication_race statuses=%s,%s logs=%s|%s\n' \
      "$status_a" "$status_b" "$(cat "$destination/a.log")" "$(cat "$destination/b.log")" >&2
    exit 1
  fi
  winner=a
  ((status_b == 0)) && winner=b
  [[ "$(cat "$destination/image")" = "image-$winner" &&
    "$(cat "$destination/manifest")" = "candidate-$winner" ]] || {
    printf 'guest_builder_test_failed publication_mixed_winner=%s image=%s manifest=%s\n' \
      "$winner" "$(cat "$destination/image")" "$(cat "$destination/manifest")" >&2
    exit 1
  }
  if find "$destination" -maxdepth 1 -name '.*.orquesta-stage.*' -print -quit | grep -q .; then
    printf 'guest_builder_test_failed publication_stage_residual\n' >&2
    exit 1
  fi
}

test_publish_supports_distinct_durable_parents() {
  local destination="$TEST_ROOT/publish-distinct-parents"
  mkdir -p "$destination/run" "$destination/images" "$destination/manifests"
  printf 'distinct-image\n' >"$destination/run/orquesta-test-guest.cpio.gz"
  printf 'distinct-manifest\n' >"$destination/run/manifest.json"
  RUN_DIR="$destination/run"
  OUTPUT_IMAGE="$destination/images/image"
  OUTPUT_MANIFEST="$destination/manifests/manifest"
  publish_outputs
  [[ "$(cat "$OUTPUT_IMAGE")" = "distinct-image" &&
    "$(cat "$OUTPUT_MANIFEST")" = "distinct-manifest" ]] || {
    printf 'guest_builder_test_failed distinct_parent_publication\n' >&2
    exit 1
  }
}

test_partial_cleanup_preserves_replaced_inode() {
  local destination="$TEST_ROOT/publish-owned-cleanup"
  local output status=0
  mkdir "$destination"
  OUTPUT_IMAGE="$destination/image"
  OUTPUT_MANIFEST="$destination/manifest"
  printf 'owned\n' >"$OUTPUT_IMAGE"
  PUBLISHED_IMAGE_IDENTITY="$(path_identity "$OUTPUT_IMAGE")"
  PUBLISHED_IMAGE_PENDING=1
  mv "$OUTPUT_IMAGE" "$destination/moved-owned"
  printf 'foreign\n' >"$OUTPUT_IMAGE"
  output="$(cleanup_partial_publication 2>&1)" || status=$?
  [[ "$status" != "0" && "$output" = *"reason=identity_changed"* &&
    "$(cat "$OUTPUT_IMAGE")" = "foreign" ]] || {
    printf 'guest_builder_test_failed cleanup_ownership status=%s output=%s content=%s\n' \
      "$status" "$output" "$(cat "$OUTPUT_IMAGE")" >&2
    exit 1
  }
  PUBLISHED_IMAGE_PENDING=0
}

test_signal_recovers_owned_partial_publication() {
  local destination="$TEST_ROOT/publish-signal"
  local output status=0
  mkdir "$destination"
  output="$(bash -c '
    source "$1"
    RUN_DIR=""
    OUTPUT_IMAGE="$2"
    OUTPUT_MANIFEST="$3"
    PUBLISH_IMAGE_STAGE="$4"
    printf "partial\n" >"$PUBLISH_IMAGE_STAGE"
    PUBLISH_IMAGE_STAGE_IDENTITY="$(path_identity "$PUBLISH_IMAGE_STAGE")"
    PUBLISHED_IMAGE_IDENTITY="$PUBLISH_IMAGE_STAGE_IDENTITY"
    PUBLISHED_IMAGE_PENDING=1
    ln -- "$PUBLISH_IMAGE_STAGE" "$OUTPUT_IMAGE"
    trap cleanup EXIT
    trap "exit 130" INT
    trap "exit 143" TERM
    kill -TERM "$$"
  ' bash "$BUILDER" "$destination/image" "$destination/manifest" \
    "$destination/.image.orquesta-stage.signal" 2>&1)" || status=$?
  [[ "$status" = "143" && "$output" = *"state=recovered kind=image"* &&
    ! -e "$destination/image" && ! -e "$destination/manifest" &&
    ! -e "$destination/.image.orquesta-stage.signal" ]] || {
    printf 'guest_builder_test_failed signal_cleanup status=%s output=%s\n' \
      "$status" "$output" >&2
    exit 1
  }
}

test_cleanup_preserves_every_committed_transition() {
  local state destination output
  for state in marker_and_image_pending marker_pending marker_cleared; do
    destination="$TEST_ROOT/publish-committed-$state"
    mkdir "$destination"
    OUTPUT_IMAGE="$destination/image"
    OUTPUT_MANIFEST="$destination/manifest"
    printf 'committed-image\n' >"$OUTPUT_IMAGE"
    printf 'committed-manifest\n' >"$OUTPUT_MANIFEST"
    PUBLISHED_IMAGE_IDENTITY="$(path_identity "$OUTPUT_IMAGE")"
    PUBLISHED_MANIFEST_IDENTITY="$(path_identity "$OUTPUT_MANIFEST")"
    PUBLISHED_IMAGE_PENDING=1
    PUBLISHED_MANIFEST_PENDING=1
    PUBLISHED_MANIFEST_DURABLE=1
    case "$state" in
      marker_pending)
        PUBLISHED_IMAGE_PENDING=0
        ;;
      marker_cleared)
        PUBLISHED_IMAGE_PENDING=0
        PUBLISHED_MANIFEST_PENDING=0
        ;;
    esac
    output="$(cleanup_partial_publication 2>&1)"
    [[ "$(cat "$OUTPUT_IMAGE")" = "committed-image" &&
      "$(cat "$OUTPUT_MANIFEST")" = "committed-manifest" ]] || {
      printf 'guest_builder_test_failed committed_transition state=%s output=%s\n' \
        "$state" "$output" >&2
      exit 1
    }
  done
}

test_memory_contract_alignment() {
  grep -Fq 'scratchFixedReserveBytes = 64 << 20' \
    "$ROOT/internal/adapters/attestor/firecrackerguest/runner_linux.go"
  grep -Fq 'scratchCacheReserveBytes = 256 << 20' \
    "$ROOT/internal/adapters/attestor/firecrackerguest/runner_linux.go"
  grep -Fq 'scratchOptions  = "mode=0711,size=75%"' \
    "$ROOT/cmd/orquesta-test-guest/main_linux.go"
}

test_reproducible_fake_build() {
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
  grep -Fqx '/bin/busybox mount -t devtmpfs -o mode=0755,nosuid devtmpfs /dev' <<<"$init"
  grep -Fqx 'exec /orquesta-test-guest' <<<"$init"
  if grep -Eq '^export ' <<<"$init" || grep -Eqi 'ssh|cloud-init|wget|curl' <<<"$init"; then
    printf 'guest_builder_test_failed forbidden_init_surface\n' >&2
    exit 1
  fi

  local extracted="$TEST_ROOT/extracted"
  mkdir "$extracted"
  (
    cd "$extracted"
    gzip -dc "$TEST_ROOT/first/guest.cpio.gz" | cpio --quiet -idm
  )
  [[ "$(stat -c '%a' "$extracted/toolchain")" = "555" &&
    "$(stat -c '%a' "$extracted/toolchain/bin/go")" = "555" &&
    "$(stat -c '%a' "$extracted/toolchain/pkg/tool/linux_amd64/compile")" = "555" &&
    "$(stat -c '%a' "$extracted/toolchain/src/runtime/runtime.go")" = "444" ]] || {
    printf 'guest_builder_test_failed normalized_modes\n' >&2
    exit 1
  }
  [[ "$(readlink "$extracted/toolchain/src/runtime/alias.go")" = "runtime.go" ]] || {
    printf 'guest_builder_test_failed packaged_symlink\n' >&2
    exit 1
  }
  [[ -z "$(find "$extracted/toolchain" \( -type d -o -type f \) -perm /0222 -print -quit)" &&
    -z "$(find "$extracted/toolchain" ! -type d ! -type f ! -type l -print -quit)" ]] || {
    printf 'guest_builder_test_failed writable_or_special_toolchain\n' >&2
    exit 1
  }
  env -i PATH="$PATH_SAFE" LC_ALL=C bwrap \
    --unshare-user --unshare-net --uid 65534 --gid 65534 --die-with-parent \
    --clearenv --setenv GOROOT /toolchain --setenv PATH /toolchain/bin \
    --setenv GOENV off --setenv GOPROXY off --setenv GOSUMDB off --setenv GOTOOLCHAIN local \
    --ro-bind "$extracted/toolchain" /toolchain --proc /proc --dev /dev \
    /toolchain/bin/go version >/dev/null

  image_sha="$(sha256sum "$TEST_ROOT/first/guest.cpio.gz" | cut -d' ' -f1)"
  busybox_sha="$(sha256sum "$extracted/bin/busybox" | cut -d' ' -f1)"
  unpacked_bytes="$(gzip -dc "$TEST_ROOT/first/guest.cpio.gz" | wc -c)"
  minimum_required_bytes=$((unpacked_bytes +
    SCRATCH_FIXED_RESERVE_BYTES + SCRATCH_CACHE_RESERVE_BYTES))
  minimum_required_mib=$(((minimum_required_bytes + MEMORY_MIB_BYTES - 1) /
    MEMORY_MIB_BYTES))
  minimum_guest_memory_mib=$(((minimum_required_mib * 100 +
    SCRATCH_TMPFS_PERCENT - 1) / SCRATCH_TMPFS_PERCENT))
  ((minimum_guest_memory_mib <= 4096)) || {
    printf 'guest_builder_test_failed profile_4096_below_manifest_minimum minimum=%s\n' \
      "$minimum_guest_memory_mib" >&2
    exit 1
  }
  grep -Fq "\"image_sha256\":\"sha256:$image_sha\"" "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"busybox_sha256\":\"sha256:$busybox_sha\"" "$TEST_ROOT/first/manifest.json"
  grep -Eq '"busybox_version":"v[0-9A-Za-z._+-]+"' "$TEST_ROOT/first/manifest.json"
  grep -Fq '"toolchain_version":"go-fake"' "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"source_commit\":\"$SOURCE_COMMIT\"" "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"unpacked_bytes\":$unpacked_bytes" "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"minimum_guest_memory_mib\":$minimum_guest_memory_mib" \
    "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"scratch_fixed_reserve_bytes\":$SCRATCH_FIXED_RESERVE_BYTES" \
    "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"scratch_cache_reserve_bytes\":$SCRATCH_CACHE_RESERVE_BYTES" \
    "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"tmpfs_percent\":$SCRATCH_TMPFS_PERCENT" "$TEST_ROOT/first/manifest.json"
  grep -Fq "\"kernel_runtime_headroom_percent\":$KERNEL_RUNTIME_HEADROOM_PERCENT" \
    "$TEST_ROOT/first/manifest.json"
  grep -Fq '"runner_double_build":true' "$TEST_ROOT/first/manifest.json"
  grep -Fq '"source":"exact_commit_private_export"' "$TEST_ROOT/first/manifest.json"
  grep -Fq '"toolchain_directories":"0555"' "$TEST_ROOT/first/manifest.json"
  grep -Fq '"toolchain_nobody_go_test":true' "$TEST_ROOT/first/manifest.json"
}

make_fake_toolchain "$TEST_ROOT/toolchain"
test_guards
test_exact_commit_export_and_double_compile
test_submount_guard
test_busybox_pin_race
test_toolchain_pin_race
test_root_ownership_validation_uses_resolved_toolchain
test_real_go_nobody_probe
test_publish_no_clobber_and_partial_diagnostic
test_publish_race_has_single_matching_winner
test_publish_supports_distinct_durable_parents
test_partial_cleanup_preserves_replaced_inode
test_signal_recovers_owned_partial_publication
test_cleanup_preserves_every_committed_transition
test_memory_contract_alignment
test_reproducible_fake_build
printf 'ORQUESTA_TEST_GUEST_BUILDER_TEST_V0 status=pass mode=fake\n'
