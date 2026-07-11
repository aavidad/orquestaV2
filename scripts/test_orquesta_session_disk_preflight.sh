#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_session_disk_preflight.sh"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-session-disk-preflight-test.XXXXXX")"
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/home" "$fixture/tmp" "$fixture/work" "$fixture/session-parent" "$fixture/bin"
cat >"$fixture/bin/df" <<'DF'
#!/usr/bin/env bash
set -euo pipefail
path="${@: -1}"
available_home="${FAKE_HOME_AVAILABLE_KIB:-4096}"
case "$path" in
  *"/home"*) device=/dev/home; mount=/home; available="$available_home" ;;
  *"/work"*|*"/session-base"*) device=/dev/work; mount=/work; available="${FAKE_WORK_AVAILABLE_KIB:-4096}" ;;
  *"/tmp"*) device=/dev/tmp; mount=/tmp; available=4096 ;;
  /) device=/dev/root; mount=/; available=4096 ;;
  *) echo "unexpected df path: $path" >&2; exit 9 ;;
esac
printf 'Filesystem 1024-blocks Used Available Capacity Mounted on\n%s 8192 1 %s 1%% %s\n' "$device" "$available" "$mount"
DF
chmod +x "$fixture/bin/df"

common=(--session-ref bug-208ah --session-base "$fixture/session-base" --workdir "$fixture/work" --budget-bytes 1048576)
mkdir -m 700 "$fixture/session-base"

status_output="$(HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --status "${common[@]}")"
grep -q '^mode=status$' <<<"$status_output"
grep -q '^state=ready$' <<<"$status_output"
[ "$(grep -c $'device=/dev/work\tmount=/work' <<<"$status_output")" -eq 1 ]
[ ! -e "$fixture/session-base/bug-208ah" ] || { echo 'status creo rutas de sesion' >&2; exit 1; }
large_mount_output="$(HOME="$fixture/home" FAKE_WORK_AVAILABLE_KIB=4000000000 ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --status "${common[@]}")"
grep -q 'available_bytes=4096000000000' <<<"$large_mount_output"
budget_mib_output="$(HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --status --session-ref mib --session-base "$fixture/session-base" --workdir "$fixture/work" --budget-mib 1)"
grep -q '^budget_bytes=1048576$' <<<"$budget_mib_output"
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --status --session-ref mixed --session-base "$fixture/session-base" --workdir "$fixture/work" --budget-bytes 1 --budget-mib 1 >/dev/null 2>&1; then
  echo 'preflight acepto unidades de presupuesto mezcladas' >&2
  exit 1
fi
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --status --session-ref home-root --session-base "$fixture/home" --workdir "$fixture/work" --budget-bytes 1 >/dev/null 2>&1; then
  echo 'preflight acepto HOME como raiz de sesion' >&2
  exit 1
fi
mkdir -m 700 "$fixture/real-base"
ln -s "$fixture/real-base" "$fixture/base-link"
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight --session-ref symlink --session-base "$fixture/base-link" --init-session-base --workdir "$fixture/work" --budget-bytes 1 >/dev/null 2>&1; then
  echo 'preflight acepto base symlink' >&2
  exit 1
fi
mkdir -m 700 "$fixture/undeclared-base"
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight --session-ref outside --session-base "$fixture/undeclared-base" --workdir "$fixture/work" --budget-bytes 1 >/dev/null 2>&1; then
  echo 'preflight acepto base explicita no declarada' >&2
  exit 1
fi
mkdir -m 700 "$fixture/declared-base" "$fixture/session-target"
printf 'schema_version=orquesta_session_disk_base.v0\n' >"$fixture/declared-base/.orquesta-session-base.v0"
ln -s "$fixture/session-target" "$fixture/declared-base/root-link"
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight --session-ref root-link --session-base "$fixture/declared-base" --workdir "$fixture/work" --budget-bytes 1 >/dev/null 2>&1; then
  echo 'preflight acepto session_root symlink' >&2
  exit 1
fi

if HOME="$fixture/home" FAKE_HOME_AVAILABLE_KIB=10 ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight --init-session-base "${common[@]}" >"$fixture/insufficient.out" 2>&1; then
  echo 'preflight acepto HOME insuficiente' >&2
  exit 1
fi
grep -q '^state=insufficient$' "$fixture/insufficient.out"
receipt="$fixture/session-base/bug-208ah/session_disk_receipt.json"
grep -q '"state": "insufficient"' "$receipt"
[ ! -d "$fixture/session-base/bug-208ah/caches/go-cache" ] || { echo 'preflight insuficiente creo cache' >&2; exit 1; }

preflight_output="$(HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight "${common[@]}")"
grep -q '^state=ready$' <<<"$preflight_output"
grep -q '"schema_version": "orquesta_session_disk_preflight.v0"' "$receipt"
grep -q '"unit": "bytes"' "$receipt"
grep -q '"paths":"workdir,session_root"' "$receipt"
python3 -m json.tool "$receipt" >/dev/null
[ -f "$fixture/session-base/bug-208ah/caches/go-cache/.orquesta-session-owned.v0" ]

cleanup_dry="$(HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --cleanup "${common[@]}")"
grep -q '^mode=dry-run$' <<<"$cleanup_dry"
[ -d "$fixture/session-base/bug-208ah/caches/go-cache" ]
cleanup_receipt="$fixture/session-base/bug-208ah/session_disk_cleanup_receipt.json"
python3 - "$cleanup_receipt" "$receipt" <<'PY'
import json, sys
cleanup=json.load(open(sys.argv[1], encoding='utf-8'))
assert cleanup['schema_version'] == 'orquesta_session_disk_cleanup_receipt.v0'
assert cleanup['mode'] == 'dry-run' and cleanup['preflight_receipt'] == sys.argv[2]
assert len(cleanup['actions']) == 9 and {action['action'] for action in cleanup['actions']} == {'report'}
assert {action['reason'] for action in cleanup['actions']} == {'dry_run'}
PY
if HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --cleanup --confirm-cleanup wrong "${common[@]}" >/dev/null 2>&1; then
  echo 'cleanup acepto confirmacion ajena' >&2
  exit 1
fi
HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --cleanup --confirm-cleanup bug-208ah "${common[@]}" >/dev/null
[ ! -e "$fixture/session-base/bug-208ah/caches/go-cache" ]
[ -f "$receipt" ] || { echo 'cleanup borro receipt' >&2; exit 1; }
python3 - "$cleanup_receipt" <<'PY'
import json, sys
cleanup=json.load(open(sys.argv[1], encoding='utf-8'))
assert cleanup['mode'] == 'cleanup'
assert len(cleanup['actions']) == 9 and {action['action'] for action in cleanup['actions']} == {'delete'}
assert {action['reason'] for action in cleanup['actions']} == {'marker_verified'}
PY

HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --preflight "${common[@]}" >/dev/null
rm "$fixture/session-base/bug-208ah/caches/go-cache/.orquesta-session-owned.v0"
blocked="$(HOME="$fixture/home" ORQUESTA_SESSION_DISK_DF_BIN="$fixture/bin/df" "$script" --cleanup --confirm-cleanup bug-208ah "${common[@]}")"
grep -q 'action=preserve.*missing_or_invalid_marker' <<<"$blocked"
[ -d "$fixture/session-base/bug-208ah/caches/go-cache" ]
python3 - "$cleanup_receipt" <<'PY'
import json, sys
cleanup=json.load(open(sys.argv[1], encoding='utf-8'))
assert any(action['owned_path'].endswith('/go-cache') and action['action'] == 'preserve' and action['reason'] == 'missing_or_invalid_marker' for action in cleanup['actions'])
PY

echo 'orquesta_session_disk_preflight_test=ok'
