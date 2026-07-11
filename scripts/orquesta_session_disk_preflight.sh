#!/usr/bin/env bash
set -euo pipefail

# Preflight local para sesiones largas. No inspecciona ni elimina caches ajenas.

mode="preflight"
session_ref=""
session_base=""
session_base_explicit=0
initialize_session_base=0
workdir="$(pwd -P)"
budget_bytes=""
budget_specified=0
receipt=""
confirm_cleanup=""
df_bin="${ORQUESTA_SESSION_DISK_DF_BIN:-df}"

usage() {
  cat <<'USAGE'
Uso: scripts/orquesta_session_disk_preflight.sh [--status|--preflight|--cleanup] --session-ref REF [opciones]

Opciones:
  --session-base PATH     Base privada permitida. Requiere declaracion explicita fuera del default.
  --init-session-base     Inicializa el marcador de una base explicita privada.
  --workdir PATH          Directorio de trabajo que se medira (por defecto: pwd).
  --budget-bytes N        Presupuesto minimo libre por mount, en bytes.
  --budget-mib N          Presupuesto minimo libre por mount, en MiB enteros.
  --receipt PATH          Recibo JSON (por defecto: SESSION_BASE/REF/session_disk_receipt.json).
  --confirm-cleanup REF   Requerido con --cleanup para borrar caches de REF.

--status solo mide. --preflight (por defecto) crea caches de sesion y un
recibo solo si cada mount unico de HOME, /, /tmp, workdir y session-root tiene
el presupuesto libre. --cleanup es dry-run salvo confirmacion literal y nunca
borra la raiz de sesion, el recibo ni rutas sin marcador de propiedad.
USAGE
}

fail() {
  printf 'orquesta_session_disk_preflight_failed: %s\n' "$1" >&2
  exit 2
}

json_escape() {
  local value="$1"
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  printf '%s' "$value"
}

absolute_existing_parent() {
  local path="$1"
  while [ ! -e "$path" ]; do
    local parent
    parent="$(dirname "$path")"
    [ "$parent" != "$path" ] || return 1
    path="$parent"
  done
  [ -d "$path" ] || path="$(dirname "$path")"
  (cd "$path" && pwd -P)
}

absolute_path() {
  local path="$1"
  case "$path" in
    /*) printf '%s' "$path" ;;
    *) printf '%s/%s' "$(pwd -P)" "$path" ;;
  esac
}

canonical_session_base() {
  local path="$1" create="$2"
  python3 - "$path" "$create" <<'PY'
import os, pathlib, stat, sys
raw = pathlib.Path(sys.argv[1])
create = sys.argv[2] == "1"
if not raw.is_absolute() or raw == pathlib.Path("/"):
    raise SystemExit("session_base_invalid")
current = pathlib.Path("/")
for part in raw.parts[1:]:
    current /= part
    if os.path.lexists(current) and stat.S_ISLNK(os.lstat(current).st_mode):
        raise SystemExit("session_base_symlink")
if create:
    raw.mkdir(mode=0o700, parents=True, exist_ok=True)
if not raw.exists():
    parent = raw.parent.resolve(strict=True)
    print(str(parent / raw.name))
    raise SystemExit(0)
info = raw.stat()
resolved = raw.resolve(strict=True)
if resolved != raw or not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
    raise SystemExit("session_base_not_private_owned_canonical")
print(str(resolved))
PY
}

prepare_session_root() {
  local path="$1" create="$2"
  python3 - "$path" "$create" <<'PY'
import os, pathlib, stat, sys
path = pathlib.Path(sys.argv[1])
create = sys.argv[2] == "1"
if os.path.lexists(path) and stat.S_ISLNK(os.lstat(path).st_mode):
    raise SystemExit("session_root_symlink")
if create:
    path.mkdir(mode=0o700, exist_ok=True)
if path.exists():
    info = path.stat()
    resolved = path.resolve(strict=True)
    if resolved != path or not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise SystemExit("session_root_not_private_owned_canonical")
PY
}

marker_path() { printf '%s/.orquesta-session-owned.v0' "$1"; }
session_base_marker_path() { printf '%s/.orquesta-session-base.v0' "$1"; }

has_session_base_marker() {
  local marker
  marker="$(session_base_marker_path "$session_base")"
  [ -f "$marker" ] && grep -Fxq 'schema_version=orquesta_session_disk_base.v0' "$marker"
}

write_session_base_marker() {
  printf 'schema_version=orquesta_session_disk_base.v0\n' >"$(session_base_marker_path "$session_base")"
}

write_marker() {
  local path="$1"
  {
    printf 'schema_version=orquesta_session_disk_owned_path.v0\n'
    printf 'session_ref=%s\n' "$session_ref"
  } >"$(marker_path "$path")"
}

has_marker() {
  local path="$1"
  local marker
  marker="$(marker_path "$path")"
  [ -f "$marker" ] && grep -Fxq 'schema_version=orquesta_session_disk_owned_path.v0' "$marker" &&
    grep -Fxq "session_ref=$session_ref" "$marker"
}

write_cleanup_receipt() {
  local cleanup_receipt="$session_root/session_disk_cleanup_receipt.json"
  local temporary index
  temporary="$(mktemp "$session_root/.session_disk_cleanup_receipt.XXXXXX")"
  {
    printf '{\n'
    printf '  "schema_version": "orquesta_session_disk_cleanup_receipt.v0",\n'
    printf '  "session_ref": "%s",\n' "$(json_escape "$session_ref")"
    printf '  "session_base": "%s",\n' "$(json_escape "$session_base")"
    printf '  "session_root": "%s",\n' "$(json_escape "$session_root")"
    printf '  "mode": "%s",\n' "$cleanup_mode"
    printf '  "preflight_receipt": "%s",\n' "$(json_escape "$receipt")"
    printf '  "actions": [\n'
    for index in "${!cleanup_paths[@]}"; do
      printf '    {"owned_path":"%s","action":"%s","reason":"%s"}%s\n' \
        "$(json_escape "${cleanup_paths[$index]}")" "${cleanup_actions[$index]}" "${cleanup_reasons[$index]}" \
        "$( [ "$index" -lt $((${#cleanup_paths[@]} - 1)) ] && printf , )"
    done
    printf '  ]\n}\n'
  } >"$temporary"
  mv -f -- "$temporary" "$cleanup_receipt"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --status) mode="status"; shift ;;
    --preflight) mode="preflight"; shift ;;
    --cleanup) mode="cleanup"; shift ;;
    --session-ref) [ "$#" -ge 2 ] || fail 'session_ref_sin_valor'; session_ref="$2"; shift 2 ;;
    --session-base) [ "$#" -ge 2 ] || fail 'session_base_sin_valor'; session_base="$2"; session_base_explicit=1; shift 2 ;;
    --init-session-base) initialize_session_base=1; shift ;;
    --workdir) [ "$#" -ge 2 ] || fail 'workdir_sin_valor'; workdir="$2"; shift 2 ;;
    --budget-bytes) [ "$#" -ge 2 ] || fail 'budget_bytes_sin_valor'; [ "$budget_specified" -eq 0 ] || fail 'budget_unidad_duplicada'; budget_bytes="$2"; budget_specified=1; shift 2 ;;
    --budget-mib) [ "$#" -ge 2 ] || fail 'budget_mib_sin_valor'; [ "$budget_specified" -eq 0 ] || fail 'budget_unidad_duplicada'; case "$2" in ''|*[!0-9]*) fail 'budget_mib_invalido';; esac; budget_bytes=$(( $2 * 1024 * 1024 )); budget_specified=1; shift 2 ;;
    --receipt) [ "$#" -ge 2 ] || fail 'receipt_sin_valor'; receipt="$2"; shift 2 ;;
    --confirm-cleanup) [ "$#" -ge 2 ] || fail 'confirm_cleanup_sin_valor'; confirm_cleanup="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) fail "opcion_incompatible:$1" ;;
  esac
done

case "$session_ref" in ''|*[!A-Za-z0-9._-]*) fail 'session_ref_invalida';; esac
[ "$initialize_session_base" -eq 0 ] || [ "$mode" = 'preflight' ] || fail 'init_session_base_solo_preflight'
if [ "$budget_specified" -eq 0 ]; then budget_bytes="${ORQUESTA_SESSION_DISK_BUDGET_BYTES:-2147483648}"; fi
case "$budget_bytes" in 0|*[!0-9]*) fail 'budget_bytes_invalido';; esac

workdir="$(absolute_existing_parent "$(absolute_path "$workdir")")" || fail 'workdir_inaccesible'
if [ -z "$session_base" ]; then session_base="${TMPDIR:-/tmp}/orquesta-sessions"; fi
session_base="$(absolute_path "$session_base")"
if [ "$mode" = 'preflight' ]; then
  session_base="$(canonical_session_base "$session_base" 1)" || fail 'session_base_no_permitida'
else
  session_base="$(canonical_session_base "$session_base" 0)" || fail 'session_base_no_permitida'
fi
if [ "$mode" = 'preflight' ]; then
  if ! has_session_base_marker; then
    if [ "$session_base_explicit" -eq 1 ] && [ "$initialize_session_base" -ne 1 ]; then
      fail 'session_base_no_declarada'
    fi
    write_session_base_marker
  fi
elif [ "$mode" = 'cleanup' ] && ! has_session_base_marker; then
  fail 'session_base_marker_ausente'
fi
session_root="$session_base/$session_ref"
case "$session_root" in /|"$HOME"|"$HOME"/*) fail 'session_root_no_aislada';; esac
if [ -z "$receipt" ]; then receipt="$session_root/session_disk_receipt.json"; fi
receipt="$(absolute_path "$receipt")"
[ "$receipt" = "$session_root/session_disk_receipt.json" ] || fail 'receipt_fuera_session_root'

owned_paths=(
  "$session_root/caches/tmp"
  "$session_root/caches/go-tmp"
  "$session_root/caches/go-cache"
  "$session_root/caches/go-mod-cache"
  "$session_root/caches/go-path"
  "$session_root/caches/codex-home"
  "$session_root/caches/xdg-runtime"
  "$session_root/caches/flaky-cache"
  "$session_root/caches/runtime"
)

if [ "$mode" = 'cleanup' ]; then
  [ -z "$confirm_cleanup" ] || [ "$confirm_cleanup" = "$session_ref" ] || fail 'confirm_cleanup_invalida'
  [ -f "$receipt" ] || fail 'receipt_ausente'
  grep -Fqx "  \"session_ref\": \"$(json_escape "$session_ref")\"," "$receipt" || fail 'receipt_session_ref_invalida'
  grep -Fqx "  \"session_base\": \"$(json_escape "$session_base")\"," "$receipt" || fail 'receipt_session_base_invalida'
  grep -Fqx "  \"session_root\": \"$(json_escape "$session_root")\"," "$receipt" || fail 'receipt_session_root_invalida'
  prepare_session_root "$session_root" 0 || fail 'session_root_identidad_invalida'
  for path in "${owned_paths[@]}"; do
    grep -Fqx "    \"$(json_escape "$path")\"" "$receipt" ||
      grep -Fqx "    \"$(json_escape "$path")\"," "$receipt" || fail 'receipt_owned_paths_invalido'
  done
  cleanup_mode="$( [ "$confirm_cleanup" = "$session_ref" ] && printf cleanup || printf dry-run )"
  cleanup_paths=()
  cleanup_actions=()
  cleanup_reasons=()
  cleanup_failed=0
  printf 'schema_version=orquesta_session_disk_cleanup.v0\nmode=%s\nsession_ref=%s\n' "$cleanup_mode" "$session_ref"
  for path in "${owned_paths[@]}"; do
    if ! has_marker "$path"; then
      printf 'owned_path\taction=preserve\treason=missing_or_invalid_marker\tpath=%s\n' "$path"
      cleanup_paths+=("$path")
      cleanup_actions+=("preserve")
      cleanup_reasons+=("missing_or_invalid_marker")
      continue
    fi
    if [ "$confirm_cleanup" = "$session_ref" ]; then
      if rm -rf -- "$path"; then
        printf 'owned_path\taction=delete\treason=marker_verified\tpath=%s\n' "$path"
        cleanup_paths+=("$path")
        cleanup_actions+=("delete")
        cleanup_reasons+=("marker_verified")
      else
        printf 'owned_path\taction=preserve\treason=delete_failed\tpath=%s\n' "$path" >&2
        cleanup_paths+=("$path")
        cleanup_actions+=("preserve")
        cleanup_reasons+=("delete_failed")
        cleanup_failed=1
      fi
    else
      printf 'owned_path\taction=report\treason=dry_run\tpath=%s\n' "$path"
      cleanup_paths+=("$path")
      cleanup_actions+=("report")
      cleanup_reasons+=("dry_run")
    fi
  done
  write_cleanup_receipt
  printf 'cleanup_receipt=%s\n' "$session_root/session_disk_cleanup_receipt.json"
  [ "$cleanup_failed" -eq 0 ] || exit 1
  exit 0
fi

[ -x "$df_bin" ] || command -v "$df_bin" >/dev/null 2>&1 || fail 'df_no_disponible'
declare -a mount_devices mount_points mount_available mount_paths
measure_path() {
  local label="$1" path="$2" probe output device mount available_kib available index
  probe="$(absolute_existing_parent "$path")" || fail "ruta_inaccesible:$label"
  output="$($df_bin -Pk "$probe")" || fail "df_fallo:$label"
  read -r device mount available_kib < <(printf '%s\n' "$output" | awk 'NR == 2 {print $1, $6, $4}')
  [ -n "${device:-}" ] && [ -n "${mount:-}" ] && [ -n "${available_kib:-}" ] || fail "df_salida_invalida:$label"
  case "$available_kib" in *[!0-9]*|'') fail "df_disponible_invalido:$label";; esac
  available=$((available_kib * 1024))
  for index in "${!mount_devices[@]}"; do
    if [ "${mount_devices[$index]}" = "$device" ] && [ "${mount_points[$index]}" = "$mount" ]; then
      mount_paths[$index]="${mount_paths[$index]},$label"
      return
    fi
  done
  mount_devices+=("$device")
  mount_points+=("$mount")
  mount_available+=("$available")
  mount_paths+=("$label")
}

measure_path 'home' "$HOME"
measure_path 'root' '/'
measure_path 'tmp' '/tmp'
measure_path 'workdir' "$workdir"
measure_path 'session_root' "$session_root"

state='ready'
for index in "${!mount_available[@]}"; do
  if [ "${mount_available[$index]}" -lt "$budget_bytes" ]; then state='insufficient'; fi
done

if [ "$mode" = 'preflight' ]; then
  prepare_session_root "$session_root" 1 || fail 'session_root_identidad_invalida'
fi
if [ "$mode" = 'preflight' ] && [ "$state" = 'ready' ]; then
  umask 077
  for path in "${owned_paths[@]}"; do mkdir -p "$path"; chmod 700 "$path"; write_marker "$path"; done
fi

if [ "$mode" = 'preflight' ]; then
  mkdir -p "$(dirname "$receipt")"
  {
    printf '{\n'
    printf '  "schema_version": "orquesta_session_disk_preflight.v0",\n'
    printf '  "session_ref": "%s",\n' "$(json_escape "$session_ref")"
    printf '  "session_base": "%s",\n' "$(json_escape "$session_base")"
    printf '  "session_root": "%s",\n' "$(json_escape "$session_root")"
    printf '  "state": "%s",\n' "$state"
    printf '  "budget": {"unit": "bytes", "required_bytes": %s},\n' "$budget_bytes"
    printf '  "owned_paths": [\n'
    for index in "${!owned_paths[@]}"; do printf '    "%s"%s\n' "$(json_escape "${owned_paths[$index]}")" "$( [ "$index" -lt $((${#owned_paths[@]} - 1)) ] && printf , )"; done
    printf '  ],\n  "mounts": [\n'
    for index in "${!mount_devices[@]}"; do
      printf '    {"device":"%s","mount":"%s","available_bytes":%s,"required_bytes":%s,"paths":"%s"}%s\n' \
        "$(json_escape "${mount_devices[$index]}")" "$(json_escape "${mount_points[$index]}")" "${mount_available[$index]}" "$budget_bytes" "${mount_paths[$index]}" \
        "$( [ "$index" -lt $((${#mount_devices[@]} - 1)) ] && printf , )"
    done
    printf '  ],\n'
    printf '  "cache_environment": {"TMPDIR":"%s","GOTMPDIR":"%s","GOCACHE":"%s","GOMODCACHE":"%s","GOPATH":"%s","CODEX_HOME":"%s","XDG_RUNTIME_DIR":"%s","ORQUESTA_FLAKY_HARNESS_CACHE_ROOT":"%s","ORQUESTA_TEST_RUNTIME_ROOT":"%s"}\n' \
      "$(json_escape "${owned_paths[0]}")" "$(json_escape "${owned_paths[1]}")" "$(json_escape "${owned_paths[2]}")" "$(json_escape "${owned_paths[3]}")" "$(json_escape "${owned_paths[4]}")" "$(json_escape "${owned_paths[5]}")" "$(json_escape "${owned_paths[6]}")" "$(json_escape "${owned_paths[7]}")" "$(json_escape "${owned_paths[8]}")"
    printf '}\n'
  } >"$receipt"
fi

printf 'schema_version=orquesta_session_disk_preflight.v0\nmode=%s\nsession_ref=%s\nstate=%s\nbudget_bytes=%s\nreceipt=%s\n' \
  "$mode" "$session_ref" "$state" "$budget_bytes" "$receipt"
for index in "${!mount_devices[@]}"; do
  printf 'mount\tdevice=%s\tmount=%s\tavailable_bytes=%s\trequired_bytes=%s\tpaths=%s\n' \
    "${mount_devices[$index]}" "${mount_points[$index]}" "${mount_available[$index]}" "$budget_bytes" "${mount_paths[$index]}"
done
if [ "$state" != 'ready' ]; then
  exit 1
fi
if [ "$mode" = 'preflight' ]; then
  printf 'export TMPDIR=%q GOTMPDIR=%q GOCACHE=%q GOMODCACHE=%q GOPATH=%q CODEX_HOME=%q XDG_RUNTIME_DIR=%q ORQUESTA_FLAKY_HARNESS_CACHE_ROOT=%q ORQUESTA_TEST_RUNTIME_ROOT=%q\n' \
    "${owned_paths[0]}" "${owned_paths[1]}" "${owned_paths[2]}" "${owned_paths[3]}" "${owned_paths[4]}" "${owned_paths[5]}" "${owned_paths[6]}" "${owned_paths[7]}" "${owned_paths[8]}"
fi
