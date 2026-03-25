/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		webRender(w, r, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	webRender(w, r, webTplLayout+webTplGobernanza, data)
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
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "governance.flash.rule_saved")), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
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
	prioridad, err := parseOptionalInt(r.FormValue("prioridad"), 100)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if _, err := webCrearSkillPorAPI(apiSkillCrearRequest{
		Actor:              "alberto",
		TipoAgente:         tipoAgente,
		Nombre:             r.FormValue("nombre"),
		Descripcion:        r.FormValue("descripcion"),
		CuandoUsar:         r.FormValue("cuando_usar"),
		Escenario:          r.FormValue("escenario"),
		Prioridad:          prioridad,
		AliasesJSON:        aliasesJSON,
		HerramientasJSON:   herramientasJSON,
		Origen:             r.FormValue("origen"),
		NivelRiesgo:        r.FormValue("nivel_riesgo"),
		RequiereAprobacion: strings.TrimSpace(r.FormValue("requiere_aprobacion")) != "",
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "governance.flash.skill_saved")), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillImportar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	sourceURL := strings.TrimSpace(r.FormValue("url"))
	repo := strings.TrimSpace(r.FormValue("repo"))
	skill := strings.TrimSpace(r.FormValue("skill"))
	result, err := webImportarSkillPorAPI(apiSkillImportarRequest{
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
	msgKey := "governance.flash.skill_imported"
	if result.Existente {
		msgKey = "governance.flash.skill_import_exists"
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, msgKey, result.Skill.Nombre)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillActivar(w http.ResponseWriter, r *http.Request, rawID string, activa bool) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if err := webSetSkillActivaPorAPI(id, "alberto", activa); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msgKey := "governance.flash.skill_deactivated"
	if activa {
		msgKey = "governance.flash.skill_activated"
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, msgKey, id)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillBorrar(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if err := webBorrarSkillPorAPI(id, "alberto"); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "governance.flash.skill_deleted", id)), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillEditar(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parsePositiveInt64(rawID)
	if err != nil {
		http.Redirect(w, r, "/gobernanza?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	skill, err := webCargarSkillPorAPI(id)
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
	if err := webActualizarSkillPorAPI(id, apiSkillActualizarRequest{
		Actor:              "alberto",
		TipoAgente:         skill.TipoAgente,
		Nombre:             skill.Nombre,
		Descripcion:        skill.Descripcion,
		CuandoUsar:         skill.CuandoUsar,
		Escenario:          skill.Escenario,
		Prioridad:          skill.Prioridad,
		AliasesJSON:        skill.AliasesJSON,
		HerramientasJSON:   skill.HerramientasJSON,
		Origen:             skill.Origen,
		NivelRiesgo:        skill.NivelRiesgo,
		RequiereAprobacion: skill.RequiereAprobacion,
		Activa:             skill.Activa,
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "governance.flash.skill_updated", id)), http.StatusSeeOther)
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
	resultado, err := webDetectarCarenciaSkillPorAPI(apiSkillDeteccionRequest{
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
		webRender(w, r, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	data.DeteccionSkill = resultado
	data.SolicitudSkill = solicitud
	webRender(w, r, webTplLayout+webTplGobernanza, data)
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
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "governance.flash.workflow_saved")), http.StatusSeeOther)
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

func cargarDatosGobernanza(r *http.Request, tipoAgente string) (webGobernanzaData, error) {
	itemsReglas, err := gobernanzaService.ListRules(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	itemsSkills, err := webCargarSkillsPorAPI(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	itemsWorkflows, err := gobernanzaService.ListWorkflows(tipoAgente, nil)
	if err != nil {
		return webGobernanzaData{}, err
	}
	filtroSkillRemota := strings.TrimSpace(r.URL.Query().Get("skill_q"))
	skillsRemotas, errSkillsRemotas := webListarSkillsRemotasPorAPI(filtroSkillRemota, 24)
	return webGobernanzaData{
		TipoAgente:        tipoAgente,
		Reglas:            itemsReglas,
		Skills:            itemsSkills,
		VersionesSkill:    versionarSkills(itemsSkills),
		SkillsRemotas:     skillsRemotas,
		FiltroSkillRemota: filtroSkillRemota,
		Workflows:         workflowsToView(itemsWorkflows),
		Msg:               r.URL.Query().Get("ok"),
		Err:               r.URL.Query().Get("err"),
		ErrSkillsRemotas:  errString(errSkillsRemotas),
	}, nil
}

func webInvocarAPIJSON(method, path string, payload any, dst any) error {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code >= http.StatusBadRequest {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return errors.New(apiErr.Error)
		}
		return fmt.Errorf("error API %s %s: status %d", method, path, rec.Code)
	}
	if dst == nil {
		return nil
	}
	return json.NewDecoder(rec.Body).Decode(dst)
}

func webCargarSkillsPorAPI(rol string, activa *bool) ([]*db.Skill, error) {
	query := url.Values{}
	if rol = strings.TrimSpace(rol); rol != "" {
		query.Set("tipo_agente", rol)
	}
	if activa != nil {
		query.Set("activa", strconv.FormatBool(*activa))
	}
	path := "/api/skills"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiSkillsResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Skills, nil
}

func webCrearSkillPorAPI(req apiSkillCrearRequest) (int64, error) {
	var resp apiCatalogoMutationResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/skills", req, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

func webCargarSkillPorAPI(id int64) (*db.Skill, error) {
	var resp apiSkillResponse
	if err := webInvocarAPIJSON(http.MethodGet, fmt.Sprintf("/api/skills/%d", id), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Skill, nil
}

func webActualizarSkillPorAPI(id int64, req apiSkillActualizarRequest) error {
	return webInvocarAPIJSON(http.MethodPost, fmt.Sprintf("/api/skills/%d", id), req, nil)
}

func webSetSkillActivaPorAPI(id int64, actor string, activa bool) error {
	return webInvocarAPIJSON(http.MethodPost, fmt.Sprintf("/api/skills/%d/activa", id), apiCatalogoActivacionRequest{
		Actor:  actor,
		Activa: activa,
	}, nil)
}

func webBorrarSkillPorAPI(id int64, actor string) error {
	return webInvocarAPIJSON(http.MethodPost, fmt.Sprintf("/api/skills/%d/borrar", id), apiSkillBorrarRequest{
		Actor: actor,
	}, nil)
}

func webCargarVersionesSkillPorAPI(id int64) ([]*db.SkillVersion, error) {
	var resp apiSkillVersionesResponse
	if err := webInvocarAPIJSON(http.MethodGet, fmt.Sprintf("/api/skills/%d/versiones", id), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Versiones, nil
}

func webListarSkillsRemotasPorAPI(filtro string, limite int) ([]*skillsapp.SkillRemota, error) {
	query := url.Values{}
	if filtro = strings.TrimSpace(filtro); filtro != "" {
		query.Set("q", filtro)
	}
	if limite > 0 {
		query.Set("limit", strconv.Itoa(limite))
	}
	path := "/api/skills/remotas"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiSkillsRemotasResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func webImportarSkillPorAPI(req apiSkillImportarRequest) (*apiSkillImportarResponse, error) {
	var resp apiSkillImportarResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/skills/importar", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webDetectarCarenciaSkillPorAPI(req apiSkillDeteccionRequest) (*db.ResultadoDeteccionSkill, error) {
	var resp apiSkillDeteccionResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/skills/detectar-carencia", req, &resp); err != nil {
		return nil, err
	}
	return resp.Resultado, nil
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
		versiones, err := webCargarVersionesSkillPorAPI(item.ID)
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
  <h2 style="margin:0">{{tr "governance.title"}}</h2>
  <p style="color:#64748b">{{tr "governance.subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="get" action="/gobernanza">
    <label>{{tr "governance.agent_type"}}
      <select name="tipo_agente">
        <option value="programador" {{if eqStr .TipoAgente "programador"}}selected{{end}}>{{tr "programador"}}</option>
        <option value="documentador" {{if eqStr .TipoAgente "documentador"}}selected{{end}}>{{tr "documentador"}}</option>
        <option value="admin" {{if eqStr .TipoAgente "admin"}}selected{{end}}>{{tr "admin"}}</option>
      </select>
    </label>
    <button type="submit" class="btn-sm">{{tr "common.filter"}}</button>
  </form>
</section>

<section class="container grid">
  <article>
    <h3>{{tr "governance.rules.title"}}</h3>
    {{range .Reglas}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Titulo}}</strong> · {{.Categoria}}</summary>
        <p>{{.Descripcion}}</p>
      </details>
    {{else}}
      <p>{{tr "governance.rules.none"}}</p>
    {{end}}
    <form method="post" action="/gobernanza/reglas">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>{{tr "projects.category"}} <input name="categoria" required></label>
      <label>{{tr "projects.title_label"}} <input name="titulo" required></label>
      <label>{{tr "governance.description"}} <textarea name="descripcion" required></textarea></label>
      <label><input type="checkbox" name="activa" checked> {{tr "governance.active_feminine"}}</label>
      <button type="submit">{{tr "governance.rules.save"}}</button>
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
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.source"}}: {{tr .Origen}}</span>
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.risk"}}: {{tr .NivelRiesgo}}</span>
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
            <label>{{tr "common.name"}} <input name="nombre" value="{{.Nombre}}" required></label>
            <label>{{tr "governance.description"}} <textarea name="descripcion" required>{{.Descripcion}}</textarea></label>
            <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar">{{.CuandoUsar}}</textarea></label>
            <label>{{tr "governance.skills.scenario"}} <input name="escenario" value="{{.Escenario}}"></label>
            <label>{{tr "governance.skills.priority"}} <input type="number" name="prioridad" value="{{.Prioridad}}"></label>
            <label>{{tr "governance.skills.source"}}
              <select name="origen">
                <option value="builtin" {{if eqStr .Origen "builtin"}}selected{{end}}>{{tr "builtin"}}</option>
                <option value="local" {{if eqStr .Origen "local"}}selected{{end}}>{{tr "local"}}</option>
                <option value="third_party" {{if eqStr .Origen "third_party"}}selected{{end}}>{{tr "third_party"}}</option>
              </select>
            </label>
            <label>{{tr "governance.skills.risk"}}
              <select name="nivel_riesgo">
                <option value="bajo" {{if eqStr .NivelRiesgo "bajo"}}selected{{end}}>{{tr "bajo"}}</option>
                <option value="medio" {{if eqStr .NivelRiesgo "medio"}}selected{{end}}>{{tr "medio"}}</option>
                <option value="alto" {{if eqStr .NivelRiesgo "alto"}}selected{{end}}>{{tr "alto"}}</option>
              </select>
            </label>
            <label>{{tr "governance.skills.aliases"}} <textarea name="aliases">{{jsonLines .AliasesJSON}}</textarea></label>
            <label>{{tr "governance.skills.tools"}} <textarea name="herramientas">{{jsonLines .HerramientasJSON}}</textarea></label>
            <label><input type="checkbox" name="requiere_aprobacion" {{if .RequiereAprobacion}}checked{{end}}> {{tr "governance.skills.requires_approval"}}</label>
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
      <label>{{tr "common.name"}} <input name="nombre" required></label>
      <label>{{tr "governance.description"}} <textarea name="descripcion" required></textarea></label>
      <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar"></textarea></label>
      <label>{{tr "governance.skills.scenario"}} <input name="escenario"></label>
      <label>{{tr "governance.skills.priority"}} <input type="number" name="prioridad" value="100"></label>
      <label>{{tr "governance.skills.source"}}
        <select name="origen">
          <option value="builtin" selected>{{tr "builtin"}}</option>
          <option value="local">{{tr "local"}}</option>
          <option value="third_party">{{tr "third_party"}}</option>
        </select>
      </label>
      <label>{{tr "governance.skills.risk"}}
        <select name="nivel_riesgo">
          <option value="bajo" selected>{{tr "bajo"}}</option>
          <option value="medio">{{tr "medio"}}</option>
          <option value="alto">{{tr "alto"}}</option>
        </select>
      </label>
      <label>{{tr "governance.skills.aliases"}} <textarea name="aliases"></textarea></label>
      <label>{{tr "governance.skills.tools"}} <textarea name="herramientas"></textarea></label>
      <label><input type="checkbox" name="requiere_aprobacion"> {{tr "governance.skills.requires_approval"}}</label>
      <label><input type="checkbox" name="activa" checked> {{tr "governance.active_feminine"}}</label>
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
      <label>{{tr "common.name"}} <input name="nombre" value="{{.SolicitudSkill.Nombre}}"></label>
      <label>{{tr "governance.description"}} <textarea name="descripcion">{{.SolicitudSkill.Descripcion}}</textarea></label>
      <label>{{tr "governance.skills.when_to_use"}} <textarea name="cuando_usar">{{.SolicitudSkill.CuandoUsar}}</textarea></label>
      <label>{{tr "governance.skills.scenario"}} <input name="escenario" value="{{.SolicitudSkill.Escenario}}"></label>
      <label>{{tr "governance.skills.aliases"}} <textarea name="aliases">{{.SolicitudSkill.Aliases}}</textarea></label>
      <label>{{tr "governance.skills.tools"}} <textarea name="herramientas">{{.SolicitudSkill.Herramientas}}</textarea></label>
      <button type="submit">{{tr "governance.skills.detect.button"}}</button>
    </form>
    {{with .DeteccionSkill}}
    <section style="margin-top:1rem;border:1px solid #e2e8f0;padding:.75rem;border-radius:.5rem">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.detect.result_title"}}</h4>
      <p><strong>{{tr "common.reason"}}:</strong> {{.Motivo}}</p>
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
        <button type="submit" class="btn-sm">{{tr "common.filter"}}</button>
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
    <h3>{{tr "governance.workflows.title"}}</h3>
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
      <p>{{tr "governance.workflows.none"}}</p>
    {{end}}
    <form method="post" action="/gobernanza/workflows">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>{{tr "common.name"}} <input name="nombre" required></label>
      <label>{{tr "governance.description"}} <textarea name="descripcion" required></textarea></label>
      <label>{{tr "governance.workflows.steps"}} <textarea name="pasos" required></textarea></label>
      <label><input type="checkbox" name="activo" checked> {{tr "common.active"}}</label>
      <button type="submit">{{tr "governance.workflows.save"}}</button>
    </form>
  </article>
</section>
{{end}}`
