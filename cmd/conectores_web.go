/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/url"
	"strings"

	"orquesta/conectoresapp"
	"orquesta/db"
)

var conectoresService = conectoresapp.NewService(conectoresapp.Repository{})

type webConectoresData struct {
	Conectores []*db.Conector
	Msg        string
	Err        string
}

type webConectorDetalleData struct {
	Conector *db.Conector
	Msg      string
	Err      string
}

func webRouterConectores(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/conectores", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerConectorDetalle(w, r, parts[1])
	case len(parts) == 2 && r.Method == http.MethodPost:
		webHandlerConectorGuardar(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webHandlerConectores(w http.ResponseWriter, r *http.Request) {
	items, err := conectoresService.ListConnectors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, r, webTplLayout+webTplConectores, webConectoresData{
		Conectores: items,
		Msg:        r.URL.Query().Get("ok"),
		Err:        r.URL.Query().Get("err"),
	})
}

func webHandlerConectorDetalle(w http.ResponseWriter, r *http.Request, ref string) {
	item, err := conectoresService.GetConnector(ref)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplConectorDetalle, webConectorDetalleData{
		Conector: item,
		Msg:      r.URL.Query().Get("ok"),
		Err:      r.URL.Query().Get("err"),
	})
}

func webHandlerConectorGuardar(w http.ResponseWriter, r *http.Request, ref string) {
	_ = r.ParseForm()
	target := strings.TrimSpace(ref)
	if target == "nuevo" {
		target = ""
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	if target != "" && slug == "" {
		actual, err := conectoresService.GetConnector(target)
		if err == nil && actual != nil {
			slug = strings.TrimSpace(actual.Slug)
		}
	}
	if _, err := conectoresService.SaveConnector(conectoresapp.SaveConnectorInput{
		Slug:         slug,
		Nombre:       strings.TrimSpace(r.FormValue("nombre")),
		Transporte:   strings.TrimSpace(r.FormValue("transporte")),
		Comando:      strings.TrimSpace(r.FormValue("comando")),
		ArgsJSON:     strings.TrimSpace(r.FormValue("args_json")),
		EnvJSON:      strings.TrimSpace(r.FormValue("env_json")),
		MetadataJSON: strings.TrimSpace(r.FormValue("metadata_json")),
		Activo:       r.FormValue("activo") == "on" || r.FormValue("activo") == "1" || r.FormValue("activo") == "true",
	}); err != nil {
		redirect := "/conectores"
		if target != "" {
			redirect = "/conectores/" + url.PathEscape(target)
		}
		http.Redirect(w, r, redirect+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msg := webTranslateRequestf(r, "connectors.flash.saved")
	if slug != "" {
		http.Redirect(w, r, "/conectores/"+url.PathEscape(slug)+"?ok="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/conectores?ok="+url.QueryEscape(msg), http.StatusSeeOther)
}

const webTplConectores = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "connectors.title"}}</h2>
  <p style="color:#64748b">{{tr "connectors.subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <table>
    <thead><tr><th>ID</th><th>{{tr "common.slug"}}</th><th>{{tr "common.name"}}</th><th>{{tr "connectors.transport"}}</th><th>{{tr "common.active"}}</th><th>{{tr "connectors.command"}}</th><th></th></tr></thead>
    <tbody>
      {{range .Conectores}}
      <tr>
        <td>{{.ID}}</td>
        <td><code>{{.Slug}}</code></td>
        <td>{{.Nombre}}</td>
        <td>{{.Transporte}}</td>
        <td>{{if .Activo}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</td>
        <td><code>{{.Comando}}</code></td>
        <td><a href="/conectores/{{.Slug}}">{{tr "common.view"}}</a></td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>

<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "connectors.new_title"}}</h3>
  <form method="post" action="/conectores/nuevo" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "common.slug"}} <input name="slug" required></label>
      <label>{{tr "common.name"}} <input name="nombre" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "connectors.transport"}} <input name="transporte" value="cli"></label>
      <label>{{tr "connectors.command"}} <input name="comando"></label>
    </div>
    <div class="grid2">
      <label>{{tr "connectors.args_json"}} <textarea name="args_json" rows="4">[]</textarea></label>
      <label>{{tr "connectors.env_json"}} <textarea name="env_json" rows="4">{}</textarea></label>
    </div>
    <label>{{tr "connectors.metadata_json"}} <textarea name="metadata_json" rows="6">{}</textarea></label>
    <label><input type="checkbox" name="activo" checked> {{tr "common.active"}}</label>
    <button type="submit">{{tr "connectors.save"}}</button>
  </form>
</section>
{{end}}`

const webTplConectorDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/conectores">← {{tr "connectors.back"}}</a></p>
  <h2 style="margin:0">{{tr "connectors.detail_title"}} {{.Conector.Nombre}}</h2>
  <p style="color:#64748b">{{tr "connectors.detail_subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="post" action="/conectores/{{.Conector.Slug}}" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "common.slug"}} <input name="slug" value="{{.Conector.Slug}}" required></label>
      <label>{{tr "common.name"}} <input name="nombre" value="{{.Conector.Nombre}}" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "connectors.transport"}} <input name="transporte" value="{{.Conector.Transporte}}"></label>
      <label>{{tr "connectors.command"}} <input name="comando" value="{{.Conector.Comando}}"></label>
    </div>
    <div class="grid2">
      <label>{{tr "connectors.args_json"}} <textarea name="args_json" rows="4">{{.Conector.ArgsJSON}}</textarea></label>
      <label>{{tr "connectors.env_json"}} <textarea name="env_json" rows="4">{{.Conector.EnvJSON}}</textarea></label>
    </div>
    <label>{{tr "connectors.metadata_json"}} <textarea name="metadata_json" rows="8">{{.Conector.MetadataJSON}}</textarea></label>
    <label><input type="checkbox" name="activo"{{if .Conector.Activo}} checked{{end}}> {{tr "common.active"}}</label>
    <button type="submit">{{tr "connectors.save"}}</button>
  </form>
</section>
{{end}}`
