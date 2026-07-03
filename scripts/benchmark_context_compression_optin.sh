#!/usr/bin/env bash
set -euo pipefail

CONFIRM_VALUE="CTX_COMPRESSION_BENCHMARK_OPT_IN"

usage() {
  cat <<'EOF'
Usage:
  ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN \
    bash scripts/benchmark_context_compression_optin.sh \
      --input docs/non-critical.md \
      --output /tmp/context-compression-benchmark.json \
      [--compressed-output /tmp/context-compression-preview.md] \
      [--target-ratio 0.50] \
      [--compressor-command "local-command-reading-stdin"]

Opt-in benchmark only. It refuses protected inputs such as AGENTS.md,
contracts, hard-rule blocks and write-set material.
EOF
}

inputs=()
output="-"
compressed_output=""
target_ratio="${ORQUESTA_CONTEXT_COMPRESSION_TARGET_RATIO:-0.50}"
compressor_command="${ORQUESTA_CONTEXT_COMPRESSION_COMMAND:-}"
python_bin="${PYTHON_BIN:-python3}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --input" >&2
        exit 64
      fi
      inputs+=("$2")
      shift 2
      ;;
    --output)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --output" >&2
        exit 64
      fi
      output="$2"
      shift 2
      ;;
    --compressed-output)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --compressed-output" >&2
        exit 64
      fi
      compressed_output="$2"
      shift 2
      ;;
    --target-ratio)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --target-ratio" >&2
        exit 64
      fi
      target_ratio="$2"
      shift 2
      ;;
    --compressor-command)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --compressor-command" >&2
        exit 64
      fi
      compressor_command="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 64
      ;;
  esac
done

if [[ "${ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM:-}" != "$CONFIRM_VALUE" ]]; then
  echo "missing opt-in confirmation: set ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=$CONFIRM_VALUE" >&2
  exit 64
fi

if [[ ${#inputs[@]} -eq 0 ]]; then
  echo "at least one --input file is required" >&2
  usage >&2
  exit 64
fi

"$python_bin" - "$output" "$compressed_output" "$target_ratio" "$compressor_command" "${inputs[@]}" <<'PY'
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path, PurePosixPath


def fail(message, code=1):
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(code)


output_path = sys.argv[1]
compressed_output_path = sys.argv[2]
target_ratio_raw = sys.argv[3]
compressor_command = sys.argv[4].strip()
input_paths = sys.argv[5:]

try:
    target_ratio = float(target_ratio_raw)
except ValueError:
    fail(f"invalid --target-ratio {target_ratio_raw!r}", 64)

if target_ratio <= 0 or target_ratio >= 1:
    fail("--target-ratio must be greater than 0 and lower than 1", 64)

protected_content_patterns = [
    ("agent_instructions_block", re.compile(r"(?im)^\s*<INSTRUCTIONS>\s*$")),
    ("agents_header", re.compile(r"(?im)^\s*#\s*Contexto\s+para\s+agentes\b")),
    ("write_set", re.compile(r"(?i)\bwrite[-_ ]set\b")),
    ("hard_rules", re.compile(r"(?i)\b(reglas\s+hard|hard\s+rules)\b")),
    ("hard_contracts", re.compile(r"(?i)\b(contratos?\s+duros|hard\s+contracts)\b")),
]

ref_patterns = [
    re.compile(r"\b[a-z][a-z0-9_-]*_ref:[A-Za-z0-9._:/-]+"),
    re.compile(r"\b[a-z][a-z0-9_-]+-ref-[A-Za-z0-9._:/-]+"),
    re.compile(r"\b(?:CTX-TASK-[0-9A-Z]+|T[0-9]{2,4}|CODEX-[A-Z0-9-]+|EXT-[A-Z0-9-]+|OPES-[A-Z0-9-]+)\b"),
    re.compile(r"\b(?:docs|modulos|scripts|cmd)/[A-Za-z0-9._/-]+(?::(?:line:)?[0-9]+)?(?::sha256:[a-f0-9]{64})?\b"),
    re.compile(r"\bsha256:[a-f0-9]{64}\b"),
    re.compile(r"\b[a-f0-9]{64}\b"),
]


def normalized_path(path):
    return str(PurePosixPath(Path(path).as_posix()))


def protected_path_reasons(path):
    norm = normalized_path(path)
    lower = norm.lower()
    name = PurePosixPath(norm).name.lower()
    reasons = []
    if name == "agents.md":
        reasons.append("agents_file")
    if "/agents.md" in lower:
        reasons.append("agents_path")
    if name in {"contratos.md", "contracts.md"}:
        reasons.append("contract_file")
    if "/contratos/" in lower or "/contracts/" in lower:
        reasons.append("contract_path")
    return reasons


def content_protection_reasons(text):
    reasons = []
    for reason, pattern in protected_content_patterns:
        if pattern.search(text):
            reasons.append(reason)
    return reasons


def extract_refs(text):
    refs = set()
    for pattern in ref_patterns:
        refs.update(match.group(0) for match in pattern.finditer(text))
    return refs


def estimate_tokens(text):
    return len(re.findall(r"\w+|[^\w\s]", text, flags=re.UNICODE))


def line_has_ref(line, refs):
    return any(ref in line for ref in refs)


def local_extractive_compress(text, refs, ratio):
    source_bytes = len(text.encode("utf-8"))
    target_bytes = max(1, int(source_bytes * ratio))
    current_bytes = 0
    kept_lines = []
    blank_kept = False
    for line in text.splitlines():
        stripped = line.strip()
        mandatory = stripped.startswith("#") or line_has_ref(line, refs)
        optional = bool(stripped) and current_bytes < target_bytes
        if mandatory or optional:
            kept_lines.append(line)
            current_bytes += len((line + "\n").encode("utf-8"))
            blank_kept = False
        elif not stripped and not blank_kept and kept_lines:
            kept_lines.append("")
            current_bytes += 1
            blank_kept = True
    if not kept_lines and text:
        kept_lines.append(text[:target_bytes])
    return "\n".join(kept_lines).rstrip() + "\n"


def external_compress(text, command):
    timeout = int(os.environ.get("ORQUESTA_CONTEXT_COMPRESSION_COMMAND_TIMEOUT_SECONDS", "120"))
    proc = subprocess.run(
        command,
        input=text,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        shell=True,
        timeout=timeout,
        check=False,
    )
    if proc.returncode != 0:
        stderr = proc.stderr[-2000:]
        fail(f"compressor command failed with exit {proc.returncode}: {stderr}", 70)
    return proc.stdout


input_records = []
source_parts = []
for raw_path in input_paths:
    path = Path(raw_path)
    if not path.is_file():
        fail(f"input is not a file: {raw_path}", 66)
    text = path.read_text(encoding="utf-8")
    reasons = protected_path_reasons(raw_path) + content_protection_reasons(text)
    if reasons:
        fail(f"protected input refused: {raw_path} reasons={','.join(sorted(set(reasons)))}", 65)
    refs = extract_refs(text)
    encoded = text.encode("utf-8")
    input_records.append(
        {
            "path": raw_path,
            "bytes": len(encoded),
            "tokens_estimated": estimate_tokens(text),
            "refs_count": len(refs),
        }
    )
    source_parts.append(f"## Source: {raw_path}\n\n{text.rstrip()}\n")

source_text = "\n".join(source_parts)
source_refs = extract_refs(source_text)
started = time.perf_counter()
if compressor_command:
    compressed_text = external_compress(source_text, compressor_command)
    compressor_name = "external_local_command"
else:
    compressed_text = local_extractive_compress(source_text, source_refs, target_ratio)
    compressor_name = "stdlib_extractive_baseline"
elapsed_ms = round((time.perf_counter() - started) * 1000, 3)

compressed_refs = extract_refs(compressed_text)
lost_refs = sorted(source_refs - compressed_refs)
source_bytes = len(source_text.encode("utf-8"))
compressed_bytes = len(compressed_text.encode("utf-8"))
source_tokens = estimate_tokens(source_text)
compressed_tokens = estimate_tokens(compressed_text)

payload = {
    "schema_version": "orquesta_context_compression_benchmark.v0",
    "status": "valid_benchmark" if not lost_refs else "refs_lost",
    "mode": "opt_in_local_only",
    "confirmation_env": "ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM",
    "compressor": {
        "name": compressor_name,
        "external_command_used": bool(compressor_command),
        "target_ratio": target_ratio,
    },
    "policy": {
        "protected_material_never_compressed": [
            "AGENTS.md",
            "hard_rules",
            "write_set",
            "contracts",
        ],
        "productive_adapter_allowed_by_this_script": False,
        "productive_adapter_recommendation": "do_not_enable_product_adapter_from_benchmark_alone",
    },
    "inputs": input_records,
    "metrics": {
        "source_bytes": source_bytes,
        "compressed_bytes": compressed_bytes,
        "source_tokens_estimated": source_tokens,
        "compressed_tokens_estimated": compressed_tokens,
        "byte_ratio": round(compressed_bytes / source_bytes, 4) if source_bytes else 0,
        "token_ratio": round(compressed_tokens / source_tokens, 4) if source_tokens else 0,
        "lost_refs_count": len(lost_refs),
        "lost_refs": lost_refs,
        "elapsed_ms": elapsed_ms,
    },
    "decision_gate": {
        "decision_required_before_product_adapter": True,
        "minimum_manual_evidence": [
            "zero lost refs on selected non-critical corpus",
            "acceptable local runtime cost",
            "separate product adapter design reviewed in modulos/orquesta-context/docs/decisiones.md",
        ],
    },
}

serialized = json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
if output_path == "-":
    sys.stdout.write(serialized)
else:
    output = Path(output_path)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(serialized, encoding="utf-8")

if compressed_output_path:
    compressed_output = Path(compressed_output_path)
    compressed_output.parent.mkdir(parents=True, exist_ok=True)
    compressed_output.write_text(compressed_text, encoding="utf-8")
PY
