package orquestaweb

import (
	"html/template"
	"net/http"
)

type nuevaAppHTMLDataV0 struct {
	Page   NuevaAppWebPageV0
	Labels map[string]string
	Help   map[string]string
	HTML   map[string]string
}

func writeNuevaAppHTMLPageV0(w http.ResponseWriter, status int, page NuevaAppWebPageV0) WebHTMLWriteResultV0 {
	return writeNuevaAppHTMLPageWithTemplateV0(w, status, page, nuevaAppHTMLTemplateV0)
}

func writeNuevaAppHTMLPageWithTemplateV0(
	w http.ResponseWriter,
	status int,
	page NuevaAppWebPageV0,
	tmpl *template.Template,
) WebHTMLWriteResultV0 {
	return writeWebHTMLTemplateResponseV0(w, status, tmpl, nuevaAppHTMLDataFromPageV0(page), page.Locale)
}

func nuevaAppHTMLDataFromPageV0(page NuevaAppWebPageV0) nuevaAppHTMLDataV0 {
	return nuevaAppHTMLDataV0{
		Page:   page,
		Labels: nuevaAppHTMLLabelsV0(page.Formulario.Campos),
		Help:   nuevaAppHTMLHelpV0(page.Locale, NewNuevaAppI18nCatalogV0()),
		HTML:   nuevaAppHTMLTextosV0(page.Locale, NewNuevaAppI18nCatalogV0()),
	}
}

func nuevaAppHTMLLabelsV0(fields []NuevaAppWebCampoV0) map[string]string {
	labels := map[string]string{}
	for _, field := range fields {
		if field.Label != "" {
			labels[field.Path] = field.Label
		}
	}
	return labels
}

func nuevaAppHTMLTextosV0(locale string, catalog NuevaAppI18nCatalogV0) map[string]string {
	keys := []string{
		"nueva_app.html.resultado",
		"nueva_app.html.resumen",
		"nueva_app.html.errores_publicos",
		"nueva_app.html.warnings",
		"nueva_app.html.defaults",
		"nueva_app.html.sin_resultado",
		"nueva_app.html.backlog_vacio",
	}
	out := map[string]string{}
	for _, key := range keys {
		out[key] = nuevaAppWebLookupV0(catalog, locale, key)
	}
	return out
}

func nuevaAppHTMLHelpV0(locale string, catalog NuevaAppI18nCatalogV0) map[string]string {
	return map[string]string{
		"request_kind":   nuevaAppWebLookupV0(catalog, locale, "nueva_app.ayuda.request_kind"),
		"execution_mode": nuevaAppWebLookupV0(catalog, locale, "nueva_app.ayuda.execution_mode"),
	}
}

var nuevaAppHTMLTemplateV0 = template.Must(template.New("nueva_app_html_v0").Parse(`<!doctype html>
<html lang="{{.Page.Locale}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Page.Titulo}}</title>
  <style>
    body{font-family:system-ui,sans-serif;margin:0;background:#f6f7f9;color:#1f2933}
    main{max-width:1120px;margin:0 auto;padding:24px}
    form{display:grid;gap:16px}
    fieldset,.panel{border:1px solid #d7dde5;border-radius:8px;background:#fff;padding:16px}
    legend,h2{font-size:1rem;font-weight:700;margin:0 0 12px}
    label{display:grid;gap:6px;margin:0 0 12px;font-size:.93rem}
    input,select,textarea{font:inherit;padding:9px;border:1px solid #b8c2cc;border-radius:6px;background:#fff}
    textarea{min-height:82px}
    .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}
    .checks{display:flex;flex-wrap:wrap;gap:10px}
    .checks label{display:flex;align-items:center;gap:6px;margin:0}
    button{width:max-content;padding:10px 14px;border:0;border-radius:6px;background:#1d4ed8;color:#fff;font-weight:700}
    .status{border-left:4px solid #1d4ed8}
    .issue{border-left:4px solid #b91c1c}
    code{background:#eef2f7;padding:2px 4px;border-radius:4px}
  </style>
</head>
<body>
<main>
  <h1>{{.Page.Titulo}}</h1>
  <form method="post" action="/nueva-app">
    <fieldset>
      <legend>{{index .Labels "nombre"}}</legend>
      <div class="grid">
        <label>{{index .Labels "request_id"}}<input name="request_id" autocomplete="off"></label>
        <label>{{index .Labels "locale"}}<select name="locale">{{range .Page.Opciones.Locales}}<option value="{{.Valor}}">{{.Label}}</option>{{end}}</select></label>
        <label title="{{index .Help "request_kind"}}">{{index .Labels "request_kind"}}<select name="request_kind"><option value="crear_app_completa">crear_app_completa</option><option value="documentar_app">documentar_app</option><option value="analizar_app">analizar_app</option><option value="brainstorming_arquitectura">brainstorming_arquitectura</option><option value="planificar_app">planificar_app</option><option value="programar_modulo">programar_modulo</option><option value="modificar_app_existente">modificar_app_existente</option><option value="revisar_codigo">revisar_codigo</option><option value="pruebas_y_validacion">pruebas_y_validacion</option><option value="seguridad">seguridad</option><option value="deploy">deploy</option><option value="operacion_soporte">operacion_soporte</option><option value="integracion_externa">integracion_externa</option><option value="i18n_l10n">i18n_l10n</option><option value="migracion_refactor">migracion_refactor</option><option value="investigacion_tecnica">investigacion_tecnica</option></select></label>
        <label title="{{index .Help "execution_mode"}}">{{index .Labels "execution_mode"}}<select name="execution_mode"><option value="normal">normal</option><option value="debug">debug</option></select></label>
        <label>{{index .Labels "nombre"}}<input name="nombre" required></label>
        <label>{{index .Labels "tipo_app"}}<select name="tipo_app" required><option value="web">web</option><option value="api">api</option><option value="cli">cli</option><option value="desktop">desktop</option><option value="mobile">mobile</option><option value="automation">automation</option><option value="data">data</option><option value="plugin">plugin</option><option value="mixed">mixed</option></select></label>
      </div>
      <label>{{index .Labels "objetivo"}}<textarea name="objetivo" required></textarea></label>
      <label>{{index .Labels "descripcion"}}<textarea name="descripcion"></textarea></label>
      <div class="checks"><label><input type="checkbox" name="plataformas" value="web">web</label><label><input type="checkbox" name="plataformas" value="mobile">mobile</label><label><input type="checkbox" name="plataformas" value="desktop">desktop</label><label><input type="checkbox" name="plataformas" value="api">api</label></div>
    </fieldset>
    <fieldset><legend>{{index .Labels "project_source"}}</legend><div class="grid">
      <label>{{index .Labels "project_source.kind"}}<select name="project_source.kind"><option value=""></option><option value="new">new</option><option value="github">github</option><option value="local_path">local_path</option></select></label>
      <label>{{index .Labels "project_source.git_url"}}<input name="project_source.git_url"></label>
      <label>{{index .Labels "project_source.branch"}}<input name="project_source.branch"></label>
      <label>{{index .Labels "project_source.local_path"}}<input name="project_source.local_path"></label>
      <label>{{index .Labels "project_source.project_ref"}}<input name="project_source.project_ref"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "preferencias_tecnicas"}}</legend><div class="grid">
      <label>{{index .Labels "preferencias_tecnicas.arquitectura"}}<select name="preferencias_tecnicas.arquitectura"><option value="hexagonal">hexagonal</option><option value="modular">modular</option><option value="monolito_modular">monolito_modular</option></select></label>
      <label>{{index .Labels "preferencias_tecnicas.lenguaje"}}<input name="preferencias_tecnicas.lenguaje"></label>
      <label>{{index .Labels "preferencias_tecnicas.framework"}}<input name="preferencias_tecnicas.framework"></label>
      <label>{{index .Labels "preferencias_tecnicas.preferencias"}}<input name="preferencias_tecnicas.preferencias"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "i18n"}}</legend><div class="grid">
      <label><span>{{index .Labels "i18n.enabled"}}</span><select name="i18n.enabled"><option value="true">true</option><option value="false">false</option></select></label>
      <label>{{index .Labels "i18n.default_locale"}}<input name="i18n.default_locale" value="{{.Page.Locale}}"></label>
      <label>{{index .Labels "i18n.locales"}}<input name="i18n.locales" placeholder="es-ES,en-US"></label>
      <label>{{index .Labels "i18n.justificacion"}}<input name="i18n.justificacion"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "datos"}}</legend><div class="grid">
      <label><span>{{index .Labels "datos.db_required"}}</span><select name="datos.db_required"><option value="false">false</option><option value="true">true</option></select></label>
      <label>{{index .Labels "datos.necesidad_funcional"}}<input name="datos.necesidad_funcional"></label>
      <label>{{index .Labels "datos.tipos_datos"}}<input name="datos.tipos_datos" placeholder="usuarios,eventos"></label>
      <label>{{index .Labels "datos.sensibilidad"}}<input name="datos.sensibilidad"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "deploy"}}</legend><div class="grid">
      <label>{{index .Labels "deploy.target"}}<select name="deploy.target"><option value="sin_preferencia">sin_preferencia</option><option value="local">local</option><option value="contenedor">contenedor</option><option value="paas">paas</option><option value="serverless">serverless</option><option value="kubernetes">kubernetes</option><option value="desktop">desktop</option><option value="mobile_store">mobile_store</option></select></label>
      <label>{{index .Labels "deploy.restricciones"}}<input name="deploy.restricciones"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "calidad"}}</legend><div class="grid">
      <label>{{index .Labels "calidad.pruebas"}}<select name="calidad.pruebas"><option value="basica">basica</option><option value="media">media</option><option value="alta">alta</option></select></label>
      <label>{{index .Labels "calidad.accesibilidad"}}<select name="calidad.accesibilidad"><option value="basica">basica</option><option value="wcag_aa">wcag_aa</option></select></label>
      <label><span>{{index .Labels "calidad.observabilidad"}}</span><select name="calidad.observabilidad"><option value="true">true</option><option value="false">false</option></select></label>
      <label>{{index .Labels "calidad.compliance"}}<input name="calidad.compliance"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "agentes"}}</legend><div class="grid">
      <label><span>{{index .Labels "agentes.revision_humana"}}</span><select name="agentes.revision_humana"><option value="true">true</option><option value="false">false</option></select></label>
      <label>{{index .Labels "agentes.autonomia"}}<select name="agentes.autonomia"><option value="media">media</option><option value="baja">baja</option><option value="alta">alta</option></select></label>
      <label>{{index .Labels "agentes.preferencias"}}<input name="agentes.preferencias"></label>
      <label>{{index .Labels "restricciones"}}<input name="restricciones"></label>
    </div></fieldset>
    <fieldset><legend>{{index .Labels "integraciones"}}</legend><div class="grid">
      <label>{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.0.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option></select></label>
      <label>{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.0.nombre"></label>
      <label>{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.0.proposito"></label>
      <label><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.0.requerido"><option value="false">false</option><option value="true">true</option></select></label>
    </div></fieldset>
    <button type="submit">{{.Page.Formulario.Acciones.Submit}}</button>
  </form>
  <section class="panel status"><h2>{{index .HTML "nueva_app.html.resultado"}}</h2><p>{{.Page.Textos.Estado}}</p>{{if .Page.ViewModel.RequestID}}<p><code>{{.Page.ViewModel.RequestID}}</code></p>{{end}}</section>
  {{if .Page.ViewModel.ErroresPublicos}}<section class="panel issue"><h2>{{index .HTML "nueva_app.html.errores_publicos"}}</h2><ul>{{range .Page.ViewModel.ErroresPublicos}}<li><code>{{.Code}}</code> {{index $.Page.Textos.ErroresPublicos .Code}}</li>{{end}}</ul></section>{{end}}
  {{if .Page.ViewModel.ResumenApp.Nombre}}<section class="panel"><h2>{{index .HTML "nueva_app.html.resumen"}}</h2><p>{{.Page.ViewModel.ResumenApp.Nombre}} · {{.Page.ViewModel.ResumenApp.TipoApp}}</p><p>{{.Page.ViewModel.ResumenApp.Objetivo}}</p></section>{{end}}
  <section class="panel"><h2>{{.Page.Textos.BacklogPreview.Titulo}}</h2>{{if .Page.ViewModel.BacklogPreview.TotalMicrotareas}}<ol>{{range .Page.ViewModel.BacklogPreview.Microtareas}}<li><strong>{{.Key}}</strong> {{.Titulo}} <code>{{.ModuloFrontera}}</code></li>{{end}}</ol>{{else}}<p>{{index .HTML "nueva_app.html.backlog_vacio"}}</p>{{end}}</section>
</main>
</body>
</html>`))
