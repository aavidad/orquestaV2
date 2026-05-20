#!/usr/bin/env python3
"""Render OPES topic HTML with the canonical v1 template.

The script owns presentation. Agents provide content as Markdown plus optional
metadata; they should not handcraft a different HTML shell.
"""

from __future__ import annotations

import argparse
import datetime as _dt
import html
import json
import re
import sys
from html.parser import HTMLParser
from pathlib import Path
from string import Template


CONNECTOR_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_TEMPLATE = CONNECTOR_ROOT / "templates" / "opes_html_topic_template_v1.html"


def slugify(value: str) -> str:
    value = value.strip().lower()
    value = value.replace("ñ", "n")
    value = re.sub(r"\{#([^}]+)\}$", "", value).strip()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    return value.strip("-") or "seccion"


def split_heading_id(text: str) -> tuple[str, str]:
    text = text.strip()
    match = re.search(r"\s+\{#([^}]+)\}$", text)
    if match:
        title = text[: match.start()].strip()
        return title, match.group(1).strip()
    return text, slugify(text)


def inline_html(text: str) -> str:
    text = html.escape(text.strip(), quote=False)
    text = re.sub(r"`([^`]+)`", r"<code>\1</code>", text)
    text = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", text)
    text = re.sub(r"\*([^*]+)\*", r"<em>\1</em>", text)
    return text


def plain_heading(text: str) -> str:
    text = re.sub(r"\{#([^}]+)\}$", "", text).strip()
    text = re.sub(r"^[#]+\s+", "", text)
    return text


def parse_table(lines: list[str], start: int) -> tuple[str, int]:
    table_lines: list[str] = []
    idx = start
    while idx < len(lines) and lines[idx].strip().startswith("|"):
        table_lines.append(lines[idx].strip())
        idx += 1
    rows: list[list[str]] = []
    for line in table_lines:
        cells = [cell.strip() for cell in line.strip("|").split("|")]
        if cells and all(re.fullmatch(r":?-{3,}:?", cell or "") for cell in cells):
            continue
        rows.append(cells)
    if not rows:
        return "", idx
    head = rows[0]
    body = rows[1:]
    out = ["<div class=\"table-wrap\"><table>"]
    out.append(
        "<thead><tr>"
        + "".join(f"<th>{inline_html(cell)}</th>" for cell in head)
        + "</tr></thead>"
    )
    out.append("<tbody>")
    for row in body:
        out.append(
            "<tr>"
            + "".join(f"<td>{inline_html(cell)}</td>" for cell in row)
            + "</tr>"
        )
    out.append("</tbody></table></div>")
    return "\n".join(out), idx


def render_blockquote(raw_lines: list[str]) -> str:
    text = " ".join(line.strip()[1:].strip() for line in raw_lines).strip()
    lower = text.lower()
    text = re.sub(r"^\*\*(modo tutor|nota de test)\.?\*\*\s*", "", text, flags=re.I)
    if "modo tutor" in lower[:40]:
        return f"<aside class=\"tutor\"><strong>Modo tutor</strong><p>{inline_html(text)}</p></aside>"
    if "nota de test" in lower[:50]:
        return f"<aside class=\"test-note\"><strong>Nota de test</strong><p>{inline_html(text)}</p></aside>"
    return f"<blockquote><p>{inline_html(text)}</p></blockquote>"


def render_image(line: str) -> str | None:
    match = re.fullmatch(r"!\[([^\]]*)\]\(([^)]+)\)", line.strip())
    if not match:
        return None
    alt = html.escape(match.group(1).strip(), quote=True)
    src = html.escape(match.group(2).strip(), quote=True)
    caption = alt or "Visual de estudio"
    return (
        "<figure class=\"visual-card\"><div class=\"diagram-scroll\">"
        f"<img src=\"{src}\" alt=\"{alt}\" loading=\"lazy\">"
        "</div><input class=\"diagram-slider\" type=\"range\" min=\"0\" value=\"0\" "
        "aria-label=\"Desplazar esquema horizontalmente\">"
        f"<figcaption>{html.escape(caption, quote=False)}</figcaption></figure>"
    )


def render_markdown(markdown_text: str) -> tuple[str, list[tuple[str, str]], str]:
    lines = markdown_text.splitlines()
    sections: list[str] = []
    nav: list[tuple[str, str]] = []
    first_h1 = ""
    current: list[str] = []

    def ensure_section() -> None:
        if not current:
            current.append('<section id="contenido" class="topic-section">')
            current.append("<h2>Contenido</h2>")
            nav.append(("contenido", "Contenido"))

    def close_section() -> None:
        nonlocal current
        if current:
            current.append("</section>")
            sections.append("\n".join(current))
            current = []

    idx = 0
    while idx < len(lines):
        line = lines[idx]
        stripped = line.strip()
        if not stripped:
            idx += 1
            continue
        if stripped.startswith("# "):
            if not first_h1:
                first_h1 = plain_heading(stripped)
            idx += 1
            continue
        if stripped.startswith("## "):
            close_section()
            title, section_id = split_heading_id(stripped[3:])
            current = [f'<section id="{html.escape(section_id, quote=True)}" class="topic-section">']
            current.append(f"<h2>{inline_html(title)}</h2>")
            nav.append((section_id, title))
            idx += 1
            continue
        ensure_section()
        if stripped.startswith("### "):
            title, section_id = split_heading_id(stripped[4:])
            current.append(f"<h3 id=\"{html.escape(section_id, quote=True)}\">{inline_html(title)}</h3>")
            idx += 1
            continue
        if stripped.startswith(">"):
            block: list[str] = []
            while idx < len(lines) and lines[idx].strip().startswith(">"):
                block.append(lines[idx])
                idx += 1
            current.append(render_blockquote(block))
            continue
        if stripped.startswith("|"):
            table, idx = parse_table(lines, idx)
            current.append(table)
            continue
        image = render_image(stripped)
        if image is not None:
            current.append(image)
            idx += 1
            continue
        if re.match(r"^[-*]\s+", stripped):
            items: list[str] = []
            while idx < len(lines) and re.match(r"^[-*]\s+", lines[idx].strip()):
                items.append(re.sub(r"^[-*]\s+", "", lines[idx].strip()))
                idx += 1
            current.append("<ul>" + "".join(f"<li>{inline_html(item)}</li>" for item in items) + "</ul>")
            continue
        if re.match(r"^\d+\.\s+", stripped):
            items = []
            while idx < len(lines) and re.match(r"^\d+\.\s+", lines[idx].strip()):
                items.append(re.sub(r"^\d+\.\s+", "", lines[idx].strip()))
                idx += 1
            current.append("<ol>" + "".join(f"<li>{inline_html(item)}</li>" for item in items) + "</ol>")
            continue
        if stripped.startswith("<") and stripped.endswith(">"):
            current.append(stripped)
            idx += 1
            continue
        para = [stripped]
        idx += 1
        while idx < len(lines):
            nxt = lines[idx].strip()
            if (
                not nxt
                or nxt.startswith("#")
                or nxt.startswith(">")
                or nxt.startswith("|")
                or re.match(r"^[-*]\s+", nxt)
                or re.match(r"^\d+\.\s+", nxt)
                or render_image(nxt) is not None
            ):
                break
            para.append(nxt)
            idx += 1
        current.append(f"<p>{inline_html(' '.join(para))}</p>")
    close_section()
    return "\n".join(sections), nav, first_h1


class StrictHTMLParser(HTMLParser):
    pass


def validate_html(output: Path, base_dir: Path, html_text: str) -> list[str]:
    errors: list[str] = []
    try:
        StrictHTMLParser().feed(html_text)
    except Exception as exc:  # pragma: no cover - HTMLParser rarely raises.
        errors.append(f"html_parse_error={exc}")
    for match in re.finditer(r"\b(?:src|href)=\"([^\"]+)\"", html_text):
        ref = match.group(1).strip()
        if ref.startswith("#") or ref.startswith("mailto:"):
            continue
        if ref.startswith(("http://", "https://", "file:")):
            errors.append(f"external_or_file_ref={ref}")
            continue
        candidate = (base_dir / ref).resolve()
        try:
            candidate.relative_to(base_dir.resolve())
        except ValueError:
            errors.append(f"ref_outside_base={ref}")
            continue
        if not candidate.exists():
            errors.append(f"missing_ref={ref}")
    if "first-reading-on" not in html_text:
        errors.append("missing_first_reading_default")
    output.write_text(html_text, encoding="utf-8")
    return errors


def load_manifest(path: Path | None) -> dict[str, str]:
    if path is None:
        return {}
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError("manifest must be a JSON object")
    return {str(k): "" if v is None else str(v) for k, v in data.items()}


def render(args: argparse.Namespace) -> int:
    markdown_path = Path(args.markdown)
    output = Path(args.output)
    base_dir = Path(args.base_dir) if args.base_dir else output.parent
    manifest = load_manifest(Path(args.manifest) if args.manifest else None)
    content, nav, inferred_title = render_markdown(markdown_path.read_text(encoding="utf-8"))
    title = args.title or manifest.get("title") or inferred_title or "Tema OPES"
    topic_key = args.topic_key or manifest.get("topic_key") or slugify(title)
    eyebrow = args.eyebrow or manifest.get("eyebrow") or "OPES"
    subtitle = args.subtitle or manifest.get("subtitle") or "Tema de estudio con primera lectura, modo tutor, notas de test y visuales locales."
    hero_image = args.hero_image or manifest.get("hero_image", "")
    hero_alt = args.hero_alt or manifest.get("hero_alt") or title
    hero_media = ""
    if hero_image:
        hero_media = (
            '<div class="hero-media">'
            f'<img src="{html.escape(hero_image, quote=True)}" alt="{html.escape(hero_alt, quote=True)}">'
            "</div>"
        )
    nav_links = "\n".join(
        f'<a href="#{html.escape(section_id, quote=True)}">{html.escape(label, quote=False)}</a>'
        for section_id, label in nav
    )
    template_path = Path(args.template) if args.template else DEFAULT_TEMPLATE
    template = Template(template_path.read_text(encoding="utf-8"))
    html_text = template.safe_substitute(
        title=html.escape(title, quote=False),
        page_title=html.escape(title, quote=False),
        topic_key=html.escape(topic_key, quote=True),
        body_class="first-reading-on",
        hero_media=hero_media,
        eyebrow=html.escape(eyebrow, quote=False),
        subtitle=html.escape(subtitle, quote=False),
        nav_links=nav_links,
        content_sections=content,
        generated_at=_dt.date.today().isoformat(),
    )
    output.parent.mkdir(parents=True, exist_ok=True)
    errors = validate_html(output, base_dir, html_text)
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print(f"rendered={output}")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Render OPES topic HTML with template v1.")
    parser.add_argument("--markdown", required=True, help="Markdown source for the topic.")
    parser.add_argument("--output", required=True, help="HTML output path.")
    parser.add_argument("--base-dir", help="Directory used to validate relative assets.")
    parser.add_argument("--manifest", help="Optional JSON metadata manifest.")
    parser.add_argument("--template", help="Optional template path.")
    parser.add_argument("--title", help="Override topic title.")
    parser.add_argument("--topic-key", help="Override topic key.")
    parser.add_argument("--eyebrow", help="Hero eyebrow text.")
    parser.add_argument("--subtitle", help="Hero subtitle text.")
    parser.add_argument("--hero-image", help="Optional local hero image relative to base-dir.")
    parser.add_argument("--hero-alt", help="Hero image alt text.")
    return render(parser.parse_args(argv))


if __name__ == "__main__":
    raise SystemExit(main())
