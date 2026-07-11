#!/usr/bin/env python3
"""Concrete git/process adapter for the pure APG-006 evaluation domain."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

import domain

RECEIPT_SCHEMA = "orquesta.context_rotation.receipt.v1"
PROVIDER_SCHEMA = "orquesta.context_rotation.provider_result.v1"


def load_json(path: str | Path) -> dict:
    def unique(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                raise ValueError(f"duplicate JSON key: {key}")
            value[key] = item
        return value

    with open(path, encoding="utf-8") as handle:
        return json.load(handle, object_pairs_hook=unique)


def dump(value) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True) + "\n").encode()


def print_json(value):
    sys.stdout.buffer.write(dump(value))


def atomic_write(path: Path, value: bytes, mode=0o600):
    path.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix=".tmp-", dir=path.parent)
    try:
        os.fchmod(fd, mode)
        with os.fdopen(fd, "wb") as handle:
            handle.write(value)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def file_hash(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def git(directory: Path, *args: str) -> str:
    result = subprocess.run(["git", "-C", str(directory), *args], text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False)
    if result.returncode:
        raise RuntimeError(f"git {' '.join(args)} failed: {result.stdout[-2000:]}")
    return result.stdout.strip()


def operation(prefix: str, *parts: str) -> str:
    return domain.stable_ref(prefix, *parts)


def receipt_path(manifest: dict, operation_ref: str) -> Path:
    return Path(manifest["state_root"], "receipts", f"{operation_ref}.json")


def read_receipt(manifest: dict, operation_ref: str, input_hash: str | None = None) -> dict | None:
    path = receipt_path(manifest, operation_ref)
    if not path.exists():
        return None
    receipt = load_json(path)
    if receipt.get("schema_version") != RECEIPT_SCHEMA or receipt.get("operation_ref") != operation_ref or receipt.get("status") != "completed":
        raise ValueError(f"invalid receipt: {operation_ref}")
    if input_hash is not None and receipt.get("input_sha256") != input_hash:
        raise ValueError(f"receipt input hash mismatch: {operation_ref}")
    return receipt


def write_receipt(manifest: dict, operation_ref: str, kind: str, input_hash: str, result_hash: str, evidence: list[dict]) -> dict:
    receipt = {
        "schema_version": RECEIPT_SCHEMA,
        "receipt_ref": operation("receipt", operation_ref, input_hash),
        "operation_ref": operation_ref,
        "operation_kind": kind,
        "experiment_ref": manifest["experiment_ref"],
        "input_sha256": input_hash,
        "result_sha256": result_hash,
        "status": "completed",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "evidence": evidence,
    }
    atomic_write(receipt_path(manifest, operation_ref), dump(receipt))
    return receipt


def manifest_from(args) -> dict:
    manifest = load_json(args.manifest)
    issues = domain.validate_manifest(manifest)
    if issues:
        raise ValueError("invalid manifest: " + ",".join(issues))
    return manifest


def require_apply(args, manifest: dict, *, provider=False):
    if args.confirm != manifest["experiment_ref"]:
        raise ValueError("--confirm must equal experiment_ref")
    if provider and not args.allow_provider:
        raise ValueError("provider execution also requires --allow-provider")


def worktree_for(plan: dict, pair_ref: str, mode: str) -> dict:
    return next(item for item in plan["worktrees"] if item["pair_ref"] == pair_ref and item["mode"] == mode)


def validate_paths(manifest: dict, repo: Path):
    temp = Path(manifest["temporary_root"]).resolve()
    if temp == repo or repo in temp.parents or temp in repo.parents:
        raise ValueError("temporary_root and repository must be disjoint")
    parent = temp
    while not parent.exists():
        parent = parent.parent
    required = manifest["execution"]["budget"]["max_workspace_bytes"] * len(manifest["pairs"]) * 2
    if shutil.disk_usage(parent).free < required:
        raise ValueError("workspace disk budget unavailable")


def prepare(args, manifest: dict):
    planned = domain.plan(manifest)
    if not args.apply:
        print_json(planned)
        return
    require_apply(args, manifest)
    repo = Path(manifest["repository_path"]).resolve()
    validate_paths(manifest, repo)
    git(repo, "cat-file", "-e", f"{manifest['baseline_commit']}^{{commit}}")
    receipts = []
    for item in planned["worktrees"]:
        input_hash = domain.digest(item)
        prior = read_receipt(manifest, item["operation_ref"], input_hash)
        target = Path(item["path"])
        if prior:
            if git(target, "rev-parse", "HEAD") != manifest["baseline_commit"]:
                raise ValueError(f"receipt/worktree mismatch: {target}")
            receipts.append(prior)
            continue
        if target.exists():
            raise ValueError(f"unreceipted path exists: {target}")
        target.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
        git(repo, "worktree", "add", "--detach", str(target), manifest["baseline_commit"])
        receipts.append(write_receipt(manifest, item["operation_ref"], "prepare_worktree", input_hash, domain.digest(item), [manifest["snapshot"]]))
    print_json(receipts)


def provider_result_valid(result: dict) -> bool:
    evidence = result.get("artifacts", []) + result.get("receipts", [])
    return result.get("schema_version") == PROVIDER_SCHEMA and result.get("status") == "completed" and domain.REF.fullmatch(result.get("result_ref", "")) is not None and bool(result.get("artifacts")) and bool(result.get("receipts")) and all(domain._evidence(item) for item in evidence)


def run_stage(args, manifest: dict):
    request, previous_operation = domain.stage_request(manifest, args.pair, args.mode, args.stage)
    if not args.apply:
        print_json(request)
        return
    require_apply(args, manifest, provider=True)
    command = Path(args.provider_command)
    if not command.is_absolute() or not command.is_file() or not os.access(command, os.X_OK):
        raise ValueError("--provider-command must be an absolute executable file")
    planned = domain.plan(manifest)
    worktree = worktree_for(planned, args.pair, args.mode)
    if read_receipt(manifest, worktree["operation_ref"], domain.digest(worktree)) is None:
        raise ValueError("prepared worktree receipt required")
    if previous_operation:
        prior = read_receipt(manifest, previous_operation)
        if prior is None:
            raise ValueError("prior treatment stage receipt required")
        prior_path = receipt_path(manifest, previous_operation)
        request["prior_receipt"] = {"ref": prior["receipt_ref"], "sha256": file_hash(prior_path)}
    input_hash = domain.digest(request)
    operation_ref = operation("run", manifest["experiment_ref"], args.pair, args.mode, args.stage)
    prior = read_receipt(manifest, operation_ref, input_hash)
    result_path = Path(manifest["state_root"], "provider_results", f"{operation_ref}.json")
    if prior:
        if not result_path.is_file() or file_hash(result_path) != prior["result_sha256"]:
            raise ValueError("provider receipt/result mismatch")
        print_json(prior)
        return
    result = subprocess.run([str(command), *args.provider_arg], cwd=worktree["path"], input=json.dumps(request).encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=manifest["execution"]["budget"]["max_duration_ms"] / 1000, check=False)
    if result.returncode:
        raise RuntimeError(f"provider failed ({result.returncode}): {result.stderr[-2000:].decode(errors='replace')}")
    if len(result.stdout) > 8 << 20 or len(result.stderr) > 1 << 20:
        raise ValueError("provider output limit exceeded")
    provider_result = json.loads(result.stdout)
    if not provider_result_valid(provider_result):
        raise ValueError("provider result contract rejected")
    atomic_write(result_path, result.stdout)
    evidence = provider_result["artifacts"] + provider_result["receipts"]
    print_json(write_receipt(manifest, operation_ref, "run_stage", input_hash, file_hash(result_path), evidence))


def blind(args, manifest: dict):
    dataset = load_json(args.dataset)
    key = Path(args.key_file).read_bytes().strip()
    package, mapping = domain.blind(manifest, dataset, args.pair, args.mode, key)
    if not args.apply:
        print_json(package)
        return
    require_apply(args, manifest)
    base = Path(manifest["state_root"], "blind", args.pair)
    atomic_write(base / f"{package['candidate_ref']}.package.json", dump(package), 0o644)
    atomic_write(base / f"{package['candidate_ref']}.mapping.json", dump(mapping), 0o600)
    print_json({"package_ref": package["candidate_ref"], "package_sha256": domain.digest(package), "mapping": "persisted_separately_mode_0600"})


def evaluate(args, manifest: dict):
    decision = domain.evaluate(manifest, load_json(args.dataset))
    if args.apply:
        require_apply(args, manifest)
        atomic_write(Path(manifest["state_root"], "decision.json"), dump(decision))
    print_json(decision)


def cleanup(args, manifest: dict):
    retention = manifest["retention"]
    if not retention.get("allow_worktree_cleanup") or not retention["preserve_receipts"]:
        raise ValueError("retention policy forbids cleanup")
    state, dataset_path = Path(manifest["state_root"]).resolve(), Path(args.dataset).resolve()
    if state not in dataset_path.parents:
        raise ValueError("dataset must be retained under state_root")
    decision = domain.evaluate(manifest, load_json(dataset_path))
    if load_json(state / "decision.json") != decision:
        raise ValueError("durable decision integrity failed")
    planned = domain.plan(manifest)
    cutoff = datetime.now(timezone.utc) - timedelta(hours=retention["minimum_retention_hours"])
    for item in planned["worktrees"]:
        receipt = read_receipt(manifest, item["operation_ref"], domain.digest(item))
        if not receipt or datetime.fromisoformat(receipt["created_at"]) > cutoff:
            raise ValueError(f"retention blocks cleanup: {item['operation_ref']}")
        if git(Path(item["path"]), "status", "--porcelain=v1"):
            raise ValueError(f"dirty worktree retained: {item['path']}")
    for pair in manifest["pairs"]:
        stages = [("control", "continuous-session")] + [("treatment", item["stage_ref"]) for item in pair["treatment_stages"]]
        for mode, stage in stages:
            op = operation("run", manifest["experiment_ref"], pair["pair_ref"], mode, stage)
            receipt = read_receipt(manifest, op)
            result_path = state / "provider_results" / f"{op}.json"
            if not receipt or datetime.fromisoformat(receipt["created_at"]) > cutoff or not result_path.is_file() or file_hash(result_path) != receipt["result_sha256"]:
                raise ValueError(f"terminal integrity/retention blocks cleanup: {op}")
    result = {"dry_run": not args.apply, "eligible_worktrees": len(planned["worktrees"]), "receipts_preserved": True, "production_promoted": False}
    if not args.apply:
        print_json(result)
        return
    require_apply(args, manifest)
    repo = Path(manifest["repository_path"]).resolve()
    for item in planned["worktrees"]:
        git(repo, "worktree", "remove", item["path"])
    print_json(result)


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser()
    commands = root.add_subparsers(dest="command", required=True)
    for name in ("validate", "plan", "prepare", "run-stage", "blind", "evaluate", "cleanup"):
        item = commands.add_parser(name)
        item.add_argument("--manifest", required=True)
        if name in ("prepare", "run-stage", "blind", "evaluate", "cleanup"):
            item.add_argument("--apply", action="store_true")
            item.add_argument("--confirm", default="")
        if name == "run-stage":
            item.add_argument("--pair", required=True); item.add_argument("--mode", choices=("control", "treatment"), required=True); item.add_argument("--stage", required=True)
            item.add_argument("--allow-provider", action="store_true"); item.add_argument("--provider-command", default=""); item.add_argument("--provider-arg", action="append", default=[])
        if name == "blind":
            item.add_argument("--dataset", required=True); item.add_argument("--pair", required=True); item.add_argument("--mode", choices=("control", "treatment"), required=True); item.add_argument("--key-file", required=True)
        if name in ("evaluate", "cleanup"):
            item.add_argument("--dataset", required=True)
    return root


def main():
    args = parser().parse_args()
    manifest = manifest_from(args)
    if args.command == "validate": print_json({"valid": True})
    elif args.command == "plan": print_json(domain.plan(manifest))
    elif args.command == "prepare": prepare(args, manifest)
    elif args.command == "run-stage": run_stage(args, manifest)
    elif args.command == "blind": blind(args, manifest)
    elif args.command == "evaluate": evaluate(args, manifest)
    elif args.command == "cleanup": cleanup(args, manifest)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired, StopIteration) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
