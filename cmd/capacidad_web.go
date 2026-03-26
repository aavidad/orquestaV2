package cmd

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
)

type webPoolsData struct {
	Pools []*db.PoolCapacidadResumen
	Msg   string
	Err   string
}

type webPoolDetalleData struct {
	Detalle *capacidadapp.PoolDetail
	Msg     string
	Err     string
}

type webModeloResolverData struct {
	Proyecto   string
	Fase       string
	Perfil     string
	Resolucion *db.ResolucionModelo
	Politicas  []*db.PoliticaModelo
	Msg        string
	Err        string
}

func webPoolFormBool(r *http.Request, key string) bool {
	v := strings.TrimSpace(strings.ToLower(r.FormValue(key)))
	return v == "1" || v == "true" || v == "on" || v == "yes"
}

func webRouterPools(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/pools", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerPoolDetalle(w, r, parts[1])
	case len(parts) == 2 && r.Method == http.MethodPost && parts[1] == "guardar":
		webHandlerPoolGuardar(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerPools(w http.ResponseWriter, r *http.Request) {
	var resp apiPoolsResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/pools", nil, &resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, r, webTplLayout+webTplPools, webPoolsData{
		Pools: resp.Pools,
		Msg:   r.URL.Query().Get("ok"),
		Err:   r.URL.Query().Get("err"),
	})
}

func webHandlerPoolDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	var resp apiPoolResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/pools/"+url.PathEscape(strings.TrimSpace(slug)), nil, &resp); err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplPoolDetalle, webPoolDetalleData{
		Detalle: resp.Detalle,
		Msg:     r.URL.Query().Get("ok"),
		Err:     r.URL.Query().Get("err"),
	})
}

func webHandlerPoolGuardar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	capacidadTotal, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("capacidad_total")))
	capacidadReservada, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("capacidad_reservada")))
	req := &db.PoolCapacidad{
		Slug:                strings.TrimSpace(r.FormValue("slug")),
		Proveedor:           strings.TrimSpace(r.FormValue("proveedor")),
		Runtime:             strings.TrimSpace(r.FormValue("runtime")),
		Plan:                strings.TrimSpace(r.FormValue("plan")),
		EsDePago:            webPoolFormBool(r, "es_de_pago"),
		CapacidadTotal:      capacidadTotal,
		CapacidadReservada:  capacidadReservada,
		PermiteHijos:        webPoolFormBool(r, "permite_hijos"),
		PermiteModelosMulti: webPoolFormBool(r, "permite_modelos_multi"),
		PermiteSobrecoste:   webPoolFormBool(r, "permite_sobrecoste"),
		PoliticaHandoff:     strings.TrimSpace(r.FormValue("politica_handoff")),
		FuenteTelemetria:    strings.TrimSpace(r.FormValue("fuente_telemetria")),
		MetadataJSON:        strings.TrimSpace(r.FormValue("metadata_json")),
		Activo:              webPoolFormBool(r, "activo"),
	}
	var resp apiPoolSaveResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/pools", apiPoolSaveRequest{
		Slug:                req.Slug,
		Proveedor:           req.Proveedor,
		Runtime:             req.Runtime,
		Plan:                req.Plan,
		EsDePago:            req.EsDePago,
		CapacidadTotal:      req.CapacidadTotal,
		CapacidadReservada:  req.CapacidadReservada,
		PermiteHijos:        req.PermiteHijos,
		PermiteModelosMulti: req.PermiteModelosMulti,
		PermiteSobrecoste:   req.PermiteSobrecoste,
		PoliticaHandoff:     req.PoliticaHandoff,
		FuenteTelemetria:    req.FuenteTelemetria,
		MetadataJSON:        req.MetadataJSON,
		Activo:              req.Activo,
	}, &resp); err != nil {
		http.Redirect(w, r, "/pools?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	destino := "/pools"
	if req.Slug != "" {
		destino = "/pools/" + url.PathEscape(req.Slug)
	}
	http.Redirect(w, r, destino+"?ok="+url.QueryEscape(webTranslateRequestf(r, "pools.flash.saved", req.Slug)), http.StatusSeeOther)
}

func webHandlerModelo(w http.ResponseWriter, r *http.Request) {
	data := webModeloResolverData{
		Proyecto: strings.TrimSpace(r.URL.Query().Get("proyecto")),
		Fase:     strings.TrimSpace(r.URL.Query().Get("fase")),
		Perfil:   strings.TrimSpace(r.URL.Query().Get("perfil")),
		Msg:      r.URL.Query().Get("ok"),
		Err:      r.URL.Query().Get("err"),
	}
	var politicasResp apiPoliticasModeloResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/politicas-modelo?activa=true", nil, &politicasResp); err == nil {
		data.Politicas = politicasResp.Politicas
	}
	if data.Proyecto != "" || data.Fase != "" || data.Perfil != "" {
		path := "/api/modelo/resolver"
		q := url.Values{}
		if data.Proyecto != "" {
			q.Set("proyecto", data.Proyecto)
		}
		if data.Fase != "" {
			q.Set("fase", data.Fase)
		}
		if data.Perfil != "" {
			q.Set("perfil", data.Perfil)
		}
		if len(q) > 0 {
			path += "?" + q.Encode()
		}
		var resp apiResolucionModeloResponse
		if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
			data.Err = err.Error()
		} else {
			data.Resolucion = resp.Resolucion
		}
	}
	webRender(w, r, webTplLayout+webTplModeloResolver, data)
}

func webHandlerModeloPoliticaGuardar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	prioridad, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("prioridad")))
	req := apiPoliticaModeloSaveRequest{
		ScopeTipo:       strings.TrimSpace(r.FormValue("scope_tipo")),
		ScopeRef:        strings.TrimSpace(r.FormValue("scope_ref")),
		PerfilTarea:     strings.TrimSpace(r.FormValue("perfil")),
		PoolSlug:        strings.TrimSpace(r.FormValue("pool_slug")),
		ModelSlug:       strings.TrimSpace(r.FormValue("model_slug")),
		ReasoningEffort: strings.TrimSpace(r.FormValue("reasoning")),
		Prioridad:       prioridad,
		Activa:          webPoolFormBool(r, "activa"),
		MetadataJSON:    strings.TrimSpace(r.FormValue("metadata_json")),
	}
	var resp apiPoliticaModeloSaveResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/politicas-modelo", req, &resp); err != nil {
		http.Redirect(w, r, "/modelo?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/modelo?ok="+url.QueryEscape(webTranslateRequestf(r, "model.policy_saved", req.PerfilTarea)), http.StatusSeeOther)
}

const webTplPools = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "pools.title"}}</h2>
  <p style="color:#64748b">{{tr "pools.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <table>
    <thead><tr><th>Slug</th><th>{{tr "pools.provider"}}</th><th>Runtime</th><th>{{tr "pools.total"}}</th><th>{{tr "pools.active_sessions"}}</th><th>{{tr "pools.available"}}</th><th></th></tr></thead>
    <tbody>
      {{range .Pools}}
      <tr>
        <td><code>{{.Pool.Slug}}</code></td>
        <td>{{.Pool.Proveedor}}</td>
        <td>{{.Pool.Runtime}}</td>
        <td>{{.Pool.CapacidadTotal}}</td>
        <td>{{.SesionesActivas}}</td>
        <td>{{.CapacidadDisponible}}</td>
        <td><a href="/pools/{{.Pool.Slug}}">{{tr "common.view"}}</a></td>
      </tr>
      {{else}}
      <tr><td colspan="7">{{tr "pools.none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>

<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "pools.new_title"}}</h3>
  <form method="post" action="/pools/guardar" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>Slug <input name="slug" required></label>
      <label>{{tr "pools.provider"}} <input name="proveedor" required></label>
    </div>
    <div class="grid2">
      <label>Runtime <input name="runtime" required></label>
      <label>Plan <input name="plan" value="default"></label>
    </div>
    <div class="grid2">
      <label>{{tr "pools.total"}} <input type="number" name="capacidad_total" value="1"></label>
      <label>{{tr "pools.reserved"}} <input type="number" name="capacidad_reservada" value="0"></label>
    </div>
    <div class="grid2">
      <label>{{tr "pools.handoff_policy"}} <input name="politica_handoff" value="preventivo"></label>
      <label>{{tr "pools.telemetry_source"}} <input name="fuente_telemetria" value="manual"></label>
    </div>
    <label>{{tr "memory.metadata"}} <textarea name="metadata_json" rows="4">{}</textarea></label>
    <label><input type="checkbox" name="activo" checked> {{tr "common.active"}}</label>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>
{{end}}`

const webTplPoolDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/pools">← {{tr "pools.back"}}</a></p>
  <h2 style="margin:0">{{tr "pools.detail_title"}} {{.Detalle.Pool.Slug}}</h2>
  <p style="color:#64748b">{{tr "pools.detail_subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <table>
    <tbody>
      <tr><th>Slug</th><td><code>{{.Detalle.Pool.Slug}}</code></td></tr>
      <tr><th>{{tr "pools.provider"}}</th><td>{{.Detalle.Pool.Proveedor}}</td></tr>
      <tr><th>Runtime</th><td>{{.Detalle.Pool.Runtime}}</td></tr>
      <tr><th>Plan</th><td>{{.Detalle.Pool.Plan}}</td></tr>
      <tr><th>{{tr "pools.total"}}</th><td>{{.Detalle.Pool.CapacidadTotal}}</td></tr>
      <tr><th>{{tr "pools.reserved"}}</th><td>{{.Detalle.Pool.CapacidadReservada}}</td></tr>
      <tr><th>{{tr "pools.active_sessions"}}</th><td>{{.Detalle.SesionesActivas}}</td></tr>
      <tr><th>{{tr "pools.available"}}</th><td>{{.Detalle.CapacidadDisponible}}</td></tr>
    </tbody>
  </table>
  {{if .Detalle.Modelos}}
  <h3>{{tr "pools.models_title"}}</h3>
  <table>
    <thead><tr><th>{{tr "pools.model"}}</th><th>{{tr "pools.priority"}}</th><th>{{tr "pools.cost"}}</th><th>{{tr "common.active"}}</th></tr></thead>
    <tbody>
      {{range .Detalle.Modelos}}
      <tr><td><code>{{.ModelSlug}}</code></td><td>{{.Prioridad}}</td><td>{{.CosteRelativo}}</td><td>{{if .Activo}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</td></tr>
      {{end}}
    </tbody>
  </table>
  {{end}}
</section>
{{end}}`

const webTplModeloResolver = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "model.title"}}</h2>
  <p style="color:#64748b">{{tr "model.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="get" action="/modelo" style="display:grid;gap:.7rem;margin-bottom:1rem">
    <div class="grid2">
      <label>{{tr "projects.project"}} <input name="proyecto" value="{{.Proyecto}}"></label>
      <label>{{tr "Progreso"}} <input name="fase" value="{{.Fase}}"></label>
    </div>
    <label>{{tr "Perfil"}} <input name="perfil" value="{{.Perfil}}"></label>
    <button type="submit">{{tr "model.resolve"}}</button>
  </form>
  {{if .Resolucion}}
  <table>
    <tbody>
      <tr><th>{{tr "Perfil"}}</th><td>{{.Resolucion.PerfilTarea}}</td></tr>
      <tr><th>Pool</th><td>{{.Resolucion.PoolSlug}}</td></tr>
      <tr><th>{{tr "Modelo"}}</th><td>{{.Resolucion.ModelSlug}}</td></tr>
      <tr><th>{{tr "Reasoning"}}</th><td>{{.Resolucion.ReasoningEffort}}</td></tr>
    </tbody>
  </table>
  {{end}}
  <h3>{{tr "model.policies_title"}}</h3>
  <table>
    <thead><tr><th>{{tr "config.key"}}</th><th>{{tr "Perfil"}}</th><th>Pool</th><th>{{tr "Modelo"}}</th><th>{{tr "Reasoning"}}</th><th>{{tr "pools.priority"}}</th></tr></thead>
    <tbody>
      {{range .Politicas}}
      <tr><td>{{.ScopeTipo}}{{if .ScopeRef}}/{{.ScopeRef}}{{end}}</td><td>{{.PerfilTarea}}</td><td>{{.PoolSlug}}</td><td>{{.ModelSlug}}</td><td>{{.ReasoningEffort}}</td><td>{{.Prioridad}}</td></tr>
      {{else}}
      <tr><td colspan="6">{{tr "model.policies_none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
  <h3>{{tr "model.policy_new_title"}}</h3>
  <form method="post" action="/modelo/politicas/guardar" style="display:grid;gap:.7rem;margin-top:1rem">
    <div class="grid2">
      <label>{{tr "config.key"}} <input name="scope_tipo" value="global"></label>
      <label>Ref <input name="scope_ref"></label>
    </div>
    <div class="grid2">
      <label>{{tr "Perfil"}} <input name="perfil" value="*"></label>
      <label>Pool <input name="pool_slug"></label>
    </div>
    <div class="grid2">
      <label>{{tr "Modelo"}} <input name="model_slug"></label>
      <label>{{tr "Reasoning"}} <input name="reasoning"></label>
    </div>
    <div class="grid2">
      <label>{{tr "pools.priority"}} <input type="number" name="prioridad" value="100"></label>
      <label>{{tr "memory.metadata"}} <input name="metadata_json" value="{}"></label>
    </div>
    <label><input type="checkbox" name="activa" checked> {{tr "common.active"}}</label>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>
{{end}}`
