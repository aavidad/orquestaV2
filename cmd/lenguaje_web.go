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
	"strconv"
	"strings"

	"orquesta/lenguajeapp"
)

type webLenguajeData struct {
	Politica         *lenguajeapp.LanguagePolicy
	AllowedLanguages string
	Matriz           []webLenguajeMatrixRow
	Resolucion       *lenguajeapp.LanguageResolution
	ResolverProyecto string
	ResolverTarea    string
	ResolverContexto string
	Msg              string
	Err              string
}

type webLenguajeMatrixRow struct {
	Scope     string
	Selector  string
	Context   string
	Language  string
	Reason    string
	UpdatedAt string
}

func webRouterLenguaje(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/lenguaje", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "politica" && r.Method == http.MethodPost:
		webHandlerLenguajePolitica(w, r)
	case len(parts) == 2 && parts[1] == "matriz" && r.Method == http.MethodPost:
		webHandlerLenguajeMatriz(w, r)
	case len(parts) == 3 && parts[1] == "matriz" && parts[2] == "borrar" && r.Method == http.MethodPost:
		webHandlerLenguajeMatrizBorrar(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerLenguaje(w http.ResponseWriter, r *http.Request) {
	politica, err := webCargarPoliticaLenguajePorAPI()
	if err != nil {
		webRender(w, r, webTplLayout+webTplLenguaje, webLenguajeData{Err: err.Error()})
		return
	}
	matriz, err := webCargarMatrizLenguajePorAPI()
	if err != nil {
		webRender(w, r, webTplLayout+webTplLenguaje, webLenguajeData{
			Politica: politica,
			Err:      err.Error(),
		})
		return
	}

	data := webLenguajeData{
		Politica:         politica,
		AllowedLanguages: strings.Join(politica.AllowedLanguages, ", "),
		ResolverProyecto: strings.TrimSpace(r.URL.Query().Get("proyecto")),
		ResolverTarea:    strings.TrimSpace(r.URL.Query().Get("tarea")),
		ResolverContexto: strings.TrimSpace(r.URL.Query().Get("contexto")),
		Msg:              r.URL.Query().Get("ok"),
		Err:              r.URL.Query().Get("err"),
	}
	if data.ResolverContexto == "" {
		data.ResolverContexto = "all"
	}
	if data.ResolverProyecto != "" || data.ResolverTarea != "" {
		var tareaID *int64
		if raw := strings.TrimSpace(data.ResolverTarea); raw != "" {
			value, convErr := strconv.ParseInt(raw, 10, 64)
			if convErr != nil || value <= 0 {
				data.Err = webTranslateRequestf(r, "langweb.flash.invalid_task")
				webRender(w, r, webTplLayout+webTplLenguaje, data)
				return
			}
			tareaID = &value
		}
		resolucion, resolveErr := webResolverLenguajePorAPI(data.ResolverProyecto, tareaID, data.ResolverContexto)
		if resolveErr != nil {
			data.Err = resolveErr.Error()
		} else {
			data.Resolucion = resolucion
		}
	}
	for _, entry := range matriz {
		if entry == nil {
			continue
		}
		data.Matriz = append(data.Matriz, webLenguajeMatrixRow{
			Scope:     entry.Scope,
			Selector:  entry.Selector,
			Context:   entry.Context,
			Language:  entry.Language,
			Reason:    entry.Reason,
			UpdatedAt: formatTime(entry.UpdatedAt),
		})
	}
	webRender(w, r, webTplLayout+webTplLenguaje, data)
}

func webHandlerLenguajePolitica(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	politica := &lenguajeapp.LanguagePolicy{
		DefaultLanguage:          strings.TrimSpace(r.FormValue("default_language")),
		DocumentationMultilang:   r.FormValue("documentation_multilang") == "on",
		AppsMultilang:            r.FormValue("apps_multilang") == "on",
		DocumentationDefaultLang: strings.TrimSpace(r.FormValue("documentation_default_language")),
		AppsDefaultLang:          strings.TrimSpace(r.FormValue("apps_default_language")),
		AllowedLanguages:         splitLanguages(r.FormValue("allowed_languages")),
		Notes:                    strings.TrimSpace(r.FormValue("notes")),
	}
	if err := webFijarPoliticaLenguajePorAPI(politica, "web"); err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/lenguaje?ok="+url.QueryEscape(webTranslateRequestf(r, "langweb.flash.policy_updated")), http.StatusSeeOther)
}

func webHandlerLenguajeMatriz(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	err := webFijarEntradaMatrizLenguajePorAPI(
		strings.TrimSpace(r.FormValue("scope")),
		strings.TrimSpace(r.FormValue("selector")),
		strings.TrimSpace(r.FormValue("context")),
		strings.TrimSpace(r.FormValue("language")),
		strings.TrimSpace(r.FormValue("reason")),
		"web",
	)
	if err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/lenguaje?ok="+url.QueryEscape(webTranslateRequestf(r, "langweb.flash.matrix_saved")), http.StatusSeeOther)
}

func webHandlerLenguajeMatrizBorrar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	err := webBorrarEntradaMatrizLenguajePorAPI(
		strings.TrimSpace(r.FormValue("scope")),
		strings.TrimSpace(r.FormValue("selector")),
		strings.TrimSpace(r.FormValue("context")),
	)
	if err != nil {
		http.Redirect(w, r, "/lenguaje?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/lenguaje?ok="+url.QueryEscape(webTranslateRequestf(r, "langweb.flash.matrix_deleted")), http.StatusSeeOther)
}

func webCargarPoliticaLenguajePorAPI() (*lenguajeapp.LanguagePolicy, error) {
	var resp apiLenguajePoliticaResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/lenguaje/politica", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Politica, nil
}

func webCargarMatrizLenguajePorAPI() ([]*lenguajeapp.LanguageMatrixEntry, error) {
	var resp apiLenguajeMatrizResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/lenguaje/matriz", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Matriz, nil
}

func webResolverLenguajePorAPI(proyecto string, tareaID *int64, contexto string) (*lenguajeapp.LanguageResolution, error) {
	query := url.Values{}
	if proyecto = strings.TrimSpace(proyecto); proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	if tareaID != nil && *tareaID > 0 {
		query.Set("tarea", strconv.FormatInt(*tareaID, 10))
	}
	if contexto = strings.TrimSpace(contexto); contexto != "" {
		query.Set("contexto", contexto)
	}
	path := "/api/lenguaje/resolver"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiLenguajeResolucionResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Resolucion, nil
}

func webFijarPoliticaLenguajePorAPI(politica *lenguajeapp.LanguagePolicy, por string) error {
	return webInvocarAPIJSON(http.MethodPost, "/api/lenguaje/politica", apiLenguajePoliticaSetRequest{
		Politica: politica,
		Por:      strings.TrimSpace(por),
	}, nil)
}

func webFijarEntradaMatrizLenguajePorAPI(scope, selector, contexto, idioma, razon, por string) error {
	return webInvocarAPIJSON(http.MethodPost, "/api/lenguaje/matriz", apiLenguajeMatrizSetRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
		Idioma:   strings.TrimSpace(idioma),
		Razon:    strings.TrimSpace(razon),
		Por:      strings.TrimSpace(por),
	}, nil)
}

func webBorrarEntradaMatrizLenguajePorAPI(scope, selector, contexto string) error {
	return webInvocarAPIJSON(http.MethodPost, "/api/lenguaje/matriz/borrar", apiLenguajeMatrizDeleteRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
	}, nil)
}

const webTplLenguaje = `{{define "content"}}
<section>
  <h2>{{tr "langweb.title"}}</h2>
  <p style="color:#64748b">{{tr "langweb.subtitle"}}</p>
  {{if .Msg}}<div class="alert-ok">{{.Msg}}</div>{{end}}
  {{if .Err}}<div class="alert-err">{{.Err}}</div>{{end}}

  <div class="grid2">
    <div>
      <article>
        <header><strong>{{tr "langweb.policy.title"}}</strong></header>
        {{if .Politica}}
        <form method="post" action="/lenguaje/politica" style="display:grid;gap:.7rem">
          <div><label>{{tr "langweb.policy.default"}}</label><input type="text" name="default_language" value="{{.Politica.DefaultLanguage}}"></div>
          <div><label>{{tr "langweb.policy.docs_default"}}</label><input type="text" name="documentation_default_language" value="{{.Politica.DocumentationDefaultLang}}"></div>
          <div><label>{{tr "langweb.policy.apps_default"}}</label><input type="text" name="apps_default_language" value="{{.Politica.AppsDefaultLang}}"></div>
          <div><label>{{tr "langweb.policy.allowed"}}</label><input type="text" name="allowed_languages" value="{{.AllowedLanguages}}"></div>
          <label><input type="checkbox" name="documentation_multilang" {{if .Politica.DocumentationMultilang}}checked{{end}}> {{tr "langweb.policy.docs_multilang"}}</label>
          <label><input type="checkbox" name="apps_multilang" {{if .Politica.AppsMultilang}}checked{{end}}> {{tr "langweb.policy.apps_multilang"}}</label>
          <div><label>{{tr "langweb.policy.notes"}}</label><textarea name="notes" rows="5">{{.Politica.Notes}}</textarea></div>
          <div><button type="submit" class="btn-sm">{{tr "langweb.policy.save"}}</button></div>
        </form>
        {{end}}
      </article>

      <article style="margin-top:1rem">
        <header><strong>{{tr "langweb.resolve.title"}}</strong></header>
        <form method="get" action="/lenguaje" style="display:grid;gap:.7rem">
          <div><label>{{tr "Proyecto"}}</label><input type="text" name="proyecto" value="{{.ResolverProyecto}}" placeholder="orquestador"></div>
          <div><label>{{tr "langweb.resolve.task"}}</label><input type="text" name="tarea" value="{{.ResolverTarea}}" placeholder="123"></div>
          <div><label>{{tr "langweb.resolve.context"}}</label><input type="text" name="contexto" value="{{.ResolverContexto}}" placeholder="apps"></div>
          <div><button type="submit" class="btn-sm">{{tr "langweb.resolve.run"}}</button></div>
        </form>
        {{if .Resolucion}}
        <div style="margin-top:1rem">
          <p><strong>{{tr "langweb.resolve.language"}}:</strong> {{.Resolucion.Idioma}}</p>
          <p><strong>{{tr "langweb.resolve.origin"}}:</strong> {{.Resolucion.Origen}}</p>
          <p><strong>{{tr "langweb.resolve.context"}}:</strong> {{.Resolucion.Contexto}}</p>
          {{if .Resolucion.Entrada}}
          <p><strong>{{tr "langweb.resolve.entry"}}:</strong> {{.Resolucion.Entrada.Scope}}/{{.Resolucion.Entrada.Selector}} [{{.Resolucion.Entrada.Context}}]</p>
          {{if .Resolucion.Entrada.Reason}}<p><strong>{{tr "Motivo"}}:</strong> {{.Resolucion.Entrada.Reason}}</p>{{end}}
          {{end}}
        </div>
        {{end}}
      </article>
    </div>

    <div>
      <article>
        <header><strong>{{tr "langweb.matrix.title"}}</strong></header>
        <form method="post" action="/lenguaje/matriz" style="display:grid;grid-template-columns:1fr 1fr 1fr 1fr;gap:.6rem;align-items:end;margin-bottom:1rem">
          <div><label>{{tr "langweb.matrix.scope"}}</label><input type="text" name="scope" value="project" placeholder="project"></div>
          <div><label>{{tr "langweb.matrix.selector"}}</label><input type="text" name="selector" placeholder="orquestador"></div>
          <div><label>{{tr "langweb.matrix.context"}}</label><input type="text" name="context" value="all" placeholder="apps"></div>
          <div><label>{{tr "langweb.matrix.language"}}</label><input type="text" name="language" placeholder="en"></div>
          <div style="grid-column:1/4"><label>{{tr "Motivo"}}</label><input type="text" name="reason" placeholder="{{tr "langweb.matrix.reason_placeholder"}}"></div>
          <div><button type="submit" class="btn-sm">{{tr "langweb.matrix.save"}}</button></div>
        </form>

        {{if .Matriz}}
        <table>
          <thead><tr><th>{{tr "langweb.matrix.scope"}}</th><th>{{tr "langweb.matrix.selector"}}</th><th>{{tr "langweb.matrix.context"}}</th><th>{{tr "langweb.matrix.language"}}</th><th>{{tr "Motivo"}}</th><th>{{tr "langweb.matrix.updated"}}</th><th></th></tr></thead>
          <tbody>
            {{range .Matriz}}
            <tr>
              <td>{{.Scope}}</td>
              <td>{{.Selector}}</td>
              <td>{{.Context}}</td>
              <td>{{.Language}}</td>
              <td>{{if .Reason}}{{.Reason}}{{else}}—{{end}}</td>
              <td>{{.UpdatedAt}}</td>
              <td>
                <form method="post" action="/lenguaje/matriz/borrar" style="margin:0">
                  <input type="hidden" name="scope" value="{{.Scope}}">
                  <input type="hidden" name="selector" value="{{.Selector}}">
                  <input type="hidden" name="context" value="{{.Context}}">
                  <button type="submit" class="btn-sm">{{tr "langweb.matrix.delete"}}</button>
                </form>
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
        {{else}}
        <p>{{tr "langweb.matrix.empty"}}</p>
        {{end}}
      </article>
    </div>
  </div>
</section>
{{end}}`
