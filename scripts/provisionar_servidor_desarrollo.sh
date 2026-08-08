#!/usr/bin/env bash
set -Eeuo pipefail

# Aprovisiona el host local donde se ejecuta. No abre SSH, no clona repositorios
# y no lee ni copia credenciales. Las descargas quedan fijadas por digest.

readonly SAFE_PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
readonly EXPECTED_OS_ID="ubuntu"
readonly EXPECTED_OS_VERSION="26.04"
readonly EXPECTED_UNAME_ARCH="x86_64"
readonly EXPECTED_DPKG_ARCH="amd64"

readonly GO_VERSION="1.25.11"
readonly GO_ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
readonly GO_URL="https://dl.google.com/go/${GO_ARCHIVE}"
readonly GO_ARCHIVE_SHA256="34f14304e856893f4ba30c2cacfe93906e9de7915c5f6aaaf3a81cdccd7ba30b"
readonly GO_BINARY_SHA256="e2ceecdc8170a43e196e4410030c8c6f9530bafa0efa71f6e8ad1a71ba3998e4"

readonly NODE_VERSION="20.19.2"
readonly NODE_ARCHIVE="node-v${NODE_VERSION}-linux-x64.tar.xz"
readonly NODE_URL="https://nodejs.org/dist/v${NODE_VERSION}/${NODE_ARCHIVE}"
readonly NODE_ARCHIVE_SHA256="cbe59620b21732313774df4428586f7222a84af29e556f848abf624ba41caf90"
readonly NODE_BINARY_SHA256="b9640779a1fffec1b6beeb90cb05019637457df2bd3e459034216b260a9ad0ac"

readonly NPM_VERSION="11.12.1"
readonly NPM_ARCHIVE="npm-${NPM_VERSION}.tgz"
readonly NPM_URL="https://registry.npmjs.org/npm/-/${NPM_ARCHIVE}"
readonly NPM_ARCHIVE_SHA512="cdca14b85d647b3192028d02aadbe82d75f79a446aceea9874be98e6d768f20ebd3555770a48d0e9906106007877bbc690f715e9372f2e2dc644a3c3157fb14c"
readonly NPM_CLI_SHA256="8e5f6f3429f8cdbe693cdc29904e9d5a7b127a494bd15c804bd54c7403bfcbe7"

readonly CODEX_VERSION="0.147.0"
readonly CODEX_ARCHIVE="codex-${CODEX_VERSION}.tgz"
readonly CODEX_URL="https://registry.npmjs.org/@openai/codex/-/${CODEX_ARCHIVE}"
readonly CODEX_ARCHIVE_SHA512="1102c45de7001b6a6dc48ed4a41328d9347f81ae79f7afdcfceb1817fd0ba140e1e4900d67b2281aa97304459bb84550efa25e3c86ed4d6fe2842929d5aed9df"
readonly CODEX_WRAPPER_SHA256="134063e133f0b4244fa3b251acf973d4fe4b4aeeacbdc135211bf480f59f1477"
readonly CODEX_NATIVE_ARCHIVE="codex-${CODEX_VERSION}-linux-x64.tgz"
readonly CODEX_NATIVE_URL="https://registry.npmjs.org/@openai/codex/-/${CODEX_NATIVE_ARCHIVE}"
readonly CODEX_NATIVE_ARCHIVE_SHA512="d16f4c0713e9596d1c4a436aad30cdda347baf3cd3ee834c850639e38ea54f62f0e5ccf9ca10d3724e156bdae3910126f87945ccffdd98431265b5df26c20d9b"
readonly CODEX_NATIVE_SHA256="cb0a15567e9a60a5820d54b0f6ae86d504dc3805c1eab21a47f70e3eb7b73a40"

readonly RUST_VERSION="1.97.1"
readonly RUST_TOOLCHAIN="${RUST_VERSION}-x86_64-unknown-linux-gnu"
readonly RUST_MUSL_TARGET="x86_64-unknown-linux-musl"
readonly RUST_MANIFEST_URL="https://static.rust-lang.org/dist/channel-rust-${RUST_VERSION}.toml"
readonly RUST_MANIFEST_SHA256="03569b1886ceb5c05276b50c8431ab111de944cd6140fe1fa7d821dd8e0f29cf"
readonly RUSTUP_VERSION="1.28.2"
readonly RUSTUP_URL="https://static.rust-lang.org/rustup/archive/${RUSTUP_VERSION}/x86_64-unknown-linux-gnu/rustup-init"
readonly RUSTUP_SHA256="20a06e644b0d9bd2fbdbfd52d42540bdde820ea7df86e92e533c073da0cdd43c"

readonly FIRECRACKER_VERSION="1.16.1"
readonly FIRECRACKER_ARCHIVE="firecracker-v${FIRECRACKER_VERSION}-x86_64.tgz"
readonly FIRECRACKER_URL="https://github.com/firecracker-microvm/firecracker/releases/download/v${FIRECRACKER_VERSION}/${FIRECRACKER_ARCHIVE}"
readonly FIRECRACKER_ARCHIVE_SHA256="382a02a869e4d6d5cb14c40577f9545e8458021ea8b0b2d3fc10ec14d9c242e6"
readonly FIRECRACKER_BINARY_SHA256="2fd0171309af7e24cf8dafc8a6f921c1434c49b5f9349bb996b7ed0a4deb8aa7"
readonly JAILER_BINARY_SHA256="1f3a0c1fe86212d0001819bfe0819071c01208b3ccc9398c3b3bc1b84cf21edd"

readonly TOOLCHAIN_ROOT="/opt/orquesta/toolchains"
readonly GO_ROOT="${TOOLCHAIN_ROOT}/go/${GO_VERSION}"
readonly NODE_ROOT="${TOOLCHAIN_ROOT}/node/${NODE_VERSION}"
readonly NPM_ROOT="${TOOLCHAIN_ROOT}/npm/${NPM_VERSION}"
readonly CODEX_ROOT="${TOOLCHAIN_ROOT}/codex/${CODEX_VERSION}"
readonly RUST_ROOT="${TOOLCHAIN_ROOT}/rust/${RUST_VERSION}"
readonly FIRECRACKER_ROOT="${TOOLCHAIN_ROOT}/firecracker/${FIRECRACKER_VERSION}"
readonly EVIDENCE_DIR="/var/lib/orquesta/provisioning"
readonly EVIDENCE_PATH="${EVIDENCE_DIR}/development-server-v1.json"
readonly WORK_MARKER_SCHEMA="orquesta_development_server_provision_tmp.v1"

WORK_DIR=""
EVIDENCE_TMP=""

export PATH="$SAFE_PATH"

usage() {
  cat <<'EOF'
Uso: sudo scripts/provisionar_servidor_desarrollo.sh

Aprovisiona exclusivamente el host local Ubuntu 26.04 x86_64. No acepta
destino SSH ni credenciales. Las versiones y digests están fijados en el script.
EOF
}

fail() {
  printf 'ORQUESTA_DEV_SERVER_PROVISION_ERROR code=%s\n' "$1" >&2
  exit 1
}

warn() {
  printf 'ORQUESTA_DEV_SERVER_PROVISION_WARNING code=%s\n' "$1" >&2
}

cleanup() {
  if [[ -n "$EVIDENCE_TMP" && "$EVIDENCE_TMP" == "$EVIDENCE_DIR"/.development-server-v1.* && -f "$EVIDENCE_TMP" ]]; then
    find "$EVIDENCE_TMP" -xdev -delete 2>/dev/null || true
  fi
  if [[ -z "$WORK_DIR" ]]; then
    return
  fi
  if [[ "$WORK_DIR" != "$TOOLCHAIN_ROOT"/.provision.* || ! -f "$WORK_DIR/.owner" ]]; then
    warn "temporary_workdir_retained"
    return
  fi
  if [[ "$(<"$WORK_DIR/.owner")" != "$WORK_MARKER_SCHEMA:$$" ]]; then
    warn "temporary_workdir_owner_mismatch_retained"
    return
  fi
  find "$WORK_DIR" -xdev -depth -delete 2>/dev/null || warn "temporary_workdir_cleanup_failed"
}

require_root_and_platform() {
  if [[ $# == 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
    usage
    exit 0
  fi
  if [[ $# != 0 ]]; then
    usage >&2
    fail "arguments_not_supported"
  fi
  [[ "$(id -u)" == "0" ]] || fail "root_required"
  [[ -r /etc/os-release ]] || fail "os_release_missing"
  # shellcheck disable=SC1091
  source /etc/os-release
  [[ "${ID:-}" == "$EXPECTED_OS_ID" ]] || fail "unsupported_os_${ID:-unknown}"
  [[ "${VERSION_ID:-}" == "$EXPECTED_OS_VERSION" ]] || fail "unsupported_ubuntu_${VERSION_ID:-unknown}"
  [[ "$(uname -m)" == "$EXPECTED_UNAME_ARCH" ]] || fail "unsupported_machine_arch"
  [[ "$(dpkg --print-architecture)" == "$EXPECTED_DPKG_ARCH" ]] || fail "unsupported_dpkg_arch"
}

prepare_workspace() {
  install -d -o root -g root -m 0755 "$TOOLCHAIN_ROOT"
  WORK_DIR="$(mktemp -d "$TOOLCHAIN_ROOT/.provision.XXXXXX")"
  chmod 0700 "$WORK_DIR"
  printf '%s:%s\n' "$WORK_MARKER_SCHEMA" "$$" >"$WORK_DIR/.owner"
  chmod 0600 "$WORK_DIR/.owner"
}

install_apt_packages() {
  local packages=(
    build-essential ca-certificates clang cmake cpio curl file git gzip jq
    libseccomp-dev libssl-dev lld musl-tools pkg-config python3 qemu-system-x86
    qemu-utils rsync tar unzip xz-utils
  )
  apt-get update
  apt-get install -y --no-install-recommends "${packages[@]}"
  local package
  for package in "${packages[@]}"; do
    dpkg-query -W -f='${db:Status-Status}\n' "$package" 2>/dev/null |
      grep -Fxq 'installed' || fail "apt_package_not_installed_${package}"
  done
}

download_checked() {
  local algorithm="$1" expected="$2" url="$3" destination="$4" observed
  [[ "$algorithm" == "sha256" || "$algorithm" == "sha512" ]] || fail "digest_algorithm_invalid"
  [[ "$expected" =~ ^[0-9a-f]+$ ]] || fail "expected_digest_invalid"
  curl --proto '=https' --proto-redir '=https' --tlsv1.2 --location --fail \
    --silent --show-error --retry 3 --retry-all-errors \
    --output "$destination" "$url"
  chmod 0600 "$destination"
  observed="$("${algorithm}"sum "$destination" | cut -d' ' -f1)"
  [[ "$observed" == "$expected" ]] || fail "download_checksum_mismatch_$(basename "$destination")"
}

verify_file_sha256() {
  local path="$1" expected="$2" observed
  [[ -f "$path" && ! -L "$path" ]] || fail "verified_file_missing_$(basename "$path")"
  observed="$(sha256sum "$path" | cut -d' ' -f1)"
  [[ "$observed" == "$expected" ]] || fail "installed_checksum_mismatch_$(basename "$path")"
}

require_real_directory_or_missing() {
  local path="$1"
  [[ ! -L "$path" ]] || fail "install_root_symlink_forbidden"
  [[ ! -e "$path" || -d "$path" ]] || fail "install_root_not_directory"
}

publish_directory() {
  local stage="$1" target="$2"
  [[ -d "$stage" && ! -L "$stage" ]] || fail "stage_directory_invalid"
  [[ ! -e "$target" && ! -L "$target" ]] || fail "install_target_raced"
  install -d -o root -g root -m 0755 "$(dirname "$target")"
  chmod -R go-w "$stage"
  mv -- "$stage" "$target"
}

ensure_link() {
  local target="$1" link="$2" current
  [[ "$target" = /* && "$link" == /usr/local/bin/* ]] || fail "link_scope_invalid"
  [[ -e "$target" && ! -d "$target" ]] || fail "link_target_invalid_$(basename "$link")"
  if [[ -L "$link" ]]; then
    current="$(readlink -- "$link")"
    if [[ "$current" != "$target" ]]; then
      ln -sfn -- "$target" "$link"
    fi
  elif [[ -e "$link" ]]; then
    fail "link_destination_owned_elsewhere_$(basename "$link")"
  else
    ln -s -- "$target" "$link"
  fi
}

probe_codex_version() {
  local executable="$1" probe_root="$WORK_DIR/codex-version-probe"
  install -d -m 0700 \
    "$probe_root/home" "$probe_root/config" "$probe_root/cache" \
    "$probe_root/data" "$probe_root/state" "$probe_root/tmp"
  env -i \
    PATH="$SAFE_PATH" \
    HOME="$probe_root/home" \
    CODEX_HOME="$probe_root/home" \
    XDG_CONFIG_HOME="$probe_root/config" \
    XDG_CACHE_HOME="$probe_root/cache" \
    XDG_DATA_HOME="$probe_root/data" \
    XDG_STATE_HOME="$probe_root/state" \
    TMPDIR="$probe_root/tmp" \
    "$executable" --version
}

install_go() {
  require_real_directory_or_missing "$GO_ROOT"
  if [[ ! -d "$GO_ROOT" ]]; then
    local archive="$WORK_DIR/$GO_ARCHIVE" stage="$WORK_DIR/go-root"
    download_checked sha256 "$GO_ARCHIVE_SHA256" "$GO_URL" "$archive"
    install -d -m 0755 "$stage"
    tar -xzf "$archive" -C "$stage" --strip-components=1
    publish_directory "$stage" "$GO_ROOT"
  fi
  verify_file_sha256 "$GO_ROOT/bin/go" "$GO_BINARY_SHA256"
  [[ "$($GO_ROOT/bin/go version)" == "go version go${GO_VERSION} linux/amd64" ]] || fail "go_version_mismatch"
  ensure_link "$GO_ROOT/bin/go" /usr/local/bin/go
  ensure_link "$GO_ROOT/bin/gofmt" /usr/local/bin/gofmt
}

install_node_npm_codex() {
  require_real_directory_or_missing "$NODE_ROOT"
  if [[ ! -d "$NODE_ROOT" ]]; then
    local node_archive="$WORK_DIR/$NODE_ARCHIVE" node_stage="$WORK_DIR/node-root"
    download_checked sha256 "$NODE_ARCHIVE_SHA256" "$NODE_URL" "$node_archive"
    install -d -m 0755 "$node_stage"
    tar -xJf "$node_archive" -C "$node_stage" --strip-components=1
    publish_directory "$node_stage" "$NODE_ROOT"
  fi
  verify_file_sha256 "$NODE_ROOT/bin/node" "$NODE_BINARY_SHA256"
  [[ "$($NODE_ROOT/bin/node --version)" == "v${NODE_VERSION}" ]] || fail "node_version_mismatch"

  require_real_directory_or_missing "$NPM_ROOT"
  if [[ ! -d "$NPM_ROOT" ]]; then
    local npm_archive="$WORK_DIR/$NPM_ARCHIVE" npm_stage="$WORK_DIR/npm-root"
    download_checked sha512 "$NPM_ARCHIVE_SHA512" "$NPM_URL" "$npm_archive"
    install -d -m 0755 "$npm_stage"
    tar -xzf "$npm_archive" -C "$npm_stage" --strip-components=1
    publish_directory "$npm_stage" "$NPM_ROOT"
  fi
  verify_file_sha256 "$NPM_ROOT/bin/npm-cli.js" "$NPM_CLI_SHA256"
  [[ "$($NODE_ROOT/bin/node "$NPM_ROOT/bin/npm-cli.js" --version)" == "$NPM_VERSION" ]] || fail "npm_version_mismatch"

  require_real_directory_or_missing "$CODEX_ROOT"
  if [[ ! -d "$CODEX_ROOT" ]]; then
    local codex_archive="$WORK_DIR/$CODEX_ARCHIVE"
    local native_archive="$WORK_DIR/$CODEX_NATIVE_ARCHIVE"
    local codex_stage="$WORK_DIR/codex-root"
    download_checked sha512 "$CODEX_ARCHIVE_SHA512" "$CODEX_URL" "$codex_archive"
    download_checked sha512 "$CODEX_NATIVE_ARCHIVE_SHA512" "$CODEX_NATIVE_URL" "$native_archive"
    install -d -m 0755 \
      "$codex_stage/lib/node_modules/@openai/codex" \
      "$codex_stage/lib/node_modules/@openai/codex-linux-x64"
    tar -xzf "$codex_archive" -C "$codex_stage/lib/node_modules/@openai/codex" --strip-components=1
    tar -xzf "$native_archive" -C "$codex_stage/lib/node_modules/@openai/codex-linux-x64" --strip-components=1
    chmod 0755 \
      "$codex_stage/lib/node_modules/@openai/codex/bin/codex.js" \
      "$codex_stage/lib/node_modules/@openai/codex-linux-x64/vendor/x86_64-unknown-linux-musl/bin/codex"
    publish_directory "$codex_stage" "$CODEX_ROOT"
  fi
  local codex_wrapper="$CODEX_ROOT/lib/node_modules/@openai/codex/bin/codex.js"
  local codex_native="$CODEX_ROOT/lib/node_modules/@openai/codex-linux-x64/vendor/x86_64-unknown-linux-musl/bin/codex"
  verify_file_sha256 "$codex_wrapper" "$CODEX_WRAPPER_SHA256"
  verify_file_sha256 "$codex_native" "$CODEX_NATIVE_SHA256"
  [[ "$(probe_codex_version "$codex_native")" == "codex-cli ${CODEX_VERSION}" ]] || fail "codex_version_mismatch"

  ensure_link "$NODE_ROOT/bin/node" /usr/local/bin/node
  ensure_link "$NPM_ROOT/bin/npm-cli.js" /usr/local/bin/npm
  ensure_link "$NPM_ROOT/bin/npx-cli.js" /usr/local/bin/npx
  ensure_link "$codex_wrapper" /usr/local/bin/codex
}

rustup_command() {
  env RUSTUP_HOME="$1" CARGO_HOME="$2" "$2/bin/rustup" "${@:3}"
}

install_rust() {
  local rust_manifest="$WORK_DIR/channel-rust-${RUST_VERSION}.toml"
  download_checked sha256 "$RUST_MANIFEST_SHA256" "$RUST_MANIFEST_URL" "$rust_manifest"
  require_real_directory_or_missing "$RUST_ROOT"
  if [[ ! -d "$RUST_ROOT" ]]; then
    local rustup_init="$WORK_DIR/rustup-init"
    local rust_stage="$WORK_DIR/rust-root"
    local stage_rustup="$rust_stage/rustup" stage_cargo="$rust_stage/cargo"
    download_checked sha256 "$RUSTUP_SHA256" "$RUSTUP_URL" "$rustup_init"
    chmod 0700 "$rustup_init"
    install -d -m 0755 "$stage_rustup" "$stage_cargo"
    env RUSTUP_HOME="$stage_rustup" CARGO_HOME="$stage_cargo" \
      "$rustup_init" -y --no-modify-path --profile minimal --default-toolchain none
    rustup_command "$stage_rustup" "$stage_cargo" set auto-self-update disable
    rustup_command "$stage_rustup" "$stage_cargo" toolchain install "$RUST_TOOLCHAIN" \
      --profile minimal --component clippy --component rustfmt --target "$RUST_MUSL_TARGET" \
      --no-self-update
    publish_directory "$rust_stage" "$RUST_ROOT"
  fi

  local rustup_home="$RUST_ROOT/rustup" cargo_home="$RUST_ROOT/cargo"
  [[ -x "$cargo_home/bin/rustup" ]] || fail "rustup_missing"
  [[ "$(rustup_command "$rustup_home" "$cargo_home" --version | awk '{print $2}')" == "$RUSTUP_VERSION" ]] || fail "rustup_version_mismatch"
  rustup_command "$rustup_home" "$cargo_home" toolchain install "$RUST_TOOLCHAIN" \
    --profile minimal --component clippy --component rustfmt --target "$RUST_MUSL_TARGET" \
    --no-self-update

  local rust_sysroot components
  rust_sysroot="$(rustup_command "$rustup_home" "$cargo_home" run "$RUST_TOOLCHAIN" rustc --print sysroot)"
  [[ "$rust_sysroot" == "$rustup_home/toolchains/$RUST_TOOLCHAIN" ]] || fail "rust_sysroot_unexpected"
  [[ "$("$rust_sysroot/bin/rustc" --version | awk '{print $2}')" == "$RUST_VERSION" ]] || fail "rust_version_mismatch"
  components="$(rustup_command "$rustup_home" "$cargo_home" component list --toolchain "$RUST_TOOLCHAIN" --installed)"
  grep -Eq '^clippy-x86_64-unknown-linux-gnu( \(installed\))?$' <<<"$components" || fail "rust_clippy_missing"
  grep -Eq '^rustfmt-x86_64-unknown-linux-gnu( \(installed\))?$' <<<"$components" || fail "rustfmt_missing"
  grep -Eq "^rust-std-${RUST_MUSL_TARGET}( \\(installed\\))?$" <<<"$components" || fail "rust_musl_target_missing"

  local binary
  for binary in cargo cargo-clippy cargo-fmt clippy-driver rustc rustdoc rustfmt; do
    ensure_link "$rust_sysroot/bin/$binary" "/usr/local/bin/$binary"
  done
}

install_firecracker() {
  require_real_directory_or_missing "$FIRECRACKER_ROOT"
  if [[ ! -d "$FIRECRACKER_ROOT" ]]; then
    local archive="$WORK_DIR/$FIRECRACKER_ARCHIVE" stage="$WORK_DIR/firecracker-root"
    download_checked sha256 "$FIRECRACKER_ARCHIVE_SHA256" "$FIRECRACKER_URL" "$archive"
    install -d -m 0755 "$stage"
    tar -xzf "$archive" -C "$stage" --strip-components=1
    (cd "$stage" && sha256sum --check --quiet SHA256SUMS) || fail "firecracker_internal_checksums_invalid"
    publish_directory "$stage" "$FIRECRACKER_ROOT"
  fi

  local firecracker="$FIRECRACKER_ROOT/firecracker-v${FIRECRACKER_VERSION}-x86_64"
  local jailer="$FIRECRACKER_ROOT/jailer-v${FIRECRACKER_VERSION}-x86_64"
  verify_file_sha256 "$firecracker" "$FIRECRACKER_BINARY_SHA256"
  verify_file_sha256 "$jailer" "$JAILER_BINARY_SHA256"
  [[ "$($firecracker --version 2>/dev/null | sed -n '1p')" == "Firecracker v${FIRECRACKER_VERSION}" ]] || fail "firecracker_version_mismatch"
  [[ "$($jailer --version 2>/dev/null | sed -n '1p')" == "Jailer v${FIRECRACKER_VERSION}" ]] || fail "jailer_version_mismatch"
  ensure_link "$firecracker" /usr/local/bin/firecracker
  ensure_link "$jailer" /usr/local/bin/jailer
}

detect_kvm() {
  if [[ ! -e /dev/kvm ]]; then
    printf 'absent\n'
  elif [[ ! -c /dev/kvm ]]; then
    printf 'not_character_device\n'
  elif [[ ! -r /dev/kvm || ! -w /dev/kvm ]]; then
    printf 'present_not_rw\n'
  else
    printf 'available_rw\n'
  fi
}

record_evidence() {
  local kvm_state="$1" provision_status="$2" qemu_version qemu_img_version generated_at
  qemu_version="$(qemu-system-x86_64 --version)"
  qemu_version="${qemu_version%%$'\n'*}"
  qemu_img_version="$(qemu-img --version)"
  qemu_img_version="${qemu_img_version%%$'\n'*}"
  generated_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  install -d -o root -g root -m 0755 "$EVIDENCE_DIR"
  EVIDENCE_TMP="$(mktemp "$EVIDENCE_DIR/.development-server-v1.XXXXXX")"
  jq -cn \
    --arg generated_at "$generated_at" \
    --arg status "$provision_status" \
    --arg kvm "$kvm_state" \
    --arg qemu "$qemu_version" \
    --arg qemu_img "$qemu_img_version" \
    --arg go "go${GO_VERSION}" \
    --arg node "v${NODE_VERSION}" \
    --arg npm "$NPM_VERSION" \
    --arg codex "$CODEX_VERSION" \
    --arg rust "$RUST_VERSION" \
    --arg rust_target "$RUST_MUSL_TARGET" \
    --arg firecracker "$FIRECRACKER_VERSION" \
    --arg go_sha "$GO_ARCHIVE_SHA256" \
    --arg node_sha "$NODE_ARCHIVE_SHA256" \
    --arg npm_sha "$NPM_ARCHIVE_SHA512" \
    --arg codex_sha "$CODEX_ARCHIVE_SHA512" \
    --arg codex_native_sha "$CODEX_NATIVE_ARCHIVE_SHA512" \
    --arg rustup_sha "$RUSTUP_SHA256" \
    --arg rust_manifest_sha "$RUST_MANIFEST_SHA256" \
    --arg firecracker_sha "$FIRECRACKER_ARCHIVE_SHA256" \
    '{schema:"orquesta_development_server_provision.v1",generated_at:$generated_at,status:$status,host:{os:"ubuntu",version:"26.04",arch:"x86_64",kvm:$kvm},tools:{go:$go,node:$node,npm:$npm,codex:$codex,rust:$rust,rust_target:$rust_target,firecracker:$firecracker,jailer:$firecracker,qemu:$qemu,qemu_img:$qemu_img},source_digests:{go_sha256:$go_sha,node_sha256:$node_sha,npm_sha512:$npm_sha,codex_sha512:$codex_sha,codex_native_sha512:$codex_native_sha,rustup_sha256:$rustup_sha,rust_manifest_sha256:$rust_manifest_sha,firecracker_sha256:$firecracker_sha}}' \
    >"$EVIDENCE_TMP"
  chmod 0644 "$EVIDENCE_TMP"
  chown root:root "$EVIDENCE_TMP"
  mv -f -- "$EVIDENCE_TMP" "$EVIDENCE_PATH"
  EVIDENCE_TMP=""
}

main() {
  require_root_and_platform "$@"
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  prepare_workspace
  install_apt_packages
  install_go
  install_node_npm_codex
  install_rust
  install_firecracker

  command -v qemu-system-x86_64 >/dev/null || fail "qemu_system_missing"
  command -v qemu-img >/dev/null || fail "qemu_img_missing"
  hash -r
  [[ "$(go version)" == "go version go${GO_VERSION} linux/amd64" ]] || fail "active_go_version_mismatch"
  [[ "$(node --version)" == "v${NODE_VERSION}" ]] || fail "active_node_version_mismatch"
  [[ "$(npm --version)" == "$NPM_VERSION" ]] || fail "active_npm_version_mismatch"
  [[ "$(probe_codex_version /usr/local/bin/codex)" == "codex-cli ${CODEX_VERSION}" ]] || fail "active_codex_version_mismatch"
  [[ "$(rustc --version | awk '{print $2}')" == "$RUST_VERSION" ]] || fail "active_rust_version_mismatch"

  local kvm_state provision_status
  kvm_state="$(detect_kvm)"
  provision_status="ready"
  if [[ "$kvm_state" != "available_rw" ]]; then
    provision_status="ready_without_kvm"
    warn "kvm_${kvm_state}_firecracker_not_exercised"
  fi
  record_evidence "$kvm_state" "$provision_status"
  printf 'ORQUESTA_DEV_SERVER_PROVISION_V1 status=%s evidence=%s\n' \
    "$provision_status" "$EVIDENCE_PATH"
}

main "$@"
