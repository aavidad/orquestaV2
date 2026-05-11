package orquestaweb

import (
	"html/template"
	"net/http"
)

type appChangePageDataV0 struct {
	Locale string
	VM     WebAppChangeViewModelV0
	Text   map[string]string
}

func appChangePageV0(locale string, vm WebAppChangeViewModelV0) appChangePageDataV0 {
	if locale == "" {
		locale = "es"
	}
	keys := []string{"title", "submit", "run_ref", "app_ref", "change_ref", "locale", "user_intent", "target_area", "criteria", "write_set", "current_refs", "constraints", "accepted", "initial", "error", "help_run", "help_change", "help_intent", "help_write_set"}
	text := map[string]string{}
	for _, key := range keys {
		text[key] = appChangeTextV0(locale, key)
	}
	return appChangePageDataV0{Locale: locale, VM: vm, Text: text}
}

func writeAppChangeHTMLV0(w http.ResponseWriter, status int, page appChangePageDataV0) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = appChangeHTMLTemplateV0.Execute(w, page)
}

var appChangeHTMLTemplateV0 = template.Must(template.New("app_change_html_v0").Parse(`<!doctype html>
<html lang="{{.Locale}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{index .Text "title"}}</title>
  <style>
    body{font-family:system-ui,sans-serif;margin:0;background:#f7f8fb;color:#172033}
    main{max-width:960px;margin:0 auto;padding:24px}
    form{display:grid;gap:14px}
    fieldset,.panel{border:1px solid #d8dee8;border-radius:8px;background:#fff;padding:16px}
    .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}
    label{display:grid;gap:6px;font-size:.94rem}
    .hint{margin:0;color:#566276;font-size:.88rem}
    input,select,textarea{font:inherit;padding:9px;border:1px solid #b7c1ce;border-radius:6px}
    textarea{min-height:110px}
    button{width:max-content;padding:10px 14px;border:0;border-radius:6px;background:#0f766e;color:#fff;font-weight:700}
    code{background:#eef2f7;padding:2px 4px;border-radius:4px}
  </style>
</head>
<body><main>
  <h1>{{index .Text "title"}}</h1>
  <form method="post" action="/app-change">
    <fieldset><div class="grid">
      <label title="{{index .Text "help_run"}}">{{index .Text "run_ref"}}<input name="run_ref" required></label>
      <label>{{index .Text "app_ref"}}<input name="app_ref"></label>
      <label title="{{index .Text "help_change"}}">{{index .Text "change_ref"}}<input name="change_ref" required></label>
      <label>{{index .Text "locale"}}<input name="locale" value="{{.Locale}}"></label>
    </div></fieldset>
    <fieldset>
      <label title="{{index .Text "help_intent"}}">{{index .Text "user_intent"}}<textarea name="user_intent" required aria-describedby="app-change-intent-help"></textarea></label>
      <p id="app-change-intent-help" class="hint">{{index .Text "help_intent"}}</p>
      <div class="grid">
        <label>{{index .Text "target_area"}}<select name="target_area"><option value="web">web</option><option value="api">api</option><option value="docs">docs</option><option value="quality">quality</option></select></label>
        <label>{{index .Text "criteria"}}<input name="acceptance_criteria"></label>
        <label title="{{index .Text "help_write_set"}}">{{index .Text "write_set"}}<input name="allowed_write_set"></label>
        <label>{{index .Text "current_refs"}}<input name="current_state_refs"></label>
        <label>{{index .Text "constraints"}}<input name="constraints"></label>
      </div>
    </fieldset>
    <button type="submit">{{index .Text "submit"}}</button>
  </form>
  <section class="panel">
    {{if eq .VM.Estado "accepted"}}<p>{{index .Text "accepted"}} <code>{{.VM.DirectorQuestionRef}}</code></p>{{else if eq .VM.Estado "error"}}<p>{{index .Text "error"}}</p>{{else}}<p>{{index .Text "initial"}}</p>{{end}}
    {{if .VM.RunRef}}<p><code>{{.VM.RunRef}}</code> <code>{{.VM.ChangeRef}}</code></p>{{end}}
    {{if .VM.ErroresPublicos}}<ul>{{range .VM.ErroresPublicos}}<li><code>{{.Code}}</code> {{.Field}}</li>{{end}}</ul>{{end}}
  </section>
</main></body></html>`))
