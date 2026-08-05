#!/usr/bin/env bash
set -Eeuo pipefail

umask 077
export LC_ALL=C

die() {
  printf 'pfc01_activos_test: %s\n' "$1" >&2
  exit 1
}

usage() {
  cat >&2 <<'EOF'
Uso: test_build_agent_microvm_codex_assets.sh [--cheap] \
  --agente-microvm-repo RUTA_ABSOLUTA \
  --codex RUTA_ABSOLUTA --busybox RUTA_ABSOLUTA \
  --go RUTA_ABSOLUTA --cargo RUTA_ABSOLUTA --rustc RUTA_ABSOLUTA \
  --cargo-registry RUTA_ABSOLUTA --temporal RUTA_ABSOLUTA \
  [--kernel RUTA_ABSOLUTA]

--cheap valida sintaxis, allowlists y la guarda tracked/HEAD sin builds, red ni
temporales productivos. El gate completo requiere builder confirmado y kernel
fijado; cubre fallo, cancelación, padre readonly y descarga con curl fake.
EOF
}

CHEAP=0
AGENT_REPO=""
CODEX=""
BUSYBOX=""
GO_BIN=""
CARGO_BIN=""
RUSTC_BIN=""
CARGO_REGISTRY=""
TMP_PARENT=""
KERNEL=""
while (($# > 0)); do
  case "$1" in
    --cheap) CHEAP=1; shift ;;
    --agente-microvm-repo) AGENT_REPO="${2:-}"; shift 2 ;;
    --codex) CODEX="${2:-}"; shift 2 ;;
    --busybox) BUSYBOX="${2:-}"; shift 2 ;;
    --go) GO_BIN="${2:-}"; shift 2 ;;
    --cargo) CARGO_BIN="${2:-}"; shift 2 ;;
    --rustc) RUSTC_BIN="${2:-}"; shift 2 ;;
    --cargo-registry) CARGO_REGISTRY="${2:-}"; shift 2 ;;
    --temporal) TMP_PARENT="${2:-}"; shift 2 ;;
    --kernel) KERNEL="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "argumento_no_admitido" ;;
  esac
done

[[ "$AGENT_REPO" == /* && "$CODEX" == /* && "$BUSYBOX" == /* && "$GO_BIN" == /* &&
  "$CARGO_BIN" == /* && "$RUSTC_BIN" == /* && "$CARGO_REGISTRY" == /* &&
  "$TMP_PARENT" == /* ]] || die "argumentos_incompletos"
[[ -d "$TMP_PARENT" && ! -L "$TMP_PARENT" ]] || die "temporal_invalido"
[[ "$(stat -c '%u' -- "$TMP_PARENT")" == "$(id -u)" ]] || die "temporal_propietario"
(( (8#$(stat -c '%a' -- "$TMP_PARENT") & 8#077) == 0 )) || die "temporal_modo"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd -P)"
BUILDER="$SCRIPT_DIR/build_agent_microvm_codex_assets.sh"
bash -n "$BUILDER"
bash -n "${BASH_SOURCE[0]}"

# Las expansiones son texto literal que debe existir en el constructor.
# shellcheck disable=SC2016
for contract in \
  'run_preflight_clean "${TOOL_PATH[git]}" -C "$ORQUESTA_REPO" ls-files --error-unmatch' \
  '"${TOOL_PATH[cmp]}" -s - "$SCRIPT_PATH"' \
  'readonly -a ORQUESTA_ARCHIVE_PATHS=(' \
  'readonly -a AGENT_ARCHIVE_PATHS=(' \
  '"${ORQUESTA_ARCHIVE_PATHS[@]}"' \
  '"${AGENT_ARCHIVE_PATHS[@]}"' \
  '"${TOOL_PATH[env]}" -i CGO_ENABLED=0 GOAMD64=v1' \
  'CARGO_NET_OFFLINE=true' \
  'WORK_ID=' \
  'STAGE_ID=' \
  'trap cleanup EXIT' \
  'OUTPUT_ID="$STAGE_ID"' \
  'OUTPUT_OWNED=1' \
  'remove_owned_tree "$OUTPUT" "$OUTPUT_PARENT" "$OUTPUT_ID"' \
  'revalidate_tools' \
  'configuration_reference:{'; do
  grep -Fq -- "$contract" "$BUILDER" || die "contrato_estatico_ausente"
done
! grep -Fq -- '--prueba-senal-tras-publicacion' "$BUILDER" ||
  die "hook_senal_en_builder_productivo"
first_trap_line="$(grep -n -m1 '^trap cleanup EXIT$' "$BUILDER" | cut -d: -f1)"
first_temp_line="$(grep -n -m1 '^create_owned_temp_dir WORK_DIR ' "$BUILDER" | cut -d: -f1)"
[[ "$first_trap_line" =~ ^[0-9]+$ && "$first_temp_line" =~ ^[0-9]+$ &&
  "$first_trap_line" -lt "$first_temp_line" ]] || die "trap_posterior_a_temporal"
# shellcheck disable=SC2016
for forbidden_archive in 'git archive --format=tar "$ORQUESTA_COMMIT" |' \
  'git archive --format=tar "$AGENT_COMMIT" |'; do
  ! grep -Fq -- "$forbidden_archive" "$BUILDER" || die "archive_amplio_detectado"
done
executor_archive_count="$(sed -n '/readonly -a ORQUESTA_ARCHIVE_PATHS=(/,/^)/p' "$BUILDER" |
  grep -c '^  cmd/orquesta-codex-work-executor$')"
[[ "$executor_archive_count" == 1 ]] || die "dependencia_executor_duplicada"

ROOT="$(mktemp -d "$TMP_PARENT/pfc01-activos-test.XXXXXX")"
chmod 0700 -- "$ROOT"
cleanup() {
  local status=$? cleanup_failed=0
  trap - EXIT INT TERM
  set +e
  if [[ -d "$ROOT" && ! -L "$ROOT" && "$ROOT" == "$TMP_PARENT"/pfc01-activos-test.* ]]; then
    chmod -R u+w -- "$ROOT" || cleanup_failed=1
    rm -rf --one-file-system -- "$ROOT" || cleanup_failed=1
  fi
  if ((status == 0 && cleanup_failed != 0)); then
    return 1
  fi
  return "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

builder_args=(
  --agente-microvm-repo "$AGENT_REPO"
  --codex "$CODEX"
  --busybox "$BUSYBOX"
  --go "$GO_BIN"
  --cargo "$CARGO_BIN"
  --rustc "$RUSTC_BIN"
  --cargo-registry "$CARGO_REGISTRY"
  --temporal "$ROOT"
)

if ! git -C "$REPO_DIR" ls-files --error-unmatch -- \
  scripts/build_agent_microvm_codex_assets.sh >/dev/null 2>&1; then
  if "$BUILDER" "${builder_args[@]}" --salida "$ROOT/untracked" \
    >"$ROOT/untracked.out" 2>"$ROOT/untracked.err"; then
    die "constructor_untracked_aceptado"
  fi
  grep -Fxq 'error.pfc01.activos: constructor_no_trackeado' "$ROOT/untracked.err" ||
    die "codigo_untracked_incorrecto"
  [[ ! -e "$ROOT/untracked" ]] || die "salida_untracked_creada"
  [[ -z "$(find "$ROOT" -mindepth 1 -maxdepth 1 -type d \
    \( -name 'pfc01-activos.*' -o -name '.pfc01-activos.*' \) -print -quit)" ]] ||
    die "temporal_untracked_creado"
  printf 'pfc01_activos_test: cheap correcto bloqueo=constructor_no_trackeado\n'
  exit 0
fi

git -C "$REPO_DIR" show HEAD:scripts/build_agent_microvm_codex_assets.sh |
  cmp -s - "$BUILDER" || die "constructor_trackeado_difiere"
if ((CHEAP == 1)); then
  printf 'pfc01_activos_test: cheap correcto estado=tracked_head\n'
  exit 0
fi

[[ "$KERNEL" == /* && -f "$KERNEL" && ! -L "$KERNEL" ]] || die "kernel_requerido"
AGENT_HEAD="$(git -C "$AGENT_REPO" rev-parse --verify 'HEAD^{commit}')"
KERNEL_MANIFEST="$ROOT/kernel-manifest-head.json"
git -C "$AGENT_REPO" show \
  "$AGENT_HEAD:activos/kernel_desarrollo_x86_64.json" >"$KERNEL_MANIFEST"
KERNEL_NAME="$(jq -er '.nombre' "$KERNEL_MANIFEST")"
KERNEL_SHA="$(jq -er '.sha256' "$KERNEL_MANIFEST")"
KERNEL_SIZE="$(jq -er '.tamano_bytes' "$KERNEL_MANIFEST")"
[[ "$(sha256sum "$KERNEL" | cut -d' ' -f1)" == "$KERNEL_SHA" &&
  "$(stat -c '%s' -- "$KERNEL")" == "$KERNEL_SIZE" ]] || die "kernel_no_fijado"

assert_no_builder_residue() {
  [[ -z "$(find "$ROOT" -mindepth 1 -maxdepth 1 -type d \
    \( -name 'pfc01-activos.*' -o -name '.pfc01-activos.*' \) -print -quit)" ]] ||
    die "temporal_build_residual"
}

# El preflight de espacio falla antes de mktemp. df fake evita depender del
# disco del host y no se usa durante ninguna construcción.
FAKE_DF_BIN="$ROOT/fake-df-bin"
mkdir -m 0700 -- "$FAKE_DF_BIN"
# Variables pertenecen al script fake generado.
# shellcheck disable=SC2016
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'set -Eeuo pipefail' \
  'if [[ "${1:-}" == "--version" ]]; then printf "df pfc01-fake 1\n"; exit 0; fi' \
  'printf "Avail\n1\n"' > "$FAKE_DF_BIN/df"
chmod 0500 -- "$FAKE_DF_BIN/df"
if PATH="$FAKE_DF_BIN:$PATH" "$BUILDER" "${builder_args[@]}" --kernel "$KERNEL" \
  --salida "$ROOT/sin-espacio" >"$ROOT/sin-espacio.out" 2>"$ROOT/sin-espacio.err"; then
  die "espacio_insuficiente_aceptado"
fi
if ! grep -Fxq 'error.pfc01.activos: espacio_insuficiente' "$ROOT/sin-espacio.err"; then
  sed -n '1,20p' "$ROOT/sin-espacio.err" >&2
  die "codigo_espacio_incorrecto"
fi
[[ ! -e "$ROOT/sin-espacio" ]] || die "salida_sin_espacio_residual"
assert_no_builder_residue

# Fallo recuperable: input inválido no deja staging, salida ni cache propia.
INVALID_CODEX="$(readlink -f -- "$(type -P true)")"
[[ -f "$INVALID_CODEX" && ! -L "$INVALID_CODEX" && -x "$INVALID_CODEX" ]] ||
  die "fixture_codex_invalido_ausente"
if "$BUILDER" "${builder_args[@]}" --codex "$INVALID_CODEX" --kernel "$KERNEL" \
  --salida "$ROOT/fallo" >"$ROOT/fallo.out" 2>"$ROOT/fallo.err"; then
  die "codex_invalido_aceptado"
fi
if ! grep -Fxq 'error.pfc01.activos: codex_no_fijado' "$ROOT/fallo.err"; then
  sed -n '1,20p' "$ROOT/fallo.err" >&2
  die "codigo_codex_invalido"
fi
[[ ! -e "$ROOT/fallo" ]] || die "salida_fallo_residual"
assert_no_builder_residue

# El segundo mktemp falla con padre readonly; el primero ya está bajo trap.
READONLY_PARENT="$ROOT/readonly"
mkdir -m 0500 -- "$READONLY_PARENT"
if "$BUILDER" "${builder_args[@]}" --kernel "$KERNEL" \
  --salida "$READONLY_PARENT/candidato" >"$ROOT/readonly.out" 2>"$ROOT/readonly.err"; then
  die "padre_readonly_aceptado"
fi
chmod 0700 -- "$READONLY_PARENT"
[[ ! -e "$READONLY_PARENT/candidato" ]] || die "salida_readonly_residual"
assert_no_builder_residue

# Cancelación cooperativa por INT y TERM: cero output y cero temporales.
for cancel_signal in INT TERM; do
  cancel_ref="${cancel_signal,,}"
  setsid env --default-signal=INT --default-signal=TERM \
    "$BUILDER" "${builder_args[@]}" --kernel "$KERNEL" \
    --salida "$ROOT/cancelado-$cancel_ref" \
    >"$ROOT/cancelado-$cancel_ref.out" 2>"$ROOT/cancelado-$cancel_ref.err" &
  cancel_pid=$!
  cancel_seen=0
  for ((attempt = 0; attempt < 500; attempt++)); do
    if find "$ROOT" -mindepth 1 -maxdepth 1 -type d -name 'pfc01-activos.*' -print -quit |
      grep -q .; then
      cancel_seen=1
      break
    fi
    kill -0 "$cancel_pid" 2>/dev/null || break
    sleep 0.01
  done
  ((cancel_seen == 1)) || die "ventana_cancelacion_no_alcanzada_$cancel_ref"
  kill -s "$cancel_signal" -- "-$cancel_pid"
  cancel_finished=0
  for ((attempt = 0; attempt < 1000; attempt++)); do
    if [[ ! -r "/proc/$cancel_pid/stat" ]]; then
      cancel_finished=1
      break
    fi
    read -r _ _ cancel_state _ < "/proc/$cancel_pid/stat" || true
    if [[ "${cancel_state:-}" == Z ]]; then
      cancel_finished=1
      break
    fi
    sleep 0.01
  done
  if ((cancel_finished == 0)); then
    kill -s KILL -- "-$cancel_pid" 2>/dev/null || true
    wait "$cancel_pid" 2>/dev/null || true
    die "cancelacion_no_finalizo_$cancel_ref"
  fi
  if wait "$cancel_pid"; then
    die "cancelacion_aceptada_como_exito_$cancel_ref"
  fi
  [[ ! -e "$ROOT/cancelado-$cancel_ref" ]] || die "salida_cancelada_residual_$cancel_ref"
  assert_no_builder_residue
done

# El builder productivo no contiene hooks. Un mv falso, resuelto y digerido
# como cualquier otra herramienta, delega al mv real y señala al PPID justo
# después del rename final. El modo sustitución devuelve el inode propio a la
# ruta de staging, coloca otro inode en salida y prueba que cleanup solo borra
# el primero.
make_mv_wrapper() {
  local directory="$1" mode="$2" target="$3"
  local real_mv real_kill real_mkdir
  real_mv="$(type -P mv)"
  real_kill="$(type -P kill)"
  real_mkdir="$(type -P mkdir)"
  mkdir -m 0700 -- "$directory"
  {
    printf '%s\n' '#!/usr/bin/env bash' 'set -Eeuo pipefail'
    printf 'real_mv=%q\nreal_kill=%q\nreal_mkdir=%q\nmode=%q\ntarget=%q\n' \
      "$real_mv" "$real_kill" "$real_mkdir" "$mode" "$target"
    # Variables pertenecen al wrapper generado.
    # shellcheck disable=SC2016
    printf '%s\n' \
      'if [[ "${1:-}" == "--version" ]]; then printf "mv pfc01-test-wrapper 1\\n"; exit 0; fi' \
      'source_path="${4:-}"' \
      'destination="${5:-}"' \
      '"$real_mv" "$@"' \
      'if [[ "$#" == 5 && "${1:-}" == "-T" && "${2:-}" == "--no-clobber" && "${3:-}" == "--" && "$destination" == "$target" ]]; then' \
      '  if [[ "$mode" == "sustitucion" ]]; then' \
      '    "$real_mv" -T --no-clobber -- "$destination" "$source_path"' \
      '    "$real_mkdir" -m 0700 -- "$destination"' \
      '    printf "sustitucion-preservada\\n" > "$destination/sustitucion-test"' \
      '  fi' \
      '  "$real_kill" -s TERM "$PPID"' \
      'fi'
  } > "$directory/mv"
  chmod 0500 -- "$directory/mv"
}

for publish_mode in propio sustitucion; do
  publish_output="$ROOT/publicacion-$publish_mode"
  fake_mv_bin="$ROOT/fake-mv-$publish_mode"
  make_mv_wrapper "$fake_mv_bin" "$publish_mode" "$publish_output"
  set +e
  PATH="$fake_mv_bin:$PATH" "$BUILDER" "${builder_args[@]}" --kernel "$KERNEL" \
    --salida "$publish_output" >"$ROOT/publicacion-$publish_mode.out" \
    2>"$ROOT/publicacion-$publish_mode.err"
  publish_status=$?
  set -e
  if [[ "$publish_status" != 143 ]]; then
    printf 'pfc01_activos_test: publicacion_%s_estado=%s\n' \
      "$publish_mode" "$publish_status" >&2
    sed -n '1,40p' "$ROOT/publicacion-$publish_mode.err" >&2
    die "senal_publicacion_estado_$publish_mode"
  fi
  if [[ "$publish_mode" == propio ]]; then
    [[ ! -e "$publish_output" && ! -L "$publish_output" ]] ||
      die "senal_publicacion_dejo_inode_propio"
  else
    [[ -d "$publish_output" && ! -L "$publish_output" ]] ||
      die "sustitucion_publicacion_eliminada"
    grep -Fxq 'sustitucion-preservada' "$publish_output/sustitucion-test" ||
      die "sustitucion_publicacion_alterada"
  fi
  assert_no_builder_residue
  if [[ "$publish_mode" == sustitucion ]]; then
    unlink "$publish_output/sustitucion-test"
    rmdir "$publish_output"
  fi
done

# La ruta sin --kernel se ejerce con curl fake: no existe tráfico de red y el
# preparador hermano debe seguir verificando tamaño y digest del manifiesto.
FAKE_BIN="$ROOT/fake-bin"
FAKE_MARKER="$ROOT/curl-fake.calls"
mkdir -m 0700 -- "$FAKE_BIN"
{
  printf '%s\n' '#!/usr/bin/env bash' 'set -Eeuo pipefail'
  printf 'kernel=%q\n' "$KERNEL"
  printf 'marker=%q\n' "$FAKE_MARKER"
  # Variables pertenecen al script fake generado.
  # shellcheck disable=SC2016
  printf '%s\n' \
    'if [[ "${1:-}" == "--version" ]]; then printf "curl pfc01-fake 1\\n"; exit 0; fi' \
    'output=""' \
    'while (($# > 0)); do' \
    '  if [[ "$1" == "--output" ]]; then output="${2:-}"; shift 2; else shift; fi' \
    'done' \
    '[[ "$output" == /* ]]' \
    'cp -- "$kernel" "$output"' \
    'printf "download_verified\\n" >> "$marker"'
} > "$FAKE_BIN/curl"
chmod 0500 -- "$FAKE_BIN/curl"

for run in one two; do
  PATH="$FAKE_BIN:$PATH" "$BUILDER" "${builder_args[@]}" --salida "$ROOT/$run" \
    >"$ROOT/$run.out" 2>"$ROOT/$run.err"
  assert_no_builder_residue
done
[[ "$(wc -l < "$FAKE_MARKER")" == 2 ]] || die "ruta_download_no_ejercida"

for asset in candidate-manifest.json kernel-manifest.json perfil-manifest.json \
  agente-microvm-huesped agente-microvm-huesped.Cargo.lock \
  agente-microvm-initramfs.cpio.gz \
  orquesta-codex-work-executor perfil-codex.ext4 "$KERNEL_NAME"; do
  cmp -- "$ROOT/one/$asset" "$ROOT/two/$asset" || die "no_reproducible_$asset"
done
[[ -z "$(find "$ROOT/one" "$ROOT/two" \( -name '*.tmp' -o -name '*.parcial' \) \
  -print -quit)" ]] || die "temporal_antiguo_usado_como_evidencia"

MANIFEST="$ROOT/one/candidate-manifest.json"
jq -e --arg kernel_sha "$KERNEL_SHA" --argjson kernel_size "$KERNEL_SIZE" '
  .schema == "orquesta.pfc01_agent_microvm_codex_assets.v1"
  and .status == "built_not_physically_exercised"
  and .builder.tracked_head == true
  and .environment == {
    "cargo_build":"env_i","cargo_offline":true,
    "filesystem_tools":"resolved_absolute_paths","go_build":"env_i",
    "goamd64":"v1","goenv":"off","gotoolchain":"local",
    "helper_scripts":"env_i",
    "rustflags":[
      "--remap-path-prefix=WORK_DIR=/orquesta-pfc01"
    ],
    "rustc_wrapper":"builder_generated_selective_metadata",
    "source_commands":"env_i"
  }
  and .physical_evidence == {"codex_session":false,"firecracker":false,"kvm":false}
  and .assets.kernel.origin == "downloaded_verified"
  and .assets.kernel.sha256 == $kernel_sha
  and .assets.kernel.size_bytes == $kernel_size
  and .assets.kernel.configuration_reference.active == false
  and (.assets.kernel.configuration_reference.source | startswith("https://"))
  and (.assets.kernel.configuration_reference.sha256 | test("^[0-9a-f]{64}$"))
  and (.sources.orquesta.archive_paths | index("modulos") | not)
  and (.sources.orquesta.archive_paths | index("docs") | not)
  and (.sources.agente_microvm.archive_paths | index("docs") | not)
  and ([.tools[] |
    (.path_ref | (test("^[A-Za-z0-9._/-]+$") and (startswith("/") | not)))
    and (.sha256 | test("^[0-9a-f]{64}$"))
    and (.version | length > 0)
  ] | all)
  and ([
    "basename","cmp","cut","df","dirname","grep","id","mkdir","mktemp",
    "readlink","rm","sed","tr"
  ] - (.tools | keys) | length == 0)
  and .effective_toolchain.claim == "same_environment_reproducibility"
  and .effective_toolchain.hermetic_supply_chain == false
  and .effective_toolchain.verification == {
    "environment":"same_host_and_inputs","performed_by_builder":false,"required_runs":2
  }
  and (.effective_toolchain.go.internal_tools | length > 0)
  and ([.effective_toolchain.go.internal_tools[] |
    (.path_ref | startswith("pkg/tool/"))
    and (.sha256 | test("^[0-9a-f]{64}$"))
    and (.version | length > 0)
  ] | all)
  and (.effective_toolchain.rust.sysroot_ref | startswith("toolchain/"))
  and (.effective_toolchain.rust.target_libdir_ref | startswith("lib/rustlib/"))
  and .effective_toolchain.rust.wrapper.scope ==
    ["agente_microvm_huesped","contrato_huesped"]
  and .effective_toolchain.rust.wrapper.path_ref ==
    "builder_generated/rustc-pfc01-wrapper"
  and (.effective_toolchain.rust.wrapper.sha256 | test("^[0-9a-f]{64}$"))
  and ([.effective_toolchain.rust.linkers[] |
    (.path_ref | startswith("lib/rustlib/"))
    and (.sha256 | test("^[0-9a-f]{64}$"))
    and (.version | length > 0)
  ] | all)
  and .effective_toolchain.cargo_dependencies.method ==
    "locked_offline_graph_read_only_snapshot"
  and .effective_toolchain.cargo_dependencies.package_count == 22
  and (.effective_toolchain.cargo_dependencies.archives | length == 22)
  and ([.effective_toolchain.cargo_dependencies.archives[] |
    (.package | test("^[A-Za-z0-9._+-]+-[0-9][A-Za-z0-9._+-]*$"))
    and (.sha256 | test("^[0-9a-f]{64}$"))
  ] | all)
  and .effective_toolchain.cargo_dependencies.effective_lock_file ==
    "agente-microvm-huesped.Cargo.lock"
  and (.effective_toolchain.cargo_dependencies.cargo_lock_sha256 |
    test("^[0-9a-f]{64}$"))
  and (.effective_toolchain.cargo_dependencies.source_cargo_lock_sha256 |
    test("^[0-9a-f]{64}$"))
  and (.effective_toolchain.cargo_dependencies.snapshot_sha256 |
    test("^[0-9a-f]{64}$"))
  and .effective_toolchain.cargo_dependencies.read_only_during_build == true
  and .effective_toolchain.cargo_dependencies.original_index_and_cache_not_consumed_by_build == true
  and ([.assets[] | .sha256 | test("^[0-9a-f]{64}$")] | all)
' "$MANIFEST" >/dev/null || die "manifiesto_invalido"

EXPECTED_KERNEL_MANIFEST_SHA="$(git -C "$AGENT_REPO" show \
  "$(jq -r '.sources.agente_microvm.commit' "$MANIFEST"):activos/kernel_desarrollo_x86_64.json" |
  sha256sum | cut -d' ' -f1)"
[[ "$(jq -r '.assets.kernel_manifest.sha256' "$MANIFEST")" == \
  "$EXPECTED_KERNEL_MANIFEST_SHA" ]] || die "kernel_manifiesto_no_corresponde_head"
EXPECTED_SOURCE_CARGO_LOCK_SHA="$(git -C "$AGENT_REPO" show \
  "$(jq -r '.sources.agente_microvm.commit' "$MANIFEST"):Cargo.lock" |
  sha256sum | cut -d' ' -f1)"
[[ "$(jq -r '.effective_toolchain.cargo_dependencies.source_cargo_lock_sha256' \
  "$MANIFEST")" == "$EXPECTED_SOURCE_CARGO_LOCK_SHA" ]] ||
  die "cargo_lock_fuente_no_corresponde_head"
[[ "$(jq -r '.assets.effective_cargo_lock.sha256' "$MANIFEST")" == \
  "$(jq -r '.effective_toolchain.cargo_dependencies.cargo_lock_sha256' \
    "$MANIFEST")" ]] || die "cargo_lock_efectivo_no_ligado"
for local_path in "$ROOT" "$TMP_PARENT" "$AGENT_REPO" "$CARGO_REGISTRY"; do
  ! grep -aFq -- "$local_path" "$ROOT/one/agente-microvm-huesped" ||
    die "ruta_local_embebida_guest"
  ! grep -Fq -- "$local_path" "$MANIFEST" || die "ruta_local_embebida_manifiesto"
done
for key in effective_cargo_lock executor guest initramfs kernel profile profile_manifest; do
  file="$(jq -er --arg key "$key" '.assets[$key].file' "$MANIFEST")"
  expected="$(jq -er --arg key "$key" '.assets[$key].sha256' "$MANIFEST")"
  printf '%s  %s\n' "$expected" "$ROOT/one/$file" | sha256sum --check --status ||
    die "digest_invalido_$key"
done

printf 'pfc01_activos_test: correcto manifiesto_sha256=%s\n' \
  "$(sha256sum "$MANIFEST" | cut -d' ' -f1)"
