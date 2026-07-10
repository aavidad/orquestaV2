#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: scripts/orquesta_auditoria_codigo.sh [--json] [--root DIR] [--out-dir DIR] [--json-out FILE] [--sqlite-out FILE] [--deadcode-file FILE] [--no-sqlite]
EOF
}

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
root="$repo_root"
out_dir="${ORQUESTA_AUDIT_RESULTS_DIR:-${TMPDIR:-/tmp}/orquesta-code-audit}"
json_out=""
sqlite_out=""
deadcode_file=""
print_json=false
write_sqlite=true

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --json)
      print_json=true
      shift
      ;;
    --root)
      [[ "$#" -ge 2 ]] || { usage; exit 2; }
      root="$2"
      shift 2
      ;;
    --out-dir)
      [[ "$#" -ge 2 ]] || { usage; exit 2; }
      out_dir="$2"
      shift 2
      ;;
    --json-out)
      [[ "$#" -ge 2 ]] || { usage; exit 2; }
      json_out="$2"
      shift 2
      ;;
    --sqlite-out)
      [[ "$#" -ge 2 ]] || { usage; exit 2; }
      sqlite_out="$2"
      shift 2
      ;;
    --deadcode-file)
      [[ "$#" -ge 2 ]] || { usage; exit 2; }
      deadcode_file="$2"
      shift 2
      ;;
    --no-sqlite)
      write_sqlite=false
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ ! -d "$root/modulos" || ! -d "$root/cmd" || ! -f "$root/go.mod" ]]; then
  echo "orquesta_auditoria_codigo: root must contain go.mod, modulos/ and cmd/" >&2
  exit 2
fi

mkdir -p "$out_dir"
run_id="${ORQUESTA_AUDIT_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
json_out="${json_out:-$out_dir/auditoria_codigo_${run_id}.json}"
if [[ "$write_sqlite" == true ]]; then
  sqlite_out="${sqlite_out:-$out_dir/auditoria_codigo_${run_id}.sqlite}"
else
  sqlite_out=""
fi

python3 - "$root" "$json_out" "$sqlite_out" "$deadcode_file" <<'PY'
import json
import os
import re
import shlex
import shutil
import sqlite3
import subprocess
import sys
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path

root = Path(sys.argv[1]).resolve()
json_out = Path(sys.argv[2]).resolve()
sqlite_arg = sys.argv[3].strip()
sqlite_out = Path(sqlite_arg).resolve() if sqlite_arg else None
deadcode_arg = sys.argv[4].strip()
deadcode_file = Path(deadcode_arg).resolve() if deadcode_arg else None

def rel(path):
    return path.resolve().relative_to(root).as_posix()

def module_for_rel(path):
    parts = path.split("/")
    if len(parts) >= 2 and parts[0] in {"modulos", "cmd"}:
        return "/".join(parts[:2])
    return parts[0] if parts else ""

def iter_project_go_files():
    for base in (root / "cmd", root / "modulos"):
        if not base.exists():
            continue
        for path in sorted(base.rglob("*.go")):
            yield path

def read_go_module():
    for raw in (root / "go.mod").read_text(encoding="utf-8", errors="replace").splitlines():
        raw = raw.strip()
        if raw.startswith("module "):
            return raw.split(None, 1)[1].strip()
    return ""

def parse_deadcode_line(line):
    match = re.match(r"^(.*?):(\d+):(\d+):\s+unreachable\s+func:\s+(.+)$", line)
    if not match:
        return None
    path, line_no, col_no, symbol = match.groups()
    path = path.strip()
    try:
        candidate = Path(path)
        if candidate.is_absolute():
            path = candidate.resolve().relative_to(root).as_posix()
    except ValueError:
        pass
    return {
        "path": path,
        "line": int(line_no),
        "column": int(col_no),
        "symbol": symbol.strip(),
        "module": module_for_rel(path),
        "raw": line,
    }

def go_env_value(name):
    try:
        proc = subprocess.run(
            ["go", "env", name],
            cwd=root,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            env={**os.environ, "GOFLAGS": os.environ.get("GOFLAGS", "-buildvcs=false")},
        )
    except OSError:
        return ""
    if proc.returncode != 0:
        return ""
    return proc.stdout.strip()

def find_deadcode_binary():
    binary = shutil.which("deadcode")
    if binary:
        return binary

    candidates = []
    gobin = os.environ.get("GOBIN", "").strip() or go_env_value("GOBIN")
    if gobin:
        candidates.append(Path(gobin) / "deadcode")

    gopath = os.environ.get("GOPATH", "").strip() or go_env_value("GOPATH")
    for raw in gopath.split(os.pathsep):
        if raw.strip():
            candidates.append(Path(raw.strip()) / "bin" / "deadcode")

    for candidate in candidates:
        if candidate.is_file() and os.access(candidate, os.X_OK):
            return str(candidate)
    return ""

def collect_deadcode():
    if deadcode_file:
        if not deadcode_file.exists():
            raise SystemExit(f"deadcode file not found: {deadcode_file}")
        text = deadcode_file.read_text(encoding="utf-8", errors="replace")
        source = "provided_file"
    else:
        env_cmd = os.environ.get("ORQUESTA_AUDIT_DEADCODE_CMD", "").strip()
        command = shlex.split(env_cmd) if env_cmd else []
        if not command:
            binary = find_deadcode_binary()
            if binary:
                command = [binary, "./cmd/..."]
        if command:
            proc = subprocess.run(
                command,
                cwd=root,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                env={**os.environ, "GOFLAGS": os.environ.get("GOFLAGS", "-buildvcs=false")},
            )
            text = proc.stdout
            source = "deadcode_tool" if proc.returncode == 0 or text.strip() else "deadcode_tool_failed"
        else:
            snapshot = root / "docs" / "auditoria_codigo_deadcode_2026-07-04.txt"
            if snapshot.exists():
                text = snapshot.read_text(encoding="utf-8", errors="replace")
                source = "snapshot_file"
            else:
                text = ""
                source = "unavailable"
    entries = []
    unparsable = 0
    for raw in text.splitlines():
        line = raw.strip()
        if not line:
            continue
        parsed = parse_deadcode_line(line)
        if parsed:
            entries.append(parsed)
        else:
            unparsable += 1
    return source, entries, unparsable

def collect_large_files():
    records = []
    for path in iter_project_go_files():
        relative = rel(path)
        if relative.endswith("_test.go"):
            continue
        try:
            line_count = len(path.read_text(encoding="utf-8", errors="replace").splitlines())
        except OSError:
            continue
        if line_count > 800:
            records.append({
                "path": relative,
                "lines": line_count,
                "module": module_for_rel(relative),
            })
    return records

FUNC_RE = re.compile(r"^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(")

def helper_family(name):
    lowered = name.lower()
    if lowered.startswith("firstnonempty"):
        return "firstNonEmpty"
    if lowered.startswith("compact"):
        return "compact"
    if lowered.startswith("contains"):
        return "contains"
    return ""

def collect_helper_copies():
    records = []
    for path in iter_project_go_files():
        relative = rel(path)
        if relative.endswith("_test.go"):
            continue
        try:
            lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        except OSError:
            continue
        for index, raw in enumerate(lines, start=1):
            match = FUNC_RE.match(raw)
            if not match:
                continue
            name = match.group(1)
            family = helper_family(name)
            if not family:
                continue
            records.append({
                "family": family,
                "name": name,
                "path": relative,
                "line": index,
                "module": module_for_rel(relative),
            })
    return records

def collect_function_index(deadcode_entries):
    deadcode_keys = {(entry["path"], entry["line"], entry["symbol"].split(".")[-1]) for entry in deadcode_entries}
    all_source = []
    records = []
    for path in iter_project_go_files():
        relative = rel(path)
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        all_source.append(text)
        for line_no, raw in enumerate(text.splitlines(), start=1):
            match = FUNC_RE.match(raw)
            if not match:
                continue
            name = match.group(1)
            exported = bool(name) and name[0].isupper()
            if relative.endswith("_test.go"):
                classification = "test_only"
            elif (relative, line_no, name) in deadcode_keys:
                classification = "static_candidate_requires_review"
            elif exported:
                classification = "exported_or_contract_requires_review"
            else:
                classification = "unclassified_private"
            records.append({
                "path": relative,
                "line": line_no,
                "module": module_for_rel(relative),
                "name": name,
                "exported": exported,
                "test_file": relative.endswith("_test.go"),
                "classification": classification,
            })
    identifier_counts = Counter(
        identifier
        for text in all_source
        for identifier in re.findall(r"\b[A-Za-z_][A-Za-z0-9_]*\b", text)
    )
    for record in records:
        record["text_reference_count"] = identifier_counts[record["name"]]
    records.sort(key=lambda item: (item["path"], item["line"], item["name"]))
    return records

def collect_orphan_modules(module_path):
    module_dirs = []
    for path in sorted((root / "modulos").iterdir()):
        if not path.is_dir():
            continue
        if any(child.suffix == ".go" for child in path.rglob("*.go")):
            module_dirs.append(path.name)

    packages = []
    go_list_error = ""
    try:
        proc = subprocess.run(
            ["go", "list", "-f", "{{.ImportPath}}\t{{join .Imports \" \"}}\t{{join .TestImports \" \"}}\t{{join .XTestImports \" \"}}", "./cmd/...", "./modulos/..."],
            cwd=root,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env={**os.environ, "GOFLAGS": os.environ.get("GOFLAGS", "-buildvcs=false")},
        )
        if proc.returncode == 0:
            for raw in proc.stdout.splitlines():
                parts = raw.split("\t")
                while len(parts) < 4:
                    parts.append("")
                packages.append({
                    "path": parts[0],
                    "imports": set(" ".join(parts[1:]).split()),
                })
        else:
            go_list_error = proc.stderr.strip()
    except OSError as exc:
        go_list_error = str(exc)

    records = []
    for name in module_dirs:
        prefix = f"{module_path}/modulos/{name}" if module_path else f"modulos/{name}"
        importers = sorted(
            package["path"]
            for package in packages
            if not (package["path"] == prefix or package["path"].startswith(prefix + "/"))
            and any(imported == prefix or imported.startswith(prefix + "/") for imported in package["imports"])
        )
        if not importers:
            records.append({
                "module": f"modulos/{name}",
                "path": f"modulos/{name}",
                "importer_count": 0,
            })
    return records, go_list_error

module_path = read_go_module()
deadcode_source, deadcode_entries, deadcode_unparsable = collect_deadcode()
large_files = collect_large_files()
helper_copies = collect_helper_copies()
function_index = collect_function_index(deadcode_entries)
orphan_modules, go_list_error = collect_orphan_modules(module_path)

deadcode_by_module = Counter(entry["module"] for entry in deadcode_entries)
helper_family_counts = Counter(entry["family"] for entry in helper_copies)
helper_duplicate_definitions = sum(max(0, count - 1) for count in helper_family_counts.values())
function_classification_counts = Counter(entry["classification"] for entry in function_index)

payload = {
    "schema_version": "orquesta_code_audit.v0",
    "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "repo_root": str(root),
    "module_path": module_path,
    "deadcode": {
        "source": deadcode_source,
        "candidate_count": len(deadcode_entries),
        "unparsable_lines": deadcode_unparsable,
        "by_module": dict(sorted(deadcode_by_module.items())),
        "entries": deadcode_entries,
    },
    "orphan_modules": orphan_modules,
    "helper_copies": {
        "families": dict(sorted(helper_family_counts.items())),
        "duplicate_definitions": helper_duplicate_definitions,
        "entries": helper_copies,
    },
    "function_index": {
        "parser": "go_function_declaration_lexical_v0",
        "classification_counts": dict(sorted(function_classification_counts.items())),
        "entries": function_index,
    },
    "large_files": large_files,
    "metrics": {
        "deadcode_candidates": len(deadcode_entries),
        "deadcode_unparsable_lines": deadcode_unparsable,
        "orphan_modules": len(orphan_modules),
        "helper_duplicate_definitions": helper_duplicate_definitions,
        "helper_family_definitions": len(helper_copies),
        "large_files_over_800": len(large_files),
        "functions_indexed": len(function_index),
    },
    "notes": [
        "SQLite output is a derived cache/report, not operational truth.",
        "deadcode candidates require per-module verification before deletion.",
        "function index classifications and text_reference_count are triage signals, not proof of removability.",
    ],
}
if go_list_error:
    payload["go_list_error"] = go_list_error

json_out.parent.mkdir(parents=True, exist_ok=True)
tmp_json = json_out.with_suffix(json_out.suffix + ".tmp")
tmp_json.write_text(json.dumps(payload, ensure_ascii=True, indent=2, sort_keys=True) + "\n", encoding="utf-8")
tmp_json.replace(json_out)

if sqlite_out:
    sqlite_out.parent.mkdir(parents=True, exist_ok=True)
    tmp_sqlite = sqlite_out.with_suffix(sqlite_out.suffix + ".tmp")
    if tmp_sqlite.exists():
        tmp_sqlite.unlink()
    con = sqlite3.connect(tmp_sqlite)
    try:
        con.executescript(
            """
            create table metadata(key text primary key, value text not null);
            create table metrics(metric text primary key, value integer not null);
            create table deadcode(path text not null, line integer not null, column integer not null, symbol text not null, module text not null, raw text not null);
            create table orphan_modules(module text primary key, path text not null, importer_count integer not null);
            create table helper_copies(family text not null, name text not null, path text not null, line integer not null, module text not null);
            create table large_files(path text primary key, lines integer not null, module text not null);
            create table function_index(path text not null, line integer not null, module text not null, name text not null, exported integer not null, test_file integer not null, classification text not null, text_reference_count integer not null);
            """
        )
        con.executemany(
            "insert into metadata(key, value) values(?, ?)",
            [
                ("schema_version", payload["schema_version"]),
                ("generated_at_utc", payload["generated_at_utc"]),
                ("repo_root", payload["repo_root"]),
                ("module_path", payload["module_path"]),
                ("deadcode_source", deadcode_source),
            ],
        )
        con.executemany(
            "insert into metrics(metric, value) values(?, ?)",
            sorted((key, int(value)) for key, value in payload["metrics"].items()),
        )
        con.executemany(
            "insert into deadcode(path, line, column, symbol, module, raw) values(?, ?, ?, ?, ?, ?)",
            [(item["path"], item["line"], item["column"], item["symbol"], item["module"], item["raw"]) for item in deadcode_entries],
        )
        con.executemany(
            "insert into orphan_modules(module, path, importer_count) values(?, ?, ?)",
            [(item["module"], item["path"], item["importer_count"]) for item in orphan_modules],
        )
        con.executemany(
            "insert into helper_copies(family, name, path, line, module) values(?, ?, ?, ?, ?)",
            [(item["family"], item["name"], item["path"], item["line"], item["module"]) for item in helper_copies],
        )
        con.executemany(
            "insert into large_files(path, lines, module) values(?, ?, ?)",
            [(item["path"], item["lines"], item["module"]) for item in large_files],
        )
        con.executemany(
            "insert into function_index(path, line, module, name, exported, test_file, classification, text_reference_count) values(?, ?, ?, ?, ?, ?, ?, ?)",
            [(item["path"], item["line"], item["module"], item["name"], int(item["exported"]), int(item["test_file"]), item["classification"], item["text_reference_count"]) for item in function_index],
        )
        con.commit()
    finally:
        con.close()
    tmp_sqlite.replace(sqlite_out)

print(f"code_audit_json={json_out}")
print(f"code_audit_sqlite={sqlite_out or ''}")
print(f"code_audit_source={deadcode_source}")
for key in [
    "deadcode_candidates",
    "helper_duplicate_definitions",
    "orphan_modules",
    "large_files_over_800",
    "functions_indexed",
]:
    print(f"code_audit_{key}={payload['metrics'][key]}")
PY

if [[ "$print_json" == true ]]; then
  cat "$json_out"
fi
