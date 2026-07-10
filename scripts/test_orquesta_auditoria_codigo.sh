#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$root/scripts/orquesta_auditoria_codigo.sh"

tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-code-audit-test.XXXXXX")"
trap 'rm -rf "$tmp_root"' EXIT

fixture="$tmp_root/repo"
out_dir="$tmp_root/out"
mkdir -p \
  "$fixture/cmd/app" \
  "$fixture/modulos/live" \
  "$fixture/modulos/helpers" \
  "$fixture/modulos/orphan" \
  "$fixture/modulos/large" \
  "$out_dir"

cat >"$fixture/go.mod" <<'GO'
module fixture.test

go 1.22
GO

cat >"$fixture/cmd/app/main.go" <<'GO'
package main

import "fixture.test/modulos/live"

func main() {
	live.Live()
}
GO

cat >"$fixture/modulos/live/live.go" <<'GO'
package live

func Live() {}

func firstNonEmptyLiveV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
GO

cat >"$fixture/modulos/helpers/helpers.go" <<'GO'
package helpers

func firstNonEmptyHelperV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func compactHelperV0(value string) string {
	return value
}

func compactOtherHelperV0(value string) string {
	return value
}
GO

cat >"$fixture/modulos/orphan/orphan.go" <<'GO'
package orphan

func OrphanV0() {}
GO

{
  echo "package large"
  echo
  echo "func LargeV0() {}"
  for index in $(seq 1 805); do
    printf '// filler %03d\n' "$index"
  done
} >"$fixture/modulos/large/large.go"

deadcode_file="$tmp_root/deadcode.txt"
cat >"$deadcode_file" <<'EOF'
modulos/live/live.go:5:6: unreachable func: firstNonEmptyLiveV0
modulos/orphan/orphan.go:3:6: unreachable func: OrphanV0
EOF

json_out="$out_dir/audit.json"
sqlite_out="$out_dir/audit.sqlite"

"$script" \
  --root "$fixture" \
  --out-dir "$out_dir" \
  --deadcode-file "$deadcode_file" \
  --json-out "$json_out" \
  --sqlite-out "$sqlite_out" >"$tmp_root/script.out"

grep -q '^code_audit_json=' "$tmp_root/script.out"
grep -q '^code_audit_sqlite=' "$tmp_root/script.out"
grep -q '^code_audit_deadcode_candidates=2$' "$tmp_root/script.out"

python3 - "$json_out" "$sqlite_out" <<'PY'
import json
import sqlite3
import sys

json_path, sqlite_path = sys.argv[1:3]
with open(json_path, encoding="utf-8") as fh:
    payload = json.load(fh)

assert payload["schema_version"] == "orquesta_code_audit.v0", payload
assert payload["deadcode"]["source"] == "provided_file", payload["deadcode"]
assert payload["metrics"]["deadcode_candidates"] == 2, payload["metrics"]
assert payload["metrics"]["helper_duplicate_definitions"] >= 2, payload["metrics"]
assert payload["metrics"]["large_files_over_800"] == 1, payload["metrics"]
assert payload["metrics"]["functions_indexed"] >= 7, payload["metrics"]
assert any(item["name"] == "firstNonEmptyLiveV0" and item["classification"] == "static_candidate_requires_review" for item in payload["function_index"]["entries"]), payload["function_index"]
assert any(item["module"] == "modulos/orphan" for item in payload["orphan_modules"]), payload["orphan_modules"]
assert any(item["path"] == "modulos/large/large.go" for item in payload["large_files"]), payload["large_files"]

con = sqlite3.connect(sqlite_path)
try:
    metrics = dict(con.execute("select metric, value from metrics"))
    assert metrics["deadcode_candidates"] == 2, metrics
    assert metrics["large_files_over_800"] == 1, metrics
    assert metrics["functions_indexed"] >= 7, metrics
    deadcode_rows = con.execute("select count(*) from deadcode").fetchone()[0]
    helper_rows = con.execute("select count(*) from helper_copies").fetchone()[0]
    assert deadcode_rows == 2, deadcode_rows
    assert helper_rows >= 4, helper_rows
    function_rows = con.execute("select count(*) from function_index").fetchone()[0]
    assert function_rows >= 7, function_rows
finally:
    con.close()
PY

before_files="$(cd "$fixture" && find . -type f -printf '%P\n' | sort)"
"$script" --root "$fixture" --out-dir "$out_dir" --deadcode-file "$deadcode_file" --no-sqlite >/dev/null
after_files="$(cd "$fixture" && find . -type f -printf '%P\n' | sort)"
if [[ "$before_files" != "$after_files" ]]; then
  echo "code audit modified fixture files" >&2
  exit 1
fi

gopath="$tmp_root/go-path"
mkdir -p "$gopath/bin"
cat >"$gopath/bin/deadcode" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
echo "modulos/live/live.go:5:6: unreachable func: firstNonEmptyLiveV0"
echo "modulos/orphan/orphan.go:3:6: unreachable func: OrphanV0"
SH
chmod +x "$gopath/bin/deadcode"

go_bin_dir="$(dirname "$(command -v go)")"
python_bin_dir="$(dirname "$(command -v python3)")"
json_gopath="$out_dir/audit-gopath.json"
PATH="$go_bin_dir:$python_bin_dir:/usr/bin:/bin" \
GOPATH="$gopath" \
  "$script" --root "$fixture" --out-dir "$out_dir" --json-out "$json_gopath" --no-sqlite >"$tmp_root/gopath.out"
grep -q '^code_audit_source=deadcode_tool$' "$tmp_root/gopath.out"
grep -q '^code_audit_deadcode_candidates=2$' "$tmp_root/gopath.out"

echo "orquesta_auditoria_codigo_ok=true"
