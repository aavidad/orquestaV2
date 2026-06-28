package orquestaweb

import (
	"html/template"
	"net/http"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

type nuevaAppHTMLDataV0 struct {
	Page         NuevaAppWebPageV0
	Labels       map[string]string
	Help         map[string]string
	HTML         map[string]string
	StorageTypes []string
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
		Page:         page,
		Labels:       nuevaAppHTMLLabelsV0(page.Formulario.Campos),
		Help:         nuevaAppHTMLHelpV0(page.Locale, NewNuevaAppI18nCatalogV0()),
		HTML:         nuevaAppHTMLTextosV0(page.Locale, NewNuevaAppI18nCatalogV0()),
		StorageTypes: orquestafactory.SupportedDataStorageTypesV0(),
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
		"nueva_app.goal.titulo",
		"nueva_app.goal.run_ref",
		"nueva_app.goal.director_execution_mode",
		"nueva_app.goal.goal_ref",
		"nueva_app.goal.external_goal_ref",
		"nueva_app.goal.goal_status",
		"nueva_app.goal.run_status",
		"nueva_app.goal.closure_status",
		"nueva_app.goal.update",
		"nueva_app.goal.updating",
		"nueva_app.goal.updated",
		"nueva_app.goal.error",
		"nueva_app.goal_preview.titulo",
		"nueva_app.goal_preview.objective",
		"nueva_app.goal_preview.write_set",
		"nueva_app.goal_preview.required_tests",
		"nueva_app.goal_preview.acceptance",
		"nueva_app.goal_preview.artifacts",
		"nueva_app.goal_preview.estimate",
		"nueva_app.goal_preview.cost_tier",
		"nueva_app.goal_preview.tokens",
		"nueva_app.goal_preview.subgoals",
		"nueva_app.wizard.nav.home",
		"nueva_app.wizard.nav.ops",
		"nueva_app.wizard.nav.autoprogramming",
		"nueva_app.wizard.lead",
		"nueva_app.wizard.steps_label",
		"nueva_app.wizard.step.idea",
		"nueva_app.wizard.step.tipo",
		"nueva_app.wizard.step.tecnologia",
		"nueva_app.wizard.step.datos",
		"nueva_app.wizard.step.calidad",
		"nueva_app.wizard.step.revisar",
		"nueva_app.wizard.idea_objetivo",
		"nueva_app.wizard.guided_title",
		"nueva_app.wizard.guided_need",
		"nueva_app.wizard.guided_analyze",
		"nueva_app.wizard.guided_followups",
		"nueva_app.wizard.guided_mobile_both",
		"nueva_app.wizard.guided_mobile_ios",
		"nueva_app.wizard.guided_mobile_android",
		"nueva_app.wizard.guided_data_external",
		"nueva_app.wizard.guided_data_management",
		"nueva_app.wizard.guided_maps_generic",
		"nueva_app.wizard.guided_maps_osm",
		"nueva_app.wizard.guided_architecture_default",
		"nueva_app.wizard.guided_architecture_event",
		"nueva_app.wizard.guided_architecture_modular",
		"nueva_app.wizard.guided_quality_public",
		"nueva_app.wizard.guided_review",
		"nueva_app.wizard.guided_msg_analyzed",
		"nueva_app.wizard.guided_msg_mobile",
		"nueva_app.wizard.guided_msg_data",
		"nueva_app.wizard.guided_msg_maps",
		"nueva_app.wizard.guided_msg_architecture",
		"nueva_app.wizard.guided_msg_quality",
		"nueva_app.wizard.guided_msg_review",
		"nueva_app.wizard.presets",
		"nueva_app.wizard.preset.webapp",
		"nueva_app.wizard.preset.api",
		"nueva_app.wizard.preset.ops",
		"nueva_app.wizard.identidad_avanzada",
		"nueva_app.wizard.compatibilidad_historica",
		"nueva_app.wizard.compatibilidad_legacy_intro",
		"nueva_app.wizard.force_legacy_director_loop",
		"nueva_app.wizard.plataformas_origen",
		"nueva_app.wizard.origen_proyecto",
		"nueva_app.wizard.datos_experto",
		"nueva_app.wizard.dato_1",
		"nueva_app.wizard.dato_2",
		"nueva_app.wizard.almacenamiento_1",
		"nueva_app.wizard.almacenamiento_2",
		"nueva_app.wizard.integracion_1",
		"nueva_app.wizard.integracion_2",
		"nueva_app.wizard.integracion_3",
		"nueva_app.wizard.integraciones_adicionales",
		"nueva_app.wizard.revision_final",
		"nueva_app.wizard.resumen_vivo",
		"nueva_app.wizard.back",
		"nueva_app.wizard.next",
		"nueva_app.wizard.placeholder.objetivo",
		"nueva_app.wizard.placeholder.descripcion",
		"nueva_app.wizard.summary.name",
		"nueva_app.wizard.summary.type",
		"nueva_app.wizard.summary.goal",
		"nueva_app.wizard.summary.platforms",
		"nueva_app.wizard.summary.architecture",
		"nueva_app.wizard.summary.data",
		"nueva_app.wizard.summary.quality",
		"nueva_app.wizard.summary.autonomy",
		"nueva_app.wizard.summary.no_name",
		"nueva_app.wizard.summary.pending",
		"nueva_app.wizard.summary.db_required",
		"nueva_app.wizard.summary.no_db_required",
		"nueva_app.validation.summary_title",
		"nueva_app.validation.summary_intro",
		"nueva_app.validation.required",
	}
	out := map[string]string{}
	for _, key := range keys {
		out[key] = nuevaAppWebLookupV0(catalog, locale, key)
	}
	return out
}

func nuevaAppHTMLHelpV0(locale string, catalog NuevaAppI18nCatalogV0) map[string]string {
	out := map[string]string{}
	for _, key := range nuevaAppHTMLHelpKeysV0 {
		out[key] = nuevaAppWebLookupV0(catalog, locale, "nueva_app.ayuda."+key)
	}
	return out
}

func nuevaAppHTMLHelpI18nKeysV0() []string {
	keys := make([]string, 0, len(nuevaAppHTMLHelpKeysV0))
	for _, key := range nuevaAppHTMLHelpKeysV0 {
		keys = append(keys, "nueva_app.ayuda."+key)
	}
	return keys
}

var nuevaAppHTMLHelpKeysV0 = []string{
	"step.idea",
	"step.tipo",
	"step.tecnologia",
	"step.datos",
	"step.calidad",
	"step.revisar",
	"guided.analyze",
	"guided.review",
	"guided.mobile_both",
	"guided.mobile_ios",
	"guided.mobile_android",
	"guided.data_external",
	"guided.data_management",
	"guided.maps_generic",
	"guided.maps_osm",
	"guided.architecture_default",
	"guided.architecture_event",
	"guided.architecture_modular",
	"guided.quality_public",
	"preset.webapp",
	"preset.api",
	"preset.ops",
	"action.preview",
	"action.launch",
	"nav.back",
	"nav.next",
	"nombre",
	"tipo_app",
	"objetivo",
	"descripcion",
	"request_id",
	"locale",
	"request_kind",
	"execution_mode",
	"director_execution_mode",
	"plataformas.web",
	"plataformas.mobile",
	"plataformas.desktop",
	"plataformas.api",
	"project_source.kind",
	"project_source.git_url",
	"project_source.branch",
	"project_source.project_ref",
	"project_source.local_path",
	"preferencias_tecnicas.arquitectura",
	"preferencias_tecnicas.lenguaje",
	"preferencias_tecnicas.framework",
	"preferencias_tecnicas.preferencias",
	"i18n.enabled",
	"i18n.default_locale",
	"i18n.locales",
	"i18n.justificacion",
	"datos.db_required",
	"datos.necesidad_funcional",
	"datos.tipos_datos",
	"datos.tipos_detallados.nombre",
	"datos.tipos_detallados.proposito",
	"datos.tipos_detallados.sensibilidad",
	"datos.tipos_detallados.retencion",
	"datos.tipos_detallados.volumen",
	"datos.tipos_detallados.restricciones",
	"datos.storage.tipo",
	"datos.storage.proposito",
	"datos.storage.requerido",
	"datos.storage.restricciones",
	"datos.sensibilidad",
	"datos.retencion",
	"integraciones.0.tipo",
	"integraciones.0.nombre",
	"integraciones.0.proposito",
	"integraciones.0.requerido",
	"integraciones.0.restricciones",
	"calidad.pruebas",
	"calidad.accesibilidad",
	"calidad.accesibilidad_opciones",
	"calidad.observabilidad",
	"calidad.compliance",
	"deploy.target",
	"deploy.restricciones",
	"documentacion.usuario",
	"documentacion.desarrollo",
	"documentacion.sistemas",
	"documentacion.locales",
	"agentes.revision_humana",
	"agentes.autonomia",
	"agentes.preferencias",
	"restricciones",
}

var nuevaAppHTMLTemplateV0 = template.Must(template.New("nueva_app_html_v0").Parse(`<!doctype html>
<html lang="{{.Page.Locale}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Page.Titulo}}</title>
  <style>
    :root{color-scheme:light;--bg:#f3f5ef;--ink:#18211b;--muted:#65736a;--panel:#ffffff;--line:#d7dfd5;--brand:#1f7a4d;--brand-2:#c7f04b;--warn:#b45309;--bad:#b42318}
    *{box-sizing:border-box}
    body{font-family:"Trebuchet MS","Gill Sans",Verdana,sans-serif;margin:0;background:radial-gradient(circle at 10% 0,rgba(199,240,75,.35),transparent 28rem),linear-gradient(160deg,#f8faf4,#edf3ee);color:var(--ink)}
    main{max-width:1260px;margin:0 auto;padding:24px}
    header{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:18px}
    h1{font-family:Georgia,"Times New Roman",serif;font-size:clamp(30px,4vw,56px);line-height:.95;letter-spacing:0;margin:0}
    .topnav{display:flex;gap:8px;flex-wrap:wrap}
    .topnav a,.ghost{border:1px solid var(--line);background:rgba(255,255,255,.72);color:var(--ink);border-radius:999px;padding:8px 11px;text-decoration:none}
    .lead{color:var(--muted);max-width:760px;margin:10px 0 0;font-size:17px}
    form{display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:16px;align-items:start}
    .wizard{border:1px solid var(--line);border-radius:18px;background:rgba(255,255,255,.88);box-shadow:0 24px 70px rgba(30,50,35,.12);overflow:visible}
    .steps{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:1px;background:var(--line)}
    .step-tab{border:0;border-radius:0;background:#f8fbf5;color:var(--muted);padding:12px 8px;font-weight:800;cursor:pointer}
    .step-tab.active{background:var(--brand);color:#fff}
    .step{padding:18px;display:grid;gap:14px}
    .wizard-ready .step[hidden]{display:none}
    fieldset,.panel{border:1px solid var(--line);border-radius:14px;background:var(--panel);padding:16px}
    legend,h2{font-size:1rem;font-weight:850;margin:0 0 12px}
    label{display:grid;gap:6px;margin:0 0 12px;font-size:.93rem;color:#2a382f}
    input,select,textarea{font:inherit;padding:10px;border:1px solid #b8c5bb;border-radius:9px;background:#fff;color:var(--ink)}
    textarea{min-height:92px}
    .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}
    .checks{display:flex;flex-wrap:wrap;gap:10px}
    .checks label{display:flex;align-items:center;gap:6px;margin:0;border:1px solid var(--line);border-radius:999px;padding:7px 10px;background:#f8fbf5}
    [data-help]{position:relative;cursor:help}
    [data-help]::after{content:attr(data-help);display:none;position:absolute;left:0;bottom:calc(100% + 8px);z-index:50;width:300px;max-width:calc(100vw - 32px);padding:9px 10px;border-radius:8px;background:#102017;color:#f7fff2;box-shadow:0 12px 28px rgba(15,30,20,.24);font-size:.82rem;line-height:1.35;font-weight:600;white-space:normal;overflow-wrap:anywhere;pointer-events:none}
    [data-help]::before{content:"";display:none;position:absolute;left:12px;bottom:calc(100% + 2px);z-index:51;border:6px solid transparent;border-top-color:#102017;pointer-events:none}
    [data-help]:hover::after,[data-help]:focus::after,[data-help]:focus-within::after,[data-help].help-open::after{display:block}
    [data-help]:hover::before,[data-help]:focus::before,[data-help]:focus-within::before,[data-help].help-open::before{display:block}
    .help-text{position:absolute;width:1px;height:1px;margin:-1px;padding:0;overflow:hidden;clip:rect(0 0 0 0);clip-path:inset(50%);border:0;white-space:nowrap}
    button{width:max-content;padding:10px 14px;border:0;border-radius:9px;background:var(--brand);color:#fff;font-weight:850;cursor:pointer}
    button.secondary{background:#e8efe8;color:var(--ink);border:1px solid var(--line)}
    .wizard-actions{display:flex;justify-content:space-between;gap:10px;padding:14px 18px;border-top:1px solid var(--line);background:#f8fbf5}
    .presets{display:flex;gap:8px;flex-wrap:wrap}
    .presets button{background:#102017;color:#dcffd6}
    .guided-assistant{display:grid;gap:12px;border-color:#a7c8b2;background:#f7fcf5}
    .guided-assistant textarea{min-height:110px}
    .guided-thread{display:grid;gap:8px;min-height:38px}
    .guided-message{border-left:4px solid var(--brand);background:#fff;padding:9px 10px;border-radius:8px;color:#334137}
    .guided-actions{display:flex;gap:8px;flex-wrap:wrap}
    .guided-actions button{background:#e8efe8;color:var(--ink);border:1px solid var(--line)}
    .guided-actions button.primary{background:var(--brand);color:#fff;border-color:var(--brand)}
    .expert-block{display:grid;gap:12px;margin-top:10px}
    .expert-row{border:1px solid var(--line);border-radius:10px;padding:10px;background:#fff}
    .expert-row-title{font-weight:850;margin:0 0 8px;color:var(--brand)}
    details{border:1px dashed #bfd0c3;border-radius:12px;padding:10px;background:#fbfdf9}
    summary{cursor:pointer;font-weight:800;color:var(--brand)}
    .side{position:sticky;top:18px;display:grid;gap:12px}
    .summary-list{display:grid;gap:8px;color:var(--muted)}
    .summary-list strong{color:var(--ink)}
    .status{border-left:4px solid var(--brand)}
    .goal-panel{border-left:4px solid #2563eb}
    .goal-grid{display:grid;gap:7px;color:var(--muted)}
    .goal-grid div{overflow-wrap:anywhere}
    .goal-grid strong{color:var(--ink)}
    .goal-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-top:12px}
    .goal-feedback{color:var(--muted);font-size:.9rem}
    .final-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-top:12px}
    .goal-preview-panel{border-left:4px solid #0f766e}
    .goal-preview-panel ul{margin:6px 0 12px;padding-left:20px}
    .goal-preview-panel li{margin-bottom:5px;overflow-wrap:anywhere}
    .issue{border-left:4px solid var(--bad)}
    .form-error-summary{margin:14px 18px 0;border:1px solid #f0b4ae;border-left:4px solid var(--bad);border-radius:10px;background:#fff7f5;padding:12px;color:#621b16}
    .form-error-summary strong{display:block;margin-bottom:4px}
    .form-error-summary p{margin:0 0 8px;color:#7c2d24}
    .form-error-summary ul{margin:0;padding-left:20px}
    .form-error-summary button{width:auto;padding:0;border:0;border-radius:0;background:transparent;color:#8a1f15;text-align:left;text-decoration:underline;font-weight:800}
    .field-error{margin-top:-4px;margin-bottom:10px;color:#9f241a;font-size:.86rem;font-weight:800}
    [aria-invalid="true"]{border-color:#b42318;box-shadow:0 0 0 3px rgba(180,35,24,.12)}
    code{background:#eef5ed;padding:2px 4px;border-radius:4px}
    .wizard-ready .hidden-final{display:none}
    .wizard-ready .wizard-step-final .hidden-final{display:inline-flex}
    @media(max-width:920px){main{padding:14px}header,form{grid-template-columns:1fr;display:grid}.side{position:static}.steps{grid-template-columns:repeat(2,minmax(0,1fr))}}
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>{{.Page.Titulo}}</h1>
      <p class="lead">{{index .HTML "nueva_app.wizard.lead"}}</p>
    </div>
    <nav class="topnav"><a href="/">{{index .HTML "nueva_app.wizard.nav.home"}}</a><a href="/ops">{{index .HTML "nueva_app.wizard.nav.ops"}}</a><a href="/autoprogramming">{{index .HTML "nueva_app.wizard.nav.autoprogramming"}}</a></nav>
  </header>
  <form method="post" action="/nueva-app" novalidate>
    <section class="wizard" id="nueva-app-wizard"
      data-summary-name="{{index .HTML "nueva_app.wizard.summary.name"}}"
      data-summary-type="{{index .HTML "nueva_app.wizard.summary.type"}}"
      data-summary-goal="{{index .HTML "nueva_app.wizard.summary.goal"}}"
      data-summary-platforms="{{index .HTML "nueva_app.wizard.summary.platforms"}}"
      data-summary-architecture="{{index .HTML "nueva_app.wizard.summary.architecture"}}"
      data-summary-data="{{index .HTML "nueva_app.wizard.summary.data"}}"
      data-summary-quality="{{index .HTML "nueva_app.wizard.summary.quality"}}"
      data-summary-autonomy="{{index .HTML "nueva_app.wizard.summary.autonomy"}}"
      data-summary-no-name="{{index .HTML "nueva_app.wizard.summary.no_name"}}"
      data-summary-pending="{{index .HTML "nueva_app.wizard.summary.pending"}}"
      data-summary-db-required="{{index .HTML "nueva_app.wizard.summary.db_required"}}"
      data-summary-no-db-required="{{index .HTML "nueva_app.wizard.summary.no_db_required"}}"
      data-validation-summary-title="{{index .HTML "nueva_app.validation.summary_title"}}"
      data-validation-summary-intro="{{index .HTML "nueva_app.validation.summary_intro"}}"
      data-validation-required="{{index .HTML "nueva_app.validation.required"}}"
      data-guided-msg-analyzed="{{index .HTML "nueva_app.wizard.guided_msg_analyzed"}}"
      data-guided-msg-mobile="{{index .HTML "nueva_app.wizard.guided_msg_mobile"}}"
      data-guided-msg-data="{{index .HTML "nueva_app.wizard.guided_msg_data"}}"
      data-guided-msg-maps="{{index .HTML "nueva_app.wizard.guided_msg_maps"}}"
      data-guided-msg-architecture="{{index .HTML "nueva_app.wizard.guided_msg_architecture"}}"
      data-guided-msg-quality="{{index .HTML "nueva_app.wizard.guided_msg_quality"}}"
      data-guided-msg-review="{{index .HTML "nueva_app.wizard.guided_msg_review"}}">
      <div class="steps" role="tablist" aria-label="{{index .HTML "nueva_app.wizard.steps_label"}}">
        <button class="step-tab active" type="button" role="tab" id="nueva-app-step-tab-0" aria-controls="nueva-app-step-panel-0" aria-selected="true" data-goto-step="0" data-help="{{index .Help "step.idea"}}">{{index .HTML "nueva_app.wizard.step.idea"}}</button>
        <button class="step-tab" type="button" role="tab" id="nueva-app-step-tab-1" aria-controls="nueva-app-step-panel-1" aria-selected="false" data-goto-step="1" data-help="{{index .Help "step.tipo"}}">{{index .HTML "nueva_app.wizard.step.tipo"}}</button>
        <button class="step-tab" type="button" role="tab" id="nueva-app-step-tab-2" aria-controls="nueva-app-step-panel-2" aria-selected="false" data-goto-step="2" data-help="{{index .Help "step.tecnologia"}}">{{index .HTML "nueva_app.wizard.step.tecnologia"}}</button>
        <button class="step-tab" type="button" role="tab" id="nueva-app-step-tab-3" aria-controls="nueva-app-step-panel-3" aria-selected="false" data-goto-step="3" data-help="{{index .Help "step.datos"}}">{{index .HTML "nueva_app.wizard.step.datos"}}</button>
        <button class="step-tab" type="button" role="tab" id="nueva-app-step-tab-4" aria-controls="nueva-app-step-panel-4" aria-selected="false" data-goto-step="4" data-help="{{index .Help "step.calidad"}}">{{index .HTML "nueva_app.wizard.step.calidad"}}</button>
        <button class="step-tab" type="button" role="tab" id="nueva-app-step-tab-5" aria-controls="nueva-app-step-panel-5" aria-selected="false" data-goto-step="5" data-help="{{index .Help "step.revisar"}}">{{index .HTML "nueva_app.wizard.step.revisar"}}</button>
      </div>
      <div class="step" data-step="0" role="tabpanel" id="nueva-app-step-panel-0" aria-labelledby="nueva-app-step-tab-0" tabindex="-1">
        <fieldset>
          <legend>{{index .HTML "nueva_app.wizard.idea_objetivo"}}</legend>
          <div class="guided-assistant" id="guided-assistant">
            <p class="expert-row-title">{{index .HTML "nueva_app.wizard.guided_title"}}</p>
            <label data-help="{{index .Help "objetivo"}}">{{index .HTML "nueva_app.wizard.guided_need"}}<textarea id="guided-need" placeholder="{{index .HTML "nueva_app.wizard.placeholder.objetivo"}}"></textarea></label>
            <div class="guided-actions"><button class="primary" type="button" data-guided-action="analyze" data-help="{{index .Help "guided.analyze"}}">{{index .HTML "nueva_app.wizard.guided_analyze"}}</button><button type="button" data-guided-action="review" data-help="{{index .Help "guided.review"}}">{{index .HTML "nueva_app.wizard.guided_review"}}</button></div>
            <div class="guided-thread" id="guided-thread" aria-live="polite"></div>
            <div id="guided-followups" hidden>
              <p class="expert-row-title">{{index .HTML "nueva_app.wizard.guided_followups"}}</p>
              <div class="guided-actions">
                <button type="button" data-guided-action="mobile_both" data-help="{{index .Help "guided.mobile_both"}}">{{index .HTML "nueva_app.wizard.guided_mobile_both"}}</button>
                <button type="button" data-guided-action="mobile_ios" data-help="{{index .Help "guided.mobile_ios"}}">{{index .HTML "nueva_app.wizard.guided_mobile_ios"}}</button>
                <button type="button" data-guided-action="mobile_android" data-help="{{index .Help "guided.mobile_android"}}">{{index .HTML "nueva_app.wizard.guided_mobile_android"}}</button>
                <button type="button" data-guided-action="data_external" data-help="{{index .Help "guided.data_external"}}">{{index .HTML "nueva_app.wizard.guided_data_external"}}</button>
                <button type="button" data-guided-action="data_management" data-help="{{index .Help "guided.data_management"}}">{{index .HTML "nueva_app.wizard.guided_data_management"}}</button>
                <button type="button" data-guided-action="maps_generic" data-help="{{index .Help "guided.maps_generic"}}">{{index .HTML "nueva_app.wizard.guided_maps_generic"}}</button>
                <button type="button" data-guided-action="maps_osm" data-help="{{index .Help "guided.maps_osm"}}">{{index .HTML "nueva_app.wizard.guided_maps_osm"}}</button>
                <button type="button" data-guided-action="architecture_default" data-help="{{index .Help "guided.architecture_default"}}">{{index .HTML "nueva_app.wizard.guided_architecture_default"}}</button>
                <button type="button" data-guided-action="architecture_event" data-help="{{index .Help "guided.architecture_event"}}">{{index .HTML "nueva_app.wizard.guided_architecture_event"}}</button>
                <button type="button" data-guided-action="architecture_modular" data-help="{{index .Help "guided.architecture_modular"}}">{{index .HTML "nueva_app.wizard.guided_architecture_modular"}}</button>
                <button type="button" data-guided-action="quality_public" data-help="{{index .Help "guided.quality_public"}}">{{index .HTML "nueva_app.wizard.guided_quality_public"}}</button>
              </div>
            </div>
          </div>
          <div class="presets" aria-label="{{index .HTML "nueva_app.wizard.presets"}}">
            <button type="button" data-preset="webapp" data-help="{{index .Help "preset.webapp"}}">{{index .HTML "nueva_app.wizard.preset.webapp"}}</button>
            <button type="button" data-preset="api" data-help="{{index .Help "preset.api"}}">{{index .HTML "nueva_app.wizard.preset.api"}}</button>
            <button type="button" data-preset="ops" data-help="{{index .Help "preset.ops"}}">{{index .HTML "nueva_app.wizard.preset.ops"}}</button>
          </div>
          <div class="grid">
            <label data-help="{{index .Help "nombre"}}">{{index .Labels "nombre"}}<input name="nombre" data-required="true" aria-required="true" data-label="{{index .Labels "nombre"}}" autocomplete="off"></label>
            <label data-help="{{index .Help "tipo_app"}}">{{index .Labels "tipo_app"}}<select name="tipo_app" data-required="true" aria-required="true" data-label="{{index .Labels "tipo_app"}}"><option value="web">web</option><option value="api">api</option><option value="cli">cli</option><option value="desktop">desktop</option><option value="mobile">mobile</option><option value="automation">automation</option><option value="data">data</option><option value="plugin">plugin</option><option value="mixed">mixed</option></select></label>
          </div>
          <label data-help="{{index .Help "objetivo"}}">{{index .Labels "objetivo"}}<textarea name="objetivo" data-required="true" aria-required="true" data-label="{{index .Labels "objetivo"}}" placeholder="{{index .HTML "nueva_app.wizard.placeholder.objetivo"}}"></textarea></label>
          <label data-help="{{index .Help "descripcion"}}">{{index .Labels "descripcion"}}<textarea name="descripcion" placeholder="{{index .HTML "nueva_app.wizard.placeholder.descripcion"}}"></textarea></label>
          <details><summary>{{index .HTML "nueva_app.wizard.identidad_avanzada"}}</summary><div class="grid">
            <label data-help="{{index .Help "request_id"}}">{{index .Labels "request_id"}}<input name="request_id" autocomplete="off"></label>
            <label data-help="{{index .Help "locale"}}">{{index .Labels "locale"}}<select name="locale">{{range .Page.Opciones.Locales}}<option value="{{.Valor}}">{{.Label}}</option>{{end}}</select></label>
            <label data-help="{{index .Help "request_kind"}}">{{index .Labels "request_kind"}}<select name="request_kind"><option value="crear_app_completa">crear_app_completa</option><option value="documentar_app">documentar_app</option><option value="analizar_app">analizar_app</option><option value="brainstorming_arquitectura">brainstorming_arquitectura</option><option value="planificar_app">planificar_app</option><option value="programar_modulo">programar_modulo</option><option value="modificar_app_existente">modificar_app_existente</option><option value="revisar_codigo">revisar_codigo</option><option value="pruebas_y_validacion">pruebas_y_validacion</option><option value="seguridad">seguridad</option><option value="deploy">deploy</option><option value="operacion_soporte">operacion_soporte</option><option value="integracion_externa">integracion_externa</option><option value="i18n_l10n">i18n_l10n</option><option value="migracion_refactor">migracion_refactor</option><option value="investigacion_tecnica">investigacion_tecnica</option></select></label>
            <label data-help="{{index .Help "execution_mode"}}">{{index .Labels "execution_mode"}}<select name="execution_mode"><option value="normal">normal</option><option value="debug">debug</option></select></label>
          </div>
          <details><summary>{{index .HTML "nueva_app.wizard.compatibilidad_historica"}}</summary>
            <p class="expert-row-title">{{index .HTML "nueva_app.wizard.compatibilidad_legacy_intro"}}</p>
            <div class="checks"><label data-help="{{index .Help "director_execution_mode"}}"><input type="checkbox" name="director_execution_mode" value="legacy_director_loop">{{index .HTML "nueva_app.wizard.force_legacy_director_loop"}}</label></div>
          </details>
          <input type="hidden" name="director_execution_mode" value="">
          </details>
        </fieldset>
      </div>
      <div class="step" data-step="1" role="tabpanel" id="nueva-app-step-panel-1" aria-labelledby="nueva-app-step-tab-1" tabindex="-1">
        <fieldset><legend>{{index .HTML "nueva_app.wizard.plataformas_origen"}}</legend>
          <div class="checks"><label data-help="{{index .Help "plataformas.web"}}"><input type="checkbox" name="plataformas" value="web">web</label><label data-help="{{index .Help "plataformas.mobile"}}"><input type="checkbox" name="plataformas" value="mobile">mobile</label><label data-help="{{index .Help "plataformas.desktop"}}"><input type="checkbox" name="plataformas" value="desktop">desktop</label><label data-help="{{index .Help "plataformas.api"}}"><input type="checkbox" name="plataformas" value="api">api</label></div>
          <details open><summary>{{index .HTML "nueva_app.wizard.origen_proyecto"}}</summary><div class="grid">
            <label data-help="{{index .Help "project_source.kind"}}">{{index .Labels "project_source.kind"}}<select name="project_source.kind"><option value=""></option><option value="new">new</option><option value="github">github</option><option value="local_path">local_path</option></select></label>
            <label data-help="{{index .Help "project_source.git_url"}}">{{index .Labels "project_source.git_url"}}<input name="project_source.git_url"></label>
            <label data-help="{{index .Help "project_source.branch"}}">{{index .Labels "project_source.branch"}}<input name="project_source.branch"></label>
            <label data-help="{{index .Help "project_source.project_ref"}}">{{index .Labels "project_source.project_ref"}}<input name="project_source.project_ref"></label>
            <label data-help="{{index .Help "project_source.local_path"}}">{{index .Labels "project_source.local_path"}}<input name="project_source.local_path"></label>
          </div></details>
        </fieldset>
      </div>
      <div class="step" data-step="2" role="tabpanel" id="nueva-app-step-panel-2" aria-labelledby="nueva-app-step-tab-2" tabindex="-1">
        <fieldset><legend>{{index .Labels "preferencias_tecnicas"}}</legend><div class="grid">
          <label data-help="{{index .Help "preferencias_tecnicas.arquitectura"}}">{{index .Labels "preferencias_tecnicas.arquitectura"}}<select name="preferencias_tecnicas.arquitectura"><option value="">sin_preferencia</option><option value="hexagonal">hexagonal</option><option value="clean_architecture">clean_architecture</option><option value="onion">onion</option><option value="modular_monolith">modular_monolith</option><option value="layered">layered</option><option value="event_driven">event_driven</option><option value="microservices">microservices</option><option value="serverless">serverless</option><option value="plugin_based">plugin_based</option><option value="data_pipeline">data_pipeline</option></select></label>
          <label data-help="{{index .Help "preferencias_tecnicas.lenguaje"}}">{{index .Labels "preferencias_tecnicas.lenguaje"}}<input name="preferencias_tecnicas.lenguaje" placeholder="go, typescript..."></label>
          <label data-help="{{index .Help "preferencias_tecnicas.framework"}}">{{index .Labels "preferencias_tecnicas.framework"}}<input name="preferencias_tecnicas.framework"></label>
          <label data-help="{{index .Help "preferencias_tecnicas.preferencias"}}">{{index .Labels "preferencias_tecnicas.preferencias"}}<input name="preferencias_tecnicas.preferencias"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "i18n"}}</legend><div class="grid">
          <label data-help="{{index .Help "i18n.enabled"}}"><span>{{index .Labels "i18n.enabled"}}</span><select name="i18n.enabled"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "i18n.default_locale"}}">{{index .Labels "i18n.default_locale"}}<input name="i18n.default_locale" value="{{.Page.Locale}}"></label>
          <label data-help="{{index .Help "i18n.locales"}}">{{index .Labels "i18n.locales"}}<input name="i18n.locales" placeholder="es-ES,en-US"></label>
          <label data-help="{{index .Help "i18n.justificacion"}}">{{index .Labels "i18n.justificacion"}}<input name="i18n.justificacion"></label>
        </div></fieldset>
      </div>
      <div class="step" data-step="3" role="tabpanel" id="nueva-app-step-panel-3" aria-labelledby="nueva-app-step-tab-3" tabindex="-1">
        <fieldset><legend>{{index .Labels "datos"}}</legend><div class="grid">
          <label data-help="{{index .Help "datos.db_required"}}"><span>{{index .Labels "datos.db_required"}}</span><select name="datos.db_required"><option value="false">false</option><option value="true">true</option></select></label>
          <label data-help="{{index .Help "datos.necesidad_funcional"}}">{{index .Labels "datos.necesidad_funcional"}}<input name="datos.necesidad_funcional"></label>
          <label data-help="{{index .Help "datos.tipos_datos"}}">{{index .Labels "datos.tipos_datos"}}<input name="datos.tipos_datos" placeholder="usuarios,eventos"></label>
          <label data-help="{{index .Help "datos.sensibilidad"}}">{{index .Labels "datos.sensibilidad"}}<input name="datos.sensibilidad"></label>
          <label data-help="{{index .Help "datos.retencion"}}">{{index .Labels "datos.retencion"}}<input name="datos.retencion"></label>
        </div>
        <details><summary>{{index .HTML "nueva_app.wizard.datos_experto"}}</summary><div class="expert-block">
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.dato_1"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.tipos_detallados.nombre"}}">{{index .Labels "datos.tipos_detallados.0.nombre"}}<input name="datos.tipos_detallados.0.nombre"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.proposito"}}">{{index .Labels "datos.tipos_detallados.0.proposito"}}<input name="datos.tipos_detallados.0.proposito"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.sensibilidad"}}">{{index .Labels "datos.tipos_detallados.0.sensibilidad"}}<input name="datos.tipos_detallados.0.sensibilidad"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.retencion"}}">{{index .Labels "datos.tipos_detallados.0.retencion"}}<input name="datos.tipos_detallados.0.retencion"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.volumen"}}">{{index .Labels "datos.tipos_detallados.0.volumen"}}<input name="datos.tipos_detallados.0.volumen"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.restricciones"}}">{{index .Labels "datos.tipos_detallados.0.restricciones"}}<input name="datos.tipos_detallados.0.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.dato_2"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.tipos_detallados.nombre"}}">{{index .Labels "datos.tipos_detallados.0.nombre"}}<input name="datos.tipos_detallados.1.nombre"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.proposito"}}">{{index .Labels "datos.tipos_detallados.0.proposito"}}<input name="datos.tipos_detallados.1.proposito"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.sensibilidad"}}">{{index .Labels "datos.tipos_detallados.0.sensibilidad"}}<input name="datos.tipos_detallados.1.sensibilidad"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.retencion"}}">{{index .Labels "datos.tipos_detallados.0.retencion"}}<input name="datos.tipos_detallados.1.retencion"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.volumen"}}">{{index .Labels "datos.tipos_detallados.0.volumen"}}<input name="datos.tipos_detallados.1.volumen"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.restricciones"}}">{{index .Labels "datos.tipos_detallados.0.restricciones"}}<input name="datos.tipos_detallados.1.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.dato_3"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.tipos_detallados.nombre"}}">{{index .Labels "datos.tipos_detallados.0.nombre"}}<input name="datos.tipos_detallados.2.nombre"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.proposito"}}">{{index .Labels "datos.tipos_detallados.0.proposito"}}<input name="datos.tipos_detallados.2.proposito"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.sensibilidad"}}">{{index .Labels "datos.tipos_detallados.0.sensibilidad"}}<input name="datos.tipos_detallados.2.sensibilidad"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.retencion"}}">{{index .Labels "datos.tipos_detallados.0.retencion"}}<input name="datos.tipos_detallados.2.retencion"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.volumen"}}">{{index .Labels "datos.tipos_detallados.0.volumen"}}<input name="datos.tipos_detallados.2.volumen"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.restricciones"}}">{{index .Labels "datos.tipos_detallados.0.restricciones"}}<input name="datos.tipos_detallados.2.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.dato_4"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.tipos_detallados.nombre"}}">{{index .Labels "datos.tipos_detallados.0.nombre"}}<input name="datos.tipos_detallados.3.nombre"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.proposito"}}">{{index .Labels "datos.tipos_detallados.0.proposito"}}<input name="datos.tipos_detallados.3.proposito"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.sensibilidad"}}">{{index .Labels "datos.tipos_detallados.0.sensibilidad"}}<input name="datos.tipos_detallados.3.sensibilidad"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.retencion"}}">{{index .Labels "datos.tipos_detallados.0.retencion"}}<input name="datos.tipos_detallados.3.retencion"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.volumen"}}">{{index .Labels "datos.tipos_detallados.0.volumen"}}<input name="datos.tipos_detallados.3.volumen"></label>
            <label data-help="{{index .Help "datos.tipos_detallados.restricciones"}}">{{index .Labels "datos.tipos_detallados.0.restricciones"}}<input name="datos.tipos_detallados.3.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.almacenamiento_1"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.storage.tipo"}}">{{index .Labels "datos.storage.0.tipo"}}<select name="datos.storage.0.tipo"><option value=""></option>{{range .StorageTypes}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>
            <label data-help="{{index .Help "datos.storage.proposito"}}">{{index .Labels "datos.storage.0.proposito"}}<input name="datos.storage.0.proposito"></label>
            <label data-help="{{index .Help "datos.storage.requerido"}}"><span>{{index .Labels "datos.storage.0.requerido"}}</span><select name="datos.storage.0.requerido"><option value="false">false</option><option value="true">true</option></select></label>
            <label data-help="{{index .Help "datos.storage.restricciones"}}">{{index .Labels "datos.storage.0.restricciones"}}<input name="datos.storage.0.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.almacenamiento_2"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.storage.tipo"}}">{{index .Labels "datos.storage.0.tipo"}}<select name="datos.storage.1.tipo"><option value=""></option>{{range .StorageTypes}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>
            <label data-help="{{index .Help "datos.storage.proposito"}}">{{index .Labels "datos.storage.0.proposito"}}<input name="datos.storage.1.proposito"></label>
            <label data-help="{{index .Help "datos.storage.requerido"}}"><span>{{index .Labels "datos.storage.0.requerido"}}</span><select name="datos.storage.1.requerido"><option value="false">false</option><option value="true">true</option></select></label>
            <label data-help="{{index .Help "datos.storage.restricciones"}}">{{index .Labels "datos.storage.0.restricciones"}}<input name="datos.storage.1.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.almacenamiento_3"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.storage.tipo"}}">{{index .Labels "datos.storage.0.tipo"}}<select name="datos.storage.2.tipo"><option value=""></option>{{range .StorageTypes}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>
            <label data-help="{{index .Help "datos.storage.proposito"}}">{{index .Labels "datos.storage.0.proposito"}}<input name="datos.storage.2.proposito"></label>
            <label data-help="{{index .Help "datos.storage.requerido"}}"><span>{{index .Labels "datos.storage.0.requerido"}}</span><select name="datos.storage.2.requerido"><option value="false">false</option><option value="true">true</option></select></label>
            <label data-help="{{index .Help "datos.storage.restricciones"}}">{{index .Labels "datos.storage.0.restricciones"}}<input name="datos.storage.2.restricciones"></label>
          </div></div>
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.almacenamiento_4"}}</p><div class="grid">
            <label data-help="{{index .Help "datos.storage.tipo"}}">{{index .Labels "datos.storage.0.tipo"}}<select name="datos.storage.3.tipo"><option value=""></option>{{range .StorageTypes}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>
            <label data-help="{{index .Help "datos.storage.proposito"}}">{{index .Labels "datos.storage.0.proposito"}}<input name="datos.storage.3.proposito"></label>
            <label data-help="{{index .Help "datos.storage.requerido"}}"><span>{{index .Labels "datos.storage.0.requerido"}}</span><select name="datos.storage.3.requerido"><option value="false">false</option><option value="true">true</option></select></label>
            <label data-help="{{index .Help "datos.storage.restricciones"}}">{{index .Labels "datos.storage.0.restricciones"}}<input name="datos.storage.3.restricciones"></label>
          </div></div>
        </div></details></fieldset>
        <fieldset><legend>{{index .Labels "integraciones"}}</legend><div class="expert-block">
          <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.integracion_1"}}</p><div class="grid">
            <label data-help="{{index .Help "integraciones.0.tipo"}}">{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.0.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option><option value="maps">maps</option><option value="file_import">file_import</option><option value="payments">payments</option><option value="auth">auth</option><option value="analytics">analytics</option><option value="search">search</option><option value="notifications">notifications</option></select></label>
            <label data-help="{{index .Help "integraciones.0.nombre"}}">{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.0.nombre"></label>
            <label data-help="{{index .Help "integraciones.0.proposito"}}">{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.0.proposito"></label>
            <label data-help="{{index .Help "integraciones.0.requerido"}}"><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.0.requerido"><option value="false">false</option><option value="true">true</option></select></label>
            <label data-help="{{index .Help "integraciones.0.restricciones"}}">{{index .Labels "integraciones.0.restricciones"}}<input name="integraciones.0.restricciones"></label>
          </div></div>
          <details><summary>{{index .HTML "nueva_app.wizard.integraciones_adicionales"}}</summary><div class="expert-block">
            <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.integracion_2"}}</p><div class="grid">
              <label data-help="{{index .Help "integraciones.0.tipo"}}">{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.1.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option><option value="maps">maps</option><option value="file_import">file_import</option><option value="payments">payments</option><option value="auth">auth</option><option value="analytics">analytics</option><option value="search">search</option><option value="notifications">notifications</option></select></label>
              <label data-help="{{index .Help "integraciones.0.nombre"}}">{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.1.nombre"></label>
              <label data-help="{{index .Help "integraciones.0.proposito"}}">{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.1.proposito"></label>
              <label data-help="{{index .Help "integraciones.0.requerido"}}"><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.1.requerido"><option value="false">false</option><option value="true">true</option></select></label>
              <label data-help="{{index .Help "integraciones.0.restricciones"}}">{{index .Labels "integraciones.0.restricciones"}}<input name="integraciones.1.restricciones"></label>
            </div></div>
            <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.integracion_3"}}</p><div class="grid">
              <label data-help="{{index .Help "integraciones.0.tipo"}}">{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.2.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option><option value="maps">maps</option><option value="file_import">file_import</option><option value="payments">payments</option><option value="auth">auth</option><option value="analytics">analytics</option><option value="search">search</option><option value="notifications">notifications</option></select></label>
              <label data-help="{{index .Help "integraciones.0.nombre"}}">{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.2.nombre"></label>
              <label data-help="{{index .Help "integraciones.0.proposito"}}">{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.2.proposito"></label>
              <label data-help="{{index .Help "integraciones.0.requerido"}}"><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.2.requerido"><option value="false">false</option><option value="true">true</option></select></label>
              <label data-help="{{index .Help "integraciones.0.restricciones"}}">{{index .Labels "integraciones.0.restricciones"}}<input name="integraciones.2.restricciones"></label>
            </div></div>
            <div class="expert-row"><p class="expert-row-title">{{index .HTML "nueva_app.wizard.integracion_4"}}</p><div class="grid">
              <label data-help="{{index .Help "integraciones.0.tipo"}}">{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.3.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option><option value="maps">maps</option><option value="file_import">file_import</option><option value="payments">payments</option><option value="auth">auth</option><option value="analytics">analytics</option><option value="search">search</option><option value="notifications">notifications</option></select></label>
              <label data-help="{{index .Help "integraciones.0.nombre"}}">{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.3.nombre"></label>
              <label data-help="{{index .Help "integraciones.0.proposito"}}">{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.3.proposito"></label>
              <label data-help="{{index .Help "integraciones.0.requerido"}}"><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.3.requerido"><option value="false">false</option><option value="true">true</option></select></label>
              <label data-help="{{index .Help "integraciones.0.restricciones"}}">{{index .Labels "integraciones.0.restricciones"}}<input name="integraciones.3.restricciones"></label>
            </div></div>
          </div></details>
        </div></fieldset>
      </div>
      <div class="step" data-step="4" role="tabpanel" id="nueva-app-step-panel-4" aria-labelledby="nueva-app-step-tab-4" tabindex="-1">
        <fieldset><legend>{{index .Labels "calidad"}}</legend><div class="grid">
          <label data-help="{{index .Help "calidad.pruebas"}}">{{index .Labels "calidad.pruebas"}}<select name="calidad.pruebas"><option value="basica">basica</option><option value="media">media</option><option value="alta">alta</option></select></label>
          <label data-help="{{index .Help "calidad.accesibilidad"}}">{{index .Labels "calidad.accesibilidad"}}<select name="calidad.accesibilidad"><option value="basica">basica</option><option value="normal">normal</option><option value="wcag_aa">wcag_aa</option><option value="no_aplica">no_aplica</option></select></label>
          <label data-help="{{index .Help "calidad.accesibilidad_opciones"}}">{{index .Labels "calidad.accesibilidad_opciones"}}<input name="calidad.accesibilidad_opciones" placeholder="normal,wcag_aa"></label>
          <label data-help="{{index .Help "calidad.observabilidad"}}"><span>{{index .Labels "calidad.observabilidad"}}</span><select name="calidad.observabilidad"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "calidad.compliance"}}">{{index .Labels "calidad.compliance"}}<input name="calidad.compliance"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "deploy"}}</legend><div class="grid">
          <label data-help="{{index .Help "deploy.target"}}">{{index .Labels "deploy.target"}}<select name="deploy.target"><option value="sin_preferencia">sin_preferencia</option><option value="local">local</option><option value="contenedor">contenedor</option><option value="paas">paas</option><option value="serverless">serverless</option><option value="kubernetes">kubernetes</option><option value="desktop">desktop</option><option value="mobile_store">mobile_store</option></select></label>
          <label data-help="{{index .Help "deploy.restricciones"}}">{{index .Labels "deploy.restricciones"}}<input name="deploy.restricciones"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "documentacion"}}</legend><div class="grid">
          <label data-help="{{index .Help "documentacion.usuario"}}"><span>{{index .Labels "documentacion.usuario"}}</span><select name="documentacion.usuario"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "documentacion.desarrollo"}}"><span>{{index .Labels "documentacion.desarrollo"}}</span><select name="documentacion.desarrollo"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "documentacion.sistemas"}}"><span>{{index .Labels "documentacion.sistemas"}}</span><select name="documentacion.sistemas"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "documentacion.locales"}}">{{index .Labels "documentacion.locales"}}<input name="documentacion.locales" placeholder="es-ES,en-US"></label>
        </div></fieldset>
      </div>
      <div class="step" data-step="5" role="tabpanel" id="nueva-app-step-panel-5" aria-labelledby="nueva-app-step-tab-5" tabindex="-1">
        <fieldset><legend>{{index .Labels "agentes"}}</legend><div class="grid">
          <label data-help="{{index .Help "agentes.revision_humana"}}"><span>{{index .Labels "agentes.revision_humana"}}</span><select name="agentes.revision_humana"><option value="true">true</option><option value="false">false</option></select></label>
          <label data-help="{{index .Help "agentes.autonomia"}}">{{index .Labels "agentes.autonomia"}}<select name="agentes.autonomia"><option value="media">media</option><option value="baja">baja</option><option value="alta">alta</option></select></label>
          <label data-help="{{index .Help "agentes.preferencias"}}">{{index .Labels "agentes.preferencias"}}<input name="agentes.preferencias"></label>
          <label data-help="{{index .Help "restricciones"}}">{{index .Labels "restricciones"}}<input name="restricciones"></label>
        </div></fieldset>
        <fieldset><legend>{{index .HTML "nueva_app.wizard.revision_final"}}</legend><div id="wizard-final-summary" class="summary-list"></div><div class="final-actions"><button class="secondary hidden-final" type="submit" name="nueva_app_action" value="preview_goal" data-help="{{index .Help "action.preview"}}">{{.Page.Formulario.Acciones.Preview}}</button><button class="hidden-final" type="submit" name="nueva_app_action" value="launch" data-help="{{index .Help "action.launch"}}">{{.Page.Formulario.Acciones.Submit}}</button></div></fieldset>
      </div>
      <div id="wizard-errors" class="form-error-summary" role="alert" aria-live="polite" hidden></div>
      <div class="wizard-actions">
        <button class="secondary" type="button" data-prev-step data-help="{{index .Help "nav.back"}}">{{index .HTML "nueva_app.wizard.back"}}</button>
        <button type="button" data-next-step data-help="{{index .Help "nav.next"}}">{{index .HTML "nueva_app.wizard.next"}}</button>
      </div>
    </section>
    <aside class="side">
      <section class="panel"><h2>{{index .HTML "nueva_app.wizard.resumen_vivo"}}</h2><div id="wizard-summary" class="summary-list"></div></section>
      <section class="panel status"><h2>{{index .HTML "nueva_app.html.resultado"}}</h2><p>{{.Page.Textos.Estado}}</p>{{if .Page.ViewModel.RequestID}}<p><code>{{.Page.ViewModel.RequestID}}</code></p>{{end}}</section>
      {{with .Page.ViewModel.Director}}<section class="panel goal-panel" data-goal-panel data-goal-auto-poll="{{if and .RunRef .GoalRef}}true{{else}}false{{end}}" data-goal-poll-interval-ms="5000" data-goal-max-polls="60" data-run-ref="{{.RunRef}}" data-goal-ref="{{.GoalRef}}" data-goal-updating="{{index $.HTML "nueva_app.goal.updating"}}" data-goal-updated="{{index $.HTML "nueva_app.goal.updated"}}" data-goal-error="{{index $.HTML "nueva_app.goal.error"}}">
        <h2>{{index $.HTML "nueva_app.goal.titulo"}}</h2>
        <div class="goal-grid">
          {{if .RunRef}}<div><strong>{{index $.HTML "nueva_app.goal.run_ref"}}:</strong> <code data-goal-run-ref>{{.RunRef}}</code></div>{{end}}
          {{if .DirectorExecutionMode}}<div><strong>{{index $.HTML "nueva_app.goal.director_execution_mode"}}:</strong> <code>{{.DirectorExecutionMode}}</code></div>{{end}}
          {{if .GoalRef}}<div><strong>{{index $.HTML "nueva_app.goal.goal_ref"}}:</strong> <code data-goal-goal-ref>{{.GoalRef}}</code></div>{{end}}
          {{if .ExternalGoalRef}}<div><strong>{{index $.HTML "nueva_app.goal.external_goal_ref"}}:</strong> <code data-goal-external-ref>{{.ExternalGoalRef}}</code></div>{{end}}
          {{if .GoalStatus}}<div><strong>{{index $.HTML "nueva_app.goal.goal_status"}}:</strong> <code data-goal-status>{{.GoalStatus}}</code></div>{{end}}
          <div data-goal-run-status-row hidden><strong>{{index $.HTML "nueva_app.goal.run_status"}}:</strong> <code data-goal-run-status></code></div>
          <div data-goal-closure-status-row hidden><strong>{{index $.HTML "nueva_app.goal.closure_status"}}:</strong> <code data-goal-closure-status></code></div>
        </div>
        {{if and .RunRef .GoalRef}}<div class="goal-actions"><button class="secondary" type="button" data-goal-observe>{{index $.HTML "nueva_app.goal.update"}}</button><span class="goal-feedback" data-goal-feedback aria-live="polite"></span></div>{{end}}
      </section>{{end}}
      {{with .Page.ViewModel.GoalPreview}}<section class="panel goal-preview-panel">
        <h2>{{index $.HTML "nueva_app.goal_preview.titulo"}}</h2>
        <div class="goal-grid">
          {{if .RunRef}}<div><strong>{{index $.HTML "nueva_app.goal.run_ref"}}:</strong> <code>{{.RunRef}}</code></div>{{end}}
          {{if .GoalRef}}<div><strong>{{index $.HTML "nueva_app.goal.goal_ref"}}:</strong> <code>{{.GoalRef}}</code></div>{{end}}
          {{if .DirectorExecutionMode}}<div><strong>{{index $.HTML "nueva_app.goal.director_execution_mode"}}:</strong> <code>{{.DirectorExecutionMode}}</code></div>{{end}}
          {{if .Objective}}<div><strong>{{index $.HTML "nueva_app.goal_preview.objective"}}:</strong> {{.Objective}}</div>{{end}}
          <div><strong>{{index $.HTML "nueva_app.goal_preview.estimate"}}:</strong> {{.Estimate.TokenBudget}} {{index $.HTML "nueva_app.goal_preview.tokens"}} · {{.Estimate.MaxSubgoals}} {{index $.HTML "nueva_app.goal_preview.subgoals"}} · {{index $.HTML "nueva_app.goal_preview.cost_tier"}} <code>{{.Estimate.CostTier}}</code></div>
        </div>
        {{if .WriteSet}}<h3>{{index $.HTML "nueva_app.goal_preview.write_set"}}</h3><ul>{{range .WriteSet}}<li><code>{{.Path}}</code>{{if .Purpose}} · {{.Purpose}}{{end}}</li>{{end}}</ul>{{end}}
        {{if .RequiredTests}}<h3>{{index $.HTML "nueva_app.goal_preview.required_tests"}}</h3><ul>{{range .RequiredTests}}<li>{{if .TestRef}}<code>{{.TestRef}}</code>{{end}} {{.Command}}</li>{{end}}</ul>{{end}}
        {{if .AcceptanceCriteria}}<h3>{{index $.HTML "nueva_app.goal_preview.acceptance"}}</h3><ul>{{range .AcceptanceCriteria}}<li>{{.}}</li>{{end}}</ul>{{end}}
        {{if .ArtifactContracts}}<h3>{{index $.HTML "nueva_app.goal_preview.artifacts"}}</h3><ul>{{range .ArtifactContracts}}<li><code>{{.ArtifactRef}}</code> {{.ArtifactType}}</li>{{end}}</ul>{{end}}
      </section>{{end}}
      {{if .Page.ViewModel.ErroresPublicos}}<section class="panel issue"><h2>{{index .HTML "nueva_app.html.errores_publicos"}}</h2><ul>{{range .Page.ViewModel.ErroresPublicos}}<li><code>{{.Code}}</code> {{index $.Page.Textos.ErroresPublicos .Code}}</li>{{end}}</ul></section>{{end}}
      {{if .Page.ViewModel.ResumenApp.Nombre}}<section class="panel"><h2>{{index .HTML "nueva_app.html.resumen"}}</h2><p>{{.Page.ViewModel.ResumenApp.Nombre}} · {{.Page.ViewModel.ResumenApp.TipoApp}}</p><p>{{.Page.ViewModel.ResumenApp.Objetivo}}</p></section>{{end}}
      <section class="panel"><h2>{{.Page.Textos.BacklogPreview.Titulo}}</h2>{{if .Page.ViewModel.BacklogPreview.TotalMicrotareas}}<ol>{{range .Page.ViewModel.BacklogPreview.Microtareas}}<li><strong>{{.Key}}</strong> {{.Titulo}} <code>{{.ModuloFrontera}}</code></li>{{end}}</ol>{{else}}<p>{{index .HTML "nueva_app.html.backlog_vacio"}}</p>{{end}}</section>
    </aside>
  </form>
</main>
<script>
  (function(){
    const form=document.querySelector('form[action="/nueva-app"]');
    const wizard=document.getElementById('nueva-app-wizard');
    if(!form||!wizard)return;
    document.documentElement.classList.add('wizard-ready');
    let step=0;
    const steps=[...wizard.querySelectorAll('[data-step]')];
    const tabs=[...wizard.querySelectorAll('[data-goto-step]')];
    const prev=wizard.querySelector('[data-prev-step]');
    const next=wizard.querySelector('[data-next-step]');
    const errors=document.getElementById('wizard-errors');
    function field(name){return form.elements[name];}
    function val(name){const el=field(name);return el?String(el.value||'').trim():'';}
    function checked(name){return [...form.querySelectorAll('input[name="'+name+'"]:checked')].map(el=>el.value).join(', ');}
    function renderSummary(){
      const copy=wizard.dataset;
      const items=[
        [copy.summaryName,val('nombre')||copy.summaryNoName],
        [copy.summaryType,val('tipo_app')||'-'],
        [copy.summaryGoal,val('objetivo')||copy.summaryPending],
        [copy.summaryPlatforms,checked('plataformas')||'-'],
        [copy.summaryArchitecture,val('preferencias_tecnicas.arquitectura')||'hexagonal'],
        [copy.summaryData,val('datos.db_required')==='true'?copy.summaryDbRequired:copy.summaryNoDbRequired],
        [copy.summaryQuality,val('calidad.pruebas')||'-'],
        [copy.summaryAutonomy,val('agentes.autonomia')||'-']
      ];
      const html=items.map(i=>'<div><strong>'+escapeHTML(i[0])+':</strong> '+escapeHTML(i[1])+'</div>').join('');
      document.getElementById('wizard-summary').innerHTML=html;
      document.getElementById('wizard-final-summary').innerHTML=html;
    }
    function escapeHTML(v){return String(v).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
    function motionBehavior(){
      return window.matchMedia&&window.matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth';
    }
    function addDescribedBy(node,id){
      if(!node||!id)return;
      const described=(node.getAttribute('aria-describedby')||'').split(/\s+/).filter(Boolean);
      if(!described.includes(id)){described.push(id);}
      node.setAttribute('aria-describedby',described.join(' '));
    }
    function initAccessibleHelp(){
      const closeHelpBubbles=function(except){
        wizard.querySelectorAll('[data-help].help-open').forEach(node=>{
          if(node!==except)node.classList.remove('help-open');
        });
      };
      wizard.querySelectorAll('[data-help]').forEach((node,index)=>{
        const help=String(node.dataset.help||'').trim();
        if(!help)return;
        const id=node.id?node.id+'-help':'nueva-app-help-'+index;
        let helper=document.getElementById(id);
        if(!helper){
          helper=document.createElement('span');
          helper.id=id;
          helper.className='help-text';
          helper.textContent=help;
          node.appendChild(helper);
        }
        addDescribedBy(node,id);
        const target=node.matches('button,a,input,select,textarea')?node:node.querySelector('input,select,textarea,button,a');
        addDescribedBy(target,id);
        node.addEventListener('pointerdown',event=>{
          if(event.pointerType==='mouse')return;
          const opened=node.classList.contains('help-open');
          closeHelpBubbles(node);
          node.classList.toggle('help-open',!opened);
        });
        node.addEventListener('keydown',event=>{
          if(event.key==='Escape')node.classList.remove('help-open');
        });
      });
      document.addEventListener('pointerdown',event=>{
        if(!event.target.closest('[data-help]'))closeHelpBubbles();
      });
    }
    async function observeGoal(panelOrButton,options){
      options=options||{};
      const panel=panelOrButton&&panelOrButton.closest?panelOrButton.closest('[data-goal-panel]'):panelOrButton;
      if(!panel||!panel.dataset.runRef)return;
      const button=panel.querySelector('[data-goal-observe]');
      const feedback=panel.querySelector('[data-goal-feedback]');
      if(feedback)feedback.textContent=panel.dataset.goalUpdating||'';
      if(button&&!options.auto)button.disabled=true;
      try{
        const response=await fetch('/api/v0/apps/director/goal/observe',{
          method:'POST',
          headers:{'Content-Type':'application/json','Accept':'application/json','X-Correlation-ID':panel.dataset.runRef},
          body:JSON.stringify({run_ref:panel.dataset.runRef,requested_by:options.auto?'orquesta-web-nueva-app-auto':'orquesta-web-nueva-app'})
        });
        const result=await response.json();
        if(!response.ok||result.estado==='error'){
          const issue=result.errores_publicos&&result.errores_publicos[0];
          throw new Error((issue&&(issue.message||issue.code))||panel.dataset.goalError||'error');
        }
        setGoalText(panel,'[data-goal-run-ref]',result.run_ref);
        setGoalText(panel,'[data-goal-goal-ref]',result.goal_ref);
        setGoalText(panel,'[data-goal-external-ref]',result.external_goal_ref);
        setGoalText(panel,'[data-goal-status]',result.goal_status);
        setGoalRow(panel,'[data-goal-run-status-row]','[data-goal-run-status]',result.run_status);
        setGoalRow(panel,'[data-goal-closure-status-row]','[data-goal-closure-status]',result.closure_status);
        panel.dataset.goalLastStatus=result.goal_status||'';
        panel.dataset.runLastStatus=result.run_status||'';
        panel.dataset.closureAccepted=result.closure_accepted?'true':'false';
        if(feedback)feedback.textContent=panel.dataset.goalUpdated||'';
        return result;
      }catch(err){
        if(feedback)feedback.textContent=(panel.dataset.goalError||'')+(err&&err.message?': '+err.message:'');
        return null;
      }finally{
        if(button&&!options.auto)button.disabled=false;
      }
    }
    function setGoalText(panel,selector,value){
      const node=panel.querySelector(selector);
      if(node&&value)node.textContent=value;
    }
    function setGoalRow(panel,rowSelector,valueSelector,value){
      const row=panel.querySelector(rowSelector);
      const node=panel.querySelector(valueSelector);
      if(!row||!node||!value)return;
      node.textContent=value;
      row.hidden=false;
    }
    function goalObservationTerminal(panel,result){
      const runStatus=String((result&&result.run_status)||panel.dataset.runLastStatus||'').trim();
      const goalStatus=String((result&&result.goal_status)||panel.dataset.goalLastStatus||'').trim();
      if(runStatus==='cerrada'||runStatus==='bloqueada'||runStatus==='closed'||runStatus==='blocked')return true;
      return goalStatus==='complete'||goalStatus==='blocked'||goalStatus==='invalid'||panel.dataset.closureAccepted==='true';
    }
    function startGoalAutoPoll(){
      const panel=document.querySelector('[data-goal-panel][data-goal-auto-poll="true"]');
      if(!panel||!panel.dataset.runRef||panel.dataset.goalPollingStarted==='true')return;
      panel.dataset.goalPollingStarted='true';
      const interval=Math.max(1000,Number(panel.dataset.goalPollIntervalMs||5000));
      const maxPolls=Math.max(1,Number(panel.dataset.goalMaxPolls||60));
      let polls=0;
      async function tick(){
        if(polls>=maxPolls||goalObservationTerminal(panel,null))return;
        polls+=1;
        const result=await observeGoal(panel,{auto:true});
        if(goalObservationTerminal(panel,result))return;
        window.setTimeout(tick,interval);
      }
      window.setTimeout(tick,800);
    }
    function show(index,moveFocus){
      step=Math.max(0,Math.min(steps.length-1,index));
      steps.forEach((node,i)=>{node.hidden=i!==step;});
      tabs.forEach((node,i)=>{
        const active=i===step;
        node.classList.toggle('active',active);
        node.setAttribute('aria-selected',active?'true':'false');
        node.tabIndex=active?0:-1;
      });
      prev.disabled=step===0;
      next.hidden=step===steps.length-1;
      wizard.classList.toggle('wizard-step-final',step===steps.length-1);
      renderSummary();
      if(moveFocus&&steps[step]){
        setTimeout(()=>{steps[step].focus({preventScroll:true});steps[step].scrollIntoView({block:'start',behavior:motionBehavior()});},0);
      }
    }
    function handleTabKeydown(event,node){
      const current=tabs.indexOf(node);
      if(current<0)return;
      let nextIndex=current;
      if(event.key==='ArrowRight'||event.key==='ArrowDown')nextIndex=current+1;
      else if(event.key==='ArrowLeft'||event.key==='ArrowUp')nextIndex=current-1;
      else if(event.key==='Home')nextIndex=0;
      else if(event.key==='End')nextIndex=tabs.length-1;
      else return;
      event.preventDefault();
      nextIndex=(nextIndex+tabs.length)%tabs.length;
      show(nextIndex,false);
      tabs[nextIndex].focus({preventScroll:true});
    }
    function errorID(el){return 'field-error-'+String(el.name||'field').replace(/[^a-zA-Z0-9_-]/g,'-');}
    function requiredFields(){return [...form.querySelectorAll('[data-required="true"]')];}
    function fieldLabel(el){return el.dataset.label||el.name||'';}
    function clearFieldError(el){
      if(!el)return;
      const id=errorID(el);
      const current=document.getElementById(id);
      if(current)current.remove();
      el.removeAttribute('aria-invalid');
      const described=(el.getAttribute('aria-describedby')||'').split(/\s+/).filter(v=>v&&v!==id);
      if(described.length){el.setAttribute('aria-describedby',described.join(' '));}else{el.removeAttribute('aria-describedby');}
    }
    function setFieldError(el,message){
      clearFieldError(el);
      const id=errorID(el);
      el.setAttribute('aria-invalid','true');
      const described=(el.getAttribute('aria-describedby')||'').split(/\s+/).filter(Boolean);
      if(!described.includes(id)){described.push(id);}
      el.setAttribute('aria-describedby',described.join(' '));
      const node=document.createElement('div');
      node.className='field-error';
      node.id=id;
      node.textContent=message;
      const label=el.closest('label');
      if(label){label.insertAdjacentElement('afterend',node);}else{el.insertAdjacentElement('afterend',node);}
    }
    function focusField(el){
      const parentStep=el.closest('[data-step]');
      if(parentStep){show(Number(parentStep.dataset.step||0),false);}
      setTimeout(()=>{el.focus({preventScroll:true});el.scrollIntoView({block:'center',behavior:motionBehavior()});},0);
    }
    function renderErrors(invalid){
      if(!errors)return;
      if(!invalid.length){errors.hidden=true;errors.innerHTML='';return;}
      const copy=wizard.dataset;
      errors.hidden=false;
      errors.innerHTML='<strong>'+escapeHTML(copy.validationSummaryTitle||'')+'</strong><p>'+escapeHTML(copy.validationSummaryIntro||'')+'</p><ul>'+invalid.map(item=>'<li><button type="button" data-error-target="'+escapeHTML(item.name)+'">'+escapeHTML(item.label)+': '+escapeHTML(item.message)+'</button></li>').join('')+'</ul>';
    }
    function validateForm(){
      const invalid=[];
      requiredFields().forEach(el=>{
        clearFieldError(el);
        if(String(el.value||'').trim()===''){
          const item={name:el.name,label:fieldLabel(el),message:wizard.dataset.validationRequired||'Completa este campo.'};
          invalid.push(item);
          setFieldError(el,item.message);
        }
      });
      renderErrors(invalid);
      if(invalid.length){const first=field(invalid[0].name);if(first)focusField(first);return false;}
      return true;
    }
    function dispatchField(el){if(!el)return;el.dispatchEvent(new Event('input',{bubbles:true}));el.dispatchEvent(new Event('change',{bubbles:true}));}
    function setValue(name,value){const el=field(name);if(el){el.value=value;clearFieldError(el);dispatchField(el);}}
    function addCSV(name,values){
      const current=val(name).split(',').map(v=>v.trim()).filter(Boolean);
      values.forEach(value=>{if(value&&!current.includes(value)){current.push(value);}});
      setValue(name,current.join(', '));
    }
    function setChecked(value,on){form.querySelectorAll('input[name="plataformas"][value="'+value+'"]').forEach(el=>{el.checked=on;dispatchField(el);});}
    function guidedLog(message){
      const thread=document.getElementById('guided-thread');
      if(!thread||!message)return;
      const node=document.createElement('div');
      node.className='guided-message';
      node.textContent=message;
      thread.appendChild(node);
    }
    function hasOwn(obj,key){return Object.prototype.hasOwnProperty.call(obj||{},key);}
    function setMaybe(name,value){
      if(value===undefined||value===null)return;
      if(typeof value==='string'&&value.trim()==='')return;
      setValue(name,Array.isArray(value)?value.join(', '):String(value));
    }
    function setCSV(name,values){if(Array.isArray(values)&&values.length){setValue(name,values.join(', '));}}
    function setPlatforms(values){
      if(!Array.isArray(values)||!values.length)return;
      form.querySelectorAll('input[name="plataformas"]').forEach(el=>{el.checked=values.includes(el.value);dispatchField(el);});
    }
    function applyIndexed(prefix,rows,fields){
      if(!Array.isArray(rows))return;
      rows.forEach((row,index)=>{
        if(!row||index>3)return;
        fields.forEach(name=>{
          if(hasOwn(row,name)){setMaybe(prefix+'.'+index+'.'+name,row[name]);}
        });
      });
    }
    function applyGuidedForm(guidedForm){
      if(!guidedForm)return;
      setMaybe('nombre',guidedForm.nombre);
      setMaybe('objetivo',guidedForm.objetivo);
      setMaybe('descripcion',guidedForm.descripcion);
      setMaybe('tipo_app',guidedForm.tipo_app);
      setPlatforms(guidedForm.plataformas);
      setCSV('usuarios_objetivo',guidedForm.usuarios_objetivo);
      setCSV('restricciones',guidedForm.restricciones);
      const pref=guidedForm.preferencias_tecnicas||{};
      setMaybe('preferencias_tecnicas.lenguaje',pref.lenguaje);
      setMaybe('preferencias_tecnicas.framework',pref.framework);
      setMaybe('preferencias_tecnicas.arquitectura',pref.arquitectura);
      setCSV('preferencias_tecnicas.restricciones',pref.restricciones);
      setCSV('preferencias_tecnicas.preferencias',pref.preferencias);
      const datos=guidedForm.datos||{};
      if(hasOwn(datos,'db_required'))setMaybe('datos.db_required',datos.db_required);
      setMaybe('datos.necesidad_funcional',datos.necesidad_funcional);
      setCSV('datos.tipos_datos',datos.tipos_datos);
      setMaybe('datos.sensibilidad',datos.sensibilidad);
      setMaybe('datos.retencion',datos.retencion);
      applyIndexed('datos.tipos_detallados',datos.tipos_detallados,['nombre','proposito','sensibilidad','retencion','volumen','restricciones']);
      applyIndexed('datos.storage',datos.storage,['tipo','proposito','requerido','restricciones']);
      applyIndexed('integraciones',guidedForm.integraciones,['tipo','nombre','proposito','requerido','restricciones']);
      const calidad=guidedForm.calidad||{};
      setMaybe('calidad.pruebas',calidad.pruebas);
      setMaybe('calidad.accesibilidad',calidad.accesibilidad);
      setCSV('calidad.accesibilidad_opciones',calidad.accesibilidad_opciones);
      setCSV('calidad.compliance',calidad.compliance);
      if(hasOwn(calidad,'observabilidad'))setMaybe('calidad.observabilidad',calidad.observabilidad);
    }
    async function requestGuided(payload){
      try{
        const response=await fetch('/api/v0/apps/intake/guided-turn',{method:'POST',headers:{'Content-Type':'application/json','Accept':'application/json'},body:JSON.stringify(payload||{})});
        if(!response.ok)return null;
        return await response.json();
      }catch(_){return null;}
    }
    async function applyServerGuided(payload,message){
      const out=await requestGuided(Object.assign({locale:val('locale')||'es'},payload||{}));
      if(!out||!out.session||!out.session.form)return false;
      applyGuidedForm(out.session.form);
      document.getElementById('guided-followups').hidden=false;
      guidedLog(message);
      renderSummary();
      return true;
    }
    function titleFromNeed(text){
      const lower=text.toLowerCase();
      if(/piso|alquiler|vivienda|rent/.test(lower)){return 'Alquileres cercanos';}
      const words=text.replace(/[^\p{L}\p{N}\s]/gu,' ').split(/\s+/).filter(Boolean).slice(0,5).join(' ');
      return words||'Nueva app';
    }
    function configureRentalData(){
      setValue('datos.db_required','true');
      setValue('datos.necesidad_funcional','Gestionar y consultar pisos en alquiler, ubicaciones, favoritos y alertas.');
      setValue('datos.tipos_datos','pisos, alquileres, ubicaciones, favoritos, alertas');
      setValue('datos.tipos_detallados.0.nombre','Pisos en alquiler');
      setValue('datos.tipos_detallados.0.proposito','Mostrar pisos cercanos y sus datos principales.');
      setValue('datos.tipos_detallados.0.sensibilidad','publica');
      setValue('datos.tipos_detallados.0.retencion','mientras el anuncio este activo');
      setValue('datos.tipos_detallados.0.volumen','alto');
      setValue('datos.tipos_detallados.1.nombre','Usuarios y favoritos');
      setValue('datos.tipos_detallados.1.proposito','Guardar favoritos, busquedas y alertas.');
      setValue('datos.tipos_detallados.1.sensibilidad','personal');
      setValue('datos.storage.0.tipo','relacional');
      setValue('datos.storage.0.proposito','Consultas consistentes de anuncios, usuarios y favoritos.');
      setValue('datos.storage.0.requerido','true');
      setValue('datos.storage.1.tipo','busqueda');
      setValue('datos.storage.1.proposito','Filtrado por ubicacion, precio y preferencias.');
      guidedLog(wizard.dataset.guidedMsgData);
    }
    function configureMaps(preferOSM){
      setValue('integraciones.0.tipo','maps');
      setValue('integraciones.0.nombre','capacidad de mapas');
      setValue('integraciones.0.proposito','Mostrar ubicacion, cercania y rutas aproximadas sin acoplar el dominio a un proveedor.');
      setValue('integraciones.0.requerido','true');
      if(preferOSM){
        addCSV('integraciones.0.restricciones',['preferir OpenStreetMap si el adaptador autorizado lo soporta']);
        addCSV('preferencias_tecnicas.preferencias',['mapas: OpenStreetMap preferido por adaptador']);
      }
      guidedLog(wizard.dataset.guidedMsgMaps);
    }
    async function applyGuidedNeed(){
      const input=document.getElementById('guided-need');
      const text=input?String(input.value||'').trim():'';
      if(!text)return;
      if(await applyServerGuided({need:text},wizard.dataset.guidedMsgAnalyzed)){return;}
      if(!val('objetivo'))setValue('objetivo',text);
      if(!val('descripcion'))setValue('descripcion',text);
      if(!val('nombre'))setValue('nombre',titleFromNeed(text));
      const lower=text.toLowerCase();
      if(/m[oó]vil|mobile|android|ios|iphone|apple/.test(lower)){setValue('tipo_app','mobile');setChecked('mobile',true);}
      if(/api|backend|servicio/.test(lower)){setChecked('api',true);}
      if(/web|panel|gestion|gesti[oó]n/.test(lower)){setChecked('web',true);}
      if(/piso|pisos|alquiler|vivienda|rent/.test(lower)){configureRentalData();}
      if(/map|mapa|cerca|cercan|ubicaci[oó]n|geo|openstreet/.test(lower)){configureMaps(/openstreet/.test(lower));}
      document.getElementById('guided-followups').hidden=false;
      guidedLog(wizard.dataset.guidedMsgAnalyzed);
      renderSummary();
    }
    async function guidedAction(action){
      if(action==='analyze'){applyGuidedNeed();return;}
      if(action==='review'){guidedLog(wizard.dataset.guidedMsgReview);show(5,true);return;}
      const actionMessages={mobile_both:wizard.dataset.guidedMsgMobile,mobile_ios:wizard.dataset.guidedMsgMobile,mobile_android:wizard.dataset.guidedMsgMobile,data_external:wizard.dataset.guidedMsgData,data_management:wizard.dataset.guidedMsgData,maps_generic:wizard.dataset.guidedMsgMaps,maps_osm:wizard.dataset.guidedMsgMaps,architecture_default:wizard.dataset.guidedMsgArchitecture,architecture_event:wizard.dataset.guidedMsgArchitecture,architecture_modular:wizard.dataset.guidedMsgArchitecture,quality_public:wizard.dataset.guidedMsgQuality};
      if(await applyServerGuided({action_id:action},actionMessages[action])){return;}
      if(action==='mobile_both'){setValue('tipo_app','mobile');setChecked('mobile',true);addCSV('preferencias_tecnicas.preferencias',['plataformas moviles: iOS y Android']);guidedLog(wizard.dataset.guidedMsgMobile);}
      if(action==='mobile_ios'){setValue('tipo_app','mobile');setChecked('mobile',true);addCSV('preferencias_tecnicas.preferencias',['plataforma movil: iOS']);guidedLog(wizard.dataset.guidedMsgMobile);}
      if(action==='mobile_android'){setValue('tipo_app','mobile');setChecked('mobile',true);addCSV('preferencias_tecnicas.preferencias',['plataforma movil: Android']);guidedLog(wizard.dataset.guidedMsgMobile);}
      if(action==='data_external'){setValue('datos.db_required','false');setValue('integraciones.1.tipo','api');setValue('integraciones.1.nombre','datos externos de alquileres');setValue('integraciones.1.proposito','Consultar o importar datos existentes de pisos en alquiler.');guidedLog(wizard.dataset.guidedMsgData);}
      if(action==='data_management'){configureRentalData();setChecked('api',true);setChecked('web',true);}
      if(action==='maps_generic'){configureMaps(false);}
      if(action==='maps_osm'){configureMaps(true);}
      if(action==='architecture_default'){setValue('preferencias_tecnicas.arquitectura','hexagonal');guidedLog(wizard.dataset.guidedMsgArchitecture);}
      if(action==='architecture_event'){setValue('preferencias_tecnicas.arquitectura','event_driven');guidedLog(wizard.dataset.guidedMsgArchitecture);}
      if(action==='architecture_modular'){setValue('preferencias_tecnicas.arquitectura','modular_monolith');guidedLog(wizard.dataset.guidedMsgArchitecture);}
      if(action==='quality_public'){setValue('calidad.pruebas','alta');setValue('calidad.accesibilidad','wcag_aa');setValue('calidad.accesibilidad_opciones','normal,wcag_aa');guidedLog(wizard.dataset.guidedMsgQuality);}
      renderSummary();
    }
    function preset(kind){
      if(kind==='webapp'){setValue('tipo_app','web');setChecked('web',true);setChecked('api',true);setValue('preferencias_tecnicas.lenguaje','go');setValue('deploy.target','contenedor');}
      if(kind==='api'){setValue('tipo_app','api');setChecked('api',true);setValue('preferencias_tecnicas.lenguaje','go');setValue('calidad.pruebas','alta');}
      if(kind==='ops'){setValue('tipo_app','web');setChecked('web',true);setValue('preferencias_tecnicas.framework','html/js + api');setValue('calidad.observabilidad','true');}
      renderSummary();
    }
    tabs.forEach(node=>{
      node.addEventListener('click',()=>show(Number(node.dataset.gotoStep||0),true));
      node.addEventListener('keydown',event=>handleTabKeydown(event,node));
    });
    prev.addEventListener('click',()=>show(step-1,true));
    next.addEventListener('click',()=>show(step+1,true));
    form.addEventListener('input',renderSummary);
    form.addEventListener('input',event=>{if(event.target&&event.target.matches('[data-required="true"]')){clearFieldError(event.target);renderErrors(requiredFields().filter(el=>el.getAttribute('aria-invalid')==='true').map(el=>({name:el.name,label:fieldLabel(el),message:wizard.dataset.validationRequired||''})));}});
    form.addEventListener('change',event=>{renderSummary();if(event.target&&event.target.matches('[data-required="true"]')){clearFieldError(event.target);}});
    form.addEventListener('submit',event=>{if(!validateForm()){event.preventDefault();}});
    if(errors){errors.addEventListener('click',event=>{const target=event.target.closest('[data-error-target]');if(!target)return;const el=field(target.dataset.errorTarget);if(el)focusField(el);});}
    const guided=document.getElementById('guided-assistant');
    if(guided){guided.addEventListener('click',event=>{const target=event.target.closest('[data-guided-action]');if(target)guidedAction(target.dataset.guidedAction);});}
    wizard.querySelectorAll('[data-preset]').forEach(node=>node.addEventListener('click',()=>preset(node.dataset.preset)));
    const goalButton=document.querySelector('[data-goal-observe]');
    if(goalButton)goalButton.addEventListener('click',()=>observeGoal(goalButton));
    initAccessibleHelp();
    show(0,false);
    startGoalAutoPoll();
  }());
</script>
</body>
</html>`))
