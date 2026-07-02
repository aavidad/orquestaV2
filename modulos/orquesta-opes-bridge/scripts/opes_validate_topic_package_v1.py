#!/usr/bin/env python3
"""Validate OPES A1 topic packages produced by Orquesta/Codex waves.

The validator is intentionally mechanical. It does not grade doctrine, but it
does enforce the gates that should not depend on taste: required files, word
range, duplicate long paragraphs, offline HTML references, JSON validity, SVG
parseability and absence of internal runtime leaks in final deliverables.
"""

from __future__ import annotations

import argparse
import datetime as _dt
import html.parser
import json
import re
import sys
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import urlsplit


DEFAULT_REQUIRED_FILES = (
    "tema_a1.md",
    "tema_a1.html",
    "fuentes.md",
    "checklist_a1.md",
    "banco_preguntas_i18n_es.json",
)

DEFAULT_LEAK_CHECK_FILES = (
    "tema_a1.md",
    "tema_a1.html",
    "fuentes.md",
    "banco_preguntas_i18n_es.json",
)

DEFAULT_MIN_QUESTIONS = 50
DEFAULT_OPTIONS_PER_QUESTION = 4

LEAK_PATTERNS = (
    "/home/",
    ".orquesta-runtime",
    "codex-waves",
    "agent_prompt",
    "codex_stderr",
    "codex_stdout",
    "write-set",
    "approval-policy",
    "sandbox danger-full-access",
)

REQUIRED_HTML_MARKERS = (
    "<!doctype html>",
    'lang="es"',
    'class="first-reading-on"',
    "data-topic-key=",
    'id="firstReadingToggle"',
    'id="tutorToggle"',
    'id="testNotesToggle"',
    'id="visualsToggle"',
    'class="mode-bar"',
    'class="side-nav"',
)


class RefCollectingHTMLParser(html.parser.HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.refs: list[tuple[str, str]] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        for name, value in attrs:
            if name in {"src", "href"} and value:
                self.refs.append((name, value.strip()))


@dataclass
class TopicValidationResult:
    topic_dir: Path
    words: int = 0
    duplicate_long_paras: int = 0
    validation_min_questions: int = DEFAULT_MIN_QUESTIONS
    validation_options_per_question: int = DEFAULT_OPTIONS_PER_QUESTION
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.errors


@dataclass
class FinalPackageValidationResult:
    package_dir: Path
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.errors


def word_count(text: str) -> int:
    return len(text.split())


def long_duplicate_count(markdown_text: str, min_words: int) -> int:
    seen: set[str] = set()
    duplicates: set[str] = set()
    for raw in re.split(r"\n\s*\n", markdown_text):
        para = raw.strip()
        if not para:
            continue
        if para.startswith(("#", "|", "!", ">", "<")):
            continue
        if re.match(r"^[-*]\s+", para) or re.match(r"^\d+\.\s+", para):
            continue
        if word_count(para) < min_words:
            continue
        normalized = re.sub(r"\W+", " ", para.casefold(), flags=re.UNICODE).strip()
        if normalized in seen:
            duplicates.add(normalized)
        seen.add(normalized)
    return len(duplicates)


def validate_html(html_path: Path, topic_dir: Path, result: TopicValidationResult) -> None:
    html_text = html_path.read_text(encoding="utf-8")
    parser = RefCollectingHTMLParser()
    try:
        parser.feed(html_text)
    except Exception as exc:  # pragma: no cover - HTMLParser rarely raises.
        result.errors.append(f"html_parse_error={exc}")
        return

    lowered = html_text.lower()
    for marker in REQUIRED_HTML_MARKERS:
        if marker.lower() not in lowered:
            result.errors.append(f"html_missing_marker={marker}")

    base = topic_dir.resolve()
    for attr, ref in parser.refs:
        if not ref or ref.startswith("#") or ref.startswith("mailto:"):
            continue
        split = urlsplit(ref)
        if split.scheme in {"http", "https", "file", "data", "javascript"}:
            result.errors.append(f"html_external_or_forbidden_ref={attr}:{ref}")
            continue
        local_ref = split.path
        if not local_ref:
            continue
        candidate = (topic_dir / local_ref).resolve()
        try:
            candidate.relative_to(base)
        except ValueError:
            result.errors.append(f"html_ref_outside_topic={attr}:{ref}")
            continue
        if not candidate.exists():
            result.errors.append(f"html_missing_ref={attr}:{ref}")


def validate_json_bank(json_path: Path, result: TopicValidationResult) -> None:
    try:
        data = json.loads(json_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        result.errors.append(f"json_invalid={exc}")
        return
    if data in ({}, []):
        result.errors.append("json_empty_bank")
        return

    questions = extract_questions(data)
    if not isinstance(questions, list):
        result.errors.append("json_questions_missing")
        return
    if len(questions) < result.validation_min_questions:
        result.errors.append(
            f"question_count_below_min={len(questions)}<{result.validation_min_questions}"
        )
    seen_stems: set[str] = set()
    for idx, question in enumerate(questions, start=1):
        if not isinstance(question, dict):
            result.errors.append(f"question_{idx}_not_object")
            continue
        stem = extract_question_stem(question)
        if not stem:
            result.errors.append(f"question_{idx}_missing_stem")
        else:
            normalized_stem = re.sub(r"\W+", " ", stem.casefold()).strip()
            if normalized_stem in seen_stems:
                result.errors.append(f"question_{idx}_duplicate_stem")
            seen_stems.add(normalized_stem)
        options = extract_question_options(question)
        if len(options) != result.validation_options_per_question:
            result.errors.append(
                f"question_{idx}_options={len(options)}!={result.validation_options_per_question}"
            )
        if len({normalize_option_text(option) for option in options}) != len(options):
            result.errors.append(f"question_{idx}_duplicate_options")
        if not has_correct_answer(question):
            result.errors.append(f"question_{idx}_missing_correct_answer")
        if has_trivial_distractors(options):
            result.errors.append(f"question_{idx}_trivial_distractor")


def extract_questions(data: object) -> object:
    if isinstance(data, list):
        return data
    if not isinstance(data, dict):
        return None
    for key in ("questions", "preguntas", "items"):
        value = data.get(key)
        if isinstance(value, list):
            return value
    return None


def extract_question_stem(question: dict[str, object]) -> str:
    for key in ("stem", "enunciado", "pregunta"):
        value = question.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    enunciado_i18n = question.get("enunciado_i18n")
    if isinstance(enunciado_i18n, dict):
        value = enunciado_i18n.get("es")
        if isinstance(value, str):
            return value.strip()
    return ""


def extract_question_options(question: dict[str, object]) -> list[str]:
    for key in ("options", "opciones"):
        value = question.get(key)
        options = normalize_options_value(value)
        if options:
            return options
    opciones_i18n = question.get("opciones_i18n")
    if isinstance(opciones_i18n, dict):
        options = normalize_options_value(opciones_i18n.get("es"))
        if options:
            return options
    return []


def normalize_options_value(value: object) -> list[str]:
    if isinstance(value, list):
        options: list[str] = []
        for item in value:
            if isinstance(item, str):
                options.append(item)
            elif isinstance(item, dict):
                text = item.get("text") or item.get("texto") or item.get("label")
                if isinstance(text, str):
                    options.append(text)
        return options
    if isinstance(value, dict):
        return [str(value[key]) for key in sorted(value) if str(value[key]).strip()]
    return []


def normalize_option_text(value: str) -> str:
    return re.sub(r"\W+", " ", value.casefold()).strip()


def has_correct_answer(question: dict[str, object]) -> bool:
    for key in (
        "correcta",
        "respuesta_correcta",
        "correct_option_id",
        "correct",
        "answer",
        "answer_id",
    ):
        if key in question and question[key] not in (None, ""):
            return True
    return False


def has_trivial_distractors(options: list[str]) -> bool:
    trivial_patterns = (
        r"\btodas las anteriores\b",
        r"\bninguna de las anteriores\b",
        r"\bno sabe\b",
        r"\bno contesta\b",
        r"\bobviamente\b",
        r"\babsurda\b",
    )
    for option in options:
        normalized = normalize_option_text(option)
        if len(normalized.split()) < 3:
            return True
        if any(re.search(pattern, normalized) for pattern in trivial_patterns):
            return True
    return False


def validate_svgs(topic_dir: Path, result: TopicValidationResult) -> None:
    for svg_path in sorted(topic_dir.glob("assets/**/*.svg")):
        try:
            root = ET.parse(svg_path).getroot()
        except ET.ParseError as exc:
            result.errors.append(f"svg_invalid={svg_path.relative_to(topic_dir)}:{exc}")
            continue
        if not root.tag.lower().endswith("svg"):
            result.errors.append(f"svg_root_not_svg={svg_path.relative_to(topic_dir)}")


def validate_leaks(topic_dir: Path, result: TopicValidationResult) -> None:
    for rel in DEFAULT_LEAK_CHECK_FILES:
        path = topic_dir / rel
        if not path.exists():
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        lowered = text.lower()
        for pattern in LEAK_PATTERNS:
            if pattern.lower() in lowered:
                result.errors.append(f"internal_leak={rel}:{pattern}")


def read_json(path: Path, errors: list[str], label: str) -> object | None:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        errors.append(f"missing_required_file={path.relative_to(path.parent)}")
    except json.JSONDecodeError as exc:
        errors.append(f"{label}_json_invalid={exc}")
    return None


def normalized_metadata_key(value: object) -> str:
    return re.sub(r"[^a-z0-9]+", "_", str(value).casefold()).strip("_")


def object_has_metadata_key(data: object, accepted_keys: set[str]) -> bool:
    if isinstance(data, dict):
        for key, value in data.items():
            if normalized_metadata_key(key) in accepted_keys and value not in (None, "", []):
                return True
            if isinstance(value, (dict, list)) and object_has_metadata_key(value, accepted_keys):
                return True
    if isinstance(data, list):
        return any(object_has_metadata_key(item, accepted_keys) for item in data)
    return False


def collect_string_values(data: object) -> list[str]:
    values: list[str] = []
    if isinstance(data, dict):
        for value in data.values():
            values.extend(collect_string_values(value))
    elif isinstance(data, list):
        for item in data:
            values.extend(collect_string_values(item))
    elif isinstance(data, str):
        values.append(data.strip())
    return values


def has_any_path_ref(data: object, wanted_path: str) -> bool:
    wanted = wanted_path.replace("\\", "/").strip("/")
    for value in collect_string_values(data):
        normalized = value.replace("\\", "/").strip("/")
        if normalized == wanted or normalized.endswith("/" + wanted):
            return True
    return False


def validate_manifest_cierre(package_dir: Path, result: FinalPackageValidationResult) -> None:
    manifest_path = package_dir / "manifest_cierre.json"
    if not manifest_path.is_file():
        result.errors.append("missing_required_file=manifest_cierre.json")
        return
    data = read_json(manifest_path, result.errors, "manifest_cierre")
    if data is None:
        return
    schema_values = {
        str(value)
        for value in collect_string_values(data)
        if "opes_final_package_evidence_manifest" in str(value)
    }
    if "opes_final_package_evidence_manifest.v0" not in schema_values:
        result.errors.append("manifest_cierre_schema_invalid")


RAG_COURSE_KEYS = {
    "course_id",
    "course_ref",
    "course_slug",
    "program_id",
    "syllabus_id",
}

RAG_SOURCE_VARIANT_KEYS = {
    "source_variant",
    "source_variants",
    "document_variant",
    "content_variant",
    "variant",
    "source_profile",
}


def validate_rag_contract(package_dir: Path, result: FinalPackageValidationResult) -> None:
    rag_dir = package_dir / "rag"
    manifest_path = rag_dir / "manifest.json"
    chunks_path = rag_dir / "corpus" / "chunks.jsonl"
    summary_path = rag_dir / "corpus" / "summary.json"

    if (rag_dir / "chunks.jsonl").exists():
        result.errors.append("rag_loose_chunks_not_canonical=rag/chunks.jsonl")
    if (rag_dir / "summary.json").exists():
        result.errors.append("rag_loose_summary_not_canonical=rag/summary.json")

    for path in (manifest_path, chunks_path, summary_path):
        if not path.is_file():
            result.errors.append(f"missing_required_file={path.relative_to(package_dir)}")

    if manifest_path.is_file():
        manifest = read_json(manifest_path, result.errors, "rag_manifest")
        if manifest is not None:
            if not has_any_path_ref(manifest, "rag/corpus/chunks.jsonl"):
                result.errors.append("rag_manifest_missing_chunks_ref=rag/corpus/chunks.jsonl")
            if not has_any_path_ref(manifest, "rag/corpus/summary.json"):
                result.errors.append("rag_manifest_missing_summary_ref=rag/corpus/summary.json")
            if not object_has_metadata_key(manifest, RAG_COURSE_KEYS):
                result.errors.append("rag_manifest_missing_course_id")

    if summary_path.is_file():
        summary = read_json(summary_path, result.errors, "rag_summary")
        if summary is not None:
            if not object_has_metadata_key(summary, RAG_COURSE_KEYS):
                result.errors.append("rag_summary_missing_course_id")
            if not object_has_metadata_key(summary, RAG_SOURCE_VARIANT_KEYS):
                result.errors.append("rag_summary_missing_source_variant")

    if chunks_path.is_file():
        missing_course = 0
        missing_source_variant = 0
        invalid_json = 0
        chunk_count = 0
        for line_no, raw_line in enumerate(chunks_path.read_text(encoding="utf-8").splitlines(), start=1):
            line = raw_line.strip()
            if not line:
                continue
            chunk_count += 1
            try:
                chunk = json.loads(line)
            except json.JSONDecodeError:
                invalid_json += 1
                if invalid_json <= 3:
                    result.errors.append(f"rag_chunk_json_invalid=line{line_no}")
                continue
            if not object_has_metadata_key(chunk, RAG_COURSE_KEYS):
                missing_course += 1
            if not object_has_metadata_key(chunk, RAG_SOURCE_VARIANT_KEYS):
                missing_source_variant += 1
        if chunk_count == 0:
            result.errors.append("rag_chunks_empty")
        if invalid_json > 3:
            result.errors.append(f"rag_chunk_json_invalid_extra={invalid_json - 3}")
        if missing_course:
            result.errors.append(f"rag_chunks_missing_course_id={missing_course}")
        if missing_source_variant:
            result.errors.append(f"rag_chunks_missing_source_variant={missing_source_variant}")


def find_visual_manifests(package_dir: Path) -> list[Path]:
    candidates = [
        package_dir / "visuals_manifest.json",
        package_dir / "visual_reuse_manifest.json",
        package_dir / "08_assets" / "visuals_manifest.json",
        package_dir / "08_assets" / "visual_reuse_manifest.json",
        package_dir / "visual_reuse" / "visual_reuse_manifest.json",
    ]
    return [path for path in candidates if path.is_file()]


def numeric_value(data: object, key: str) -> int | None:
    if isinstance(data, dict):
        for current_key, value in data.items():
            if normalized_metadata_key(current_key) == key:
                if isinstance(value, int):
                    return value
                if isinstance(value, float):
                    return int(value)
            nested = numeric_value(value, key)
            if nested is not None:
                return nested
    if isinstance(data, list):
        for item in data:
            nested = numeric_value(item, key)
            if nested is not None:
                return nested
    return None


def visual_asset_entries(data: object) -> list[dict[str, object]]:
    if isinstance(data, dict):
        entries: list[dict[str, object]] = []
        for key in ("assets", "visuals", "items", "reused_visual_assets", "copied_visual_assets"):
            value = data.get(key)
            if isinstance(value, list):
                entries.extend(item for item in value if isinstance(item, dict))
        return entries
    if isinstance(data, list):
        return [item for item in data if isinstance(item, dict)]
    return []


def path_values_from_visual_asset(asset: dict[str, object]) -> list[str]:
    values: list[str] = []
    for key in (
        "variants",
        "variant_paths",
        "html_variants",
        "file",
        "path",
        "src",
        "href",
        "asset_path",
        "asset_ref",
    ):
        value = asset.get(key)
        if isinstance(value, str):
            values.append(value)
        elif isinstance(value, list):
            values.extend(str(item) for item in value if isinstance(item, str))
    return [value.replace("\\", "/").strip("/") for value in values if value.strip()]


def html_text_for_variant(package_dir: Path, variant_dir: str) -> str:
    root = package_dir / variant_dir
    if not root.is_dir():
        return ""
    texts: list[str] = []
    for html_path in sorted(root.glob("tema_*.html")):
        texts.append(html_path.read_text(encoding="utf-8", errors="replace"))
    return "\n".join(texts)


def validate_visual_manifest(package_dir: Path, manifest_path: Path, result: FinalPackageValidationResult) -> None:
    data = read_json(manifest_path, result.errors, "visual_manifest")
    if data is None:
        return
    pending_count = numeric_value(data, "pending_count")
    if pending_count is not None and pending_count > 0:
        result.errors.append(f"visual_manifest_pending_count={pending_count}")

    entries = visual_asset_entries(data)
    for index, asset in enumerate(entries, start=1):
        status = str(asset.get("status", "")).casefold()
        if "rejected" in status or "descart" in status:
            continue
        paths = path_values_from_visual_asset(asset)
        html_variant_paths = [
            path
            for path in paths
            if path.startswith("html_final/") or path.startswith("html_ampliado/")
        ]
        if not html_variant_paths and "file" in asset:
            html_variant_paths = [
                "html_final/" + str(asset["file"]).replace("\\", "/").strip("/"),
                "html_ampliado/" + str(asset["file"]).replace("\\", "/").strip("/"),
            ]
        for rel_path in html_variant_paths:
            asset_path = package_dir / rel_path
            if not asset_path.is_file():
                result.errors.append(f"visual_manifest_asset_missing={rel_path}")
                continue
            variant_dir = rel_path.split("/", 1)[0]
            html_text = html_text_for_variant(package_dir, variant_dir)
            if not html_text:
                result.errors.append(f"visual_manifest_html_variant_missing={variant_dir}")
                continue
            basename = Path(rel_path).name
            local_ref = rel_path.split("/", 1)[1] if "/" in rel_path else rel_path
            if basename not in html_text and local_ref not in html_text and rel_path not in html_text:
                result.errors.append(f"visual_manifest_ref_not_in_html={rel_path}")
        if not paths:
            result.warnings.append(f"visual_manifest_asset_without_path=index{index}")


def validate_final_package(package_dir: Path) -> FinalPackageValidationResult:
    result = FinalPackageValidationResult(package_dir=package_dir)
    if not package_dir.is_dir():
        result.errors.append(f"final_package_dir_missing={package_dir}")
        return result
    validate_manifest_cierre(package_dir, result)
    validate_rag_contract(package_dir, result)
    for manifest_path in find_visual_manifests(package_dir):
        validate_visual_manifest(package_dir, manifest_path, result)
    return result


def validate_topic(topic_dir: Path, args: argparse.Namespace) -> TopicValidationResult:
    result = TopicValidationResult(
        topic_dir=topic_dir,
        validation_min_questions=args.min_questions,
        validation_options_per_question=args.options_per_question,
    )
    for rel in DEFAULT_REQUIRED_FILES:
        if not (topic_dir / rel).is_file():
            result.errors.append(f"missing_required_file={rel}")
    revision_dir = topic_dir / "revision"
    if not revision_dir.is_dir() or not any(revision_dir.glob("*.md")):
        result.errors.append("missing_revision_report_md")

    markdown_path = topic_dir / "tema_a1.md"
    if markdown_path.exists():
        markdown_text = markdown_path.read_text(encoding="utf-8")
        result.words = word_count(markdown_text)
        if result.words < args.min_words:
            result.errors.append(f"word_count_below_min={result.words}<{args.min_words}")
        if result.words > args.max_words:
            result.errors.append(f"word_count_above_max={result.words}>{args.max_words}")
        result.duplicate_long_paras = long_duplicate_count(markdown_text, args.duplicate_min_words)
        if result.duplicate_long_paras:
            result.errors.append(f"duplicate_long_paras={result.duplicate_long_paras}")

    html_path = topic_dir / "tema_a1.html"
    if html_path.exists():
        validate_html(html_path, topic_dir, result)

    json_path = topic_dir / "banco_preguntas_i18n_es.json"
    if json_path.exists():
        validate_json_bank(json_path, result)

    validate_svgs(topic_dir, result)
    validate_leaks(topic_dir, result)
    return result


def topic_dirs_from_manifest(manifest_path: Path, batch_index: int | None, base_dir: Path) -> list[Path]:
    data = json.loads(manifest_path.read_text(encoding="utf-8"))
    batches = data.get("batches")
    if not isinstance(batches, list):
        raise ValueError("manifest must contain a batches list")
    selected = batches if batch_index is None else [batches[batch_index]]
    topic_dirs: list[Path] = []
    for batch in selected:
        for topic in batch.get("topics", []):
            workdir = topic.get("workdir")
            if workdir:
                topic_dirs.append(base_dir / str(workdir))
    return topic_dirs


def write_report(
    results: list[TopicValidationResult],
    final_results: list[FinalPackageValidationResult],
    report_path: Path,
) -> None:
    report_path.parent.mkdir(parents=True, exist_ok=True)
    lines = [
        "# Validacion OPES A1",
        "",
        f"- fecha: {_dt.date.today().isoformat()}",
        f"- temas: {len(results)}",
        f"- paquetes_finales: {len(final_results)}",
        f"- resultado: {'OK' if all(r.ok for r in results + final_results) else 'ERROR'}",
        "",
    ]
    if results:
        lines.extend([
        "| tema | resultado | palabras | duplicados largos | incidencias |",
        "| --- | --- | ---: | ---: | --- |",
        ])
        for result in results:
            issues = result.errors + result.warnings
            issue_text = "<br>".join(issues) if issues else "OK"
            lines.append(
                "| "
                + str(result.topic_dir)
                + " | "
                + ("OK" if result.ok else "ERROR")
                + " | "
                + str(result.words)
                + " | "
                + str(result.duplicate_long_paras)
                + " | "
                + issue_text.replace("|", "\\|")
                + " |"
            )
        lines.append("")
    if final_results:
        lines.extend([
            "| paquete_final | resultado | incidencias |",
            "| --- | --- | --- |",
        ])
        for result in final_results:
            issues = result.errors + result.warnings
            issue_text = "<br>".join(issues) if issues else "OK"
            lines.append(
                "| "
                + str(result.package_dir)
                + " | "
                + ("OK" if result.ok else "ERROR")
                + " | "
                + issue_text.replace("|", "\\|")
                + " |"
            )
        lines.append("")
    report_path.write_text("\n".join(lines), encoding="utf-8")


def parse_args(argv: list[str] | None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate OPES A1 topic packages.")
    parser.add_argument("--topic-dir", action="append", default=[], help="Topic directory to validate. Can be repeated.")
    parser.add_argument(
        "--final-package-dir",
        action="append",
        default=[],
        help="Final completed_syllabus_package directory to validate. Can be repeated.",
    )
    parser.add_argument("--manifest", help="Optional batches.json manifest.")
    parser.add_argument("--batch-index", type=int, help="Zero-based batch index in manifest.")
    parser.add_argument("--base-dir", help="Base production directory for manifest workdirs.")
    parser.add_argument("--min-words", type=int, default=20250)
    parser.add_argument("--max-words", type=int, default=22500)
    parser.add_argument("--duplicate-min-words", type=int, default=45)
    parser.add_argument("--min-questions", type=int, default=DEFAULT_MIN_QUESTIONS)
    parser.add_argument("--options-per-question", type=int, default=DEFAULT_OPTIONS_PER_QUESTION)
    parser.add_argument("--report", help="Optional Markdown report path.")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    topic_dirs = [Path(path) for path in args.topic_dir]
    if args.manifest:
        if not args.base_dir:
            print("--base-dir is required when --manifest is used", file=sys.stderr)
            return 2
        try:
            topic_dirs.extend(
                topic_dirs_from_manifest(Path(args.manifest), args.batch_index, Path(args.base_dir))
            )
        except Exception as exc:
            print(f"manifest_error={exc}", file=sys.stderr)
            return 2
    final_package_dirs = [Path(path) for path in args.final_package_dir]
    if not topic_dirs and not final_package_dirs:
        print("no topic or final package directories provided", file=sys.stderr)
        return 2

    results = [validate_topic(path, args) for path in topic_dirs]
    final_results = [validate_final_package(path) for path in final_package_dirs]
    for result in results:
        status = "OK" if result.ok else "ERROR"
        print(
            f"{status} topic={result.topic_dir} words={result.words} "
            f"duplicate_long_paras={result.duplicate_long_paras}"
        )
        for issue in result.errors:
            print(f"  - {issue}")
        for warning in result.warnings:
            print(f"  - warning:{warning}")
    for result in final_results:
        status = "OK" if result.ok else "ERROR"
        print(f"{status} final_package={result.package_dir}")
        for issue in result.errors:
            print(f"  - {issue}")
        for warning in result.warnings:
            print(f"  - warning:{warning}")
    if args.report:
        write_report(results, final_results, Path(args.report))
        print(f"report={args.report}")
    return 0 if all(result.ok for result in results + final_results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
