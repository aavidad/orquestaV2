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
	content := template.HTMLEscapeString(strings.TrimSpace(markdown))
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
    p{margin:0 0 16px;color:var(--muted)}
    pre{white-space:pre-wrap;overflow-wrap:anywhere;margin:0;border:1px solid var(--line);background:var(--panel);border-radius:10px;padding:18px}
  </style>
</head>
<body>
<main>
  <nav><a href="/nueva-app">Volver al wizard</a><a href="/">Inicio</a><a href="/ops">Ops</a></nav>
  <h1>Guia de opciones de nueva app</h1>
  <p>Documento completo de uso y contrato visible para el wizard.</p>
  <pre>` + content + `</pre>
</main>
</body>
</html>`
}
