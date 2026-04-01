package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type webConfigEntry struct {
	Clave string
	Valor string
}

type webConfigQuickField struct {
	Clave       string
	TituloKey   string
	SubtitleKey string
	Placeholder string
	Valor       string
}

type webConfigQuickSection struct {
	TitleKey    string
	SubtitleKey string
	PresetName  string
	PresetKey   string
	Fields      []webConfigQuickField
}

type webConfigData struct {
	Entries       []webConfigEntry
	QuickSections []webConfigQuickSection
	Msg           string
	Err           string
}

func webOpenClawDefaultEndpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d/api/mcp", defaultServePort)
}

func webConfigQuickSections(config map[string]string) []webConfigQuickSection {
	return []webConfigQuickSection{
		{
			TitleKey:    "config.quick.openclaw.title",
			SubtitleKey: "config.quick.openclaw.subtitle",
			PresetName:  "openclaw_server_first",
			PresetKey:   "config.quick.openclaw.apply",
			Fields: []webConfigQuickField{
				{
					Clave:       "integration_openclaw_enabled",
					TituloKey:   "config.quick.openclaw.enabled.title",
					SubtitleKey: "config.quick.openclaw.enabled.subtitle",
					Placeholder: "false",
					Valor:       config["integration_openclaw_enabled"],
				},
				{
					Clave:       "integration_openclaw_transport",
					TituloKey:   "config.quick.openclaw.transport.title",
					SubtitleKey: "config.quick.openclaw.transport.subtitle",
					Placeholder: "mcp_http",
					Valor:       config["integration_openclaw_transport"],
				},
				{
					Clave:       "integration_openclaw_endpoint",
					TituloKey:   "config.quick.openclaw.endpoint.title",
					SubtitleKey: "config.quick.openclaw.endpoint.subtitle",
					Placeholder: webOpenClawDefaultEndpoint(),
					Valor:       config["integration_openclaw_endpoint"],
				},
				{
					Clave:       "integration_openclaw_workspace_mode",
					TituloKey:   "config.quick.openclaw.workspace.title",
					SubtitleKey: "config.quick.openclaw.workspace.subtitle",
					Placeholder: "worktree",
					Valor:       config["integration_openclaw_workspace_mode"],
				},
				{
					Clave:       "integration_openclaw_agent_prefix",
					TituloKey:   "config.quick.openclaw.agent_prefix.title",
					SubtitleKey: "config.quick.openclaw.agent_prefix.subtitle",
					Placeholder: "OpenClaw-",
					Valor:       config["integration_openclaw_agent_prefix"],
				},
				{
					Clave:       "integration_openclaw_require_identity",
					TituloKey:   "config.quick.openclaw.require_identity.title",
					SubtitleKey: "config.quick.openclaw.require_identity.subtitle",
					Placeholder: "true",
					Valor:       config["integration_openclaw_require_identity"],
				},
			},
		},
	}
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
		Entries:       entries,
		QuickSections: webConfigQuickSections(resp.Config),
		Msg:           r.URL.Query().Get("ok"),
		Err:           r.URL.Query().Get("err"),
	})
}

func webHandlerConfigGuardar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if preset := strings.TrimSpace(r.FormValue("preset")); preset != "" {
		if err := webGuardarConfigPreset(r, preset); err != nil {
			http.Redirect(w, r, webConfigRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, webConfigRedirectURL(r, "ok", webTranslateRequestf(r, "config.flash.openclaw_preset_saved")), http.StatusSeeOther)
		return
	}
	clave := strings.TrimSpace(r.FormValue("clave"))
	valor := r.FormValue("valor")
	if clave == "" {
		http.Redirect(w, r, webConfigRedirectURL(r, "err", webTranslateRequestf(r, "config.flash.key_required")), http.StatusSeeOther)
		return
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/config", apiConfigSetRequest{Clave: clave, Valor: valor}, nil); err != nil {
		http.Redirect(w, r, webConfigRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webConfigRedirectURL(r, "ok", webTranslateRequestf(r, "config.flash.saved", clave)), http.StatusSeeOther)
}

func webGuardarConfigPreset(r *http.Request, preset string) error {
	var valores map[string]string
	switch preset {
	case "openclaw_server_first":
		valores = map[string]string{
			"integration_openclaw_enabled":          "true",
			"integration_openclaw_transport":        "mcp_http",
			"integration_openclaw_endpoint":         webOpenClawDefaultEndpoint(),
			"integration_openclaw_workspace_mode":   "worktree",
			"integration_openclaw_agent_prefix":     "OpenClaw-",
			"integration_openclaw_require_identity": "true",
		}
	default:
		return fmt.Errorf("%s", webTranslateRequestf(r, "config.flash.unknown_preset", preset))
	}
	for clave, valor := range valores {
		if err := webInvocarAPIJSON(http.MethodPost, "/api/config", apiConfigSetRequest{Clave: clave, Valor: valor}, nil); err != nil {
			return err
		}
	}
	return nil
}

func webConfigRedirectURL(r *http.Request, flashKey, flashMsg string) string {
	params := url.Values{}
	params.Set(flashKey, flashMsg)
	params.Set("lang", resolveWebRequestLang(r))
	return "/config?" + params.Encode()
}

const webTplConfig = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "config.title"}}</h2>
  <p style="color:#64748b">{{tr "config.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  {{range .QuickSections}}
  <section style="margin:1rem 0 1.2rem 0;padding:1rem;border:1px solid #e2e8f0;border-radius:.6rem;background:#f8fafc">
    <div style="display:flex;gap:.75rem;justify-content:space-between;align-items:flex-start;flex-wrap:wrap">
      <div>
        <h3 style="margin:0">{{tr .TitleKey}}</h3>
        <p style="color:#64748b">{{tr .SubtitleKey}}</p>
      </div>
      {{if .PresetName}}
      <form method="post" action="/config?lang={{lang}}" style="margin:0">
        <input type="hidden" name="preset" value="{{.PresetName}}">
        <button type="submit">{{tr .PresetKey}}</button>
      </form>
      {{end}}
    </div>
    <div class="grid2">
      {{range .Fields}}
      <form method="post" action="/config?lang={{lang}}" style="display:grid;gap:.45rem;padding:.8rem;border:1px solid #cbd5e1;border-radius:.5rem;background:white">
        <input type="hidden" name="clave" value="{{.Clave}}">
        <div>
          <strong>{{tr .TituloKey}}</strong><br>
          <small style="color:#64748b">{{tr .SubtitleKey}}</small>
        </div>
        <label>{{tr "config.value"}} <input name="valor" value="{{.Valor}}" placeholder="{{.Placeholder}}"></label>
        <button type="submit">{{tr "common.save"}}</button>
      </form>
      {{end}}
    </div>
  </section>
  {{end}}
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
  <form method="post" action="/config?lang={{lang}}" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "config.key"}} <input name="clave" required></label>
      <label>{{tr "config.value"}} <input name="valor"></label>
    </div>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>
{{end}}`
