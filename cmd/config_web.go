package cmd

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type webConfigEntry struct {
	Clave string
	Valor string
}

type webConfigData struct {
	Entries []webConfigEntry
	Msg     string
	Err     string
}

func webHandlerConfig(w http.ResponseWriter, r *http.Request) {
	var resp apiConfigResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/config", nil, &resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entries := make([]webConfigEntry, 0, len(resp.Config))
	for k, v := range resp.Config {
		entries = append(entries, webConfigEntry{Clave: k, Valor: v})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Clave < entries[j].Clave })
	webRender(w, r, webTplLayout+webTplConfig, webConfigData{
		Entries: entries,
		Msg:     r.URL.Query().Get("ok"),
		Err:     r.URL.Query().Get("err"),
	})
}

func webHandlerConfigGuardar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clave := strings.TrimSpace(r.FormValue("clave"))
	valor := r.FormValue("valor")
	if clave == "" {
		http.Redirect(w, r, "/config?err="+url.QueryEscape("clave obligatoria"), http.StatusSeeOther)
		return
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/config", apiConfigSetRequest{Clave: clave, Valor: valor}, nil); err != nil {
		http.Redirect(w, r, "/config?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/config?ok="+url.QueryEscape(webTranslateRequestf(r, "config.flash.saved", clave)), http.StatusSeeOther)
}

const webTplConfig = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "config.title"}}</h2>
  <p style="color:#64748b">{{tr "config.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <table>
    <thead><tr><th>{{tr "config.key"}}</th><th>{{tr "config.value"}}</th></tr></thead>
    <tbody>
      {{range .Entries}}
      <tr><td><code>{{.Clave}}</code></td><td><code>{{.Valor}}</code></td></tr>
      {{else}}
      <tr><td colspan="2">{{tr "config.none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>

<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "config.edit_title"}}</h3>
  <form method="post" action="/config" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "config.key"}} <input name="clave" required></label>
      <label>{{tr "config.value"}} <input name="valor"></label>
    </div>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>
{{end}}`
