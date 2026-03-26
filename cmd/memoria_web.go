package cmd

import (
	"net/http"
	"net/url"
	"strings"

	"orquesta/db"
)

type webMemoriaData struct {
	Proyecto  string
	Tipo      string
	Entidades []*db.EntidadMemoria
	Msg       string
	Err       string
}

type webMemoriaDetalleData struct {
	Proyecto string
	Entidad  *db.EntidadMemoria
	Msg      string
	Err      string
}

func webRouterMemoria(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/memoria", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerMemoriaDetalle(w, r, parts[1])
	case len(parts) == 2 && r.Method == http.MethodPost && parts[1] == "guardar":
		webHandlerMemoriaGuardar(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerMemoria(w http.ResponseWriter, r *http.Request) {
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	tipo := strings.TrimSpace(r.URL.Query().Get("tipo"))
	items, err := webCargarMemoriaPorAPI(proyecto, tipo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, r, webTplLayout+webTplMemoria, webMemoriaData{
		Proyecto:  proyecto,
		Tipo:      tipo,
		Entidades: items,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	})
}

func webHandlerMemoriaDetalle(w http.ResponseWriter, r *http.Request, nombre string) {
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	item, err := webCargarMemoriaEntidadPorAPI(nombre, proyecto)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplMemoriaDetalle, webMemoriaDetalleData{
		Proyecto: proyecto,
		Entidad:  item,
		Msg:      r.URL.Query().Get("ok"),
		Err:      r.URL.Query().Get("err"),
	})
}

func webHandlerMemoriaGuardar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	proyecto := strings.TrimSpace(r.FormValue("proyecto"))
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	req := apiMemoriaEntidadRequest{
		Nombre:        nombre,
		Tipo:          strings.TrimSpace(r.FormValue("tipo")),
		Valor:         strings.TrimSpace(r.FormValue("valor")),
		Metadata:      strings.TrimSpace(r.FormValue("metadata")),
		VerificadoPor: strings.TrimSpace(r.FormValue("verificado_por")),
		Proyecto:      proyecto,
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/memoria", req, &map[string]any{}); err != nil {
		http.Redirect(w, r, "/memoria?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	destino := "/memoria"
	if nombre != "" {
		destino = "/memoria/" + url.PathEscape(nombre)
	}
	query := url.Values{}
	if proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	query.Set("ok", webTranslateRequestf(r, "memory.flash.saved", nombre))
	http.Redirect(w, r, destino+"?"+query.Encode(), http.StatusSeeOther)
}

func webCargarMemoriaPorAPI(proyecto, tipo string) ([]*db.EntidadMemoria, error) {
	var resp apiMemoriaEntidadesResponse
	path := "/api/memoria"
	q := url.Values{}
	if proyecto != "" {
		q.Set("proyecto", proyecto)
	}
	if tipo != "" {
		q.Set("tipo", tipo)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Entidades, nil
}

func webCargarMemoriaEntidadPorAPI(nombre, proyecto string) (*db.EntidadMemoria, error) {
	var resp apiMemoriaEntidadResponse
	path := "/api/memoria/" + url.PathEscape(strings.TrimSpace(nombre))
	if proyecto != "" {
		path += "?proyecto=" + url.QueryEscape(proyecto)
	}
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Entidad, nil
}

const webTplMemoria = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "memory.title"}}</h2>
  <p style="color:#64748b">{{tr "memory.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="get" action="/memoria" style="display:flex;gap:.7rem;flex-wrap:wrap;margin-bottom:1rem">
    <label>{{tr "projects.project"}} <input type="text" name="proyecto" value="{{.Proyecto}}"></label>
    <label>{{tr "memory.type"}} <input type="text" name="tipo" value="{{.Tipo}}"></label>
    <button type="submit">{{tr "common.filter"}}</button>
  </form>
  <table>
    <thead><tr><th>{{tr "common.name"}}</th><th>{{tr "memory.type"}}</th><th>{{tr "memory.verified_by"}}</th><th>{{tr "memory.value"}}</th><th></th></tr></thead>
    <tbody>
      {{range .Entidades}}
      <tr>
        <td><code>{{.Nombre}}</code></td>
        <td>{{.Tipo}}</td>
        <td>{{.VerificadoPor}}</td>
        <td><code>{{.ValorJSON}}</code></td>
        <td><a href="/memoria/{{.Nombre}}{{if $.Proyecto}}?proyecto={{$.Proyecto}}{{end}}">{{tr "common.view"}}</a></td>
      </tr>
      {{else}}
      <tr><td colspan="5">{{tr "memory.none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>

<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "memory.new_title"}}</h3>
  <form method="post" action="/memoria/guardar" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "projects.project"}} <input name="proyecto" value="{{.Proyecto}}"></label>
      <label>{{tr "common.name"}} <input name="nombre" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "memory.type"}} <input name="tipo" required></label>
      <label>{{tr "memory.verified_by"}} <input name="verificado_por"></label>
    </div>
    <label>{{tr "memory.value"}} <textarea name="valor" rows="6" required>{}</textarea></label>
    <label>{{tr "memory.metadata"}} <textarea name="metadata" rows="4">{}</textarea></label>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>
{{end}}`

const webTplMemoriaDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/memoria{{if .Proyecto}}?proyecto={{.Proyecto}}{{end}}">← {{tr "memory.back"}}</a></p>
  <h2 style="margin:0">{{tr "memory.detail_title"}} {{.Entidad.Nombre}}</h2>
  <p style="color:#64748b">{{tr "memory.detail_subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <table>
    <tbody>
      <tr><th>{{tr "common.name"}}</th><td><code>{{.Entidad.Nombre}}</code></td></tr>
      <tr><th>{{tr "memory.type"}}</th><td>{{.Entidad.Tipo}}</td></tr>
      <tr><th>{{tr "memory.verified_by"}}</th><td>{{.Entidad.VerificadoPor}}</td></tr>
      <tr><th>{{tr "memory.value"}}</th><td><pre style="margin:0;white-space:pre-wrap">{{.Entidad.ValorJSON}}</pre></td></tr>
      <tr><th>{{tr "memory.metadata"}}</th><td><pre style="margin:0;white-space:pre-wrap">{{.Entidad.MetadataJSON}}</pre></td></tr>
    </tbody>
  </table>
</section>
{{end}}`
