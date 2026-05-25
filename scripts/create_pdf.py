#!/usr/bin/env python3
"""
Create a simple PDF from text, Markdown-like text, HTML-ish text or JSON input.

The script has no third-party dependencies so it works in the remote Hermes
server without installing PDF packages. It is intended for operational PDFs
that Berserk can later attach to OWA/EWS emails.
"""

from __future__ import annotations

import argparse
import html
import json
import re
import sys
import textwrap
from pathlib import Path
from typing import Any


A4_WIDTH = 595.28
A4_HEIGHT = 841.89
MARGIN_X = 56.0
MARGIN_TOP = 64.0
MARGIN_BOTTOM = 58.0
LINE_GAP = 1.25


def decode_text(value: str) -> str:
    value = value.replace("\r\n", "\n").replace("\r", "\n")
    value = html.unescape(value)
    value = re.sub(r"<\s*br\s*/?\s*>", "\n", value, flags=re.I)
    value = re.sub(r"</\s*p\s*>", "\n\n", value, flags=re.I)
    value = re.sub(r"</\s*(h[1-6]|li|div|section|article)\s*>", "\n", value, flags=re.I)
    value = re.sub(r"<[^>]+>", "", value)
    value = re.sub(r"[ \t]+\n", "\n", value)
    return value.strip()


def pdf_literal(text: str) -> bytes:
    raw = text.encode("cp1252", "replace")
    out = bytearray()
    for b in raw:
        if b in (0x28, 0x29, 0x5C):
            out.extend(b"\\" + bytes([b]))
        elif b < 32 or b >= 127:
            out.extend(f"\\{b:03o}".encode("ascii"))
        else:
            out.append(b)
    return b"(" + bytes(out) + b")"


def estimate_width(text: str, font_size: float, bold: bool = False) -> float:
    factor = 0.54 if not bold else 0.58
    width = 0.0
    for char in text:
        if char in "il.,:;!|":
            width += font_size * 0.27
        elif char in "mwMW@#%&":
            width += font_size * 0.82
        elif char == " ":
            width += font_size * 0.30
        else:
            width += font_size * factor
    return width


def wrap_line(text: str, font_size: float, max_width: float, bold: bool = False) -> list[str]:
    text = re.sub(r"\s+", " ", text).strip()
    if not text:
        return [""]
    words = text.split(" ")
    lines: list[str] = []
    current = ""
    for word in words:
        candidate = word if not current else current + " " + word
        if estimate_width(candidate, font_size, bold) <= max_width:
            current = candidate
            continue
        if current:
            lines.append(current)
            current = word
        else:
            chunks = textwrap.wrap(word, max(8, int(max_width / (font_size * 0.55))))
            lines.extend(chunks[:-1])
            current = chunks[-1] if chunks else ""
    if current:
        lines.append(current)
    return lines or [""]


class PDFDocument:
    def __init__(self, title: str = "") -> None:
        self.title = title
        self.pages: list[list[bytes]] = []
        self.current: list[bytes] = []
        self.y = A4_HEIGHT - MARGIN_TOP
        self.add_page()

    def add_page(self) -> None:
        if self.current:
            self.pages.append(self.current)
        self.current = []
        self.y = A4_HEIGHT - MARGIN_TOP

    def ensure_space(self, height: float) -> None:
        if self.y - height < MARGIN_BOTTOM:
            self.add_page()

    def text_line(self, text: str, x: float, font: str, size: float) -> None:
        self.current.append(b"BT")
        self.current.append(f"/{font} {size:.2f} Tf".encode("ascii"))
        self.current.append(f"{x:.2f} {self.y:.2f} Td".encode("ascii"))
        self.current.append(pdf_literal(text) + b" Tj")
        self.current.append(b"ET")

    def add_wrapped(self, text: str, size: float = 11.0, bold: bool = False, indent: float = 0.0) -> None:
        font = "F2" if bold else "F1"
        max_width = A4_WIDTH - (MARGIN_X * 2) - indent
        for line in wrap_line(text, size, max_width, bold):
            line_height = size * LINE_GAP
            self.ensure_space(line_height)
            if line:
                self.text_line(line, MARGIN_X + indent, font, size)
            self.y -= line_height

    def add_gap(self, points: float) -> None:
        self.ensure_space(points)
        self.y -= points

    def finish_pages(self) -> None:
        if self.current:
            self.pages.append(self.current)
            self.current = []

    def render(self) -> bytes:
        self.finish_pages()
        objects: list[bytes] = []
        page_count = len(self.pages)
        catalog_id = 1
        pages_id = 2
        font_regular_id = 3
        font_bold_id = 4
        first_page_id = 5

        objects.append(b"<< /Type /Catalog /Pages 2 0 R >>")
        page_refs = []
        content_objects: list[bytes] = []
        for index, lines in enumerate(self.pages):
            page_id = first_page_id + index * 2
            content_id = page_id + 1
            page_refs.append(f"{page_id} 0 R".encode("ascii"))
            stream = b"\n".join(lines)
            content_objects.append(
                b"<< /Length " + str(len(stream)).encode("ascii") + b" >>\nstream\n" + stream + b"\nendstream"
            )

        kids = b" ".join(page_refs)
        objects.append(b"<< /Type /Pages /Kids [ " + kids + b" ] /Count " + str(page_count).encode("ascii") + b" >>")
        objects.append(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
        objects.append(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
        for index in range(page_count):
            content_id = first_page_id + index * 2 + 1
            objects.append(
                b"<< /Type /Page /Parent "
                + str(pages_id).encode("ascii")
                + b" 0 R /MediaBox [0 0 595.28 841.89] /Resources << /Font << /F1 "
                + str(font_regular_id).encode("ascii")
                + b" 0 R /F2 "
                + str(font_bold_id).encode("ascii")
                + b" 0 R >> >> /Contents "
                + str(content_id).encode("ascii")
                + b" 0 R >>"
            )
            objects.append(content_objects[index])

        output = bytearray(b"%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
        offsets = [0]
        for obj_id, obj in enumerate(objects, start=1):
            offsets.append(len(output))
            output.extend(f"{obj_id} 0 obj\n".encode("ascii"))
            output.extend(obj)
            output.extend(b"\nendobj\n")
        xref_pos = len(output)
        output.extend(f"xref\n0 {len(objects) + 1}\n".encode("ascii"))
        output.extend(b"0000000000 65535 f \n")
        for offset in offsets[1:]:
            output.extend(f"{offset:010d} 00000 n \n".encode("ascii"))
        output.extend(
            b"trailer\n<< /Size "
            + str(len(objects) + 1).encode("ascii")
            + b" /Root "
            + str(catalog_id).encode("ascii")
            + b" 0 R >>\nstartxref\n"
            + str(xref_pos).encode("ascii")
            + b"\n%%EOF\n"
        )
        return bytes(output)


def add_markdownish_text(pdf: PDFDocument, text: str, title: str = "") -> None:
    if title:
        pdf.add_wrapped(title, size=18, bold=True)
        pdf.add_gap(14)
    for raw in text.splitlines():
        line = raw.rstrip()
        stripped = line.strip()
        if not stripped:
            pdf.add_gap(7)
            continue
        heading = re.match(r"^(#{1,6})\s+(.+)$", stripped)
        if heading:
            level = len(heading.group(1))
            size = max(12, 18 - level)
            pdf.add_gap(5)
            pdf.add_wrapped(heading.group(2), size=size, bold=True)
            pdf.add_gap(4)
            continue
        bullet = re.match(r"^([-*]|\d+[.)])\s+(.+)$", stripped)
        if bullet:
            marker = bullet.group(1)
            prefix = "\u2022" if marker in {"-", "*"} else marker
            pdf.add_wrapped(prefix + " " + bullet.group(2), size=10.8, indent=14)
            continue
        pdf.add_wrapped(stripped, size=11)


def read_payload(args: argparse.Namespace) -> tuple[str, str]:
    if args.json_input:
        raw = Path(args.json_input).read_text(encoding="utf-8") if args.json_input != "-" else sys.stdin.read()
        payload = json.loads(raw)
        if not isinstance(payload, dict):
            raise SystemExit("JSON input must be an object")
        title = str(payload.get("title") or args.title or "").strip()
        text = str(payload.get("text") or payload.get("body") or "").strip()
        if not text:
            raise SystemExit("missing JSON text/body")
        return title, decode_text(text)
    if args.input:
        text = Path(args.input).read_text(encoding="utf-8")
    elif args.text is not None:
        text = args.text
    else:
        text = sys.stdin.read()
    text = decode_text(text)
    if not text:
        raise SystemExit("missing PDF text")
    return args.title or "", text


def main() -> int:
    parser = argparse.ArgumentParser(description="Create a simple PDF from text.")
    parser.add_argument("--input", help="Text/Markdown/HTML-ish input file.")
    parser.add_argument("--json-input", nargs="?", const="-", help="Read {title,text/body} JSON from file or stdin.")
    parser.add_argument("--text", help="Text content.")
    parser.add_argument("--title", default="")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    title, text = read_payload(args)
    output_path = Path(args.output).expanduser().resolve()
    output_path.parent.mkdir(parents=True, exist_ok=True)
    pdf = PDFDocument(title=title)
    add_markdownish_text(pdf, text, title)
    output_path.write_bytes(pdf.render())
    print(json.dumps({"status": "created", "output": str(output_path), "bytes": output_path.stat().st_size}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
