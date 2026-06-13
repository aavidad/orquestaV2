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
		"nueva_app.wizard.presets",
		"nueva_app.wizard.preset.webapp",
		"nueva_app.wizard.preset.api",
		"nueva_app.wizard.preset.ops",
		"nueva_app.wizard.identidad_avanzada",
		"nueva_app.wizard.plataformas_origen",
		"nueva_app.wizard.origen_proyecto",
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
    :root{color-scheme:light;--bg:#f3f5ef;--ink:#18211b;--muted:#65736a;--panel:#ffffff;--line:#d7dfd5;--brand:#1f7a4d;--brand-2:#c7f04b;--warn:#b45309;--bad:#b42318}
    *{box-sizing:border-box}
    body{font-family:"Trebuchet MS","Gill Sans",Verdana,sans-serif;margin:0;background:radial-gradient(circle at 10% 0,rgba(199,240,75,.35),transparent 28rem),linear-gradient(160deg,#f8faf4,#edf3ee);color:var(--ink)}
    main{max-width:1260px;margin:0 auto;padding:24px}
    header{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:18px}
    h1{font-family:Georgia,"Times New Roman",serif;font-size:clamp(30px,4vw,56px);line-height:.95;letter-spacing:-.05em;margin:0}
    .topnav{display:flex;gap:8px;flex-wrap:wrap}
    .topnav a,.ghost{border:1px solid var(--line);background:rgba(255,255,255,.72);color:var(--ink);border-radius:999px;padding:8px 11px;text-decoration:none}
    .lead{color:var(--muted);max-width:760px;margin:10px 0 0;font-size:17px}
    form{display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:16px;align-items:start}
    .wizard{border:1px solid var(--line);border-radius:18px;background:rgba(255,255,255,.88);box-shadow:0 24px 70px rgba(30,50,35,.12);overflow:hidden}
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
    button{width:max-content;padding:10px 14px;border:0;border-radius:9px;background:var(--brand);color:#fff;font-weight:850;cursor:pointer}
    button.secondary{background:#e8efe8;color:var(--ink);border:1px solid var(--line)}
    .wizard-actions{display:flex;justify-content:space-between;gap:10px;padding:14px 18px;border-top:1px solid var(--line);background:#f8fbf5}
    .presets{display:flex;gap:8px;flex-wrap:wrap}
    .presets button{background:#102017;color:#dcffd6}
    details{border:1px dashed #bfd0c3;border-radius:12px;padding:10px;background:#fbfdf9}
    summary{cursor:pointer;font-weight:800;color:var(--brand)}
    .side{position:sticky;top:18px;display:grid;gap:12px}
    .summary-list{display:grid;gap:8px;color:var(--muted)}
    .summary-list strong{color:var(--ink)}
    .status{border-left:4px solid var(--brand)}
    .issue{border-left:4px solid var(--bad)}
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
  <form method="post" action="/nueva-app">
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
      data-summary-no-db-required="{{index .HTML "nueva_app.wizard.summary.no_db_required"}}">
      <div class="steps" aria-label="{{index .HTML "nueva_app.wizard.steps_label"}}">
        <button class="step-tab active" type="button" data-goto-step="0">{{index .HTML "nueva_app.wizard.step.idea"}}</button>
        <button class="step-tab" type="button" data-goto-step="1">{{index .HTML "nueva_app.wizard.step.tipo"}}</button>
        <button class="step-tab" type="button" data-goto-step="2">{{index .HTML "nueva_app.wizard.step.tecnologia"}}</button>
        <button class="step-tab" type="button" data-goto-step="3">{{index .HTML "nueva_app.wizard.step.datos"}}</button>
        <button class="step-tab" type="button" data-goto-step="4">{{index .HTML "nueva_app.wizard.step.calidad"}}</button>
        <button class="step-tab" type="button" data-goto-step="5">{{index .HTML "nueva_app.wizard.step.revisar"}}</button>
      </div>
      <div class="step" data-step="0">
        <fieldset>
          <legend>{{index .HTML "nueva_app.wizard.idea_objetivo"}}</legend>
          <div class="presets" aria-label="{{index .HTML "nueva_app.wizard.presets"}}">
            <button type="button" data-preset="webapp">{{index .HTML "nueva_app.wizard.preset.webapp"}}</button>
            <button type="button" data-preset="api">{{index .HTML "nueva_app.wizard.preset.api"}}</button>
            <button type="button" data-preset="ops">{{index .HTML "nueva_app.wizard.preset.ops"}}</button>
          </div>
          <div class="grid">
            <label>{{index .Labels "nombre"}}<input name="nombre" required autocomplete="off"></label>
            <label>{{index .Labels "tipo_app"}}<select name="tipo_app" required><option value="web">web</option><option value="api">api</option><option value="cli">cli</option><option value="desktop">desktop</option><option value="mobile">mobile</option><option value="automation">automation</option><option value="data">data</option><option value="plugin">plugin</option><option value="mixed">mixed</option></select></label>
          </div>
          <label>{{index .Labels "objetivo"}}<textarea name="objetivo" required placeholder="{{index .HTML "nueva_app.wizard.placeholder.objetivo"}}"></textarea></label>
          <label>{{index .Labels "descripcion"}}<textarea name="descripcion" placeholder="{{index .HTML "nueva_app.wizard.placeholder.descripcion"}}"></textarea></label>
          <details><summary>{{index .HTML "nueva_app.wizard.identidad_avanzada"}}</summary><div class="grid">
            <label>{{index .Labels "request_id"}}<input name="request_id" autocomplete="off"></label>
            <label>{{index .Labels "locale"}}<select name="locale">{{range .Page.Opciones.Locales}}<option value="{{.Valor}}">{{.Label}}</option>{{end}}</select></label>
            <label title="{{index .Help "request_kind"}}">{{index .Labels "request_kind"}}<select name="request_kind"><option value="crear_app_completa">crear_app_completa</option><option value="documentar_app">documentar_app</option><option value="analizar_app">analizar_app</option><option value="brainstorming_arquitectura">brainstorming_arquitectura</option><option value="planificar_app">planificar_app</option><option value="programar_modulo">programar_modulo</option><option value="modificar_app_existente">modificar_app_existente</option><option value="revisar_codigo">revisar_codigo</option><option value="pruebas_y_validacion">pruebas_y_validacion</option><option value="seguridad">seguridad</option><option value="deploy">deploy</option><option value="operacion_soporte">operacion_soporte</option><option value="integracion_externa">integracion_externa</option><option value="i18n_l10n">i18n_l10n</option><option value="migracion_refactor">migracion_refactor</option><option value="investigacion_tecnica">investigacion_tecnica</option></select></label>
            <label title="{{index .Help "execution_mode"}}">{{index .Labels "execution_mode"}}<select name="execution_mode"><option value="normal">normal</option><option value="debug">debug</option></select></label>
          </div></details>
        </fieldset>
      </div>
      <div class="step" data-step="1">
        <fieldset><legend>{{index .HTML "nueva_app.wizard.plataformas_origen"}}</legend>
          <div class="checks"><label><input type="checkbox" name="plataformas" value="web">web</label><label><input type="checkbox" name="plataformas" value="mobile">mobile</label><label><input type="checkbox" name="plataformas" value="desktop">desktop</label><label><input type="checkbox" name="plataformas" value="api">api</label></div>
          <details open><summary>{{index .HTML "nueva_app.wizard.origen_proyecto"}}</summary><div class="grid">
            <label>{{index .Labels "project_source.kind"}}<select name="project_source.kind"><option value=""></option><option value="new">new</option><option value="github">github</option><option value="local_path">local_path</option></select></label>
            <label>{{index .Labels "project_source.git_url"}}<input name="project_source.git_url"></label>
            <label>{{index .Labels "project_source.branch"}}<input name="project_source.branch"></label>
            <label>{{index .Labels "project_source.project_ref"}}<input name="project_source.project_ref"></label>
            <label>{{index .Labels "project_source.local_path"}}<input name="project_source.local_path"></label>
          </div></details>
        </fieldset>
      </div>
      <div class="step" data-step="2">
        <fieldset><legend>{{index .Labels "preferencias_tecnicas"}}</legend><div class="grid">
          <label>{{index .Labels "preferencias_tecnicas.arquitectura"}}<select name="preferencias_tecnicas.arquitectura"><option value="hexagonal">hexagonal</option><option value="modular">modular</option><option value="monolito_modular">monolito_modular</option></select></label>
          <label>{{index .Labels "preferencias_tecnicas.lenguaje"}}<input name="preferencias_tecnicas.lenguaje" placeholder="go, typescript..."></label>
          <label>{{index .Labels "preferencias_tecnicas.framework"}}<input name="preferencias_tecnicas.framework"></label>
          <label>{{index .Labels "preferencias_tecnicas.preferencias"}}<input name="preferencias_tecnicas.preferencias"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "i18n"}}</legend><div class="grid">
          <label><span>{{index .Labels "i18n.enabled"}}</span><select name="i18n.enabled"><option value="true">true</option><option value="false">false</option></select></label>
          <label>{{index .Labels "i18n.default_locale"}}<input name="i18n.default_locale" value="{{.Page.Locale}}"></label>
          <label>{{index .Labels "i18n.locales"}}<input name="i18n.locales" placeholder="es-ES,en-US"></label>
          <label>{{index .Labels "i18n.justificacion"}}<input name="i18n.justificacion"></label>
        </div></fieldset>
      </div>
      <div class="step" data-step="3">
        <fieldset><legend>{{index .Labels "datos"}}</legend><div class="grid">
          <label><span>{{index .Labels "datos.db_required"}}</span><select name="datos.db_required"><option value="false">false</option><option value="true">true</option></select></label>
          <label>{{index .Labels "datos.necesidad_funcional"}}<input name="datos.necesidad_funcional"></label>
          <label>{{index .Labels "datos.tipos_datos"}}<input name="datos.tipos_datos" placeholder="usuarios,eventos"></label>
          <label>{{index .Labels "datos.sensibilidad"}}<input name="datos.sensibilidad"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "integraciones"}}</legend><div class="grid">
          <label>{{index .Labels "integraciones.0.tipo"}}<select name="integraciones.0.tipo"><option value=""></option><option value="api">api</option><option value="webhook">webhook</option><option value="email">email</option><option value="calendar">calendar</option></select></label>
          <label>{{index .Labels "integraciones.0.nombre"}}<input name="integraciones.0.nombre"></label>
          <label>{{index .Labels "integraciones.0.proposito"}}<input name="integraciones.0.proposito"></label>
          <label><span>{{index .Labels "integraciones.0.requerido"}}</span><select name="integraciones.0.requerido"><option value="false">false</option><option value="true">true</option></select></label>
        </div></fieldset>
      </div>
      <div class="step" data-step="4">
        <fieldset><legend>{{index .Labels "calidad"}}</legend><div class="grid">
          <label>{{index .Labels "calidad.pruebas"}}<select name="calidad.pruebas"><option value="basica">basica</option><option value="media">media</option><option value="alta">alta</option></select></label>
          <label>{{index .Labels "calidad.accesibilidad"}}<select name="calidad.accesibilidad"><option value="basica">basica</option><option value="wcag_aa">wcag_aa</option></select></label>
          <label><span>{{index .Labels "calidad.observabilidad"}}</span><select name="calidad.observabilidad"><option value="true">true</option><option value="false">false</option></select></label>
          <label>{{index .Labels "calidad.compliance"}}<input name="calidad.compliance"></label>
        </div></fieldset>
        <fieldset><legend>{{index .Labels "deploy"}}</legend><div class="grid">
          <label>{{index .Labels "deploy.target"}}<select name="deploy.target"><option value="sin_preferencia">sin_preferencia</option><option value="local">local</option><option value="contenedor">contenedor</option><option value="paas">paas</option><option value="serverless">serverless</option><option value="kubernetes">kubernetes</option><option value="desktop">desktop</option><option value="mobile_store">mobile_store</option></select></label>
          <label>{{index .Labels "deploy.restricciones"}}<input name="deploy.restricciones"></label>
        </div></fieldset>
      </div>
      <div class="step" data-step="5">
        <fieldset><legend>{{index .Labels "agentes"}}</legend><div class="grid">
          <label><span>{{index .Labels "agentes.revision_humana"}}</span><select name="agentes.revision_humana"><option value="true">true</option><option value="false">false</option></select></label>
          <label>{{index .Labels "agentes.autonomia"}}<select name="agentes.autonomia"><option value="media">media</option><option value="baja">baja</option><option value="alta">alta</option></select></label>
          <label>{{index .Labels "agentes.preferencias"}}<input name="agentes.preferencias"></label>
          <label>{{index .Labels "restricciones"}}<input name="restricciones"></label>
        </div></fieldset>
        <fieldset><legend>{{index .HTML "nueva_app.wizard.revision_final"}}</legend><div id="wizard-final-summary" class="summary-list"></div><button class="hidden-final" type="submit">{{.Page.Formulario.Acciones.Submit}}</button></fieldset>
      </div>
      <div class="wizard-actions">
        <button class="secondary" type="button" data-prev-step>{{index .HTML "nueva_app.wizard.back"}}</button>
        <button type="button" data-next-step>{{index .HTML "nueva_app.wizard.next"}}</button>
      </div>
    </section>
    <aside class="side">
      <section class="panel"><h2>{{index .HTML "nueva_app.wizard.resumen_vivo"}}</h2><div id="wizard-summary" class="summary-list"></div></section>
      <section class="panel status"><h2>{{index .HTML "nueva_app.html.resultado"}}</h2><p>{{.Page.Textos.Estado}}</p>{{if .Page.ViewModel.RequestID}}<p><code>{{.Page.ViewModel.RequestID}}</code></p>{{end}}</section>
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
    function show(index){
      step=Math.max(0,Math.min(steps.length-1,index));
      steps.forEach((node,i)=>{node.hidden=i!==step;});
      tabs.forEach((node,i)=>node.classList.toggle('active',i===step));
      prev.disabled=step===0;
      next.hidden=step===steps.length-1;
      wizard.classList.toggle('wizard-step-final',step===steps.length-1);
      renderSummary();
    }
    function setValue(name,value){const el=field(name);if(el)el.value=value;}
    function setChecked(value,on){form.querySelectorAll('input[name="plataformas"][value="'+value+'"]').forEach(el=>{el.checked=on;});}
    function preset(kind){
      if(kind==='webapp'){setValue('tipo_app','web');setChecked('web',true);setChecked('api',true);setValue('preferencias_tecnicas.lenguaje','go');setValue('deploy.target','contenedor');}
      if(kind==='api'){setValue('tipo_app','api');setChecked('api',true);setValue('preferencias_tecnicas.lenguaje','go');setValue('calidad.pruebas','alta');}
      if(kind==='ops'){setValue('tipo_app','web');setChecked('web',true);setValue('preferencias_tecnicas.framework','html/js + api');setValue('calidad.observabilidad','true');}
      renderSummary();
    }
    tabs.forEach(node=>node.addEventListener('click',()=>show(Number(node.dataset.gotoStep||0))));
    prev.addEventListener('click',()=>show(step-1));
    next.addEventListener('click',()=>show(step+1));
    form.addEventListener('input',renderSummary);
    form.addEventListener('change',renderSummary);
    wizard.querySelectorAll('[data-preset]').forEach(node=>node.addEventListener('click',()=>preset(node.dataset.preset)));
    show(0);
  }());
</script>
</body>
</html>`))
