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


def write_report(results: list[TopicValidationResult], report_path: Path) -> None:
    report_path.parent.mkdir(parents=True, exist_ok=True)
    lines = [
        "# Validacion OPES A1",
        "",
        f"- fecha: {_dt.date.today().isoformat()}",
        f"- temas: {len(results)}",
        f"- resultado: {'OK' if all(r.ok for r in results) else 'ERROR'}",
        "",
        "| tema | resultado | palabras | duplicados largos | incidencias |",
        "| --- | --- | ---: | ---: | --- |",
    ]
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
    report_path.write_text("\n".join(lines), encoding="utf-8")


def parse_args(argv: list[str] | None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate OPES A1 topic packages.")
    parser.add_argument("--topic-dir", action="append", default=[], help="Topic directory to validate. Can be repeated.")
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
    if not topic_dirs:
        print("no topic directories provided", file=sys.stderr)
        return 2

    results = [validate_topic(path, args) for path in topic_dirs]
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
    if args.report:
        write_report(results, Path(args.report))
        print(f"report={args.report}")
    return 0 if all(result.ok for result in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
