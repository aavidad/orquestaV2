#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
HARNESS="$ROOT/scripts/experimentos/context_rotation/harness.py"
FIXTURES="$ROOT/docs/experimentos/context_rotation/fixtures"
MANIFEST="$FIXTURES/manifest.synthetic.v1.json"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-context-rotation-test.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
export PYTHONDONTWRITEBYTECODE=1

python3 - "$ROOT/docs/experimentos/context_rotation/schemas" "$MANIFEST" <<'PY'
import json, jsonschema, pathlib, sys
schemas = pathlib.Path(sys.argv[1])
expected = {
    "manifest.v1.schema.json": "orquesta.context_rotation.manifest.v1",
    "dataset.v1.schema.json": "orquesta.context_rotation.dataset.v1",
    "receipt.v1.schema.json": "orquesta.context_rotation.receipt.v1",
    "blind_review.v1.schema.json": "orquesta.context_rotation.blind_review.v1",
}
for name, schema_id in expected.items():
    assert json.loads((schemas / name).read_text())["$id"] == schema_id
jsonschema.Draft202012Validator(json.loads((schemas / "manifest.v1.schema.json").read_text())).validate(json.load(open(sys.argv[2])))
PY

python3 "$HARNESS" validate --manifest "$MANIFEST" >/dev/null
python3 - "$MANIFEST" "$TMP/invalid-manifest.json" <<'PY'
import json, sys
manifest = json.load(open(sys.argv[1])); manifest["pairs"][0]["write_set"][0] = "../escape"
json.dump(manifest, open(sys.argv[2], "w"))
PY
if python3 "$HARNESS" validate --manifest "$TMP/invalid-manifest.json" >/dev/null 2>&1; then
  echo "unsafe write_set accepted" >&2; exit 1
fi
python3 "$HARNESS" plan --manifest "$MANIFEST" >"$TMP/plan.json"
python3 - "$TMP/plan.json" <<'PY'
import json, sys
plan = json.load(open(sys.argv[1]))
assert plan["dry_run"] is True
assert len(plan["worktrees"]) == 12
assert len({item["path"] for item in plan["worktrees"]}) == 12
assert len({item["commit"] for item in plan["worktrees"]}) == 1
assert [item["mode"] for item in plan["worktrees"][::2]].count("control") == 3
assert [item["mode"] for item in plan["worktrees"][::2]].count("treatment") == 3
PY
cmp "$TMP/plan.json" <(python3 "$HARNESS" plan --manifest "$MANIFEST")

python3 "$HARNESS" run-stage --manifest "$MANIFEST" --pair pair-api --mode treatment --stage api-implementation >"$TMP/stage.json"
python3 - "$TMP/stage.json" <<'PY'
import json, sys
stage = json.load(open(sys.argv[1]))
assert stage["fresh_worker"] is True
assert stage["compact_handoff_only"] is True
assert "prior_receipt" not in stage
assert stage["arm_budget_shared_across_stages"] is True
PY
python3 "$HARNESS" run-stage --manifest "$MANIFEST" --pair pair-api --mode control --stage continuous-session >"$TMP/control-stage.json"
python3 - "$TMP/control-stage.json" <<'PY'
import json, sys
stage = json.load(open(sys.argv[1]))
assert len(stage["context_refs"]) == 3
assert stage["fresh_worker"] is False
PY
test ! -e /tmp/orquesta-apg006-synthetic

python3 - "$ROOT" "$MANIFEST" "$FIXTURES/decision_cases.v1.json" "$TMP" <<'PY'
import hashlib, hmac, importlib.util, json, pathlib, sys
root, manifest_path, cases_path, out = map(pathlib.Path, sys.argv[1:])
spec = importlib.util.spec_from_file_location("rotation_domain", root / "scripts/experimentos/context_rotation/domain.py")
domain = importlib.util.module_from_spec(spec); spec.loader.exec_module(domain)
manifest = json.loads(manifest_path.read_text())
cases = json.loads(cases_path.read_text())["cases"]
key = (root / "docs/experimentos/context_rotation/fixtures/blinding_key.synthetic.txt").read_bytes().strip()
proof = lambda ref: {"ref": ref, "sha256": "f" * 64}
for case in cases:
    pairs = []
    for pair_index, pair in enumerate(manifest["pairs"][:case["complete_pairs"]]):
        arms = []
        for mode in ("control", "treatment"):
            ratio = 10000 if mode == "control" else case["treatment_token_ratio_bps"]
            cost_ratio = 10000 if mode == "control" else case["treatment_cost_ratio_bps"]
            tokens, cost = 100000 * ratio // 10000, 1000000 * cost_ratio // 10000
            shares = {"director": 10, "workers": 70, "handoffs": 5, "recoveries": 5, "reviewers": 10}
            usage = {role: {"input_uncached": tokens * share // 100, "output": 0, "cache_read": 0, "cache_write": 0, "cost_micros": cost * share // 100} for role, share in shares.items()}
            candidate = f"source-{pair_index}-{mode}"
            blind_candidate = "candidate-" + hmac.new(key, f"{manifest['experiment_ref']}\0{pair['pair_ref']}\0{mode}".encode(), hashlib.sha256).hexdigest()[:24]
            handoffs = [] if mode == "control" else [{"stage_ref": pair["treatment_stages"][1]["stage_ref"], "package": proof(f"handoff-{pair_index}"), "continued_from_package_and_refs": True, "requested_raw_transcript": False, "out_of_scope_reads": 0}]
            arms.append({"mode": mode, "candidate_ref": candidate, "blind_candidate_ref": blind_candidate, "status": "completed", "config_sha256": domain.config_hash(manifest, pair), "usage_by_role": usage, "metrics": {"duration_ms": 100000 if mode == "control" else 100000 * case["treatment_time_ratio_bps"] // 10000, "tool_calls": 20, "repeated_reads": 2, "reworks": 1, "failed_attempts": 0, "write_set_conflicts": 0, "human_interventions": 0, "changed_lines": 100, "changed_files": 2}, "false_green": False, "causal_loss": False, "unauthorized_external_effects": False, "external_effects": 0, "severe_defects": 0, "tests": [{"command": pair["required_tests"][0], "executed": True, "passed": case["quality_ok"], "evidence": proof(f"test-{pair_index}-{mode}")}], "handoffs": handoffs, "independent_review": {"completed": True, "accepted": case["quality_ok"], "candidate_ref": blind_candidate, "quality_score": 90, "unnecessary_changed_lines": 10 if mode == "control" else 5, "evidence": [proof(f"review-{pair_index}-{mode}")]}, "diff": proof(f"diff-{pair_index}-{mode}"), "artifacts": [proof(f"artifact-{pair_index}-{mode}")], "receipts": [proof(f"receipt-{pair_index}-{mode}")]})
        pairs.append({"pair_ref": pair["pair_ref"], "arms": arms})
    dataset = {"schema_version": domain.DATASET, "experiment_ref": manifest["experiment_ref"], "manifest_sha256": domain.digest(manifest), "pairs": pairs, "report_review": {"completed": True, "accepted": True, "reproduction_passed": True, "evidence": [proof("report-review")]}}
    (out / f"{case['name']}.json").write_text(json.dumps(dataset, indent=2) + "\n")
PY

python3 - "$ROOT/docs/experimentos/context_rotation/schemas/dataset.v1.schema.json" "$TMP/adopt.json" <<'PY'
import json, jsonschema, sys
jsonschema.Draft202012Validator(json.load(open(sys.argv[1]))).validate(json.load(open(sys.argv[2])))
PY

for verdict in adopt keep_control inconclusive; do
  python3 "$HARNESS" evaluate --manifest "$MANIFEST" --dataset "$TMP/$verdict.json" >"$TMP/$verdict.decision.json"
  python3 - "$TMP/$verdict.decision.json" "$verdict" <<'PY'
import json, sys
assert json.load(open(sys.argv[1]))["decision"] == sys.argv[2]
PY
done

python3 - "$TMP/adopt.json" "$TMP/no-code-reduction.json" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
for pair in data["pairs"]:
    treatment = next(arm for arm in pair["arms"] if arm["mode"] == "treatment")
    treatment["independent_review"]["unnecessary_changed_lines"] = 10
json.dump(data, open(sys.argv[2], "w"))
PY
python3 "$HARNESS" evaluate --manifest "$MANIFEST" --dataset "$TMP/no-code-reduction.json" >"$TMP/no-code-reduction.decision.json"
python3 - "$TMP/no-code-reduction.decision.json" <<'PY'
import json, sys
decision = json.load(open(sys.argv[1]))
assert decision["decision"] == "keep_control"
assert "unnecessary_code_not_reduced" in decision["reasons"]
PY

python3 - "$TMP/adopt.json" "$TMP/more-code.json" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
treatment = next(arm for arm in data["pairs"][0]["arms"] if arm["mode"] == "treatment")
treatment["independent_review"]["unnecessary_changed_lines"] = 11
json.dump(data, open(sys.argv[2], "w"))
PY
python3 "$HARNESS" evaluate --manifest "$MANIFEST" --dataset "$TMP/more-code.json" >"$TMP/more-code.decision.json"
python3 - "$TMP/more-code.decision.json" <<'PY'
import json, sys
decision = json.load(open(sys.argv[1]))
assert decision["decision"] == "keep_control"
assert any(reason.startswith("unnecessary_code_increased:") for reason in decision["reasons"])
PY

python3 - "$TMP/adopt.json" "$TMP/lower-quality.json" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
treatment = next(arm for arm in data["pairs"][0]["arms"] if arm["mode"] == "treatment")
treatment["independent_review"]["quality_score"] = 89
json.dump(data, open(sys.argv[2], "w"))
PY
python3 "$HARNESS" evaluate --manifest "$MANIFEST" --dataset "$TMP/lower-quality.json" >"$TMP/lower-quality.decision.json"
python3 - "$TMP/lower-quality.decision.json" <<'PY'
import json, sys
decision = json.load(open(sys.argv[1]))
assert decision["decision"] == "keep_control"
assert any(reason.startswith("quality_score_decreased:") for reason in decision["reasons"])
PY

python3 "$HARNESS" blind --manifest "$MANIFEST" --dataset "$TMP/adopt.json" --pair pair-api --mode treatment --key-file "$FIXTURES/blinding_key.synthetic.txt" >"$TMP/blind.json"
python3 - "$ROOT/docs/experimentos/context_rotation/schemas/blind_review.v1.schema.json" "$TMP/blind.json" <<'PY'
import json, jsonschema, sys
jsonschema.Draft202012Validator(json.load(open(sys.argv[1]))).validate(json.load(open(sys.argv[2])))
PY
python3 - "$TMP/blind.json" <<'PY'
import json, sys
package = json.load(open(sys.argv[1]))
assert "mode" not in package
assert "reasoning" not in package
assert package["candidate_ref"].startswith("candidate-")
assert package["review_contract"]["quality_scale"] == "0-100-higher-is-better"
PY

python3 - "$TMP/adopt.json" "$TMP/bad-blind-ref.json" <<'PY'
import json, sys
data = json.load(open(sys.argv[1])); data["pairs"][0]["arms"][1]["blind_candidate_ref"] = "candidate-" + "0" * 24
json.dump(data, open(sys.argv[2], "w"))
PY
if python3 "$HARNESS" blind --manifest "$MANIFEST" --dataset "$TMP/bad-blind-ref.json" --pair pair-api --mode treatment --key-file "$FIXTURES/blinding_key.synthetic.txt" >/dev/null 2>&1; then
  echo "blind mapping mismatch accepted" >&2; exit 1
fi

mkdir "$TMP/repo"
git -C "$TMP/repo" init -q
git -C "$TMP/repo" config user.email fixture@example.invalid
git -C "$TMP/repo" config user.name fixture
printf 'fixture\n' >"$TMP/repo/file.txt"
git -C "$TMP/repo" add file.txt
git -C "$TMP/repo" commit -qm fixture
commit="$(git -C "$TMP/repo" rev-parse HEAD)"
python3 - "$MANIFEST" "$TMP/real-manifest.json" "$TMP/repo" "$TMP/runtime" "$commit" <<'PY'
import json, sys
source, target, repo, runtime, commit = sys.argv[1:]
manifest = json.load(open(source)); manifest["repository_path"] = repo; manifest["temporary_root"] = runtime; manifest["state_root"] = runtime + "/state"; manifest["baseline_commit"] = commit
json.dump(manifest, open(target, "w"), indent=2)
PY

if python3 "$HARNESS" prepare --manifest "$TMP/real-manifest.json" --apply --confirm wrong >/dev/null 2>&1; then
  echo "prepare accepted wrong confirmation" >&2; exit 1
fi
test ! -e "$TMP/runtime"
python3 "$HARNESS" prepare --manifest "$TMP/real-manifest.json" --apply --confirm apg-006-synthetic >/dev/null
test "$(find "$TMP/runtime/state/receipts" -type f | wc -l)" -eq 12
python3 - "$ROOT/docs/experimentos/context_rotation/schemas/receipt.v1.schema.json" "$TMP/runtime/state/receipts" <<'PY'
import json, jsonschema, pathlib, sys
validator = jsonschema.Draft202012Validator(json.load(open(sys.argv[1])))
for receipt in pathlib.Path(sys.argv[2]).glob("*.json"):
    validator.validate(json.load(open(receipt)))
PY
python3 "$HARNESS" prepare --manifest "$TMP/real-manifest.json" --apply --confirm apg-006-synthetic >/dev/null
test "$(find "$TMP/runtime/state/receipts" -type f | wc -l)" -eq 12
if python3 "$HARNESS" run-stage --manifest "$TMP/real-manifest.json" --pair pair-api --mode control --stage continuous-session --apply --confirm apg-006-synthetic --provider-command /bin/false >"$TMP/provider.out" 2>"$TMP/provider.err"; then
  echo "provider ran without --allow-provider" >&2; exit 1
fi
grep -q -- '--allow-provider' "$TMP/provider.err"

echo "context rotation harness tests: ok"
