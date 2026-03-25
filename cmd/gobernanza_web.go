/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/db"
	"orquesta/gobernanzaapp"
	"orquesta/skillsapp"
)

var gobernanzaService = gobernanzaapp.NewService(db.GovernanceRepository{})

type webGobernanzaData struct {
	TipoAgente        string
	Reglas            []*db.Regla
	Skills            []*db.Skill
	VersionesSkill    map[int64][]*db.SkillVersion
	SkillsRemotas     []*skillsapp.SkillRemota
	FiltroSkillRemota string
	ErrSkillsRemotas  string
	DeteccionSkill    *db.ResultadoDeteccionSkill
	SolicitudSkill    webSolicitudSkillData
	Workflows         []webWorkflowView
	Msg               string
	Err               string
}

type webSolicitudSkillData struct {
	Nombre       string
	Descripcion  string
	CuandoUsar   string
	Escenario    string
	Aliases      string
	Herramientas string
}

type webWorkflowView struct {
	ID          int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       []string
	Activo      bool
}

func webHandlerGobernanza(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	if tipoAgente == "" {
		tipoAgente = "programador"
	}
	data, err := cargarDatosGobernanza(r, tipoAgente)
	if err != nil {
		webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	webRender(w, webTplLayout+webTplGobernanza, data)
}

func webRouterGobernanza(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/gobernanza", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "reglas" && r.Method == http.MethodPost:
		webHandlerGobernanzaReglaNueva(w, r)
	case len(parts) == 2 && parts[1] == "skills" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillNueva(w, r)
	case len(parts) == 3 && parts[1] == "skills" && parts[2] == "importar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillImportar(w, r)
	case len(parts) == 3 && parts[1] == "skills" && parts[2] == "detectar-carencia" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillDetectarCarencia(w, r)
	case len(parts) == 4 && parts[1] == "skills" && parts[3] == "activar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillActivar(w, r, parts[2], true)
	case len(parts) == 4 && parts[1] == "skills" && parts[3] == "desactivar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillActivar(w, r, parts[2], false)
	case len(parts) == 4 && parts[1] == "skills" && parts[3] == "borrar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillBorrar(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "skills" && parts[3] == "editar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillEditar(w, r, parts[2])
	case len(parts) == 2 && parts[1] == "workflows" && r.Method == http.MethodPost:
		webHandlerGobernanzaWorkflowNuevo(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIGobernanza(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "gobernanza" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 3 && parts[2] == "reglas" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaReglas(w, r)
	case len(parts) == 3 && parts[2] == "reglas" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaReglaCrear(w, r)
	case len(parts) == 3 && parts[2] == "skills" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaSkills(w, r)
	case len(parts) == 3 && parts[2] == "skills" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaSkillCrear(w, r)
	case len(parts) == 3 && parts[2] == "workflows" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaWorkflows(w, r)
	case len(parts) == 3 && parts[2] == "workflows" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaWorkflowCrear(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerGobernanzaReglaNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveRule(gobernanzaapp.SaveRuleInput{
		TipoAgente:  tipoAgente,
		Categoria:   r.FormValue("categoria"),
		Titulo:      r.FormValue("titulo"),
		Descripcion: r.FormValue("descripcion"),
		Activa:      parseBoolFormDefaultOn(r.FormValue("activa")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef("governance.flash.rule_saved")), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveSkill(gobernanzaapp.SaveSkillInput{
		TipoAgente:  tipoAgente,
		Nombre:      r.FormValue("nombre"),
		Descripcion: r.FormValue("descripcion"),
		CuandoUsar:  r.FormValue("cuando_usar"),
		Activa:      parseBoolFormDefaultOn(r.FormValue("activa")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef("governance.flash.skill_saved")), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillImportar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	sourceURL := strings.TrimSpace(r.FormValue("url"))
	repo := strings.TrimSpace(r.FormValue("repo"))
	skill := strings.TrimSpace(r.FormValue("skill"))
	result, err := skillsImportService.ImportFromWeb(context.Background(), skillsapp.ImportInput{
		Actor:      "alberto",
		TipoAgente: tipoAgente,
		URL:        sourceURL,
		Repo:       repo,
		Skill:      skill,
	})
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msg := webTranslatef("governance.flash.skill_imported", result.Skill.Nombre)
	if result.Existente {
		msg = webTranslatef("governance.flash.skill_import_exists", result.Skill.Nombre)
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(msg), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillActivar(w http.ResponseWriter, r *http.Request, rawID string, activa bool) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if err := db.SetSkillActivo("alberto", id, activa); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msgKey := "governance.flash.skill_deactivated"
	if activa {
		msgKey = "governance.flash.skill_activated"
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef(msgKey, id)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillBorrar(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if err := db.EliminarSkill("alberto", id); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef("governance.flash.skill_deleted", id)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillEditar(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	skill, err := db.GetSkill(id)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	aliasesJSON, err := listaJSONDesdeTextoPlano(r.FormValue("aliases"))
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	herramientasJSON, err := listaJSONDesdeTextoPlano(r.FormValue("herramientas"))
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	prioridad, err := parseOptionalInt(r.FormValue("prioridad"), skill.Prioridad)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	skill.TipoAgente = strings.TrimSpace(tipoAgente)
	skill.Nombre = strings.TrimSpace(r.FormValue("nombre"))
	skill.Descripcion = strings.TrimSpace(r.FormValue("descripcion"))
	skill.CuandoUsar = strings.TrimSpace(r.FormValue("cuando_usar"))
	skill.Escenario = strings.TrimSpace(r.FormValue("escenario"))
	skill.Prioridad = prioridad
	skill.Origen = strings.TrimSpace(r.FormValue("origen"))
	skill.NivelRiesgo = strings.TrimSpace(r.FormValue("nivel_riesgo"))
	skill.AliasesJSON = aliasesJSON
	skill.HerramientasJSON = herramientasJSON
	skill.RequiereAprobacion = strings.TrimSpace(r.FormValue("requiere_aprobacion")) != ""
	if err := db.ActualizarSkill("alberto", skill); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef("governance.flash.skill_updated", id)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillDetectarCarencia(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	solicitud := webSolicitudSkillData{
		Nombre:       strings.TrimSpace(r.FormValue("nombre")),
		Descripcion:  strings.TrimSpace(r.FormValue("descripcion")),
		CuandoUsar:   strings.TrimSpace(r.FormValue("cuando_usar")),
		Escenario:    strings.TrimSpace(r.FormValue("escenario")),
		Aliases:      strings.TrimSpace(r.FormValue("aliases")),
		Herramientas: strings.TrimSpace(r.FormValue("herramientas")),
	}
	aliasesJSON, err := listaJSONDesdeTextoPlano(solicitud.Aliases)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	herramientasJSON, err := listaJSONDesdeTextoPlano(solicitud.Herramientas)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	resultado, err := db.DetectarCarenciaSkill(&db.SolicitudDeteccionSkill{
		TipoAgente:       tipoAgente,
		Nombre:           solicitud.Nombre,
		Descripcion:      solicitud.Descripcion,
		CuandoUsar:       solicitud.CuandoUsar,
		Escenario:        solicitud.Escenario,
		AliasesJSON:      aliasesJSON,
		HerramientasJSON: herramientasJSON,
	})
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	data, err := cargarDatosGobernanza(r, tipoAgente)
	if err != nil {
		webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	data.DeteccionSkill = resultado
	data.SolicitudSkill = solicitud
	webRender(w, webTplLayout+webTplGobernanza, data)
}

func webHandlerGobernanzaWorkflowNuevo(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveWorkflow(gobernanzaapp.SaveWorkflowInput{
		TipoAgente:  tipoAgente,
		Nombre:      r.FormValue("nombre"),
		Descripcion: r.FormValue("descripcion"),
		Pasos:       splitPasos(r.FormValue("pasos")),
		Activo:      parseBoolFormDefaultOn(r.FormValue("activo")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslatef("governance.flash.workflow_saved")), http.StatusSeeOther)
}

func webHandlerAPIGobernanzaReglas(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListRules(tipoAgente, parseOptionalBool(r.URL.Query().Get("activa")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaReglaCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string `json:"tipo_agente"`
		Categoria   string `json:"categoria"`
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		Activa      bool   `json:"activa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveRule(gobernanzaapp.SaveRuleInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIGobernanzaSkills(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListSkills(tipoAgente, parseOptionalBool(r.URL.Query().Get("activa")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaSkillCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string `json:"tipo_agente"`
		Nombre      string `json:"nombre"`
		Descripcion string `json:"descripcion"`
		CuandoUsar  string `json:"cuando_usar"`
		Activa      bool   `json:"activa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveSkill(gobernanzaapp.SaveSkillInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIGobernanzaWorkflows(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListWorkflows(tipoAgente, parseOptionalBool(r.URL.Query().Get("activo")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaWorkflowCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string   `json:"tipo_agente"`
		Nombre      string   `json:"nombre"`
		Descripcion string   `json:"descripcion"`
		Pasos       []string `json:"pasos"`
		Activo      bool     `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveWorkflow(gobernanzaapp.SaveWorkflowInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func parseBoolFormDefaultOn(raw string) bool {
	raw = strings.TrimSpace(raw)
	return raw == "" || raw == "on" || raw == "1" || strings.EqualFold(raw, "true")
}

func parseOptionalBool(raw string) *bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	value := raw == "1" || strings.EqualFold(raw, "true")
	return &value
}

func splitPasos(raw string) []string {
	lines := strings.Split(raw, "\n")
	var pasos []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pasos = append(pasos, line)
	}
	return pasos
}

func parsePositiveInt64(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("id inválido")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id inválido")
	}
	return id, nil
}

func parseOptionalInt(raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("entero inválido")
	}
	return value, nil
}

func listaJSONDesdeTextoPlano(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	raw = strings.ReplaceAll(raw, ",", "\n")
	lines := strings.Split(raw, "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		if item := strings.TrimSpace(line); item != "" {
			items = append(items, item)
		}
	}
	return db.NormalizarListaJSONPublic(items)
}

func textoPlanoDesdeListaJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return raw
	}
	return strings.Join(items, "\n")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func webTranslatef(key string, args ...any) string {
	msg := webTranslate(key)
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

func cargarDatosGobernanza(r *http.Request, tipoAgente string) (webGobernanzaData, error) {
	itemsReglas, err := gobernanzaService.ListRules(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	itemsSkills, err := gobernanzaService.ListSkills(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	itemsWorkflows, err := gobernanzaService.ListWorkflows(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	filtroSkillRemota := strings.TrimSpace(r.URL.Query().Get("skill_q"))
	skillsRemotas, errSkillsRemotas := skillsCatalogoFetcher.Listar(context.Background(), filtroSkillRemota, 24)
	return webGobernanzaData{
		TipoAgente:        tipoAgente,
		Reglas:            itemsReglas,
		Skills:            itemsSkills,
		VersionesSkill:    versionarSkills(itemsSkills),
		SkillsRemotas:     skillsRemotas,
		FiltroSkillRemota: filtroSkillRemota,
		ErrSkillsRemotas:  errString(errSkillsRemotas),
		Workflows:         workflowsToView(itemsWorkflows),
		Msg:               r.URL.Query().Get("ok"),
		Err:               r.URL.Query().Get("err"),
	}, nil
}

func versionarSkills(items []*db.Skill) map[int64][]*db.SkillVersion {
	if len(items) == 0 {
		return nil
	}
	out := make(map[int64][]*db.SkillVersion, len(items))
	for _, item := range items {
		if item == nil || item.ID <= 0 {
			continue
		}
		versiones, err := db.ListarVersionesSkill(item.ID)
		if err != nil || len(versiones) == 0 {
			continue
		}
		if len(versiones) > 5 {
			versiones = versiones[len(versiones)-5:]
		}
		out[item.ID] = versiones
	}
	return out
}

func workflowsToView(items []*db.Workflow) []webWorkflowView {
	var out []webWorkflowView
	for _, item := range items {
		var pasos []string
		_ = json.Unmarshal([]byte(item.Pasos), &pasos)
		out = append(out, webWorkflowView{
			ID:          item.ID,
			TipoAgente:  item.TipoAgente,
			Nombre:      item.Nombre,
			Descripcion: item.Descripcion,
			Pasos:       pasos,
			Activo:      item.Activo,
		})
	}
	return out
}

const webTplGobernanza = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Gobernanza operativa</h2>
  <p style="color:#64748b">Gestión de reglas, skills y workflows desde la app.</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="get" action="/gobernanza">
    <label>Tipo de agente
      <select name="tipo_agente">
        <option value="programador" {{if eqStr .TipoAgente "programador"}}selected{{end}}>programador</option>
        <option value="documentador" {{if eqStr .TipoAgente "documentador"}}selected{{end}}>documentador</option>
        <option value="admin" {{if eqStr .TipoAgente "admin"}}selected{{end}}>admin</option>
      </select>
    </label>
    <button type="submit" class="btn-sm">Filtrar</button>
  </form>
</section>

<section class="container grid">
  <article>
    <h3>Reglas</h3>
    {{range .Reglas}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Titulo}}</strong> · {{.Categoria}}</summary>
        <p>{{.Descripcion}}</p>
      </details>
    {{else}}
      <p>No hay reglas.</p>
    {{end}}
    <form method="post" action="/gobernanza/reglas">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Categoría <input name="categoria" required></label>
      <label>Título <input name="titulo" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label><input type="checkbox" name="activa" checked> Activa</label>
      <button type="submit">Guardar regla</button>
    </form>
  </article>

  <article>
    <h3>{{tr "governance.skills.title"}}</h3>
    {{range .Skills}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Nombre}}</strong></summary>
        <p>{{.Descripcion}}</p>
        {{if .CuandoUsar}}<p><strong>{{tr "governance.skills.when_to_use"}}:</strong> {{.CuandoUsar}}</p>{{end}}
        <p style="display:flex;gap:.5rem;flex-wrap:wrap">
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.source"}}: {{.Origen}}</span>
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.risk"}}: {{.NivelRiesgo}}</span>
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">
            {{tr "governance.skills.state"}}:
            {{if .Activa}}{{tr "common.active"}}{{else}}{{tr "governance.skills.inactive"}}{{end}}
          </span>
          {{if .RequiereAprobacion}}
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #f59e0b;border-radius:999px;background:#fffbeb;color:#92400e">{{tr "governance.skills.pending_approval"}}</span>
          {{end}}
        </p>
        <div style="display:flex;gap:.5rem;flex-wrap:wrap;margin:.6rem 0">
          {{if .Activa}}
          <form method="post" action="/gobernanza/skills/{{.ID}}/desactivar">
            <input type="hidden" name="tipo_agente" value="{{$.TipoAgente}}">
            <button type="submit" class="btn-sm">{{tr "governance.skills.deactivate"}}</button>
          </form>
          {{else}}
          <form method="post" action="/gobernanza/skills/{{.ID}}/activar">
            <input type="hidden" name="tipo_agente" value="{{$.TipoAgente}}">
            <button type="submit" class="btn-sm">{{tr "governance.skills.activate"}}</button>
          </form>
          {{end}}
          <form method="post" action="/gobernanza/skills/{{.ID}}/borrar">
            <input type="hidden" name="tipo_agente" value="{{$.TipoAgente}}">
            <button type="submit" class="btn-sm">{{tr "governance.skills.delete"}}</button>
          </form>
        </div>
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;align-items:start">
          <form method="post" action="/gobernanza/skills/{{.ID}}/editar" style="border:1px solid #e2e8f0;padding:.75rem;border-radius:.5rem">
            <input type="hidden" name="tipo_agente" value="{{$.TipoAgente}}">
            <h4 style="margin:.2rem 0">{{tr "governance.skills.edit_title"}}</h4>
            <label>Nombre <input name="nombre" value="{{.Nombre}}" required></label>
            <label>Descripción <textarea name="descripcion" required>{{.Descripcion}}</textarea></label>
            <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar">{{.CuandoUsar}}</textarea></label>
            <label>{{tr "governance.skills.scenario"}} <input name="escenario" value="{{.Escenario}}"></label>
            <label>{{tr "governance.skills.priority"}} <input type="number" name="prioridad" value="{{.Prioridad}}"></label>
            <label>{{tr "governance.skills.source"}} <input name="origen" value="{{.Origen}}"></label>
            <label>{{tr "governance.skills.risk"}} <input name="nivel_riesgo" value="{{.NivelRiesgo}}"></label>
            <label>{{tr "governance.skills.aliases"}} <textarea name="aliases">{{jsonLines .AliasesJSON}}</textarea></label>
            <label>{{tr "governance.skills.tools"}} <textarea name="herramientas">{{jsonLines .HerramientasJSON}}</textarea></label>
            <label><input type="checkbox" name="requiere_aprobacion" {{if .RequiereAprobacion}}checked{{end}}> {{tr "governance.skills.pending_approval"}}</label>
            <button type="submit" class="btn-sm">{{tr "governance.skills.save_changes"}}</button>
          </form>
          <div style="border:1px solid #e2e8f0;padding:.75rem;border-radius:.5rem">
            <h4 style="margin:.2rem 0">{{tr "governance.skills.versions_title"}}</h4>
            {{with index $.VersionesSkill .ID}}
              <ul style="padding-left:1rem;margin:.4rem 0">
              {{range .}}
                <li>#{{.VersionNum}} · {{.Accion}} · {{.Actor}}</li>
              {{end}}
              </ul>
            {{else}}
              <p>{{tr "governance.skills.versions_none"}}</p>
            {{end}}
          </div>
        </div>
      </details>
    {{else}}
      <p>{{tr "governance.skills.none"}}</p>
    {{end}}
    <form method="post" action="/gobernanza/skills">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Nombre <input name="nombre" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar"></textarea></label>
      <label><input type="checkbox" name="activa" checked> Activa</label>
      <button type="submit">{{tr "governance.skills.save"}}</button>
    </form>
    <form method="post" action="/gobernanza/skills/importar" style="margin-top:1rem;border-top:1px solid #e2e8f0;padding-top:1rem">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.import.title"}}</h4>
      <p style="color:#64748b">{{tr "governance.skills.import.help"}}</p>
      <label>{{tr "governance.skills.import.url"}} <input name="url" placeholder="https://skills.sh/openai/skills/openai-docs"></label>
      <label>{{tr "governance.skills.import.repo"}} <input name="repo" placeholder="openai/skills"></label>
      <label>{{tr "governance.skills.import.skill"}} <input name="skill" placeholder="openai-docs"></label>
      <button type="submit">{{tr "governance.skills.import.button"}}</button>
    </form>
    <form method="post" action="/gobernanza/skills/detectar-carencia" style="margin-top:1rem;border-top:1px solid #e2e8f0;padding-top:1rem">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.detect.title"}}</h4>
      <label>Nombre <input name="nombre" value="{{.SolicitudSkill.Nombre}}"></label>
      <label>Descripción <textarea name="descripcion">{{.SolicitudSkill.Descripcion}}</textarea></label>
      <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar">{{.SolicitudSkill.CuandoUsar}}</textarea></label>
      <label>{{tr "governance.skills.scenario"}} <input name="escenario" value="{{.SolicitudSkill.Escenario}}"></label>
      <label>{{tr "governance.skills.aliases"}} <textarea name="aliases">{{.SolicitudSkill.Aliases}}</textarea></label>
      <label>{{tr "governance.skills.tools"}} <textarea name="herramientas">{{.SolicitudSkill.Herramientas}}</textarea></label>
      <button type="submit">{{tr "governance.skills.detect.button"}}</button>
    </form>
    {{with .DeteccionSkill}}
    <section style="margin-top:1rem;border:1px solid #e2e8f0;padding:.75rem;border-radius:.5rem">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.detect.result_title"}}</h4>
      <p><strong>Motivo:</strong> {{.Motivo}}</p>
      {{if .Equivalente}}
      <p><strong>{{tr "governance.skills.detect.equivalent"}}:</strong> {{.Equivalente.Nombre}}</p>
      {{end}}
      {{if .Candidatas}}
      <p><strong>{{tr "governance.skills.detect.candidates"}}:</strong></p>
      <ul style="padding-left:1rem">
      {{range .Candidatas}}
        <li>{{.Skill.Nombre}} ({{.Puntuacion}})</li>
      {{end}}
      </ul>
      {{end}}
      {{if .InvocacionCreador}}
      <p><strong>{{tr "governance.skills.detect.creator"}}:</strong></p>
      <pre style="white-space:pre-wrap;background:#f8fafc;border:1px solid #e2e8f0;padding:.75rem">{{.InvocacionCreador}}</pre>
      {{end}}
    </section>
    {{end}}
    <section style="margin-top:1rem;border-top:1px solid #e2e8f0;padding-top:1rem">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.remote.title"}}</h4>
      <form method="get" action="/gobernanza">
        <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
        <label>{{tr "governance.skills.remote.search"}} <input name="skill_q" value="{{.FiltroSkillRemota}}"></label>
        <button type="submit" class="btn-sm">Filtrar</button>
      </form>
      {{if .ErrSkillsRemotas}}
      <p style="color:#b91c1c">{{.ErrSkillsRemotas}}</p>
      {{end}}
      {{range .SkillsRemotas}}
      <details style="margin:.6rem 0">
        <summary><strong>{{.Skill}}</strong> · {{.Repo}}</summary>
        <p><a href="{{.URLCanonica}}" target="_blank" rel="noreferrer">{{.URLCanonica}}</a></p>
        <form method="post" action="/gobernanza/skills/importar">
          <input type="hidden" name="tipo_agente" value="{{$.TipoAgente}}">
          <input type="hidden" name="url" value="{{.URLCanonica}}">
          <input type="hidden" name="repo" value="{{.Repo}}">
          <input type="hidden" name="skill" value="{{.Skill}}">
          <button type="submit" class="btn-sm">{{tr "governance.skills.import.button"}}</button>
        </form>
      </details>
      {{else}}
      <p>{{tr "governance.skills.remote.none"}}</p>
      {{end}}
    </section>
  </article>
</section>

<section class="container">
  <article>
    <h3>Workflows</h3>
    {{range .Workflows}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Nombre}}</strong></summary>
        <p>{{.Descripcion}}</p>
        {{if .Pasos}}
        <ol>
          {{range .Pasos}}<li>{{.}}</li>{{end}}
        </ol>
        {{end}}
      </details>
    {{else}}
      <p>No hay workflows.</p>
    {{end}}
    <form method="post" action="/gobernanza/workflows">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Nombre <input name="nombre" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label>Pasos (uno por línea) <textarea name="pasos" required></textarea></label>
      <label><input type="checkbox" name="activo" checked> Activo</label>
      <button type="submit">Guardar workflow</button>
    </form>
  </article>
</section>
{{end}}`
