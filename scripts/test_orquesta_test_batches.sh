#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cache_root="${ORQUESTA_BATCH_TEST_CACHE_ROOT:-/tmp/orquesta-f3-r2-batch-tests}"
mkdir -p "$cache_root"; chmod 700 "$cache_root"
test_root="$(mktemp -d "$cache_root/test.XXXXXX")"
trap 'rm -rf "$test_root"' EXIT

sed "s#__ROOT__#$test_root#g" >"$test_root/go-fake" <<'SH'
#!/usr/bin/env bash
set -eu
if [ "$1" = list ]; then
  printf '%s\n' example/p1 example/p2
  if [ "${FAKE_LIST_PARTIAL_FAIL:-0}" = 1 ]; then exit 9; fi
  printf '%s\n' example/p3 example/p4 example/p5
  exit 0
fi
printf '%s|%s\n' "$(umask)" "$*" >>'__ROOT__/go.log'
if printf '%s\n' "$*" | grep -q example/p3 && [ "${FAKE_FAIL_P3:-0}" = 1 ]; then exit 7; fi
SH
sed "s#__ROOT__#$test_root#g" >"$test_root/timeout-fake" <<'SH'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$1 $2 $3" >>'__ROOT__/timeout.log'
shift 3
exec "$@"
SH
chmod +x "$test_root/go-fake" "$test_root/timeout-fake"
cat >"$test_root/df-fake" <<'SH'
#!/usr/bin/env bash
set -eu
printf 'Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/test 8192 1 4096 1%% /test\n'
SH
chmod +x "$test_root/df-fake"

common_env=(
  ORQUESTA_TEST_CACHE_ROOT="$test_root/cache"
  ORQUESTA_TEST_PORT_LEASE_ROOT="$test_root/port-leases"
  ORQUESTA_TEST_BATCH_SIZE=2
  ORQUESTA_TEST_BATCH_TIMEOUT=37s
  ORQUESTA_TEST_BATCH_KILL_AFTER=5s
  ORQUESTA_GO_TEST_TIMEOUT=31s
  ORQUESTA_SESSION_DISK_BUDGET_BYTES=1
  ORQUESTA_SESSION_DISK_DF_BIN="$test_root/df-fake"
  ORQUESTA_BATCH_GO_COMMAND="$test_root/go-fake"
  ORQUESTA_BATCH_TIMEOUT_COMMAND="$test_root/timeout-fake"
)

umask 022
env "${common_env[@]}" ORQUESTA_TEST_BATCH_ROOT="$test_root/pass" \
  "$ROOT/scripts/orquesta_test_batches.sh" >/dev/null

python3 - "$test_root/pass/receipt.json" "$test_root/go.log" "$test_root/timeout.log" <<'PY'
import json, os, sys
r=json.load(open(sys.argv[1], encoding='utf-8'))
assert r['schema_version']=='orquesta_test_batches_receipt.v1'
assert r['status']=='passed' and r['passes_required']==2 and r['passes_completed']==[1,2]
assert r['packages_total']==5 and r['package_executions_total']==10
assert [len(x['packages']) for x in r['batches']]==[2,2,1,2,2,1]
go_lines=open(sys.argv[2]).read().splitlines()
assert len(go_lines)==6
assert all(line.startswith('0022|') for line in go_lines), go_lines
assert open(sys.argv[3]).read().splitlines()==['--signal=TERM --kill-after=5s 37s']*6
assert all(os.path.isfile(x['receipt']) for x in r['batches'])
env_root=os.path.join(r['run_root'],'env')
receipts=[]
cleanup_receipts=[]
for current, _, files in os.walk(env_root):
    receipts += [os.path.join(current, name) for name in files if name == 'session_disk_receipt.json']
    cleanup_receipts += [os.path.join(current, name) for name in files if name == 'session_disk_cleanup_receipt.json']
assert len(receipts)==1, 'receipt de preflight debe conservarse'
assert len(cleanup_receipts)==1, 'receipt de cleanup debe conservarse'
cleanup=json.load(open(cleanup_receipts[0], encoding='utf-8'))
assert cleanup['schema_version']=='orquesta_session_disk_cleanup_receipt.v0'
assert cleanup['mode']=='cleanup' and cleanup['preflight_receipt']==receipts[0]
assert len(cleanup['actions'])==9 and {a['action'] for a in cleanup['actions']}=={'delete'}
assert not any(name == '.orquesta-session-owned.v0' for current, _, files in os.walk(env_root) for name in files), 'cleanup no debe conservar caches atestadas'
PY

set +e
env "${common_env[@]}" FAKE_FAIL_P3=1 ORQUESTA_TEST_BATCH_ROOT="$test_root/fail" \
  "$ROOT/scripts/orquesta_test_batches.sh" >"$test_root/fail.out" 2>&1
fail_rc=$?
set -e
[ "$fail_rc" -eq 1 ]
python3 - "$test_root/fail/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert r['status']=='failed' and r['passes_completed']==[]
assert len(r['batches'])==6 and sum(x['exit_code']==7 for x in r['batches'])==2
PY

# go list parcial conserva su rc real y jamás ejecuta/declara verde la lista parcial.
set +e
env "${common_env[@]}" FAKE_LIST_PARTIAL_FAIL=1 ORQUESTA_TEST_BATCH_ROOT="$test_root/list-fail" \
  "$ROOT/scripts/orquesta_test_batches.sh" >"$test_root/list-fail.out" 2>&1
list_rc=$?
set -e
[ "$list_rc" -eq 9 ]
python3 - "$test_root/list-fail/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert r['status']=='failed' and r['reason_code']=='go_list_failed'
assert r['go_list_exit_code']==9 and r['packages_total']==2 and r['batches']==[]
PY
grep -q 'orquesta_test_batches=not_ok' "$test_root/list-fail.out"

# Raíz symlink se rehúsa y dos consumidores no pueden compartir el mismo rango.
source "$ROOT/scripts/lib/isolated_test_env.sh"

# La máscara temporal de aislamiento no debe contaminar al consumidor, tampoco
# cuando la preparación falla.
umask 022
env ORQUESTA_TEST_CACHE_ROOT="$test_root/cache" ORQUESTA_TEST_PORT_LEASE_ROOT="$test_root/mode-leases" ORQUESTA_SESSION_DISK_BUDGET_BYTES=1 ORQUESTA_SESSION_DISK_DF_BIN="$test_root/df-fake" \
  bash -c 'source "$1"; umask 022; orquesta_use_isolated_test_env "$2"; test "$(umask)" = 0022; test "$(stat -c %a "$2")" = 700; test "$(stat -c %a "$ORQUESTA_TEST_PORT_LOCK_DIR"/ports-*.lock)" = 600; orquesta_cleanup_isolated_test_env "$2"; umask 022; ln -s "$2" "$2-link"; orquesta_use_isolated_test_env "$2-link" >/dev/null 2>&1 && exit 1 || true; test "$(umask)" = 0022' _ \
  "$ROOT/scripts/lib/isolated_test_env.sh" "$test_root/mode-root"

mkdir -m 700 "$test_root/real-root"
ln -s "$test_root/real-root" "$test_root/root-link"
set +e
ORQUESTA_TEST_CACHE_ROOT="$test_root/cache" orquesta_use_isolated_test_env "$test_root/root-link" >/dev/null 2>&1
symlink_rc=$?
set -e
[ "$symlink_rc" -ne 0 ]

explicit_root="$test_root/explicit-root-without-global-cache"
env -u ORQUESTA_TEST_CACHE_ROOT -u ORQUESTA_TEST_PORT_LEASE_ROOT \
  ORQUESTA_SESSION_DISK_BUDGET_BYTES=1 ORQUESTA_SESSION_DISK_DF_BIN="$test_root/df-fake" \
  bash -c 'source "$1"; orquesta_use_isolated_test_env "$2"; test "$ORQUESTA_TEST_PORT_LOCK_DIR" = "$(dirname "$2")/port-leases"; orquesta_cleanup_isolated_test_env "$2"' _ \
  "$ROOT/scripts/lib/isolated_test_env.sh" "$explicit_root"

lease_root="$test_root/shared-leases"
env ORQUESTA_TEST_CACHE_ROOT="$test_root/cache" ORQUESTA_TEST_PORT_LEASE_ROOT="$lease_root" ORQUESTA_TEST_PORT_BASE=42000 ORQUESTA_SESSION_DISK_BUDGET_BYTES=1 ORQUESTA_SESSION_DISK_DF_BIN="$test_root/df-fake" \
  bash -c 'source "$1"; orquesta_use_isolated_test_env "$2"; printf ready >"$3"; sleep 2' _ \
  "$ROOT/scripts/lib/isolated_test_env.sh" "$test_root/lease-one" "$test_root/lease.ready" & lease_pid=$!
for _ in $(seq 1 30); do [ -s "$test_root/lease.ready" ] && break; sleep 0.05; done
set +e
env ORQUESTA_TEST_CACHE_ROOT="$test_root/cache" ORQUESTA_TEST_PORT_LEASE_ROOT="$lease_root" ORQUESTA_TEST_PORT_BASE=42000 ORQUESTA_SESSION_DISK_BUDGET_BYTES=1 ORQUESTA_SESSION_DISK_DF_BIN="$test_root/df-fake" \
  bash -c 'source "$1"; orquesta_use_isolated_test_env "$2"' _ "$ROOT/scripts/lib/isolated_test_env.sh" "$test_root/lease-two" >/dev/null 2>&1
lease_rc=$?
set -e
[ "$lease_rc" -ne 0 ]; wait "$lease_pid"

# Ausencia de rg no cambia los resultados del harness.
PATH=/usr/bin:/bin bash -n "$ROOT/scripts/orquesta_test_batches.sh" "$ROOT/scripts/test_orquesta_test_batches.sh" "$ROOT/scripts/lib/isolated_test_env.sh"
echo "test_orquesta_test_batches=ok"
