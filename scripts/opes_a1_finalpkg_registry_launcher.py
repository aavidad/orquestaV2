#!/usr/bin/env python3
"""Launch local OPES A1 final-package work from the shared topic registry.

This is an opt-in composition helper for the current local A1 filesystem flow.
It does not replace the canonical OPES API temario cycle; it bridges the local
registry-backed course into Orquesta external-work runs while OPES /api/jobs is
not the producer of this course.
"""

from __future__ import annotations

import argparse
import copy
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


REQUIRED_PACKAGE_FILES = (
    "manifest_cierre.json",
    "index.html",
    "tests.json",
    "visuales_plan.md",
    "rag/manifest.json",
    "rag/corpus/chunks.jsonl",
    "rag/corpus/summary.json",
    "tutor/tutor_prompt.md",
    "qa_final.md",
)


def main() -> int:
    args = parse_args()
    if args.loop:
        while True:
            launched = run_once(args)
            if args.max_total and launched >= args.max_total:
                return 0
            time.sleep(args.interval_seconds)
    run_once(args)
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Launch OPES A1 final-package runs from REGISTRO_TRABAJO_TEMAS_OPES.json.",
    )
    parser.add_argument("--registry", default=os.environ.get("ORQUESTA_OPES_TOPIC_REGISTRY", ""))
    parser.add_argument("--course-id", default=os.environ.get("ORQUESTA_OPES_COURSE_ID", "informatica-a1-72-padres"))
    parser.add_argument("--course-root", default=os.environ.get("ORQUESTA_OPES_COURSE_ROOT", ""))
    parser.add_argument("--app-change-state", default=os.environ.get("ORQUESTA_APP_CHANGE_STATE", ""))
    parser.add_argument("--orchestration-runs-dir", default=os.environ.get("ORQUESTA_ORCHESTRATION_RUNS_DIR", ""))
    parser.add_argument("--template-run-ref", default=os.environ.get("ORQUESTA_TEMPLATE_RUN_REF", "run-ref-opes-a1-t001-finalpkg-20260612"))
    parser.add_argument("--template-topic-id", default=os.environ.get("ORQUESTA_TEMPLATE_TOPIC_ID", "001"))
    parser.add_argument("--orquesta-base-url", default=default_orquesta_base_url())
    parser.add_argument("--queue-ref", default=os.environ.get("ORQUESTA_QUEUE_REF", "global"))
    parser.add_argument("--batch-size", type=int, default=int(os.environ.get("ORQUESTA_BATCH_SIZE", "6")))
    parser.add_argument("--max-in-flight", type=int, default=int(os.environ.get("ORQUESTA_MAX_IN_FLIGHT", "6")))
    parser.add_argument("--interval-seconds", type=int, default=int(os.environ.get("ORQUESTA_INTERVAL_SECONDS", "60")))
    parser.add_argument("--loop", action="store_true", default=os.environ.get("ORQUESTA_LOOP", "") == "1")
    parser.add_argument("--dry-run", action="store_true", default=os.environ.get("ORQUESTA_DRY_RUN", "") == "1")
    parser.add_argument("--max-total", type=int, default=int(os.environ.get("ORQUESTA_MAX_TOTAL", "0")))
    return parser.parse_args()


def run_once(args: argparse.Namespace) -> int:
    validate_args(args)
    registry = read_json(Path(args.registry))
    app_change_state = read_json(Path(args.app_change_state))
    template = find_template_request(app_change_state, args.template_run_ref)
    existing_run_refs = collect_existing_run_refs(app_change_state)
    active_run_refs = collect_active_run_refs(Path(args.orchestration_runs_dir))
    active_finalpkg = [
        ref for ref in active_run_refs
        if ref.startswith("run-ref-opes-a1-t") and ref.endswith("-finalpkg-20260612")
    ]
    if len(active_finalpkg) >= args.max_in_flight:
        print(json.dumps({
            "status": "idle",
            "reason": "max_in_flight",
            "active": active_finalpkg,
        }, ensure_ascii=False))
        return 0

    candidates = select_topic_candidates(
        registry=registry,
        course_id=args.course_id,
        course_root=Path(args.course_root),
        existing_run_refs=existing_run_refs,
        active_run_refs=active_run_refs,
    )
    capacity = max(0, min(args.batch_size, args.max_in_flight - len(active_finalpkg)))
    launched = []
    for topic_id in candidates[:capacity]:
        payload = build_external_work_payload(args, template, topic_id)
        if args.dry_run:
            launched.append({"topic_id": topic_id, "status": "dry_run", "run_ref": payload["external_work_run_request"]["run_ref"]})
            continue
        result = post_external_work_run(args.orquesta_base_url, payload)
        launched.append({
            "topic_id": topic_id,
            "status": result.get("estado", result.get("status", "")),
            "run_ref": result.get("run_ref", payload["external_work_run_request"]["run_ref"]),
            "errors": result.get("errores_publicos") or result.get("issues"),
        })

    print(json.dumps({
        "status": "ok",
        "dry_run": args.dry_run,
        "queue_ref": args.queue_ref,
        "selected": launched,
        "remaining_candidates": max(0, len(candidates) - len(launched)),
    }, ensure_ascii=False))
    return len(launched)


def validate_args(args: argparse.Namespace) -> None:
    required = {
        "--registry": args.registry,
        "--course-root": args.course_root,
        "--app-change-state": args.app_change_state,
        "--orchestration-runs-dir": args.orchestration_runs_dir,
    }
    missing = [name for name, value in required.items() if not str(value).strip()]
    if missing:
        raise SystemExit("missing required args: " + ", ".join(missing))
    if args.batch_size < 1 or args.max_in_flight < 1:
        raise SystemExit("--batch-size and --max-in-flight must be positive")
    if not args.dry_run and not str(args.orquesta_base_url).strip():
        raise SystemExit(
            "--orquesta-base-url required; set ORQUESTA_SERVER_URL or ORQUESTA_RUNTIME_DIR/base_url.txt",
        )


def default_orquesta_base_url() -> str:
    for name in ("ORQUESTA_SERVER_URL", "ORQUESTA_BASE_URL"):
        value = os.environ.get(name, "").strip()
        if value:
            return value
    runtime_dir = os.environ.get("ORQUESTA_RUNTIME_DIR", "").strip()
    if runtime_dir:
        try:
            return (Path(runtime_dir) / "base_url.txt").read_text(encoding="utf-8").strip()
        except OSError:
            return ""
    return ""


def read_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def find_template_request(app_change_state: dict[str, Any], run_ref: str) -> dict[str, Any]:
    for record in app_change_state.get("records") or []:
        request = record.get("request") or {}
        if request.get("run_ref") == run_ref:
            return request
    raise SystemExit(f"template run not found: {run_ref}")


def collect_existing_run_refs(app_change_state: dict[str, Any]) -> set[str]:
    refs: set[str] = set()
    for record in app_change_state.get("records") or []:
        ref = str((record.get("request") or {}).get("run_ref") or "").strip()
        if ref:
            refs.add(ref)
    return refs


def collect_active_run_refs(runs_dir: Path) -> set[str]:
    refs: set[str] = set()
    if not runs_dir.is_dir():
        return refs
    for path in runs_dir.glob("*.json"):
        try:
            data = read_json(path)
        except (OSError, json.JSONDecodeError):
            continue
        run = data.get("run") or data
        ref = str(data.get("run_ref") or run.get("run_id") or "").strip()
        status = str(run.get("status") or "").strip()
        if ref and status not in {"cerrada", "closed", "cancelled", "stopped"}:
            refs.add(ref)
    return refs


def select_topic_candidates(
    registry: dict[str, Any],
    course_id: str,
    course_root: Path,
    existing_run_refs: set[str],
    active_run_refs: set[str],
) -> list[str]:
    course = (registry.get("courses") or {}).get(course_id)
    if not course:
        raise SystemExit(f"course not found: {course_id}")
    topics = course.get("topics") or {}
    out: list[str] = []
    for topic_id in sorted(topics):
        topic = topics[topic_id] or {}
        run_ref = finalpkg_run_ref(topic_id)
        if run_ref in existing_run_refs or run_ref in active_run_refs:
            continue
        if topic_has_active_lock(topic):
            continue
        if package_is_complete(course_root / f"tema_{topic_id}" / "paquete_final"):
            continue
        out.append(topic_id)
    return out


def topic_has_active_lock(topic: dict[str, Any]) -> bool:
    lock = topic.get("lock")
    if not isinstance(lock, dict):
        return False
    return bool(str(lock.get("agent_id") or lock.get("owner") or "").strip())


def package_is_complete(package_dir: Path) -> bool:
    for relative in REQUIRED_PACKAGE_FILES:
        path = package_dir / relative
        if not path.is_file() or path.stat().st_size <= 0:
            return False
    try:
        read_json(package_dir / "tests.json")
    except (OSError, json.JSONDecodeError):
        return False
    if not rag_manifest_is_canonical(package_dir):
        return False
    final_html = topic_html_refs(package_dir, "html_final")
    expanded_html = topic_html_refs(package_dir, "html_ampliado")
    if not final_html or not expanded_html:
        return False
    material_refs = audio_manifest_material_refs(package_dir)
    if material_refs is None:
        return False
    topic_refs = [*final_html, *expanded_html]
    if not all(ref in material_refs for ref in topic_refs):
        return False
    return legacy_audio_manifest_is_compatible(package_dir, topic_refs)


def rag_manifest_is_canonical(package_dir: Path) -> bool:
    for relative in ("rag/chunks.jsonl", "rag/summary.json"):
        if (package_dir / relative).exists():
            return False
    try:
        manifest = read_json(package_dir / "rag" / "manifest.json")
    except (OSError, json.JSONDecodeError):
        return False
    return json_contains_string(manifest, "rag/corpus/chunks.jsonl") and json_contains_string(
        manifest,
        "rag/corpus/summary.json",
    )


def topic_html_refs(package_dir: Path, root: str) -> list[str]:
    html_root = package_dir / root
    if not html_root.is_dir():
        return []
    refs: list[str] = []
    for path in html_root.glob("*.html"):
        if path.name.lower().startswith("tema_") and path.is_file() and path.stat().st_size > 0:
            refs.append(f"{root}/{path.name}")
    return sorted(set(refs))


def audio_manifest_material_refs(package_dir: Path) -> set[str] | None:
    manifest_root = package_dir / "audio" / "manifests"
    if not manifest_root.is_dir():
        return set()
    refs: set[str] = set()
    for path in manifest_root.rglob("*.json"):
        try:
            data = read_json(path)
        except (OSError, json.JSONDecodeError):
            return None
        material_path = str(data.get("material_path") or "").strip().replace("\\", "/")
        if material_path:
            refs.add(material_path)
    return refs


def legacy_audio_manifest_is_compatible(package_dir: Path, topic_html_refs: list[str]) -> bool:
    path = package_dir / "audio" / "manifest.json"
    if not path.exists():
        return True
    try:
        manifest = read_json(path)
    except (OSError, json.JSONDecodeError):
        return False
    if json_contains_path_base(manifest, "index.html"):
        return False
    count = json_declared_count(manifest)
    all_refs = all_html_refs(package_dir)
    if (
        count is not None
        and topic_html_refs
        and len(all_refs) > len(topic_html_refs)
        and count == len(all_refs)
        and count != len(topic_html_refs)
    ):
        return False
    return True


def all_html_refs(package_dir: Path) -> list[str]:
    refs: list[str] = []
    for root in ("html_final", "html_ampliado"):
        html_root = package_dir / root
        if not html_root.is_dir():
            continue
        for path in html_root.glob("*.html"):
            if path.is_file() and path.stat().st_size > 0:
                refs.append(f"{root}/{path.name}")
    return sorted(set(refs))


def json_contains_string(value: Any, wanted: str) -> bool:
    if isinstance(value, str):
        return value.strip().replace("\\", "/") == wanted
    if isinstance(value, list):
        return any(json_contains_string(item, wanted) for item in value)
    if isinstance(value, dict):
        return any(json_contains_string(item, wanted) for item in value.values())
    return False


def json_contains_path_base(value: Any, base: str) -> bool:
    base = base.strip().lower()
    if isinstance(value, str):
        return Path(value.strip().replace("\\", "/")).name.lower() == base
    if isinstance(value, list):
        return any(json_contains_path_base(item, base) for item in value)
    if isinstance(value, dict):
        return any(json_contains_path_base(item, base) for item in value.values())
    return False


def json_declared_count(value: Any) -> int | None:
    if not isinstance(value, dict):
        return None
    for key in (
        "expected_count",
        "topic_count",
        "topics_count",
        "html_topic_count",
        "audio_topic_count",
        "source_html_count",
    ):
        raw = value.get(key)
        if isinstance(raw, int) and raw >= 0:
            return raw
    return None


def build_external_work_payload(
    args: argparse.Namespace,
    template: dict[str, Any],
    topic_id: str,
) -> dict[str, Any]:
    app_change = replace_topic_id(copy.deepcopy(template), args.template_topic_id, topic_id)
    app_change["user_intent"] = (
        f"Finalizar paquete local verificable del tema {topic_id} de Informatica A1 "
        "reutilizando material existente, checkpoint y fuente; no crear checkpoint nuevo."
    )
    criteria = list(app_change.get("acceptance_criteria") or [])
    app_change["acceptance_criteria"] = compact_list([
        "paquete_final contiene manifest_cierre.json, index.html, tests.json, visuales_plan.md, rag/manifest.json, tutor/tutor_prompt.md, qa_final.md, html_final/tema_*.html y html_ampliado/tema_*.html",
        "rag/manifest.json referencia rag/corpus/chunks.jsonl y rag/corpus/summary.json canonicos",
        "audio/manifests contiene un JSON por cada html_final/tema_*.html y html_ampliado/tema_*.html con material_path a la pagina tematica; index.html, portadas y listados no cuentan",
        "tests.json es JSON valido y contiene preguntas publicables con enunciado, opciones y respuesta",
        "REGISTRO_TRABAJO_TEMAS_OPES.json usa courses[course_id].topics como diccionario por topic_id; usa claim/update/release de la herramienta y no lo trates como lista",
        "al liberar el registro, preferir status paquete_final_local_verificable si el paquete cumple; si usa alias equivalente, explicar en summary/pending",
        "no subir a produccion; paquete local verificable",
        *criteria,
    ])
    run_ref = finalpkg_run_ref(topic_id)
    return {
        "request_id": f"request-launch-opes-a1-t{topic_id}-finalpkg-autonomous-registry",
        "correlation_id": f"corr-launch-opes-a1-t{topic_id}-finalpkg-autonomous-registry",
        "external_work_run_request": {
            "schema_version": "external_work_run_request.v0",
            "request_id": f"request-ref-opes-a1-t{topic_id}-finalpkg-autonomous-registry",
            "correlation_id": f"corr-opes-a1-t{topic_id}-finalpkg-autonomous-registry",
            "run_ref": run_ref,
            "project_ref": "opes-a1-informatica",
            "app_spec_ref": "app-spec-external-work-opes-a1-informatica",
            "queue_ref": args.queue_ref,
            "priority_score": 88,
            "occurred_at": utc_now(),
            "requested_by": "orquesta-registry-launcher",
            "app_change_request": app_change,
        },
    }


def replace_topic_id(value: Any, old: str, new: str) -> Any:
    if isinstance(value, str):
        return value.replace(old, new)
    if isinstance(value, list):
        return [replace_topic_id(item, old, new) for item in value]
    if isinstance(value, dict):
        return {key: replace_topic_id(item, old, new) for key, item in value.items()}
    return value


def compact_list(values: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for value in values:
        value = str(value).strip()
        if not value or value in seen:
            continue
        seen.add(value)
        out.append(value)
    return out


def finalpkg_run_ref(topic_id: str) -> str:
    return f"run-ref-opes-a1-t{topic_id}-finalpkg-20260612"


def utc_now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def post_external_work_run(base_url: str, payload: dict[str, Any]) -> dict[str, Any]:
    url = base_url.rstrip("/") + "/api/v0/external-work/run"
    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        try:
            detail = json.loads(exc.read().decode("utf-8"))
        except Exception:
            detail = {"error": str(exc)}
        return {"estado": "error", "errores_publicos": detail}


if __name__ == "__main__":
    sys.exit(main())
