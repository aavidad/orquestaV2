#!/usr/bin/env bash
set -Eeuo pipefail

umask 077
export LC_ALL=C

usage() {
  printf '%s\n' \
    "Uso: build_agent_microvm_codex_assets.sh \\" \
    "  --agente-microvm-repo RUTA_ABSOLUTA \\" \
    "  --codex RUTA_ABSOLUTA --busybox RUTA_ABSOLUTA \\" \
    "  --go RUTA_ABSOLUTA --cargo RUTA_ABSOLUTA --rustc RUTA_ABSOLUTA \\" \
    "  --cargo-registry RUTA_ABSOLUTA --salida RUTA_ABSOLUTA_NUEVA \\" \
    '  --temporal RUTA_ABSOLUTA [--kernel RUTA_ABSOLUTA]' \
    '' \
    'Construye sin sudo ni KVM los activos PFC-01. Solo usa blobs confirmados' \
    'en HEAD y una allowlist mínima de fuentes. Si este constructor no está' \
    'confirmado o difiere de su blob HEAD, falla antes de crear temporales o red.' >&2
}

die() {
  printf 'error.pfc01.activos: %s\n' "$1" >&2
  exit 1
}

digest() {
  "${TOOL_PATH[sha256sum]}" "$1" | "${TOOL_PATH[cut]}" -d' ' -f1
}

size() {
  "${TOOL_PATH[stat]}" -c '%s' -- "$1"
}

absolute_regular() {
  local path="$1" code="$2"
  [[ "$path" == /* && -f "$path" && ! -L "$path" ]] || die "$code"
}

absolute_executable() {
  local path="$1" code="$2"
  absolute_regular "$path" "$code"
  [[ -x "$path" ]] || die "$code"
}

private_directory() {
  local path="$1" code="$2"
  [[ "$path" == /* && -d "$path" && ! -L "$path" ]] || die "$code"
  [[ "$("${TOOL_PATH[stat]}" -c '%u' -- "$path")" == \
    "$("${TOOL_PATH[id]}" -u)" ]] || die "${code}_propietario"
  (( (8#$("${TOOL_PATH[stat]}" -c '%a' -- "$path") & 8#077) == 0 )) ||
    die "${code}_modo"
}

AGENT_REPO=""
CODEX=""
BUSYBOX=""
GO_BIN=""
CARGO_BIN=""
RUSTC_BIN=""
CARGO_REGISTRY=""
OUTPUT=""
TMP_PARENT=""
KERNEL_INPUT=""
while (($# > 0)); do
  case "$1" in
    --agente-microvm-repo) AGENT_REPO="${2:-}"; shift 2 ;;
    --codex) CODEX="${2:-}"; shift 2 ;;
    --busybox) BUSYBOX="${2:-}"; shift 2 ;;
    --go) GO_BIN="${2:-}"; shift 2 ;;
    --cargo) CARGO_BIN="${2:-}"; shift 2 ;;
    --rustc) RUSTC_BIN="${2:-}"; shift 2 ;;
    --cargo-registry) CARGO_REGISTRY="${2:-}"; shift 2 ;;
    --salida) OUTPUT="${2:-}"; shift 2 ;;
    --temporal) TMP_PARENT="${2:-}"; shift 2 ;;
    --kernel) KERNEL_INPUT="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "argumento_no_admitido" ;;
  esac
done

declare -A TOOL_PATH=()
declare -A TOOL_DIGEST=()
TOOLS=(
  ar as bash basename cc chmod cmp cp cpio curl cut date df dirname e2fsck env
  file find git grep gzip id install jq ld ln mkdir mke2fs mktemp mv readelf
  readlink rm sed sha256sum sort stat tar touch tr truncate unlink
)
BOOTSTRAP_READLINK="$(type -P readlink || true)"
[[ -n "$BOOTSTRAP_READLINK" && -f "$BOOTSTRAP_READLINK" &&
  -x "$BOOTSTRAP_READLINK" ]] || die "herramienta_invalida_readlink"
for tool in "${TOOLS[@]}"; do
  resolved="$("$BOOTSTRAP_READLINK" -f -- "$(type -P "$tool" || true)")"
  [[ -n "$resolved" && -f "$resolved" && -x "$resolved" && ! -L "$resolved" ]] ||
    die "herramienta_invalida_$tool"
  TOOL_PATH["$tool"]="$resolved"
done
for tool in "${TOOLS[@]}"; do
  TOOL_DIGEST["$tool"]="$("${TOOL_PATH[sha256sum]}" "${TOOL_PATH[$tool]}" |
    "${TOOL_PATH[cut]}" -d' ' -f1)"
done

run_preflight_clean() {
  "${TOOL_PATH[env]}" -i GIT_CONFIG_NOSYSTEM=1 HOME=/nonexistent LANG=C \
    LC_ALL=C PATH=/nonexistent TZ=UTC "$@"
}

SCRIPT_PATH="$("${TOOL_PATH[readlink]}" -f -- "${BASH_SOURCE[0]}")"
ORQUESTA_REPO="$(cd -- "$("${TOOL_PATH[dirname]}" -- "$SCRIPT_PATH")/.." && pwd -P)"
readonly SCRIPT_PATH ORQUESTA_REPO

[[ -n "$AGENT_REPO" && -n "$CODEX" && -n "$BUSYBOX" && -n "$GO_BIN" &&
  -n "$CARGO_BIN" && -n "$RUSTC_BIN" && -n "$CARGO_REGISTRY" &&
  -n "$OUTPUT" && -n "$TMP_PARENT" ]] || die "argumentos_incompletos"

# La receta forma parte del sujeto. Esta guarda precede la validación de
# activos, el preflight de disco, mktemp y cualquier ruta de descarga.
ORQUESTA_COMMIT="$(run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" \
  rev-parse --verify 'HEAD^{commit}')"
ORQUESTA_TREE="$(run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" \
  rev-parse --verify 'HEAD^{tree}')"
BUILDER_RELATIVE="${SCRIPT_PATH#"$ORQUESTA_REPO"/}"
[[ "$BUILDER_RELATIVE" != "$SCRIPT_PATH" && -n "$BUILDER_RELATIVE" ]] ||
  die "constructor_fuera_repositorio"
run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" ls-files --error-unmatch -- \
  "$BUILDER_RELATIVE" >/dev/null 2>&1 ||
  die "constructor_no_trackeado"
run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" \
  show "$ORQUESTA_COMMIT:$BUILDER_RELATIVE" |
  "${TOOL_PATH[cmp]}" -s - "$SCRIPT_PATH" || die "constructor_no_coincide_head"
BUILDER_DIGEST="$(run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" \
  show "$ORQUESTA_COMMIT:$BUILDER_RELATIVE" |
  "${TOOL_PATH[sha256sum]}" | "${TOOL_PATH[cut]}" -d' ' -f1)"
readonly ORQUESTA_COMMIT ORQUESTA_TREE BUILDER_RELATIVE BUILDER_DIGEST

[[ "$AGENT_REPO" == /* && -d "$AGENT_REPO" && ! -L "$AGENT_REPO" ]] ||
  die "repositorio_agente_microvm_invalido"
AGENT_REPO="$(cd -- "$AGENT_REPO" && pwd -P)"
run_preflight_clean "${TOOL_PATH[git]}" -C "$AGENT_REPO" \
  rev-parse --is-inside-work-tree 2>/dev/null |
  "${TOOL_PATH[grep]}" -Fxq true ||
  die "agente_microvm_no_es_git"
AGENT_COMMIT="$(run_preflight_clean "${TOOL_PATH[git]}" -C "$AGENT_REPO" \
  rev-parse --verify 'HEAD^{commit}')"
AGENT_TREE="$(run_preflight_clean "${TOOL_PATH[git]}" -C "$AGENT_REPO" \
  rev-parse --verify 'HEAD^{tree}')"
readonly AGENT_COMMIT AGENT_TREE

absolute_executable "$CODEX" "codex_invalido"
absolute_executable "$BUSYBOX" "busybox_invalido"
absolute_executable "$GO_BIN" "go_invalido"
absolute_executable "$CARGO_BIN" "cargo_invalido"
absolute_executable "$RUSTC_BIN" "rustc_invalido"
GO_DIGEST="$(digest "$GO_BIN")"
CARGO_DIGEST="$(digest "$CARGO_BIN")"
RUSTC_DIGEST="$(digest "$RUSTC_BIN")"
[[ "$CARGO_REGISTRY" == /* && -d "$CARGO_REGISTRY" && ! -L "$CARGO_REGISTRY" ]] ||
  die "cargo_registry_invalido"
CARGO_REGISTRY="$(cd -- "$CARGO_REGISTRY" && pwd -P)"
[[ -z "$KERNEL_INPUT" ]] || absolute_regular "$KERNEL_INPUT" "kernel_invalido"
private_directory "$TMP_PARENT" "temporal_invalido"
[[ "$OUTPUT" == /* && ! -e "$OUTPUT" && ! -L "$OUTPUT" ]] || die "salida_invalida"
OUTPUT_PARENT="$("${TOOL_PATH[dirname]}" -- "$OUTPUT")"
private_directory "$OUTPUT_PARENT" "padre_salida_invalido"

df_available_bytes() {
  "${TOOL_PATH[df]}" -B1 --output=avail -- "$1" |
    "${TOOL_PATH[sed]}" -n '2{s/[[:space:]]//g;p;}'
}

TEMP_REQUIRED=$((3 * 1024 * 1024 * 1024))
OUTPUT_REQUIRED=$((1 * 1024 * 1024 * 1024))
TEMP_AVAILABLE="$(df_available_bytes "$TMP_PARENT")"
OUTPUT_AVAILABLE="$(df_available_bytes "$OUTPUT_PARENT")"
[[ "$TEMP_AVAILABLE" =~ ^[0-9]+$ && "$OUTPUT_AVAILABLE" =~ ^[0-9]+$ ]] ||
  die "espacio_no_medido"
if [[ "$("${TOOL_PATH[stat]}" -c '%d' -- "$TMP_PARENT")" == \
  "$("${TOOL_PATH[stat]}" -c '%d' -- "$OUTPUT_PARENT")" ]]; then
  ((TEMP_AVAILABLE >= TEMP_REQUIRED + OUTPUT_REQUIRED)) || die "espacio_insuficiente"
else
  ((TEMP_AVAILABLE >= TEMP_REQUIRED && OUTPUT_AVAILABLE >= OUTPUT_REQUIRED)) ||
    die "espacio_insuficiente"
fi

WORK_DIR=""
WORK_ID=""
STAGE_DIR=""
STAGE_ID=""
OUTPUT_ID=""
OUTPUT_OWNED=0
SUCCESS=0
TEMP_ID_CRITICAL=0
DEFERRED_SIGNAL_STATUS=0

handle_signal() {
  local status="$1"
  if ((TEMP_ID_CRITICAL == 1)); then
    DEFERRED_SIGNAL_STATUS="$status"
    return 0
  fi
  exit "$status"
}

release_temp_identity_section() {
  TEMP_ID_CRITICAL=0
  if ((DEFERRED_SIGNAL_STATUS != 0)); then
    exit "$DEFERRED_SIGNAL_STATUS"
  fi
}

create_owned_temp_dir() {
  local result_name="$1" parent="$2" prefix="$3" candidate nonce
  local attempt
  printf -v "$result_name" '%s' ''
  for ((attempt = 0; attempt < 32; attempt++)); do
    printf -v nonce '%04x%04x' "$RANDOM" "$RANDOM"
    candidate="$parent/$prefix.$BASHPID.$nonce"
    printf -v "$result_name" '%s' "$candidate"
    if "${TOOL_PATH[env]}" --ignore-signal=INT --ignore-signal=TERM \
      "${TOOL_PATH[mkdir]}" -m 0700 -- "$candidate"; then
      return 0
    fi
    printf -v "$result_name" '%s' ''
  done
  die "temporal_colisiones_agotadas"
}

unlink_external_cache_ref() {
  local registry_link
  if [[ -n "$WORK_ID" && -d "$WORK_DIR" && ! -L "$WORK_DIR" &&
    "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$WORK_DIR")" == "$WORK_ID" ]]; then
    for registry_link in "$WORK_DIR/cargo-home/registry" \
      "$WORK_DIR/cargo-seed-home/registry"; do
      if [[ -L "$registry_link" ]]; then
        "${TOOL_PATH[unlink]}" -- "$registry_link" || return 1
      fi
    done
  fi
}

remove_owned_tree() {
  local path="$1" expected_parent="$2" expected_id="$3"
  [[ -n "$path" && -n "$expected_id" && -d "$path" && ! -L "$path" &&
    "$path" == "$expected_parent"/* ]] ||
    return 1
  [[ "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$path")" == "$expected_id" ]] || return 1
  "${TOOL_PATH[chmod]}" -R u+w -- "$path" || return 1
  "${TOOL_PATH[rm]}" -rf --one-file-system -- "$path"
}

cleanup() {
  local status=$? cleanup_failed=0
  trap - EXIT INT TERM
  set +e
  if ((SUCCESS == 0 && OUTPUT_OWNED == 1)) && [[ -e "$OUTPUT" || -L "$OUTPUT" ]]; then
    remove_owned_tree "$OUTPUT" "$OUTPUT_PARENT" "$OUTPUT_ID" || cleanup_failed=1
  fi
  unlink_external_cache_ref || cleanup_failed=1
  if [[ -n "$STAGE_DIR" && (-e "$STAGE_DIR" || -L "$STAGE_DIR") ]]; then
    remove_owned_tree "$STAGE_DIR" "$OUTPUT_PARENT" "$STAGE_ID" || cleanup_failed=1
  fi
  if [[ -n "$WORK_DIR" && (-e "$WORK_DIR" || -L "$WORK_DIR") ]]; then
    remove_owned_tree "$WORK_DIR" "$TMP_PARENT" "$WORK_ID" || cleanup_failed=1
  fi
  if ((status == 0 && cleanup_failed != 0)); then
    return 1
  fi
  return "$status"
}

trap cleanup EXIT
trap 'handle_signal 130' INT
trap 'handle_signal 143' TERM
TEMP_ID_CRITICAL=1
create_owned_temp_dir WORK_DIR "$TMP_PARENT" "pfc01-activos"
WORK_ID="$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$WORK_DIR")"
[[ "$WORK_ID" =~ ^[0-9]+:[0-9]+$ ]] || die "temporal_identidad_invalida"
release_temp_identity_section
"${TOOL_PATH[chmod]}" 0700 -- "$WORK_DIR"
[[ "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$WORK_DIR")" == "$WORK_ID" ]] ||
  die "temporal_sustituido"
TEMP_ID_CRITICAL=1
create_owned_temp_dir STAGE_DIR "$OUTPUT_PARENT" ".pfc01-activos"
STAGE_ID="$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$STAGE_DIR")"
[[ "$STAGE_ID" =~ ^[0-9]+:[0-9]+$ ]] || die "staging_identidad_invalida"
release_temp_identity_section
"${TOOL_PATH[chmod]}" 0700 -- "$STAGE_DIR"
[[ "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$STAGE_DIR")" == "$STAGE_ID" ]] ||
  die "staging_sustituido"

TOOL_BIN="$WORK_DIR/tool-bin"
HOME_DIR="$WORK_DIR/home"
BUILD_TMP="$WORK_DIR/tmp"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$TOOL_BIN" "$HOME_DIR" "$BUILD_TMP"
for tool in "${TOOLS[@]}"; do
  "${TOOL_PATH[ln]}" -s -- "${TOOL_PATH[$tool]}" "$TOOL_BIN/$tool"
done

run_clean() {
  "${TOOL_PATH[env]}" -i GIT_CONFIG_NOSYSTEM=1 HOME="$HOME_DIR" \
    LANG=C LC_ALL=C PATH="$TOOL_BIN" \
    TMPDIR="$BUILD_TMP" TZ=UTC "$@"
}

relative_to_root() {
  local path="$1" root="$2" code="$3"
  [[ "$path" == "$root"/* ]] || die "$code"
  printf '%s\n' "${path#"$root"/}"
}

tree_digest() {
  local root="$1" snapshot_file relative
  local -a files=()
  mapfile -d '' -t files < <(
    cd -- "$root"
    "${TOOL_PATH[find]}" . -type f -print0 | "${TOOL_PATH[sort]}" -z
  )
  ((${#files[@]} > 0)) || die "snapshot_vacio"
  (
    cd -- "$root"
    for snapshot_file in "${files[@]}"; do
      relative="${snapshot_file#./}"
      printf '%s\0' "$relative"
      digest "$snapshot_file"
    done
  ) | "${TOOL_PATH[sha256sum]}" | "${TOOL_PATH[cut]}" -d' ' -f1
}

readonly -a ORQUESTA_ARCHIVE_PATHS=(
  go.mod
  go.sum
  vendor/modules.txt
  vendor/golang.org/x/sys/LICENSE
  vendor/golang.org/x/sys/PATENTS
  vendor/golang.org/x/sys/unix
  cmd/orquesta-codex-work-executor
  internal/adapters/executor/codexwork
  internal/adapters/protocol/codexwork
)
readonly -a AGENT_ARCHIVE_PATHS=(
  Cargo.toml
  Cargo.lock
  rust-toolchain.toml
  crates/agente_microvm_huesped
  crates/contrato_huesped
  activos/kernel_desarrollo_x86_64.json
  activos/perfil_codex_desarrollo_x86_64.json
  scripts/construir_initramfs_sonda.sh
  scripts/construir_perfil_estatico.sh
  scripts/preparar_kernel_desarrollo.sh
)
ORQUESTA_SOURCE="$WORK_DIR/orquesta"
AGENT_SOURCE="$WORK_DIR/agente-microvm"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$ORQUESTA_SOURCE" "$AGENT_SOURCE"
run_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" archive --format=tar \
  "$ORQUESTA_COMMIT" -- "${ORQUESTA_ARCHIVE_PATHS[@]}" |
  run_clean "${TOOL_PATH[tar]}" -xf - -C "$ORQUESTA_SOURCE"
run_clean "${TOOL_PATH[git]}" -C "$AGENT_REPO" archive --format=tar \
  "$AGENT_COMMIT" -- "${AGENT_ARCHIVE_PATHS[@]}" |
  run_clean "${TOOL_PATH[tar]}" -xf - -C "$AGENT_SOURCE"
for forbidden in modulos docs product acceptance opes-salidas; do
  [[ ! -e "$ORQUESTA_SOURCE/$forbidden" && ! -e "$AGENT_SOURCE/$forbidden" ]] ||
    die "archivo_congelado_materializado"
done

PROFILE_TEMPLATE="$AGENT_SOURCE/activos/perfil_codex_desarrollo_x86_64.json"
KERNEL_MANIFEST="$AGENT_SOURCE/activos/kernel_desarrollo_x86_64.json"
INITRAMFS_BUILDER="$AGENT_SOURCE/scripts/construir_initramfs_sonda.sh"
PROFILE_BUILDER="$AGENT_SOURCE/scripts/construir_perfil_estatico.sh"
KERNEL_PREPARER="$AGENT_SOURCE/scripts/preparar_kernel_desarrollo.sh"
for required in "$PROFILE_TEMPLATE" "$KERNEL_MANIFEST" "$INITRAMFS_BUILDER" \
  "$PROFILE_BUILDER" "$KERNEL_PREPARER"; do
  absolute_regular "$required" "fuente_agente_microvm_incompleta"
done

# Cargo resuelve todos los miembros del workspace aunque solo se solicite el
# paquete huésped. Esta vista de build conserva sus fuentes y Cargo.lock de
# HEAD, pero evita incorporar dependencias del servidor anfitrión al snapshot.
GUEST_BUILD_ROOT="$WORK_DIR/guest-build"
GUEST_BUILD_SOURCE="$GUEST_BUILD_ROOT/agente_microvm_huesped"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$GUEST_BUILD_ROOT"
"${TOOL_PATH[cp]}" -a -- "$AGENT_SOURCE/crates/agente_microvm_huesped" \
  "$GUEST_BUILD_SOURCE"
"${TOOL_PATH[cp]}" -a -- "$AGENT_SOURCE/crates/contrato_huesped" \
  "$GUEST_BUILD_ROOT/contrato_huesped"
"${TOOL_PATH[install]}" -m 0600 -- "$AGENT_SOURCE/Cargo.lock" \
  "$GUEST_BUILD_SOURCE/Cargo.lock"

run_clean "${TOOL_PATH[jq]}" -e '
  .schema == "agente_microvm.activo_kernel.v1"
  and .arquitectura == "x86_64"
  and (.nombre | test("^vmlinux-[A-Za-z0-9._-]+$"))
  and (.sha256 | test("^[0-9a-f]{64}$"))
  and (.tamano_bytes | type == "number" and . > 0)
  and (.fuente | type == "string" and startswith("https://"))
  and (.configuracion_sha256 | test("^[0-9a-f]{64}$"))
' "$KERNEL_MANIFEST" >/dev/null || die "kernel_manifiesto_invalido"

INPUT_DIR="$WORK_DIR/inputs"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$INPUT_DIR"
CODEX_INPUT="$INPUT_DIR/codex"
BUSYBOX_INPUT="$INPUT_DIR/busybox"
"${TOOL_PATH[install]}" -m 0500 -- "$CODEX" "$CODEX_INPUT"
"${TOOL_PATH[install]}" -m 0500 -- "$BUSYBOX" "$BUSYBOX_INPUT"
CODEX_EXPECTED_DIGEST="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.ejecutable_principal.sha256' "$PROFILE_TEMPLATE")"
CODEX_EXPECTED_SIZE="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.ejecutable_principal.tamano_bytes' "$PROFILE_TEMPLATE")"
BUSYBOX_EXPECTED_DIGEST="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.busybox.sha256' "$PROFILE_TEMPLATE")"
BUSYBOX_EXPECTED_SIZE="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.busybox.tamano_bytes' "$PROFILE_TEMPLATE")"
[[ "$(digest "$CODEX_INPUT")" == "$CODEX_EXPECTED_DIGEST" &&
  "$(size "$CODEX_INPUT")" == "$CODEX_EXPECTED_SIZE" ]] || die "codex_no_fijado"
[[ "$(digest "$BUSYBOX_INPUT")" == "$BUSYBOX_EXPECTED_DIGEST" &&
  "$(size "$BUSYBOX_INPUT")" == "$BUSYBOX_EXPECTED_SIZE" ]] || die "busybox_no_fijado"
for binary in "$CODEX_INPUT" "$BUSYBOX_INPUT"; do
  run_clean "${TOOL_PATH[file]}" -- "$binary" |
    run_clean "${TOOL_PATH[grep]}" -Fq 'ELF 64-bit LSB' || die "binario_entrada_no_elf64"
  run_clean "${TOOL_PATH[readelf]}" -h -- "$binary" |
    run_clean "${TOOL_PATH[grep]}" -Fq 'Advanced Micro Devices X86-64' ||
    die "binario_entrada_arquitectura_invalida"
  if run_clean "${TOOL_PATH[readelf]}" -lW -- "$binary" |
    run_clean "${TOOL_PATH[grep]}" -F 'INTERP' >/dev/null ||
    run_clean "${TOOL_PATH[readelf]}" -dW -- "$binary" 2>/dev/null |
    run_clean "${TOOL_PATH[grep]}" -F '(NEEDED)' >/dev/null; then
    die "binario_entrada_no_estatico"
  fi
done

REQUIRED_GO="$(run_clean "${TOOL_PATH[sed]}" -n \
  's/^toolchain[[:space:]]\+\(go[^[:space:]]*\).*$/\1/p' "$ORQUESTA_SOURCE/go.mod")"
REQUIRED_RUST="$(run_clean "${TOOL_PATH[sed]}" -n \
  's/^[[:space:]]*channel[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' \
  "$AGENT_SOURCE/rust-toolchain.toml")"
GO_VERSION="$("${TOOL_PATH[env]}" -i GOENV=off GOTOOLCHAIN=local \
  HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$GO_BIN" version)"
CARGO_VERSION="$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$CARGO_BIN" --version)"
RUST_VERSION="$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$RUSTC_BIN" --version)"
[[ "$GO_VERSION" == *" $REQUIRED_GO "* ]] || die "go_version_incompatible"
[[ "$CARGO_VERSION" == "cargo $REQUIRED_RUST "* ]] || die "cargo_version_incompatible"
[[ "$RUST_VERSION" == "rustc $REQUIRED_RUST "* ]] || die "rustc_version_incompatible"

# Procedencia efectiva acotada: entrypoints, herramientas internas ejecutables
# de Go y driver/linkers de Rust. Las librerías completas de GOROOT/sysroot se
# declaran después como dependencias no selladas; PFC-01 no pretende ser B12.
GO_ROOT="$("${TOOL_PATH[env]}" -i GOENV=off GOTOOLCHAIN=local HOME="$HOME_DIR" \
  PATH="$TOOL_BIN" "$GO_BIN" env GOROOT)"
GO_TOOL_DIR="$("${TOOL_PATH[env]}" -i GOENV=off GOTOOLCHAIN=local HOME="$HOME_DIR" \
  PATH="$TOOL_BIN" "$GO_BIN" env GOTOOLDIR)"
GO_ROOT="$("${TOOL_PATH[readlink]}" -f -- "$GO_ROOT")"
GO_TOOL_DIR="$("${TOOL_PATH[readlink]}" -f -- "$GO_TOOL_DIR")"
[[ -d "$GO_ROOT" && ! -L "$GO_ROOT" && -d "$GO_TOOL_DIR" && ! -L "$GO_TOOL_DIR" ]] ||
  die "go_toolchain_invalido"
relative_to_root "$GO_TOOL_DIR" "$GO_ROOT" "go_tooldir_fuera_goroot" >/dev/null
declare -a GO_TOOL_FILES=() GO_TOOL_REFS=() GO_TOOL_DIGESTS=() GO_TOOL_VERSIONS=()
mapfile -t GO_TOOL_FILES < <("${TOOL_PATH[find]}" "$GO_TOOL_DIR" -maxdepth 1 \
  -type f -executable -print | "${TOOL_PATH[sort]}")
((${#GO_TOOL_FILES[@]} > 0)) || die "go_tools_vacios"
for tool_path in "${GO_TOOL_FILES[@]}"; do
  [[ ! -L "$tool_path" ]] || die "go_tool_symlink"
  GO_TOOL_REFS+=("$(relative_to_root "$tool_path" "$GO_ROOT" "go_tool_fuera_goroot")")
  GO_TOOL_DIGESTS+=("$(digest "$tool_path")")
  GO_TOOL_VERSIONS+=("$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
    "$tool_path" -V=full 2>&1)")
done

RUST_SYSROOT="$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$RUSTC_BIN" --print sysroot)"
RUST_SYSROOT="$("${TOOL_PATH[readlink]}" -f -- "$RUST_SYSROOT")"
RUST_HOST="$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$RUSTC_BIN" -vV | "${TOOL_PATH[sed]}" -n 's/^host: //p')"
RUST_TARGET_LIBDIR="$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" PATH="$TOOL_BIN" \
  "$RUSTC_BIN" --print target-libdir --target x86_64-unknown-linux-musl)"
RUST_TARGET_LIBDIR="$("${TOOL_PATH[readlink]}" -f -- "$RUST_TARGET_LIBDIR")"
[[ -n "$RUST_HOST" && -d "$RUST_SYSROOT" && ! -L "$RUST_SYSROOT" &&
  -d "$RUST_TARGET_LIBDIR" && ! -L "$RUST_TARGET_LIBDIR" ]] ||
  die "rust_toolchain_invalido"
RUST_SYSROOT_REF="toolchain/$("${TOOL_PATH[basename]}" -- "$RUST_SYSROOT")"
RUST_TARGET_LIBDIR_REF="$(relative_to_root "$RUST_TARGET_LIBDIR" "$RUST_SYSROOT" \
  "rust_target_fuera_sysroot")"
declare -a RUST_LINKER_FILES=(
  "$RUST_SYSROOT/lib/rustlib/$RUST_HOST/bin/rust-lld"
  "$RUST_SYSROOT/lib/rustlib/$RUST_HOST/bin/gcc-ld/ld.lld"
)
declare -a RUST_LINKER_REFS=() RUST_LINKER_DIGESTS=() RUST_LINKER_VERSIONS=()
for linker_path in "${RUST_LINKER_FILES[@]}"; do
  absolute_executable "$linker_path" "rust_linker_invalido"
  RUST_LINKER_REFS+=("$(relative_to_root "$linker_path" "$RUST_SYSROOT" \
    "rust_linker_fuera_sysroot")")
  RUST_LINKER_DIGESTS+=("$(digest "$linker_path")")
  if [[ "$linker_path" == */rust-lld ]]; then
    RUST_LINKER_VERSIONS+=("$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" \
      PATH="$TOOL_BIN" "$linker_path" -flavor gnu --version 2>&1)")
  else
    RUST_LINKER_VERSIONS+=("$("${TOOL_PATH[env]}" -i HOME="$HOME_DIR" \
      PATH="$TOOL_BIN" "$linker_path" --version 2>&1)")
  fi
done

revalidate_effective_toolchain() {
  local index
  for index in "${!GO_TOOL_FILES[@]}"; do
    [[ "$(digest "${GO_TOOL_FILES[$index]}")" == "${GO_TOOL_DIGESTS[$index]}" ]] ||
      die "go_tool_cambio_${GO_TOOL_REFS[$index]}"
  done
  for index in "${!RUST_LINKER_FILES[@]}"; do
    [[ "$(digest "${RUST_LINKER_FILES[$index]}")" == \
      "${RUST_LINKER_DIGESTS[$index]}" ]] ||
      die "rust_linker_cambio_${RUST_LINKER_REFS[$index]}"
  done
}

EXECUTOR="$STAGE_DIR/orquesta-codex-work-executor"
GUEST="$STAGE_DIR/agente-microvm-huesped"
GO_CACHE="$WORK_DIR/go-cache"
GO_PATH="$WORK_DIR/go-path"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$GO_CACHE" "$GO_PATH"
(
  cd -- "$ORQUESTA_SOURCE"
  "${TOOL_PATH[env]}" -i CGO_ENABLED=0 GOAMD64=v1 GOARCH=amd64 GOENV=off \
    GOFLAGS=-mod=vendor \
    GOCACHE="$GO_CACHE" GOMODCACHE="$WORK_DIR/go-mod-cache" GOPATH="$GO_PATH" \
    GOPROXY=off GOSUMDB=off GOOS=linux GOTOOLCHAIN=local HOME="$HOME_DIR" \
    LANG=C LC_ALL=C PATH="$TOOL_BIN" TMPDIR="$BUILD_TMP" TZ=UTC \
    "$GO_BIN" build -trimpath -buildvcs=false -ldflags='-s -w -buildid=' \
    -o "$EXECUTOR" ./cmd/orquesta-codex-work-executor
)
"${TOOL_PATH[chmod]}" 0500 -- "$EXECUTOR"

CARGO_SEED_HOME="$WORK_DIR/cargo-seed-home"
CARGO_METADATA="$WORK_DIR/cargo-metadata.json"
CARGO_SOURCE_PATHS="$WORK_DIR/cargo-source-paths.txt"
CARGO_VENDOR="$WORK_DIR/cargo-vendor"
CARGO_HOME="$WORK_DIR/cargo-home"
CARGO_TARGET="$WORK_DIR/cargo-target"
"${TOOL_PATH[mkdir]}" -m 0700 -- "$CARGO_SEED_HOME" "$CARGO_VENDOR" \
  "$CARGO_HOME" "$CARGO_TARGET"
"${TOOL_PATH[ln]}" -s -- "$CARGO_REGISTRY" "$CARGO_SEED_HOME/registry"
(
  cd -- "$GUEST_BUILD_SOURCE"
  "${TOOL_PATH[env]}" -i CARGO_HOME="$CARGO_SEED_HOME" CARGO_NET_OFFLINE=true \
    HOME="$HOME_DIR" LANG=C LC_ALL=C PATH="$TOOL_BIN" RUSTC="$RUSTC_BIN" \
    TMPDIR="$BUILD_TMP" TZ=UTC "$CARGO_BIN" metadata --offline \
    --filter-platform x86_64-unknown-linux-musl --format-version 1 >/dev/null
)
"${TOOL_PATH[chmod]}" 0400 -- "$GUEST_BUILD_SOURCE/Cargo.lock"
SOURCE_CARGO_LOCK_DIGEST="$(digest "$AGENT_SOURCE/Cargo.lock")"
CARGO_LOCK_DIGEST="$(digest "$GUEST_BUILD_SOURCE/Cargo.lock")"
EFFECTIVE_CARGO_LOCK="$STAGE_DIR/agente-microvm-huesped.Cargo.lock"
"${TOOL_PATH[install]}" -m 0400 -- "$GUEST_BUILD_SOURCE/Cargo.lock" \
  "$EFFECTIVE_CARGO_LOCK"
(
  cd -- "$GUEST_BUILD_SOURCE"
  "${TOOL_PATH[env]}" -i CARGO_HOME="$CARGO_SEED_HOME" CARGO_NET_OFFLINE=true \
    HOME="$HOME_DIR" LANG=C LC_ALL=C PATH="$TOOL_BIN" RUSTC="$RUSTC_BIN" \
    TMPDIR="$BUILD_TMP" TZ=UTC "$CARGO_BIN" metadata --locked --offline \
    --filter-platform x86_64-unknown-linux-musl --format-version 1 > "$CARGO_METADATA"
)
# El cierre recursivo parte solo del paquete guest y fija las fuentes registry
# que su grafo efectivo para musl puede consumir.
# shellcheck disable=SC2016
run_clean "${TOOL_PATH[jq]}" -er '
  . as $doc
  | ($doc.packages[] | select(.name == "agente_microvm_huesped") | .id) as $root
  | ($doc.resolve.nodes | map({key:.id,value:.}) | from_entries) as $nodes
  | def walk($id): [$id] + [($nodes[$id].deps[]?.pkg) as $dep | walk($dep)] | flatten;
  [walk($root)[]] | unique as $ids
  | $doc.packages[]
  | select(.id as $id | $ids | index($id))
  | select(.source != null and (.source | startswith("registry+")))
  | .manifest_path | rtrimstr("/Cargo.toml")
' "$CARGO_METADATA" | "${TOOL_PATH[sort]}" -u > "$CARGO_SOURCE_PATHS"
mapfile -t CARGO_SOURCES < "$CARGO_SOURCE_PATHS"
((${#CARGO_SOURCES[@]} > 0)) || die "cargo_snapshot_vacio"
CARGO_ARCHIVES_JSON='[]'
for cargo_source in "${CARGO_SOURCES[@]}"; do
  cargo_source="$("${TOOL_PATH[readlink]}" -f -- "$cargo_source")"
  [[ "$cargo_source" == "$CARGO_REGISTRY"/src/* && -d "$cargo_source" &&
    ! -L "$cargo_source" ]] || die "cargo_source_fuera_registry"
  [[ -z "$("${TOOL_PATH[find]}" "$cargo_source" -type l -print -quit)" ]] ||
    die "cargo_source_symlink"
  [[ -z "$("${TOOL_PATH[find]}" "$cargo_source" ! -type d ! -type f -print -quit)" ]] ||
    die "cargo_source_especial"
  cargo_name="$("${TOOL_PATH[basename]}" -- "$cargo_source")"
  cargo_destination="$CARGO_VENDOR/$cargo_name"
  [[ ! -e "$cargo_destination" && ! -L "$cargo_destination" ]] ||
    die "cargo_source_duplicada"
  cargo_registry_namespace="$("${TOOL_PATH[basename]}" -- \
    "$("${TOOL_PATH[dirname]}" -- "$cargo_source")")"
  cargo_archive="$CARGO_REGISTRY/cache/$cargo_registry_namespace/$cargo_name.crate"
  absolute_regular "$cargo_archive" "cargo_archive_invalido"
  cargo_archive_digest="$(digest "$cargo_archive")"
  "${TOOL_PATH[mkdir]}" -m 0700 -- "$cargo_destination"
  run_clean "${TOOL_PATH[tar]}" -xf "$cargo_archive" -C "$cargo_destination" \
    --strip-components=1 --no-same-owner --no-same-permissions
  absolute_regular "$cargo_destination/Cargo.toml" "cargo_archive_sin_manifiesto"
  [[ -z "$("${TOOL_PATH[find]}" "$cargo_destination" -type l -print -quit)" ]] ||
    die "cargo_archive_symlink"
  [[ -z "$("${TOOL_PATH[find]}" "$cargo_destination" \
    ! -type d ! -type f -print -quit)" ]] || die "cargo_archive_especial"
  # Cargo coteja `package` con el checksum del lock. Los bytes compilados se
  # extraen del mismo .crate, no del árbol mutable registry/src.
  # shellcheck disable=SC2016
  run_clean "${TOOL_PATH[jq]}" -nS --arg package "$cargo_archive_digest" \
    '{files:{},package:$package}' > \
    "$cargo_destination/.cargo-checksum.json"
  # shellcheck disable=SC2016
  CARGO_ARCHIVES_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS \
    --argjson current "$CARGO_ARCHIVES_JSON" --arg package "$cargo_name" \
    --arg sha256 "$cargo_archive_digest" \
    '$current + [{package:$package,sha256:$sha256}]')"
done
"${TOOL_PATH[unlink]}" -- "$CARGO_SEED_HOME/registry"
while IFS= read -r -d '' cargo_directory; do
  "${TOOL_PATH[chmod]}" 0500 -- "$cargo_directory"
done < <("${TOOL_PATH[find]}" "$CARGO_VENDOR" -type d -print0)
while IFS= read -r -d '' cargo_file; do
  "${TOOL_PATH[chmod]}" 0400 -- "$cargo_file"
done < <("${TOOL_PATH[find]}" "$CARGO_VENDOR" -type f -print0)
CARGO_SNAPSHOT_DIGEST="$(tree_digest "$CARGO_VENDOR")"
printf '%s\n' \
  '[source.crates-io]' \
  'replace-with = "pfc01-vendored"' \
  '[source.pfc01-vendored]' \
  "directory = \"$CARGO_VENDOR\"" > "$CARGO_HOME/config.toml"
"${TOOL_PATH[chmod]}" 0400 -- "$CARGO_HOME/config.toml"
RUST_REMAP_FLAG="--remap-path-prefix=$WORK_DIR=/orquesta-pfc01"
RUSTC_WRAPPER_PATH="$WORK_DIR/rustc-pfc01-wrapper"
# El source-id de paquetes path incluye su ruta temporal. El wrapper sustituye
# solo `-C metadata` de los dos crates locales; conserva extra-filename y todos
# los argumentos que Cargo usa para dependencias registry.
# Variables pertenecen al wrapper generado.
# shellcheck disable=SC2016
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'set -Eeuo pipefail' \
  'rustc="${1:-}"' \
  '[[ "$rustc" == /* && -x "$rustc" ]] || exit 96' \
  'shift' \
  'args=("$@")' \
  'crate_name=""' \
  'for ((i = 0; i < ${#args[@]}; i++)); do' \
  '  if [[ "${args[$i]}" == "--crate-name" && $((i + 1)) -lt ${#args[@]} ]]; then' \
  '    crate_name="${args[$((i + 1))]}"' \
  '    break' \
  '  fi' \
  'done' \
  'case "$crate_name" in' \
  '  agente_microvm_huesped) stable="pfc01-guest-v0-1-0" ;;' \
  '  contrato_huesped) stable="pfc01-contract-v0-1-0" ;;' \
  '  *) exec "$rustc" "${args[@]}" ;;' \
  'esac' \
  'replaced=0' \
  'for ((i = 0; i < ${#args[@]}; i++)); do' \
  '  if [[ "${args[$i]}" == "-C" && $((i + 1)) -lt ${#args[@]} &&' \
  '    "${args[$((i + 1))]}" == metadata=* ]]; then' \
  '    args[$((i + 1))]="metadata=$stable"' \
  '    replaced=$((replaced + 1))' \
  '  elif [[ "${args[$i]}" == -Cmetadata=* ]]; then' \
  '    args[$i]="-Cmetadata=$stable"' \
  '    replaced=$((replaced + 1))' \
  '  fi' \
  'done' \
  '[[ "$replaced" == 1 ]] || exit 97' \
  'exec "$rustc" "${args[@]}"' > "$RUSTC_WRAPPER_PATH"
"${TOOL_PATH[chmod]}" 0500 -- "$RUSTC_WRAPPER_PATH"
RUSTC_WRAPPER_DIGEST="$(digest "$RUSTC_WRAPPER_PATH")"
(
  cd -- "$GUEST_BUILD_SOURCE"
  "${TOOL_PATH[env]}" -i CARGO_ENCODED_RUSTFLAGS="$RUST_REMAP_FLAG" \
    CARGO_HOME="$CARGO_HOME" \
    CARGO_INCREMENTAL=0 \
    CARGO_NET_OFFLINE=true CARGO_TARGET_DIR="$CARGO_TARGET" HOME="$HOME_DIR" \
    LANG=C LC_ALL=C PATH="$TOOL_BIN" RUSTC="$RUSTC_BIN" \
    RUSTC_WRAPPER="$RUSTC_WRAPPER_PATH" \
    RUSTC_WORKSPACE_WRAPPER='' RUSTDOCFLAGS='' RUSTFLAGS='' SOURCE_DATE_EPOCH=0 \
    TMPDIR="$BUILD_TMP" TZ=UTC "$CARGO_BIN" build --locked --offline --release \
    --target x86_64-unknown-linux-musl -p agente_microvm_huesped
)
[[ "$(tree_digest "$CARGO_VENDOR")" == "$CARGO_SNAPSHOT_DIGEST" ]] ||
  die "cargo_snapshot_cambio"
"${TOOL_PATH[install]}" -m 0500 -- \
  "$CARGO_TARGET/x86_64-unknown-linux-musl/release/agente-microvm-huesped" "$GUEST"
for built in "$EXECUTOR" "$GUEST"; do
  run_clean "${TOOL_PATH[file]}" -- "$built" |
    run_clean "${TOOL_PATH[grep]}" -Fq 'ELF 64-bit LSB' || die "binario_construido_no_elf64"
  run_clean "${TOOL_PATH[readelf]}" -h -- "$built" |
    run_clean "${TOOL_PATH[grep]}" -Fq 'Advanced Micro Devices X86-64' ||
    die "binario_construido_arquitectura_invalida"
  if run_clean "${TOOL_PATH[readelf]}" -lW -- "$built" |
    run_clean "${TOOL_PATH[grep]}" -F 'INTERP' >/dev/null ||
    run_clean "${TOOL_PATH[readelf]}" -dW -- "$built" 2>/dev/null |
    run_clean "${TOOL_PATH[grep]}" -F '(NEEDED)' >/dev/null; then
    die "binario_construido_no_estatico"
  fi
done

KERNEL_NAME="$(run_clean "${TOOL_PATH[jq]}" -er '.nombre' "$KERNEL_MANIFEST")"
KERNEL_DIGEST="$(run_clean "${TOOL_PATH[jq]}" -er '.sha256' "$KERNEL_MANIFEST")"
KERNEL_SIZE="$(run_clean "${TOOL_PATH[jq]}" -er '.tamano_bytes' "$KERNEL_MANIFEST")"
KERNEL_CONFIG_SOURCE="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.configuracion_fuente' "$KERNEL_MANIFEST")"
KERNEL_CONFIG_DIGEST="$(run_clean "${TOOL_PATH[jq]}" -er \
  '.configuracion_sha256' "$KERNEL_MANIFEST")"
KERNEL="$STAGE_DIR/$KERNEL_NAME"
if [[ -n "$KERNEL_INPUT" ]]; then
  [[ "$(digest "$KERNEL_INPUT")" == "$KERNEL_DIGEST" &&
    "$(size "$KERNEL_INPUT")" == "$KERNEL_SIZE" ]] || die "kernel_no_fijado"
  "${TOOL_PATH[install]}" -m 0400 -- "$KERNEL_INPUT" "$KERNEL"
  KERNEL_ORIGIN="provided_verified"
else
  run_clean "${TOOL_PATH[bash]}" "$KERNEL_PREPARER" --directorio "$STAGE_DIR"
  KERNEL_ORIGIN="downloaded_verified"
fi
[[ "$(digest "$KERNEL")" == "$KERNEL_DIGEST" && "$(size "$KERNEL")" == "$KERNEL_SIZE" ]] ||
  die "kernel_preparado_invalido"
"${TOOL_PATH[install]}" -m 0400 -- "$KERNEL_MANIFEST" "$STAGE_DIR/kernel-manifest.json"
[[ "$(digest "$STAGE_DIR/kernel-manifest.json")" == "$(digest "$KERNEL_MANIFEST")" ]] ||
  die "kernel_manifiesto_cambio"

INITRAMFS="$STAGE_DIR/agente-microvm-initramfs.cpio.gz"
run_clean "${TOOL_PATH[bash]}" "$INITRAMFS_BUILDER" --huesped "$GUEST" \
  --busybox "$BUSYBOX_INPUT" --salida "$INITRAMFS" --temporal "$WORK_DIR"

EXECUTOR_DIGEST="$(digest "$EXECUTOR")"
EXECUTOR_SIZE="$(size "$EXECUTOR")"
GUEST_DIGEST="$(digest "$GUEST")"
GUEST_SIZE="$(size "$GUEST")"
ORQUESTA_EPOCH="$(run_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" \
  show -s --format=%ct "$ORQUESTA_COMMIT")"
AGENT_EPOCH="$(run_clean "${TOOL_PATH[git]}" -C "$AGENT_REPO" \
  show -s --format=%ct "$AGENT_COMMIT")"
SOURCE_EPOCH="$ORQUESTA_EPOCH"
((AGENT_EPOCH > SOURCE_EPOCH)) && SOURCE_EPOCH="$AGENT_EPOCH"
CREATED="$(run_clean "${TOOL_PATH[date]}" -u -d "@$SOURCE_EPOCH" \
  '+%Y-%m-%dT%H:%M:%SZ')"
PROFILE_MANIFEST="$STAGE_DIR/perfil-manifest.json"
# jq recibe programa literal; los $nombres pertenecen a jq, no al shell.
# shellcheck disable=SC2016
run_clean "${TOOL_PATH[jq]}" -S \
  --arg agent_commit "$AGENT_COMMIT" --arg agent_digest "$GUEST_DIGEST" \
  --arg agent_source "https://github.com/aavidad/agente_microvm/tree/$AGENT_COMMIT" \
  --arg created "$CREATED" --arg executor_commit "$ORQUESTA_COMMIT" \
  --arg executor_digest "$EXECUTOR_DIGEST" \
  --arg executor_source "https://github.com/aavidad/orquestaV2/tree/$ORQUESTA_COMMIT/cmd/orquesta-codex-work-executor" \
  --arg namespace "https://github.com/aavidad/agente_microvm/sbom/pfc01-${GUEST_DIGEST:0:16}-${EXECUTOR_DIGEST:0:16}" \
  --argjson agent_size "$GUEST_SIZE" --argjson executor_size "$EXECUTOR_SIZE" '
    .ejecutor.version = $executor_commit
    | .ejecutor.commit = $executor_commit
    | .ejecutor.fuente = $executor_source
    | .ejecutor.sha256 = $executor_digest
    | .ejecutor.tamano_bytes = $executor_size
    | .sbom.documentNamespace = $namespace
    | .sbom.creationInfo.created = $created
    | .sbom.packages |= map(
        if .SPDXID == "SPDXRef-Package-Huesped" then
          .versionInfo = $agent_commit | .downloadLocation = $agent_source
          | .checksums = [{"algorithm":"SHA256","checksumValue":$agent_digest}]
        elif .SPDXID == "SPDXRef-Package-OrquestaCodexWorkExecutor" then
          .versionInfo = $executor_commit | .downloadLocation = $executor_source
          | .checksums = [{"algorithm":"SHA256","checksumValue":$executor_digest}]
        else . end
      )
  ' "$PROFILE_TEMPLATE" > "$PROFILE_MANIFEST"
"${TOOL_PATH[chmod]}" 0400 -- "$PROFILE_MANIFEST"

PROFILE="$STAGE_DIR/perfil-codex.ext4"
run_clean "${TOOL_PATH[bash]}" "$PROFILE_BUILDER" --manifiesto "$PROFILE_MANIFEST" \
  --agente "$CODEX_INPUT" --ejecutor "$EXECUTOR" --busybox "$BUSYBOX_INPUT" \
  --salida "$PROFILE" --temporal "$WORK_DIR"

version_line() {
  run_clean "$1" "$2" 2>&1 | run_clean "${TOOL_PATH[sed]}" -n '/./{p;q;}'
}

revalidate_tools() {
  local tool current
  for tool in "${TOOLS[@]}"; do
    current="$(digest "${TOOL_PATH[$tool]}")"
    [[ "$current" == "${TOOL_DIGEST[$tool]}" ]] || die "herramienta_cambio_$tool"
  done
  [[ "$(digest "$GO_BIN")" == "$GO_DIGEST" ]] || die "herramienta_cambio_go"
  [[ "$(digest "$CARGO_BIN")" == "$CARGO_DIGEST" ]] || die "herramienta_cambio_cargo"
  [[ "$(digest "$RUSTC_BIN")" == "$RUSTC_DIGEST" ]] || die "herramienta_cambio_rustc"
  revalidate_effective_toolchain
  [[ "$(digest "$AGENT_SOURCE/Cargo.lock")" == "$SOURCE_CARGO_LOCK_DIGEST" &&
    "$(digest "$GUEST_BUILD_SOURCE/Cargo.lock")" == "$CARGO_LOCK_DIGEST" &&
    "$(digest "$EFFECTIVE_CARGO_LOCK")" == "$CARGO_LOCK_DIGEST" ]] ||
    die "cargo_lock_cambio"
  [[ "$(digest "$RUSTC_WRAPPER_PATH")" == "$RUSTC_WRAPPER_DIGEST" ]] ||
    die "rustc_wrapper_cambio"
  [[ "$(tree_digest "$CARGO_VENDOR")" == "$CARGO_SNAPSHOT_DIGEST" ]] ||
    die "cargo_snapshot_cambio"
}

# Los digests se capturaron antes de crear candidatos. Se vuelven a comprobar
# una vez terminados los builds y antes de atribuirles versión en el manifiesto.
revalidate_tools
TOOLS_JSON='{}'
RECORDED_TOOLS=("${TOOLS[@]}" cargo go rustc)
for tool in "${RECORDED_TOOLS[@]}"; do
  case "$tool" in
    go) tool_path="$GO_BIN"; tool_digest="$GO_DIGEST"; tool_version="$GO_VERSION" ;;
    cargo) tool_path="$CARGO_BIN"; tool_digest="$CARGO_DIGEST"; tool_version="$CARGO_VERSION" ;;
    rustc) tool_path="$RUSTC_BIN"; tool_digest="$RUSTC_DIGEST"; tool_version="$RUST_VERSION" ;;
    e2fsck|mke2fs)
      tool_path="${TOOL_PATH[$tool]}"
      tool_digest="${TOOL_DIGEST[$tool]}"
      tool_version="$(version_line "$tool_path" -V)"
      ;;
    *)
      tool_path="${TOOL_PATH[$tool]}"
      tool_digest="${TOOL_DIGEST[$tool]}"
      tool_version="$(version_line "$tool_path" --version)"
      ;;
  esac
  # shellcheck disable=SC2016
  TOOLS_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS --argjson current "$TOOLS_JSON" \
    --arg name "$tool" --arg path_ref "resolved_tools/$tool" \
    --arg sha256 "$tool_digest" --arg version "$tool_version" \
    '$current + {($name):{path_ref:$path_ref,sha256:$sha256,version:$version}}')"
done

GO_INTERNAL_TOOLS_JSON='[]'
for index in "${!GO_TOOL_FILES[@]}"; do
  # shellcheck disable=SC2016
  GO_INTERNAL_TOOLS_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS \
    --argjson current "$GO_INTERNAL_TOOLS_JSON" --arg path_ref "${GO_TOOL_REFS[$index]}" \
    --arg sha256 "${GO_TOOL_DIGESTS[$index]}" --arg version "${GO_TOOL_VERSIONS[$index]}" \
    '$current + [{path_ref:$path_ref,sha256:$sha256,version:$version}]')"
done
RUST_LINKERS_JSON='[]'
for index in "${!RUST_LINKER_FILES[@]}"; do
  # shellcheck disable=SC2016
  RUST_LINKERS_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS \
    --argjson current "$RUST_LINKERS_JSON" --arg path_ref "${RUST_LINKER_REFS[$index]}" \
    --arg sha256 "${RUST_LINKER_DIGESTS[$index]}" \
    --arg version "${RUST_LINKER_VERSIONS[$index]}" \
    '$current + [{path_ref:$path_ref,sha256:$sha256,version:$version}]')"
done
# PFC-01 atribuye entrypoints y snapshots consumidos; el resto del sysroot y
# las bibliotecas dinámicas del host quedan explícitamente fuera del sello B12.
# shellcheck disable=SC2016
EFFECTIVE_TOOLCHAIN_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS \
  --argjson cargo_archives "$CARGO_ARCHIVES_JSON" \
  --arg cargo_lock_sha256 "$CARGO_LOCK_DIGEST" \
  --arg source_cargo_lock_sha256 "$SOURCE_CARGO_LOCK_DIGEST" \
  --arg cargo_sha256 "$CARGO_DIGEST" --arg cargo_version "$CARGO_VERSION" \
  --arg go_sha256 "$GO_DIGEST" --arg go_version "$GO_VERSION" \
  --arg rust_host "$RUST_HOST" --arg rust_sysroot_ref "$RUST_SYSROOT_REF" \
  --arg rust_target_libdir_ref "$RUST_TARGET_LIBDIR_REF" \
  --arg rustc_wrapper_sha256 "$RUSTC_WRAPPER_DIGEST" \
  --arg rustc_sha256 "$RUSTC_DIGEST" --arg rustc_version "$RUST_VERSION" \
  --arg snapshot_sha256 "$CARGO_SNAPSHOT_DIGEST" \
  --argjson cargo_package_count "${#CARGO_SOURCES[@]}" \
  --argjson go_internal_tools "$GO_INTERNAL_TOOLS_JSON" \
  --argjson rust_linkers "$RUST_LINKERS_JSON" '
    {
      claim:"same_environment_reproducibility",
      hermetic_supply_chain:false,
      go:{
        entrypoint:{path_ref:"provided/go",sha256:$go_sha256,version:$go_version},
        internal_tools:$go_internal_tools,
        unsealed:["goroot_library_data","host_dynamic_libraries"]
      },
      rust:{
        cargo:{path_ref:"provided/cargo",sha256:$cargo_sha256,version:$cargo_version},
        driver:{path_ref:"provided/rustc",sha256:$rustc_sha256,version:$rustc_version},
        host:$rust_host,
        sysroot_ref:$rust_sysroot_ref,
        target_libdir_ref:$rust_target_libdir_ref,
        linkers:$rust_linkers,
        wrapper:{
          path_ref:"builder_generated/rustc-pfc01-wrapper",
          scope:["agente_microvm_huesped","contrato_huesped"],
          sha256:$rustc_wrapper_sha256
        },
        unsealed:["target_sysroot_libraries","host_dynamic_libraries"]
      },
      cargo_dependencies:{
        archives:$cargo_archives,
        effective_lock_file:"agente-microvm-huesped.Cargo.lock",
        method:"locked_offline_graph_read_only_snapshot",
        cargo_lock_sha256:$cargo_lock_sha256,
        source_cargo_lock_sha256:$source_cargo_lock_sha256,
        package_count:$cargo_package_count,
        snapshot_sha256:$snapshot_sha256,
        read_only_during_build:true,
        original_index_and_cache_not_consumed_by_build:true
      },
      verification:{required_runs:2,environment:"same_host_and_inputs",performed_by_builder:false}
    }
  ')"

ORQUESTA_PATHS_JSON="$(printf '%s\n' "${ORQUESTA_ARCHIVE_PATHS[@]}" |
  run_clean "${TOOL_PATH[jq]}" -R . | run_clean "${TOOL_PATH[jq]}" -s .)"
AGENT_PATHS_JSON="$(printf '%s\n' "${AGENT_ARCHIVE_PATHS[@]}" |
  run_clean "${TOOL_PATH[jq]}" -R . | run_clean "${TOOL_PATH[jq]}" -s .)"
# shellcheck disable=SC2016
ASSETS_JSON="$(run_clean "${TOOL_PATH[jq]}" -nS \
  --arg bbusy "$(digest "$BUSYBOX_INPUT")" --arg bcodex "$(digest "$CODEX_INPUT")" \
  --arg beffective_lock "$(digest "$EFFECTIVE_CARGO_LOCK")" \
  --arg bexecutor "$(digest "$EXECUTOR")" --arg bguest "$(digest "$GUEST")" \
  --arg binitramfs "$(digest "$INITRAMFS")" --arg bkernel "$(digest "$KERNEL")" \
  --arg bkernel_manifest "$(digest "$STAGE_DIR/kernel-manifest.json")" \
  --arg bprofile "$(digest "$PROFILE")" --arg bprofile_manifest "$(digest "$PROFILE_MANIFEST")" \
  --arg kernel_config_digest "$KERNEL_CONFIG_DIGEST" \
  --arg kernel_config_source "$KERNEL_CONFIG_SOURCE" \
  --arg kernel_file "$KERNEL_NAME" --arg kernel_origin "$KERNEL_ORIGIN" \
  --argjson sbusy "$(size "$BUSYBOX_INPUT")" --argjson scodex "$(size "$CODEX_INPUT")" \
  --argjson seffective_lock "$(size "$EFFECTIVE_CARGO_LOCK")" \
  --argjson sexecutor "$(size "$EXECUTOR")" --argjson sguest "$(size "$GUEST")" \
  --argjson sinitramfs "$(size "$INITRAMFS")" --argjson skernel "$(size "$KERNEL")" \
  --argjson skernel_manifest "$(size "$STAGE_DIR/kernel-manifest.json")" \
  --argjson sprofile "$(size "$PROFILE")" --argjson sprofile_manifest "$(size "$PROFILE_MANIFEST")" '
    {
      busybox:{source:"input_snapshot",sha256:$bbusy,size_bytes:$sbusy},
      codex:{source:"input_snapshot",sha256:$bcodex,size_bytes:$scodex},
      effective_cargo_lock:{
        file:"agente-microvm-huesped.Cargo.lock",
        sha256:$beffective_lock,
        size_bytes:$seffective_lock
      },
      executor:{file:"orquesta-codex-work-executor",sha256:$bexecutor,size_bytes:$sexecutor},
      guest:{file:"agente-microvm-huesped",sha256:$bguest,size_bytes:$sguest},
      initramfs:{file:"agente-microvm-initramfs.cpio.gz",sha256:$binitramfs,size_bytes:$sinitramfs},
      kernel:{
        file:$kernel_file,
        origin:$kernel_origin,
        sha256:$bkernel,
        size_bytes:$skernel,
        configuration_reference:{
          active:false,
          source:$kernel_config_source,
          sha256:$kernel_config_digest,
          reason:"referencia_de_procedencia_no_consumida_por_este_constructor"
        }
      },
      kernel_manifest:{file:"kernel-manifest.json",sha256:$bkernel_manifest,size_bytes:$skernel_manifest},
      profile:{file:"perfil-codex.ext4",sha256:$bprofile,size_bytes:$sprofile},
      profile_manifest:{file:"perfil-manifest.json",sha256:$bprofile_manifest,size_bytes:$sprofile_manifest}
    }
  ')"

MANIFEST="$STAGE_DIR/candidate-manifest.json"
# shellcheck disable=SC2016
run_clean "${TOOL_PATH[jq]}" -nS \
  --arg agent_commit "$AGENT_COMMIT" --arg agent_tree "$AGENT_TREE" \
  --arg builder_digest "$BUILDER_DIGEST" --arg orquesta_commit "$ORQUESTA_COMMIT" \
  --arg orquesta_tree "$ORQUESTA_TREE" --argjson agent_paths "$AGENT_PATHS_JSON" \
  --argjson assets "$ASSETS_JSON" --argjson orquesta_paths "$ORQUESTA_PATHS_JSON" \
  --argjson source_epoch "$SOURCE_EPOCH" --argjson toolchain "$EFFECTIVE_TOOLCHAIN_JSON" \
  --argjson tools "$TOOLS_JSON" '
    {
      schema:"orquesta.pfc01_agent_microvm_codex_assets.v1",
      status:"built_not_physically_exercised",
      architecture:"x86_64",
      source_date_epoch:$source_epoch,
      sources:{
        orquesta:{commit:$orquesta_commit,tree:$orquesta_tree,archive_paths:$orquesta_paths},
        agente_microvm:{commit:$agent_commit,tree:$agent_tree,archive_paths:$agent_paths}
      },
      builder:{file:"scripts/build_agent_microvm_codex_assets.sh",sha256:$builder_digest,tracked_head:true},
      environment:{
        source_commands:"env_i",
        helper_scripts:"env_i",
        go_build:"env_i",
        cargo_build:"env_i",
        filesystem_tools:"resolved_absolute_paths",
        goamd64:"v1",
        goenv:"off",
        gotoolchain:"local",
        cargo_offline:true,
        rustflags:[
          "--remap-path-prefix=WORK_DIR=/orquesta-pfc01"
        ],
        rustc_wrapper:"builder_generated_selective_metadata"
      },
      tools:$tools,
      effective_toolchain:$toolchain,
      assets:$assets,
      physical_evidence:{kvm:false,firecracker:false,codex_session:false}
    }
  ' > "$MANIFEST"
"${TOOL_PATH[chmod]}" 0400 -- "$MANIFEST"

for key in executor guest initramfs kernel profile profile_manifest; do
  # shellcheck disable=SC2016
  file_name="$(run_clean "${TOOL_PATH[jq]}" -er --arg key "$key" \
    '.assets[$key].file' "$MANIFEST")"
  # shellcheck disable=SC2016
  expected="$(run_clean "${TOOL_PATH[jq]}" -er --arg key "$key" \
    '.assets[$key].sha256' "$MANIFEST")"
  (cd -- "$STAGE_DIR" && printf '%s  %s\n' "$expected" "$file_name" |
    "${TOOL_PATH[sha256sum]}" --check --status) || die "manifiesto_no_verifica_$key"
done
# shellcheck disable=SC2016
run_clean "${TOOL_PATH[jq]}" -e --arg digest \
  "$(digest "$STAGE_DIR/kernel-manifest.json")" \
  '.assets.kernel_manifest.sha256 == $digest' "$MANIFEST" >/dev/null ||
  die "kernel_manifiesto_no_ligado"
[[ "$(digest "$SCRIPT_PATH")" == "$BUILDER_DIGEST" ]] || die "constructor_cambio_durante_build"

"${TOOL_PATH[chmod]}" 0500 -- "$EXECUTOR" "$GUEST"
"${TOOL_PATH[chmod]}" 0400 -- "$INITRAMFS" "$KERNEL" "$PROFILE" "$PROFILE_MANIFEST" \
  "$STAGE_DIR/kernel-manifest.json" "$MANIFEST"
MANIFEST_DIGEST="$(digest "$MANIFEST")"
[[ "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$STAGE_DIR")" == "$STAGE_ID" ]] ||
  die "staging_sustituido_antes_publicacion"
# La propiedad queda cercada antes del rename: si llega una señal justo al
# retornar mv, cleanup solo puede retirar el inode que creó este proceso.
OUTPUT_ID="$STAGE_ID"
OUTPUT_OWNED=1
revalidate_tools
"${TOOL_PATH[mv]}" -T --no-clobber -- "$STAGE_DIR" "$OUTPUT" || die "publicacion_fallida"
if [[ ! -e "$STAGE_DIR" && -d "$OUTPUT" && ! -L "$OUTPUT" &&
  "$("${TOOL_PATH[stat]}" -c '%d:%i' -- "$OUTPUT")" == "$OUTPUT_ID" ]]; then
  :
else
  die "publicacion_incierta"
fi
[[ "$(digest "$OUTPUT/candidate-manifest.json")" == "$MANIFEST_DIGEST" ]] ||
  die "publicacion_manifiesto_invalida"
SUCCESS=1
printf 'PFC01_ACTIVOS estado=construido_no_fisico manifiesto_sha256=%s salida=%s\n' \
  "$MANIFEST_DIGEST" "$OUTPUT"
