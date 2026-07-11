#!/usr/bin/env bash
# Dos pases Go consecutivos, aislados y observables por lote.

set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT/scripts/lib/isolated_test_env.sh"

BATCH_SIZE="${ORQUESTA_TEST_BATCH_SIZE:-12}"
BATCH_TIMEOUT="${ORQUESTA_TEST_BATCH_TIMEOUT:-12m}"
BATCH_KILL_AFTER="${ORQUESTA_TEST_BATCH_KILL_AFTER:-30s}"
GO_TEST_TIMEOUT="${ORQUESTA_GO_TEST_TIMEOUT:-10m}"
PASSES="${ORQUESTA_TEST_PASSES:-2}"
TIMEOUT_COMMAND="${ORQUESTA_BATCH_TIMEOUT_COMMAND:-timeout}"
GO_COMMAND="${ORQUESTA_BATCH_GO_COMMAND:-go}"
RUN_ID="batch-ref-$(date +%s%N)-$$-${RANDOM}"
RUN_ROOT="${ORQUESTA_TEST_BATCH_ROOT:-${ORQUESTA_TEST_CACHE_ROOT:-/srv/orquesta-self/runtime/test-cache}/batches/$RUN_ID}"
RECEIPT="${ORQUESTA_TEST_BATCH_RECEIPT:-$RUN_ROOT/receipt.json}"

case "$BATCH_SIZE:$PASSES" in *[!0-9:]*) echo "orquesta_test_batches=not_ok reason=invalid_numeric_config" >&2; exit 2;; esac
[ "$BATCH_SIZE" -gt 0 ] && [ "$PASSES" -eq 2 ] || { echo "orquesta_test_batches=not_ok reason=batch_size_or_two_pass_contract_invalid" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 && command -v "$TIMEOUT_COMMAND" >/dev/null 2>&1 && command -v "$GO_COMMAND" >/dev/null 2>&1 || {
  echo "orquesta_test_batches=not_ok reason=preflight_tool_missing" >&2; exit 2;
}

caller_umask="$(umask)"
umask 077
orquesta_private_test_root "$RUN_ROOT"
orquesta_private_test_root "$(dirname "$RECEIPT")"
mkdir -m 700 -p "$RUN_ROOT/logs" "$RUN_ROOT/receipts"
orquesta_use_isolated_test_env "$RUN_ROOT/env"
umask "$caller_umask"
summaries="$RUN_ROOT/batches.jsonl"
list_file="$RUN_ROOT/packages.txt"
list_stderr="$RUN_ROOT/go-list.stderr"
: >"$summaries"
started_ns="$(date +%s%N)"
finalized=0

write_receipt() {
  local status="$1" code="$2" reason="$3" list_rc="$4"
  python3 - "$summaries" "$status" "$code" "$reason" "$list_rc" "$started_ns" "$RUN_ID" "$RUN_ROOT" "$BATCH_SIZE" "$BATCH_TIMEOUT" "$BATCH_KILL_AFTER" "$GO_TEST_TIMEOUT" "$PASSES" "$list_file" >"$RECEIPT.tmp.$$" <<'PY'
import json, os, sys, time
summary,status,code,reason,list_rc,started,run_id,root,size,batch_timeout,kill_after,go_timeout,passes,list_file=sys.argv[1:15]
batches=[json.loads(line) for line in open(summary, encoding='utf-8') if line.strip()]
packages=[]
if os.path.exists(list_file): packages=[line.strip() for line in open(list_file, encoding='utf-8') if line.strip()]
passes_int=int(passes)
passes_completed=[p for p in range(1,passes_int+1) if any(b['pass']==p for b in batches) and all(b['status']=='passed' for b in batches if b['pass']==p)]
payload={
  'schema_version':'orquesta_test_batches_receipt.v1','run_id':run_id,'status':status,
  'exit_code':int(code),'reason_code':reason,'go_list_exit_code':int(list_rc),
  'run_root':root,'batch_size':int(size),'batch_timeout':batch_timeout,
  'batch_kill_after':kill_after,'go_test_timeout':go_timeout,'passes_required':int(passes),
  'passes_completed':passes_completed,
  'packages_total':len(packages),'package_executions_total':sum(len(b['packages']) for b in batches),
  'batches':batches,'started_at_unix_ns':int(started),'finished_at_unix_ns':time.time_ns(),
}
print(json.dumps(payload,sort_keys=True))
PY
  mv "$RECEIPT.tmp.$$" "$RECEIPT"
  finalized=1
}

cleanup() {
  local code=$?
  if [ "$finalized" != 1 ]; then write_receipt failed "$code" runner_interrupted -1 || true; fi
  orquesta_cleanup_isolated_test_env "$RUN_ROOT/env" || true
}
trap cleanup EXIT

patterns=("${@:-./...}")
set +e
"$GO_COMMAND" list "${patterns[@]}" >"$list_file" 2>"$list_stderr"
list_rc=$?
set -e
if [ "$list_rc" -ne 0 ]; then
  write_receipt failed "$list_rc" go_list_failed "$list_rc"
  echo "orquesta_test_batches=not_ok reason=go_list_failed rc=$list_rc receipt=$RECEIPT" >&2
  exit "$list_rc"
fi
mapfile -t packages <"$list_file"
if [ "${#packages[@]}" -eq 0 ]; then
  write_receipt failed 2 no_packages 0
  echo "orquesta_test_batches=not_ok reason=no_packages receipt=$RECEIPT" >&2
  exit 2
fi

overall=0
batch_count=0
for pass in 1 2; do
  pass_failed=0
  for ((offset=0; offset<${#packages[@]}; offset+=BATCH_SIZE)); do
    batch=("${packages[@]:offset:BATCH_SIZE}")
    batch_count=$((batch_count + 1))
    index=$((offset / BATCH_SIZE + 1))
    stem="pass-$(printf '%03d' "$pass")-batch-$(printf '%03d' "$index")"
    log="$RUN_ROOT/logs/$stem.log"
    batch_receipt="$RUN_ROOT/receipts/$stem.json"
    batch_started="$(date +%s%N)"
    set +e
    "$TIMEOUT_COMMAND" --signal=TERM --kill-after="$BATCH_KILL_AFTER" "$BATCH_TIMEOUT" \
      "$GO_COMMAND" test -count=1 -timeout "$GO_TEST_TIMEOUT" "${batch[@]}" >"$log" 2>&1
    code=$?
    set -e
    batch_finished="$(date +%s%N)"
    if [ "$code" -eq 0 ]; then status=passed; else status=failed; pass_failed=1; overall=1; fi
    [ "$code" -ne 124 ] && [ "$code" -ne 137 ] || status=timeout
    python3 - "$pass" "$index" "$status" "$code" "$batch_started" "$batch_finished" "$log" "$batch_receipt" "${batch[@]}" >"$batch_receipt.tmp.$$" <<'PY'
import json,sys
pas,index,status,code,started,finished,log,receipt,*packages=sys.argv[1:]
print(json.dumps({'schema_version':'orquesta_test_batch_receipt.v1','pass':int(pas),'index':int(index),'status':status,'exit_code':int(code),'started_at_unix_ns':int(started),'finished_at_unix_ns':int(finished),'log':log,'receipt':receipt,'packages':packages},sort_keys=True))
PY
    mv "$batch_receipt.tmp.$$" "$batch_receipt"
    cat "$batch_receipt" >>"$summaries"
  done
  [ "$pass_failed" -eq 0 ] || overall=1
done

if [ "$overall" -eq 0 ]; then
  write_receipt passed 0 two_consecutive_passes_passed 0
  echo "orquesta_test_batches=ok receipt=$RECEIPT passes=2 batches=$batch_count packages=${#packages[@]}"
else
  write_receipt failed 1 one_or_more_batches_failed 0
  echo "orquesta_test_batches=not_ok reason=one_or_more_batches_failed receipt=$RECEIPT" >&2
fi
exit "$overall"
