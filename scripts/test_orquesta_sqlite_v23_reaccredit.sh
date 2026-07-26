#!/usr/bin/env bash

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH
PYTHONDONTWRITEBYTECODE=1
export PYTHONDONTWRITEBYTECODE

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
SUBJECT="$ROOT/scripts/orquesta_sqlite_v23_reaccredit.sh"
TEST_ROOT="$(mktemp -d)"
readonly ROOT SUBJECT TEST_ROOT

cleanup() {
  find "$TEST_ROOT" -depth -mindepth 1 -delete 2>/dev/null || true
  rmdir "$TEST_ROOT" 2>/dev/null || true
}
trap cleanup EXIT

fail_test() {
  printf 'test_orquesta_sqlite_v23_reaccredit: status=failed case=%s\n' \
    "$1" >&2
  exit 1
}

assert_failure_cleanup() {
  expected_reason="$1"
  expected_verified="$2"
  expected_unit_state="$3"
  python3 - "$OUTPUT/failure.json" "$expected_reason" \
    "$expected_verified" "$expected_unit_state" <<'PY'
import json
import pathlib
import sys

receipt = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
assert receipt["result"] == "fail"
assert receipt["reason_code"] == sys.argv[2]
assert receipt["cleanup_verified"] is (sys.argv[3] == "true")
assert receipt["residues"]["unit_load_state"] == sys.argv[4]
assert isinstance(receipt["cleanup_attempts"], list)
assert not receipt["residues"]["credential_projections"]
PY
}

sha256_of() {
  sha256sum -- "$1" | awk '{print $1}'
}

set_common_value() {
  option="$1"
  value="$2"
  for ((index = 0; index < ${#COMMON[@]}; index++)); do
    if [ "${COMMON[$index]}" = "$option" ]; then
      COMMON[index + 1]="$value"
      return 0
    fi
  done
  fail_test "common_option_missing_$option"
}

write_fixture() {
  name="$1"
  FIXTURE="$TEST_ROOT/$name"
  REPOSITORY="$FIXTURE/repository"
  PRIVATE="$FIXTURE/private"
  LIVE="$FIXTURE/live"
  COMMANDS="$FIXTURE/commands"
  PROC_ROOT="$FIXTURE/proc"
  CGROUP_ROOT="$FIXTURE/cgroup"
  OUTPUT="$FIXTURE/output"
  mkdir -p -- \
    "$REPOSITORY/scripts/lib" \
    "$REPOSITORY/internal/adapters/state/sqlite/migrations" \
    "$PRIVATE" "$LIVE" "$COMMANDS" "$PROC_ROOT" "$CGROUP_ROOT"

  cat >"$REPOSITORY/scripts/orquesta_profile_server.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
action="${1:-}"
shift
profile=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --profile) profile="$2"; shift 2 ;;
    *) shift ;;
  esac
done
case "$action" in
  status)
    printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
    exit 3
    ;;
  *) exit 90 ;;
esac
EOF
  cat >"$REPOSITORY/scripts/orquesta_profile_systemd_user.sh" <<'PY'
#!/usr/bin/env python3
import hashlib
import os
import pathlib
import signal
import sqlite3
import sys
import time
import tomllib

args = sys.argv[1:]
operation = "check"
if args[:1] == ["--apply"]:
    operation = args[1]
    args = args[2:]
values = {}
index = 0
while index < len(args):
    values[args[index]] = args[index + 1]
    index += 2
systemctl = pathlib.Path(values["--systemctl"])
state = systemctl.parent / "unit-state"
log = systemctl.parent / "operations"
with log.open("a", encoding="utf-8") as handle:
    handle.write(operation + "\n")
counter_path = systemctl.parent / "cycle-count"
invocation_path = systemctl.parent / "current-invocation"
if operation == "start":
    counter = int(counter_path.read_text(encoding="utf-8")) + 1 \
        if counter_path.exists() else 1
    counter_path.write_text(str(counter) + "\n", encoding="utf-8")
    invocation = f"{counter:032x}"
    invocation_path.write_text(invocation + "\n", encoding="utf-8")
else:
    invocation = invocation_path.read_text(encoding="utf-8").strip()
if operation == "start":
    with open(values["--config"], "rb") as handle:
        config = tomllib.load(handle)
    manager_cgroup = (
        f"/user.slice/user-{os.geteuid()}.slice/"
        f"user@{os.geteuid()}.service"
    )
    unit_cgroup = manager_cgroup + "/app.slice/" + values["--unit"]
    delegated_root = (
        pathlib.Path(values["--cgroup-mount"])
        / unit_cgroup.removeprefix("/")
    )
    assert config["runtime"]["codex"]["cgroup_root"] == str(delegated_root)
    assert (
        config["test_attestor"]["resources"]["cgroup_root"]
        == str(delegated_root)
    )
    database = config["state"]["sqlite"]["path"]
    repository = pathlib.Path(values["--repository-root"])
    with sqlite3.connect(database) as connection:
        current = connection.execute("PRAGMA user_version").fetchone()[0]
        if current == 16:
            for version in (17, 18, 19):
                migration = next(
                    (repository / "internal/adapters/state/sqlite/migrations").glob(
                        f"{version:03d}_*.sql"
                    )
                )
                checksum = "sha256:" + hashlib.sha256(
                    migration.read_bytes()
                ).hexdigest()
                if not (systemctl.parent / "skip-migration-ddl").exists():
                    connection.executescript(
                        migration.read_text(encoding="utf-8")
                    )
                connection.execute(
                    "INSERT INTO schema_migrations VALUES(?,?,?)",
                    (version, migration.name, checksum),
                )
            connection.execute("PRAGMA user_version=19")
            if (systemctl.parent / "seed-new-functional-row").exists():
                connection.execute(
                    "CREATE TABLE unexpected_v19_data(value TEXT)"
                )
                connection.execute(
                    "INSERT INTO unexpected_v19_data VALUES('unexpected')"
                )
            if (systemctl.parent / "rewrite-audit-history").exists():
                connection.execute(
                    "UPDATE command_invocations "
                    "SET ref='tampered:0' WHERE ref='status:0'"
                )
                connection.execute(
                    "UPDATE command_outcomes "
                    "SET ref='tampered:0' WHERE ref='status:0'"
                )
                connection.execute(
                    "UPDATE authorization_receipts "
                    "SET ref='tampered:0' WHERE ref='status:0'"
                )
        count = connection.execute(
            "SELECT COUNT(*) FROM command_invocations"
        ).fetchone()[0] + 1
        ref = f"status:{count}"
        connection.execute(
            "INSERT INTO command_invocations VALUES(?,?)",
            (ref, "orquesta.system.status"),
        )
        connection.execute("INSERT INTO command_outcomes VALUES(?)", (ref,))
        connection.execute(
            "INSERT INTO authorization_receipts VALUES(?)", (ref,)
        )
    process_cgroup = unit_cgroup + "/orquesta-control"
    if (systemctl.parent / "place-process-in-parent-cgroup").exists():
        process_cgroup = unit_cgroup
    for pid in (100, 101):
        proc = pathlib.Path(values["--proc-root"]) / str(pid)
        proc.mkdir(mode=0o700, exist_ok=True)
        (proc / "cgroup").write_text(
            "0::" + process_cgroup + "\n",
            encoding="utf-8",
        )
    state.write_text("loaded\n", encoding="utf-8")
    if (systemctl.parent / "break-evidence-after-unit").exists():
        evidence = pathlib.Path(values["--config"]).parent / "evidence"
        evidence.rename(evidence.with_name("evidence-before-fault"))
        evidence.write_text("fault\n", encoding="utf-8")
    if (systemctl.parent / "hang-start-after-unit").exists():
        time.sleep(30)
    if (systemctl.parent / "fail-start-after-unit").exists():
        raise SystemExit(97)
    print(
        "orquesta_profile_systemd_user: status=running action=start "
        f"unit={values['--unit']} profile={values['--profile']} "
        f"main_pid=100 daemon_pid=101 invocation_id={invocation}"
    )
elif operation == "check":
    print(
        "orquesta_profile_systemd_user: status=running action=check "
        f"unit={values['--unit']} profile={values['--profile']} "
        f"main_pid=100 daemon_pid=101 invocation_id={invocation}"
    )
elif operation == "stop-profile":
    if (systemctl.parent / "cleanup-actions-fail").exists():
        raise SystemExit(98)
    if (systemctl.parent / "cleanup-hangs-with-child").exists():
        child = os.fork()
        if child == 0:
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
            time.sleep(30)
            raise SystemExit(0)
        (systemctl.parent / "cleanup-child-pid").write_text(
            str(child) + "\n",
            encoding="utf-8",
        )
        time.sleep(30)
    print(
        "orquesta_profile_systemd_user: status=profile_stopped "
        f"action=stop-profile unit={values['--unit']} "
        f"profile={values['--profile']} invocation_id={invocation}"
    )
elif operation == "collect":
    if (systemctl.parent / "cleanup-actions-fail").exists():
        raise SystemExit(99)
    state.write_text("not-found\n", encoding="utf-8")
    print(
        "orquesta_profile_systemd_user: status=collected action=collect "
        f"unit={values['--unit']} profile={values['--profile']} "
        f"invocation_id={invocation}"
    )
else:
    raise SystemExit(91)
PY
  for helper in \
    pidfd_signal.py firecracker_launcher_probe.py profile_maintenance_marker.py; do
    printf '%s\n' '#!/usr/bin/env python3' >"$REPOSITORY/scripts/lib/$helper"
  done
  chmod 500 -- \
    "$REPOSITORY/scripts/orquesta_profile_server.sh" \
    "$REPOSITORY/scripts/orquesta_profile_systemd_user.sh" \
    "$REPOSITORY/scripts/lib/"*.py

  cat >"$COMMANDS/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
state_file="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/unit-state"
unit=""
property=""
action=""
for argument in "$@"; do
  case "$argument" in
    --property=*) property="${argument#--property=}" ;;
    stop|reset-failed) action="$argument" ;;
    *.service) unit="$argument" ;;
  esac
done
if [ -n "$action" ]; then
  if [ -e "$(dirname "$state_file")/systemctl-fallback-fails" ]; then
    exit 1
  fi
  if [ "$action" = stop ]; then
    printf '%s\n' 'not-found' >"$state_file"
  fi
elif [ "$property" = "ControlGroup" ]; then
  if [ -n "$unit" ]; then
    printf '/user.slice/user-%s.slice/user@%s.service/app.slice/%s\n' \
      "$(id -u)" "$(id -u)" "$unit"
  else
    printf '/user.slice/user-%s.slice/user@%s.service\n' \
      "$(id -u)" "$(id -u)"
  fi
elif [ -f "$state_file" ]; then
  tr -d '\n' <"$state_file"
  printf '\n'
else
  printf '%s\n' 'not-found'
fi
EOF
  cat >"$COMMANDS/systemd-run" <<'EOF'
#!/usr/bin/env bash
exit 92
EOF
  cat >"$COMMANDS/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[ "${1:-}" = version ] && [ "${2:-}" = -m ] || exit 93
printf '%s\n' \
  "${3:-}: go1.25.0" \
  $'\tpath\torquesta/cmd/orquesta' \
  $'\tbuild\tvcs.revision=0123456789abcdef0123456789abcdef01234567' \
  $'\tbuild\tvcs.modified=false'
EOF
  chmod 500 -- "$COMMANDS/systemctl" "$COMMANDS/systemd-run" "$COMMANDS/go"

  cat >"$PRIVATE/orquesta" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "version" ]; then
  printf '%s\n' '0123456789abcdef0123456789abcdef01234567'
  exit 0
fi
exit 2
EOF
  chmod 500 -- "$PRIVATE/orquesta"
  cat >"$PRIVATE/config-template.toml" <<'EOF'
[server]
listen = "@@ORQUESTA_REACCREDIT_LISTEN@@"
[state.sqlite]
path = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/state/orquesta.sqlite"
[artifact.filesystem]
root = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/artifacts"
[credentials.local]
path = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/credentials.json"
[runtime.codex]
account_home_root = "@@ORQUESTA_REACCREDIT_ACCOUNT_ROOT@@"
account_profile = "@@ORQUESTA_REACCREDIT_PROFILE@@"
command = "/bin/false"
work_root = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/work"
cache_root = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/cache"
go_toolchain_root = ""
cgroup_root = "@@ORQUESTA_REACCREDIT_CGROUP_ROOT@@"
[workspace.local]
root = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/workspaces"
[repository.local]
seed_path = "@@ORQUESTA_REACCREDIT_REPOSITORY_SEED@@"
[identity]
local_token_path = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/local.token"
[config]
effective_path = "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/effective.json"
[test_attestor]
provider = "bubblewrap"
[test_attestor.bubblewrap]
command = "@@ORQUESTA_REACCREDIT_BUBBLEWRAP@@"
[test_attestor.go]
toolchain_root = ""
[test_attestor.resources]
cgroup_root = "@@ORQUESTA_REACCREDIT_CGROUP_ROOT@@"
[test_attestor.microvm]
launcher_socket = ""
EOF
  chmod 600 -- "$PRIVATE/config-template.toml"

  for version in $(seq 1 19); do
    case "$version" in
      17)
        name="017_intake.sql"
        migration_sql='CREATE TABLE intake_states_fixture(ref TEXT PRIMARY KEY);'
        ;;
      18)
        name="018_intake_dossiers.sql"
        migration_sql='CREATE TABLE intake_dossiers_fixture(ref TEXT PRIMARY KEY);'
        ;;
      19)
        name="019_intake_dossier_confirmations.sql"
        migration_sql='CREATE TABLE intake_confirmations_fixture(ref TEXT PRIMARY KEY);'
        ;;
      *)
        name="$(printf '%03d_fixture.sql' "$version")"
        migration_sql="-- fixture migration $version"
        ;;
    esac
    printf '%s\n' "$migration_sql" \
      >"$REPOSITORY/internal/adapters/state/sqlite/migrations/$name"
    chmod 600 -- \
      "$REPOSITORY/internal/adapters/state/sqlite/migrations/$name"
  done

  python3 - "$PRIVATE/backup.sqlite" "$REPOSITORY" <<'PY'
import hashlib
import pathlib
import sqlite3
import sys

database = pathlib.Path(sys.argv[1])
repository = pathlib.Path(sys.argv[2])
with sqlite3.connect(database) as connection:
    connection.executescript(
        """
        CREATE TABLE schema_migrations(
          version INTEGER PRIMARY KEY,name TEXT,checksum TEXT
        );
        CREATE TABLE goals(ref TEXT PRIMARY KEY);
        CREATE TABLE executions(ref TEXT PRIMARY KEY);
        CREATE TABLE work_items(ref TEXT PRIMARY KEY);
        CREATE TABLE outbox(ref TEXT PRIMARY KEY);
        CREATE TABLE effect_attempts(ref TEXT PRIMARY KEY);
        CREATE TABLE command_invocations(
          ref TEXT PRIMARY KEY,command_id TEXT NOT NULL
        );
        CREATE TABLE command_outcomes(ref TEXT PRIMARY KEY);
        CREATE TABLE authorization_receipts(ref TEXT PRIMARY KEY);
        INSERT INTO goals VALUES('goal:fixture');
        INSERT INTO executions VALUES('execution:fixture');
        INSERT INTO work_items VALUES('work-item:fixture');
        INSERT INTO outbox VALUES('action:fixture');
        INSERT INTO effect_attempts VALUES('attempt:fixture');
        INSERT INTO command_invocations
          VALUES('status:0','orquesta.system.status');
        INSERT INTO command_outcomes VALUES('status:0');
        INSERT INTO authorization_receipts VALUES('status:0');
        """
    )
    migration_root = repository / "internal/adapters/state/sqlite/migrations"
    for version in range(1, 17):
        migration = next(migration_root.glob(f"{version:03d}_*.sql"))
        checksum = "sha256:" + hashlib.sha256(migration.read_bytes()).hexdigest()
        connection.execute(
            "INSERT INTO schema_migrations VALUES(?,?,?)",
            (version, migration.name, checksum),
        )
    connection.execute("PRAGMA user_version=16")
PY
  chmod 600 -- "$PRIVATE/backup.sqlite"

  COMMON=(
    --backup "$PRIVATE/backup.sqlite"
    --expected-backup-sha256 "$(sha256_of "$PRIVATE/backup.sqlite")"
    --binary "$PRIVATE/orquesta"
    --expected-binary-sha256 "$(sha256_of "$PRIVATE/orquesta")"
    --expected-revision 0123456789abcdef0123456789abcdef01234567
    --repository-root "$REPOSITORY"
    --profile-script "$REPOSITORY/scripts/orquesta_profile_server.sh"
    --expected-profile-script-sha256 \
      "$(sha256_of "$REPOSITORY/scripts/orquesta_profile_server.sh")"
    --adapter "$REPOSITORY/scripts/orquesta_profile_systemd_user.sh"
    --expected-adapter-sha256 \
      "$(sha256_of "$REPOSITORY/scripts/orquesta_profile_systemd_user.sh")"
    --config-template "$PRIVATE/config-template.toml"
    --expected-config-template-sha256 \
      "$(sha256_of "$PRIVATE/config-template.toml")"
    --output-dir "$OUTPUT"
    --forbidden-live-root "$LIVE"
    --exec-path "$COMMANDS:/usr/bin:/bin"
    --go "$COMMANDS/go"
    --expected-go-sha256 "$(sha256_of "$COMMANDS/go")"
    --git /usr/bin/git
    --expected-git-sha256 "$(sha256_of /usr/bin/git)"
    --bubblewrap /usr/bin/bwrap
    --expected-bubblewrap-sha256 "$(sha256_of /usr/bin/bwrap)"
    --systemctl "$COMMANDS/systemctl"
    --expected-systemctl-sha256 "$(sha256_of "$COMMANDS/systemctl")"
    --systemd-run "$COMMANDS/systemd-run"
    --expected-systemd-run-sha256 "$(sha256_of "$COMMANDS/systemd-run")"
    --proc-root "$PROC_ROOT"
    --cgroup-mount "$CGROUP_ROOT"
    --readiness-timeout 1
    --collection-timeout 1
    --command-timeout 5
  )
}

write_fixture success
"$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" ||
  fail_test "success_execution"
python3 - "$OUTPUT/receipt.json" "$OUTPUT" "$REPOSITORY" "$LIVE" <<'PY' ||
import hashlib
import json
import os
import pathlib
import stat
import sys
import tomllib

receipt_path = pathlib.Path(sys.argv[1])
root = pathlib.Path(sys.argv[2])
repository = pathlib.Path(sys.argv[3])
live = pathlib.Path(sys.argv[4])
receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
assert set(receipt) == {
    "schema_version", "result", "candidate", "source", "first_start",
    "restart", "result_database", "closure", "harness", "evidence",
}
assert receipt["schema_version"] == "orquesta_sqlite_upgrade_audit.v1"
assert receipt["result"] == "pass"
assert receipt["harness"]["checks"]["user_version_16_19_19"] is True
assert receipt["harness"]["checks"]["schema_manifest_exact"] is True
assert receipt["source"]["user_version"] == 16
assert receipt["first_start"]["result_user_version"] == 19
assert receipt["restart"]["result_user_version"] == 19
assert receipt["first_start"]["schema_migrations_added"] == [17, 18, 19]
assert receipt["restart"]["schema_migrations_duplicated"] is False
digests = {
    receipt[key]["functional_data_sha256"]
    for key in ("source", "first_start", "restart", "result_database")
}
assert len(digests) == 1
assert receipt["closure"]["candidate_processes"] == 0
assert receipt["closure"]["database_open_processes"] == 0
assert receipt["closure"]["unit_load_state"] == "not-found"
assert receipt["closure"]["credential_projections"] == 0
assert receipt["candidate"]["binary_vcs_modified"] is False
assert "audit_row_fingerprints" not in receipt_path.read_text(encoding="utf-8")
assert set(receipt["candidate"]["profile_helper_sha256"]) == {
    "firecracker_launcher_probe.py",
    "pidfd_signal.py",
    "profile_maintenance_marker.py",
}
expected_migrations = {}
migration_root = repository / "internal/adapters/state/sqlite/migrations"
for version in range(1, 20):
    path = next(migration_root.glob(f"{version:03d}_*.sql"))
    digest = hashlib.sha256(path.read_bytes()).hexdigest()
    expected_migrations[version] = {
        "name": path.name,
        "checksum": "sha256:" + digest,
        "sha256": digest,
    }
manifest_payload = json.dumps(
    expected_migrations, sort_keys=True, separators=(",", ":")
).encode("utf-8")
assert receipt["candidate"]["migration_manifest_sha256"] == hashlib.sha256(
    manifest_payload
).hexdigest()
result_migrations = {
    item["version"]: (item["name"], item["checksum"])
    for item in receipt["result_database"]["schema_migrations"]
}
for version in (17, 18, 19):
    expected = expected_migrations[version]
    assert result_migrations[version] == (
        expected["name"], expected["checksum"]
    )
result_database = root / receipt["result_database"]["path"]
assert result_database.is_relative_to(root)
assert hashlib.sha256(result_database.read_bytes()).hexdigest() == (
    receipt["result_database"]["sha256"]
)
for item in receipt["evidence"]:
    for stream_name in ("stdout", "stderr"):
        stream = item[stream_name]
        path = root / stream["path"]
        assert path.is_relative_to(root)
        content = path.read_bytes()
        assert len(content) == stream["bytes"]
        assert hashlib.sha256(content).hexdigest() == stream["sha256"]
assert pathlib.Path(receipt["harness"]["output_dir"]) == root
with (root / "config.toml").open("rb") as handle:
    config = tomllib.load(handle)
assert config["runtime"]["codex"]["cgroup_root"] == (
    receipt["harness"]["delegated_cgroup_root"]
)
assert config["test_attestor"]["resources"]["cgroup_root"] == (
    receipt["harness"]["delegated_cgroup_root"]
)
assert receipt["harness"]["unit_control_group"] == (
    receipt["harness"]["delegated_cgroup_root"] + "/orquesta-control"
)
assert not root.is_relative_to(live)
assert not list(root.rglob("auth.json"))
assert stat.S_IMODE(receipt_path.stat().st_mode) == 0o600
for current, directories, files in os.walk(root):
    assert stat.S_IMODE(pathlib.Path(current).stat().st_mode) == 0o700
PY
  fail_test "success_receipt"
expected_operations=$'start\ncheck\nstop-profile\ncollect\nstart\ncheck\nstop-profile\ncollect'
[ "$(<"$COMMANDS/operations")" = "$expected_operations" ] ||
  fail_test "operation_sequence"

write_fixture symlink
mv -- "$PRIVATE/backup.sqlite" "$PRIVATE/backup-real.sqlite"
ln -s -- "$PRIVATE/backup-real.sqlite" "$PRIVATE/backup.sqlite"
set_common_value --backup "$PRIVATE/backup.sqlite"
set_common_value \
  --expected-backup-sha256 "$(sha256_of "$PRIVATE/backup-real.sqlite")"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "symlink_accepted"
fi
grep -q 'reason_code=backup_symlink_or_noncanonical' "$FIXTURE/stderr" ||
  fail_test "symlink_reason"
[ ! -e "$COMMANDS/operations" ] || fail_test "symlink_adapter_called"

write_fixture migration-symlink
migration="$REPOSITORY/internal/adapters/state/sqlite/migrations/019_intake_dossier_confirmations.sql"
mv -- "$migration" "$migration.real"
ln -s -- "$migration.real" "$migration"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "migration_symlink_accepted"
fi
grep -q 'reason_code=migration_file_symlink_or_noncanonical' \
  "$FIXTURE/stderr" || fail_test "migration_symlink_reason"
[ ! -e "$COMMANDS/operations" ] ||
  fail_test "migration_symlink_adapter_called"

write_fixture hash
set_common_value --expected-backup-sha256 \
  aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "hash_mismatch_accepted"
fi
grep -q 'reason_code=backup_hash_mismatch' "$FIXTURE/stderr" ||
  fail_test "hash_mismatch_reason"
[ ! -e "$COMMANDS/operations" ] || fail_test "hash_adapter_called"

write_fixture live
mv -- "$PRIVATE/backup.sqlite" "$LIVE/backup.sqlite"
set_common_value --backup "$LIVE/backup.sqlite"
set_common_value \
  --expected-backup-sha256 "$(sha256_of "$LIVE/backup.sqlite")"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "live_path_accepted"
fi
grep -q 'reason_code=backup_points_to_live_codex12' "$FIXTURE/stderr" ||
  fail_test "live_path_reason"
[ ! -e "$COMMANDS/operations" ] || fail_test "live_adapter_called"

write_fixture modified-buildinfo
sed -i 's/vcs.modified=false/vcs.modified=true/' "$COMMANDS/go"
set_common_value --expected-go-sha256 "$(sha256_of "$COMMANDS/go")"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "modified_buildinfo_accepted"
fi
grep -q 'reason_code=binary_revision_mismatch' "$FIXTURE/stderr" ||
  fail_test "modified_buildinfo_reason"
[ ! -e "$COMMANDS/operations" ] ||
  fail_test "modified_buildinfo_adapter_called"

write_fixture start-cleanup
: >"$COMMANDS/fail-start-after-unit"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "start_failure_accepted"
fi
grep -q 'reason_code=command_failed' "$FIXTURE/stderr" ||
  fail_test "start_failure_reason"
[ "$(<"$COMMANDS/operations")" = $'start\nstop-profile\ncollect' ] ||
  fail_test "start_failure_cleanup_sequence"
[ "$("$COMMANDS/systemctl" --user show ignored.service \
  --property=LoadState --value)" = "not-found" ] ||
  fail_test "start_failure_unit_residue"
if find "$OUTPUT/accounts" -type f -name auth.json -print -quit | grep -q .; then
  fail_test "start_failure_auth_residue"
fi
assert_failure_cleanup command_failed true not-found ||
  fail_test "start_failure_cleanup_receipt"

write_fixture cleanup-fallback
: >"$COMMANDS/fail-start-after-unit"
: >"$COMMANDS/cleanup-actions-fail"
set_common_value --command-timeout 1
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "cleanup_fallback_failure_accepted"
fi
assert_failure_cleanup command_failed true not-found ||
  fail_test "cleanup_fallback_receipt"
python3 - "$OUTPUT/failure.json" <<'PY' ||
import json
import pathlib
import sys

receipt = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
returncodes = {
    item["label"]: item["returncode"] for item in receipt["cleanup_attempts"]
}
assert returncodes["failure-cleanup-stop"] == 98
assert returncodes["failure-cleanup-collect"] == 99
assert returncodes["failure-cleanup-systemctl-stop"] == 0
PY
  fail_test "cleanup_fallback_attempts"

write_fixture cleanup-residue
: >"$COMMANDS/fail-start-after-unit"
: >"$COMMANDS/cleanup-actions-fail"
: >"$COMMANDS/systemctl-fallback-fails"
set_common_value --command-timeout 1
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "cleanup_residue_failure_accepted"
fi
assert_failure_cleanup command_failed false loaded ||
  fail_test "cleanup_residue_receipt"

write_fixture start-timeout
: >"$COMMANDS/hang-start-after-unit"
set_common_value --command-timeout 1
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "start_timeout_accepted"
fi
assert_failure_cleanup command_timeout true not-found ||
  fail_test "start_timeout_cleanup_receipt"

write_fixture post-start-os-error
: >"$COMMANDS/break-evidence-after-unit"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "post_start_os_error_accepted"
fi
assert_failure_cleanup unexpected_os_error true not-found ||
  fail_test "post_start_os_error_cleanup_receipt"

for requested_signal in TERM INT HUP; do
  write_fixture "signal-${requested_signal,,}"
  : >"$COMMANDS/hang-start-after-unit"
  set_common_value --command-timeout 30
  "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr" &
  harness_pid="$!"
  loaded=false
  for _ in $(seq 1 100); do
    if [ -f "$COMMANDS/unit-state" ] &&
      [ "$(<"$COMMANDS/unit-state")" = loaded ]; then
      loaded=true
      break
    fi
    sleep 0.05
  done
  [ "$loaded" = true ] || fail_test "signal_unit_not_started"
  kill "-$requested_signal" "$harness_pid"
  set +e
  wait "$harness_pid"
  signal_status="$?"
  set -e
  [ "$signal_status" -ne 0 ] || fail_test "signal_exit_zero"
  assert_failure_cleanup interrupted_by_signal true not-found ||
    fail_test "signal_cleanup_receipt"
done

write_fixture cleanup-timeout-child
: >"$COMMANDS/fail-start-after-unit"
: >"$COMMANDS/cleanup-hangs-with-child"
set_common_value --command-timeout 1
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "cleanup_timeout_child_accepted"
fi
assert_failure_cleanup command_failed true not-found ||
  fail_test "cleanup_timeout_child_receipt"
cleanup_child_pid="$(<"$COMMANDS/cleanup-child-pid")"
if [ -r "/proc/$cleanup_child_pid/stat" ]; then
  cleanup_child_state="$(awk '{print $3}' "/proc/$cleanup_child_pid/stat")"
  [ "$cleanup_child_state" = Z ] ||
    fail_test "cleanup_timeout_child_alive"
fi

write_fixture new-functional-row
: >"$COMMANDS/seed-new-functional-row"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "new_functional_row_accepted"
fi
grep -q 'reason_code=functional_counts_changed' "$FIXTURE/stderr" ||
  fail_test "new_functional_row_reason"
[ "$("$COMMANDS/systemctl" --user show ignored.service \
  --property=LoadState --value)" = "not-found" ] ||
  fail_test "new_functional_row_unit_residue"

write_fixture rewritten-audit-history
: >"$COMMANDS/rewrite-audit-history"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "rewritten_audit_history_accepted"
fi
grep -q 'reason_code=status_audit_delta_invalid' "$FIXTURE/stderr" ||
  fail_test "rewritten_audit_history_reason"
[ "$("$COMMANDS/systemctl" --user show ignored.service \
  --property=LoadState --value)" = "not-found" ] ||
  fail_test "rewritten_audit_history_unit_residue"

write_fixture migration-receipts-without-ddl
: >"$COMMANDS/skip-migration-ddl"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "migration_receipts_without_ddl_accepted"
fi
grep -q 'reason_code=sqlite_schema_manifest_changed' "$FIXTURE/stderr" ||
  fail_test "migration_receipts_without_ddl_reason"
[ "$("$COMMANDS/systemctl" --user show ignored.service \
  --property=LoadState --value)" = "not-found" ] ||
  fail_test "migration_receipts_without_ddl_unit_residue"

write_fixture process-in-parent-cgroup
: >"$COMMANDS/place-process-in-parent-cgroup"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "process_in_parent_cgroup_accepted"
fi
grep -q 'reason_code=runtime_process_cgroup_mismatch' "$FIXTURE/stderr" ||
  fail_test "process_in_parent_cgroup_reason"
[ "$("$COMMANDS/systemctl" --user show ignored.service \
  --property=LoadState --value)" = "not-found" ] ||
  fail_test "process_in_parent_cgroup_unit_residue"

write_fixture config-live
python3 - "$PRIVATE/config-template.toml" "$LIVE/tool" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
path.write_text(
    path.read_text(encoding="utf-8").replace("/bin/false", sys.argv[2]),
    encoding="utf-8",
)
PY
chmod 600 -- "$PRIVATE/config-template.toml"
set_common_value --expected-config-template-sha256 \
  "$(sha256_of "$PRIVATE/config-template.toml")"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "config_live_path_accepted"
fi
grep -q 'reason_code=materialized_config_points_to_live_codex12' \
  "$FIXTURE/stderr" || fail_test "config_live_path_reason"
[ ! -e "$COMMANDS/operations" ] || fail_test "config_live_adapter_called"

write_fixture config-cgroup
python3 - "$PRIVATE/config-template.toml" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
marker = "@@ORQUESTA_REACCREDIT_CGROUP_ROOT@@"
text = path.read_text(encoding="utf-8")
path.write_text(
    text.replace(marker, "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@/wrong", 1),
    encoding="utf-8",
)
PY
chmod 600 -- "$PRIVATE/config-template.toml"
set_common_value --expected-config-template-sha256 \
  "$(sha256_of "$PRIVATE/config-template.toml")"
if "$SUBJECT" "${COMMON[@]}" >"$FIXTURE/stdout" 2>"$FIXTURE/stderr"; then
  fail_test "config_cgroup_mismatch_accepted"
fi
grep -q 'reason_code=materialized_config_binding_invalid' \
  "$FIXTURE/stderr" || fail_test "config_cgroup_mismatch_reason"
[ ! -e "$COMMANDS/operations" ] ||
  fail_test "config_cgroup_adapter_called"

python3 - "$ROOT/scripts/lib/orquesta_sqlite_v23_reaccredit.py" \
  "$TEST_ROOT/terminal-publication" <<'PY' ||
import importlib.util
import os
import pathlib
import stat
import sys

spec = importlib.util.spec_from_file_location("subject_terminal", sys.argv[1])
subject = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = subject
spec.loader.exec_module(subject)
root = pathlib.Path(sys.argv[2])
root.mkdir(mode=0o700)

fsync_failure = root / "fsync-failure.json"
real_fsync = subject.os.fsync

def fail_directory_fsync(descriptor):
    if stat.S_ISDIR(os.fstat(descriptor).st_mode):
        raise OSError("fixture directory fsync failure")
    return real_fsync(descriptor)

subject.os.fsync = fail_directory_fsync
try:
    try:
        subject.write_terminal_receipt(fsync_failure, {"result": "pass"})
    except OSError:
        pass
    else:
        raise AssertionError("directory fsync failure accepted")
finally:
    subject.os.fsync = real_fsync
assert not fsync_failure.exists()
assert not fsync_failure.with_name(".fsync-failure.json.pending").exists()

existing = root / "existing.json"
existing.write_text("original\n", encoding="utf-8")
existing.chmod(0o600)
try:
    subject.write_terminal_receipt(existing, {"result": "pass"})
except FileExistsError:
    pass
else:
    raise AssertionError("terminal receipt overwrite accepted")
assert existing.read_text(encoding="utf-8") == "original\n"
assert not existing.with_name(".existing.json.pending").exists()

published = root / "published.json"
subject.write_terminal_receipt(published, {"result": "pass"})
assert published.stat().st_nlink == 1
assert stat.S_IMODE(published.stat().st_mode) == 0o600
PY
  fail_test "terminal_receipt_publication"

python3 - "$ROOT/scripts/lib/orquesta_sqlite_v23_reaccredit.py" \
  "$TEST_ROOT/wal-visible.sqlite" <<'PY' ||
import importlib.util
import pathlib
import sqlite3
import sys

spec = importlib.util.spec_from_file_location("subject", sys.argv[1])
subject = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = subject
spec.loader.exec_module(subject)
database = pathlib.Path(sys.argv[2])
writer = sqlite3.connect(database)
try:
    assert writer.execute("PRAGMA journal_mode=WAL").fetchone()[0] == "wal"
    writer.execute("CREATE TABLE facts(value TEXT)")
    writer.commit()
    writer.execute("PRAGMA wal_checkpoint(TRUNCATE)")
    writer.execute("INSERT INTO facts VALUES('wal-visible')")
    writer.commit()
    reader = subject.sqlite_connect_readonly(database)
    try:
        assert reader.execute("SELECT value FROM facts").fetchone()[0] == "wal-visible"
    finally:
        reader.close()
finally:
    writer.close()
PY
  fail_test "readonly_wal_visibility"

printf '%s\n' 'test_orquesta_sqlite_v23_reaccredit: status=ok'
