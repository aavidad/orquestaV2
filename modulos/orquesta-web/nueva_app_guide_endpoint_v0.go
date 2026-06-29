package orquestaweb

import (
	_ "embed"
	"html/template"
	"net/http"
	"strings"
)

const WebNuevaAppGuideEndpointV0 = "/nueva-app/guia"

//go:embed docs/guia_nueva_app_opciones_2026-06-25.md
var nuevaAppGuideMarkdownV0 string

type NuevaAppGuideWebEndpointV0 struct{}

func NewNuevaAppGuideWebEndpointV0() NuevaAppGuideWebEndpointV0 {
	return NuevaAppGuideWebEndpointV0{}
}

func (endpoint NuevaAppGuideWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != WebNuevaAppGuideEndpointV0 {
		http.NotFound(w, r)
		return
	}
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	writeWebHTMLStringResponseV0(w, http.StatusOK, nuevaAppGuideHTMLV0(nuevaAppGuideMarkdownV0), "es")
}

func nuevaAppGuideHTMLV0(markdown string) string {
	content := nuevaAppGuideMarkdownToHTMLV0(markdown)
	return `<!doctype html>
	<html lang="es">
	<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Guia nueva app - Orquesta</title>
  <style>
	    :root{color-scheme:light;--bg:#f6f8f5;--panel:#ffffff;--line:#d8e2d8;--text:#17211d;--muted:#5f7068;--brand:#245b44}
	    *{box-sizing:border-box}
	    body{margin:0;background:var(--bg);color:var(--text);font:15px/1.6 ui-sans-serif,system-ui,sans-serif}
	    main{max-width:1080px;margin:0 auto;padding:28px}
	    nav{display:flex;gap:10px;flex-wrap:wrap;margin-bottom:18px}
	    a{color:var(--brand);font-weight:800}
	    h1{margin:0 0 8px;font-size:32px}
	    article{margin-top:20px;border:1px solid var(--line);background:var(--panel);border-radius:10px;padding:24px}
	    article h2{margin:24px 0 10px;font-size:24px;line-height:1.2}
	    article h3{margin:20px 0 8px;font-size:18px}
	    article h4{margin:16px 0 6px;font-size:15px}
	    p{margin:0 0 16px;color:var(--muted)}
	    article p{color:var(--text)}
		    ul,ol{margin:0 0 16px 22px;padding:0}
		    li{margin:4px 0}
		    table{width:100%;border-collapse:collapse;margin:0 0 16px}
		    th,td{border:1px solid var(--line);padding:8px 10px;text-align:left;vertical-align:top}
		    th{background:#edf4ec;color:var(--text)}
		    code{background:#edf4ec;border:1px solid #d8e2d8;border-radius:5px;padding:1px 4px}
		    pre{white-space:pre-wrap;overflow-wrap:anywhere;margin:0 0 16px;border:1px solid var(--line);background:#f8fbf6;border-radius:8px;padding:14px}
		  </style>
	</head>
	<body>
<main>
	  <nav><a href="/nueva-app">Volver al wizard</a><a href="/">Inicio</a><a href="/ops">Ops</a></nav>
	  <h1>Guia de opciones de nueva app</h1>
	  <p>Documento completo de uso y contrato visible para el wizard.</p>
	  <article>` + string(content) + `</article>
	</main>
	</body>
	</html>`
}

func nuevaAppGuideMarkdownToHTMLV0(markdown string) template.HTML {
	var builder strings.Builder
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	inParagraph := false
	inUL := false
	inOL := false
	inCode := false
	inTable := false
	closeParagraph := func() {
		if inParagraph {
			builder.WriteString("</p>\n")
			inParagraph = false
		}
	}
	closeTable := func() {
		if inTable {
			builder.WriteString("</tbody>\n</table>\n")
			inTable = false
		}
	}
	closeLists := func() {
		if inUL {
			builder.WriteString("</ul>\n")
			inUL = false
		}
		if inOL {
			builder.WriteString("</ol>\n")
			inOL = false
		}
	}
	for index := 0; index < len(lines); index++ {
		rawLine := lines[index]
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "```") {
			closeParagraph()
			closeLists()
			closeTable()
			if inCode {
				builder.WriteString("</code></pre>\n")
				inCode = false
			} else {
				builder.WriteString("<pre><code>")
				inCode = true
			}
			continue
		}
		if inCode {
			builder.WriteString(template.HTMLEscapeString(rawLine))
			builder.WriteByte('\n')
			continue
		}
		if line == "" {
			closeParagraph()
			closeLists()
			closeTable()
			continue
		}
		if headerCells := nuevaAppGuideTableCellsV0(line); len(headerCells) > 0 &&
			index+1 < len(lines) &&
			nuevaAppGuideTableSeparatorV0(strings.TrimSpace(lines[index+1])) {
			closeParagraph()
			closeLists()
			closeTable()
			builder.WriteString("<table>\n<thead><tr>")
			for _, cell := range headerCells {
				builder.WriteString("<th>")
				builder.WriteString(nuevaAppGuideInlineHTMLV0(cell))
				builder.WriteString("</th>")
			}
			builder.WriteString("</tr></thead>\n<tbody>\n")
			inTable = true
			index++
			continue
		}
		if inTable {
			if cells := nuevaAppGuideTableCellsV0(line); len(cells) > 0 {
				builder.WriteString("<tr>")
				for _, cell := range cells {
					builder.WriteString("<td>")
					builder.WriteString(nuevaAppGuideInlineHTMLV0(cell))
					builder.WriteString("</td>")
				}
				builder.WriteString("</tr>\n")
				continue
			}
			closeTable()
		}
		if strings.HasPrefix(line, "#") {
			closeParagraph()
			closeLists()
			level, text := nuevaAppGuideHeadingV0(line)
			builder.WriteString("<h")
			builder.WriteString(level)
			builder.WriteString(">")
			builder.WriteString(nuevaAppGuideInlineHTMLV0(text))
			builder.WriteString("</h")
			builder.WriteString(level)
			builder.WriteString(">\n")
			continue
		}
		if strings.HasPrefix(line, "- ") {
			closeParagraph()
			closeTable()
			if inOL {
				builder.WriteString("</ol>\n")
				inOL = false
			}
			if !inUL {
				builder.WriteString("<ul>\n")
				inUL = true
			}
			builder.WriteString("<li>")
			builder.WriteString(nuevaAppGuideInlineHTMLV0(strings.TrimSpace(strings.TrimPrefix(line, "- "))))
			builder.WriteString("</li>\n")
			continue
		}
		if text, ok := nuevaAppGuideOrderedListTextV0(line); ok {
			closeParagraph()
			closeTable()
			if inUL {
				builder.WriteString("</ul>\n")
				inUL = false
			}
			if !inOL {
				builder.WriteString("<ol>\n")
				inOL = true
			}
			builder.WriteString("<li>")
			builder.WriteString(nuevaAppGuideInlineHTMLV0(text))
			builder.WriteString("</li>\n")
			continue
		}
		closeLists()
		if !inParagraph {
			builder.WriteString("<p>")
			inParagraph = true
		} else {
			builder.WriteByte(' ')
		}
		builder.WriteString(nuevaAppGuideInlineHTMLV0(line))
	}
	closeParagraph()
	closeLists()
	closeTable()
	if inCode {
		builder.WriteString("</code></pre>\n")
	}
	return template.HTML(builder.String())
}

func nuevaAppGuideTableCellsV0(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "|") {
		return nil
	}
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	rawCells := strings.Split(line, "|")
	if len(rawCells) < 2 {
		return nil
	}
	cells := make([]string, 0, len(rawCells))
	for _, rawCell := range rawCells {
		cells = append(cells, strings.TrimSpace(rawCell))
	}
	return cells
}

func nuevaAppGuideTableSeparatorV0(line string) bool {
	cells := nuevaAppGuideTableCellsV0(line)
	if len(cells) < 2 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if !strings.Contains(cell, "-") {
			return false
		}
		withoutDashes := strings.ReplaceAll(cell, "-", "")
		withoutDashes = strings.ReplaceAll(withoutDashes, ":", "")
		if strings.TrimSpace(withoutDashes) != "" {
			return false
		}
	}
	return true
}

func nuevaAppGuideHeadingV0(line string) (string, string) {
	count := 0
	for count < len(line) && line[count] == '#' {
		count++
	}
	if count < 1 {
		count = 1
	}
	level := count + 1
	if level > 6 {
		level = 6
	}
	return string(rune('0' + level)), strings.TrimSpace(line[count:])
}

func nuevaAppGuideOrderedListTextV0(line string) (string, bool) {
	dot := strings.Index(line, ". ")
	if dot <= 0 {
		return "", false
	}
	for _, char := range line[:dot] {
		if char < '0' || char > '9' {
			return "", false
		}
	}
	return strings.TrimSpace(line[dot+2:]), true
}

func nuevaAppGuideInlineHTMLV0(text string) string {
	parts := strings.Split(text, "`")
	if len(parts) < 3 {
		return template.HTMLEscapeString(text)
	}
	var builder strings.Builder
	for index, part := range parts {
		escaped := template.HTMLEscapeString(part)
		if index%2 == 1 {
			builder.WriteString("<code>")
			builder.WriteString(escaped)
			builder.WriteString("</code>")
		} else {
			builder.WriteString(escaped)
		}
	}
	return builder.String()
}
