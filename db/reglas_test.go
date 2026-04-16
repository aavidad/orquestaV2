package db

import (
	"encoding/json"
	"errors"
	"testing"
)

type testGovernanceTransitionCoordinator struct{}

func (testGovernanceTransitionCoordinator) AfterCrearRegla(actor string, r *Regla, id int64) error {
	if r == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, r.TipoAgente, "crear_regla", map[string]any{
		"regla_id": id,
		"titulo":   r.Titulo,
	})
	return nil
}

func (testGovernanceTransitionCoordinator) AfterActualizarRegla(actor string, r *Regla) error {
	if r == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, r.TipoAgente, "actualizar_regla", map[string]any{
		"regla_id": r.ID,
		"titulo":   r.Titulo,
	})
	return nil
}

func (testGovernanceTransitionCoordinator) AfterSetReglaActiva(actor string, r *Regla, activa bool) error {
	if r == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, r.TipoAgente, "set_regla_activa", map[string]any{
		"regla_id": r.ID,
		"activa":   activa,
		"titulo":   r.Titulo,
	})
	return nil
}

func (testGovernanceTransitionCoordinator) AfterCrearSkill(actor string, s *Skill, id int64) error {
	if s == nil {
		return nil
	}
	NotificarRefreshSkillCatalogo(actor, s, "crear")
	return nil
}

func (testGovernanceTransitionCoordinator) AfterActualizarSkill(actor string, s *Skill) error {
	if s == nil {
		return nil
	}
	NotificarRefreshSkillCatalogo(actor, s, "actualizar")
	return nil
}

func (testGovernanceTransitionCoordinator) AfterSetSkillActiva(actor string, s *Skill, activa bool) error {
	if s == nil {
		return nil
	}
	NotificarRefreshSkillCatalogo(actor, s, "activar")
	return nil
}

func (testGovernanceTransitionCoordinator) AfterCrearWorkflow(actor string, w *Workflow, id int64) error {
	if w == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, w.TipoAgente, "crear_workflow", map[string]any{
		"workflow_id": id,
		"nombre":      w.Nombre,
	})
	return nil
}

func (testGovernanceTransitionCoordinator) AfterActualizarWorkflow(actor string, w *Workflow) error {
	if w == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, w.TipoAgente, "actualizar_workflow", map[string]any{
		"workflow_id": w.ID,
		"nombre":      w.Nombre,
	})
	return nil
}

func (testGovernanceTransitionCoordinator) AfterSetWorkflowActivo(actor string, w *Workflow, activo bool) error {
	if w == nil {
		return nil
	}
	NotificarRefreshGobernanza(actor, w.TipoAgente, "set_workflow_activo", map[string]any{
		"workflow_id": w.ID,
		"activo":      activo,
		"nombre":      w.Nombre,
	})
	return nil
}

func TestGuardarYListarReglas(t *testing.T) {
	prepararDBTemporal(t)

	id, err := GuardarRegla(&Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "Puerto de almacenamiento",
		Descripcion: "No acoplar a SQLite",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("GuardarRegla: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarReglas("programador", nil)
	if err != nil {
		t.Fatalf("ListarReglas: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin reglas")
	}
}

func TestCatalogoBuiltinIncluyeGobernanzaDeTestsDB(t *testing.T) {
	prepararDBTemporal(t)

	reglas, err := ListarReglas("programador", nil)
	if err != nil {
		t.Fatalf("ListarReglas: %v", err)
	}
	skills, err := ListarSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListarSkills: %v", err)
	}

	var reglaHarness, reglaCoherencia, skillHarness bool
	for _, item := range reglas {
		if item == nil {
			continue
		}
		switch item.Titulo {
		case "Tests DB orquestados":
			reglaHarness = true
		case "Tests coherentes, rápidos y mantenibles":
			reglaCoherencia = true
		}
	}
	for _, item := range skills {
		if item != nil && item.Nombre == "db-test-harness" {
			skillHarness = true
			break
		}
	}

	if !reglaHarness || !reglaCoherencia || !skillHarness {
		t.Fatalf("catalogo builtin incompleto: reglaHarness=%t reglaCoherencia=%t skillHarness=%t", reglaHarness, reglaCoherencia, skillHarness)
	}
}

func TestGuardarYListarSkills(t *testing.T) {
	prepararDBTemporal(t)

	id, err := GuardarSkill(&Skill{
		TipoAgente:  "programador",
		Nombre:      "consultar-schema",
		Descripcion: "Revisar esquema",
		CuandoUsar:  "Antes de migrar",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("GuardarSkill: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListarSkills: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin skills")
	}
}

func TestSkillsOrdenadasPorPrioridadYEscenarioYConVersionado(t *testing.T) {
	prepararDBTemporal(t)
	registrarAgentesBaseDBTest(t)

	idBusq, err := CrearSkill("Codex2", &Skill{
		TipoAgente:       "programador",
		Nombre:           "rg",
		Descripcion:      "busqueda rapida",
		CuandoUsar:       "buscar texto",
		Escenario:        "investigacion",
		Prioridad:        20,
		AliasesJSON:      `["ripgrep"]`,
		HerramientasJSON: `["rg"]`,
		Activa:           true,
	})
	if err != nil {
		t.Fatalf("CrearSkill rg: %v", err)
	}
	idFmt, err := CrearSkill("Codex2", &Skill{
		TipoAgente:  "programador",
		Nombre:      "gofmt",
		Descripcion: "formateo",
		CuandoUsar:  "formatear codigo",
		Escenario:   "codigo",
		Prioridad:   5,
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearSkill gofmt: %v", err)
	}
	if _, err := CrearSkill("Codex2", &Skill{
		TipoAgente:  "programador",
		Nombre:      "git-diff",
		Descripcion: "comparar cambios",
		CuandoUsar:  "revisar diferencias",
		Escenario:   "git",
		Prioridad:   10,
		Activa:      true,
	}); err != nil {
		t.Fatalf("CrearSkill git-diff: %v", err)
	}

	items, err := ListarSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListarSkills: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("skills insuficientes: %+v", items)
	}
	if items[0].Nombre != "gofmt" || items[1].Nombre != "git-diff" || items[2].Nombre != "rg" {
		t.Fatalf("orden inesperado: %+v", []string{items[0].Nombre, items[1].Nombre, items[2].Nombre})
	}

	skillBusq, err := GetSkill(idBusq)
	if err != nil {
		t.Fatalf("GetSkill rg: %v", err)
	}
	skillBusq.Descripcion = "busqueda afinada"
	skillBusq.Prioridad = 15
	if err := ActualizarSkill("Codex2", skillBusq); err != nil {
		t.Fatalf("ActualizarSkill rg: %v", err)
	}
	if err := SetSkillActiva("Codex2", idBusq, false); err != nil {
		t.Fatalf("SetSkillActiva rg: %v", err)
	}
	versiones, err := ListarVersionesSkill(idBusq)
	if err != nil {
		t.Fatalf("ListarVersionesSkill: %v", err)
	}
	if len(versiones) != 3 {
		t.Fatalf("versiones inesperadas: %+v", versiones)
	}
	if versiones[0].Prioridad != 20 || versiones[1].Prioridad != 15 || versiones[2].Activa {
		t.Fatalf("snapshot de versiones inesperado: %+v", versiones)
	}
	if got, err := GetSkill(idFmt); err != nil || got.Prioridad != 5 {
		t.Fatalf("GetSkill gofmt inesperado: %+v err=%v", got, err)
	}
}

func TestCrearSkillRechazaDuplicadoEquivalente(t *testing.T) {
	prepararDBTemporal(t)
	registrarAgentesBaseDBTest(t)

	if _, err := CrearSkill("Codex2", &Skill{
		TipoAgente:       "programador",
		Nombre:           "rg",
		Descripcion:      "busqueda rapida",
		CuandoUsar:       "buscar texto",
		Escenario:        "investigacion",
		Prioridad:        10,
		AliasesJSON:      `["ripgrep"]`,
		HerramientasJSON: `["rg"]`,
		Activa:           true,
	}); err != nil {
		t.Fatalf("CrearSkill base: %v", err)
	}

	_, err := CrearSkill("Codex2", &Skill{
		TipoAgente:       "programador",
		Nombre:           "ripgrep",
		Descripcion:      "duplicada funcional",
		CuandoUsar:       "buscar texto",
		Escenario:        "investigacion",
		Prioridad:        20,
		HerramientasJSON: `["rg"]`,
		Activa:           true,
	})
	if err == nil {
		t.Fatalf("se esperaba error por skill equivalente")
	}
	var dupErr *SkillEquivalenteError
	if !errors.As(err, &dupErr) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestSkillExternaQuedaPendienteYSoloAdminLaActiva(t *testing.T) {
	prepararDBTemporal(t)
	registrarAgentesBaseDBTest(t)

	id, err := CrearSkill("Codex2", &Skill{
		TipoAgente:       "programador",
		Nombre:           "external-linter",
		Descripcion:      "linter externo",
		CuandoUsar:       "validar dependencias de terceros",
		Escenario:        "qa",
		Prioridad:        30,
		HerramientasJSON: `["vendor-lint"]`,
		Origen:           "third_party",
		Activa:           true,
	})
	if err != nil {
		t.Fatalf("CrearSkill externa: %v", err)
	}

	skill, err := GetSkill(id)
	if err != nil {
		t.Fatalf("GetSkill externa: %v", err)
	}
	if skill.Activa {
		t.Fatalf("la skill externa deberia quedar inactiva hasta aprobacion: %+v", skill)
	}
	if !skill.RequiereAprobacion {
		t.Fatalf("la skill externa deberia requerir aprobacion: %+v", skill)
	}
	if skill.NivelRiesgo != "medio" {
		t.Fatalf("riesgo inesperado: %+v", skill)
	}

	if err := SetSkillActiva("Codex2", id, true); err == nil {
		t.Fatalf("se esperaba rechazo al activar skill externa sin admin")
	}
	if err := SetSkillActiva("alberto", id, true); err != nil {
		t.Fatalf("activar skill externa como admin: %v", err)
	}

	skill, err = GetSkill(id)
	if err != nil {
		t.Fatalf("GetSkill externa tras activar: %v", err)
	}
	if !skill.Activa || skill.RequiereAprobacion {
		t.Fatalf("estado final inesperado: %+v", skill)
	}
}

func TestCrearSkillNotificaRefreshAMailboxDeAgentesActivosDelRol(t *testing.T) {
	prepararDBTemporal(t)
	SetGovernanceTransitionCoordinator(testGovernanceTransitionCoordinator{})
	t.Cleanup(func() {
		SetGovernanceTransitionCoordinator(nil)
	})
	registrarAgentesBaseDBTest(t)
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion Codex1: %v", err)
	}
	if _, err := IniciarSesion("antigravity"); err != nil {
		t.Fatalf("IniciarSesion antigravity: %v", err)
	}

	id, err := CrearSkill("Codex2", &Skill{
		TipoAgente:  "programador",
		Nombre:      "rg-refresh",
		Descripcion: "busqueda rapida",
		CuandoUsar:  "buscar texto",
		Escenario:   "investigacion",
		Prioridad:   15,
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	toCodex1 := "Codex1"
	mailboxCodex1, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &toCodex1})
	if err != nil {
		t.Fatalf("ListarRuntimeMailbox Codex1: %v", err)
	}
	if len(mailboxCodex1) == 0 {
		t.Fatalf("se esperaba refresh en mailbox de Codex1")
	}
	msg := mailboxCodex1[0]
	if msg.Kind != MailboxKindSkillsRefresh {
		t.Fatalf("kind inesperado: %+v", msg)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &payload); err != nil {
		t.Fatalf("payload refresh invalido: %v", err)
	}
	if int64(payload["skill_id"].(float64)) != id || payload["motivo"] != "crear" {
		t.Fatalf("payload refresh inesperado: %+v", payload)
	}

	toDocs := "antigravity"
	mailboxDocs, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &toDocs})
	if err != nil {
		t.Fatalf("ListarRuntimeMailbox antigravity: %v", err)
	}
	if len(mailboxDocs) != 0 {
		t.Fatalf("no deberia haber refresh de skill programador para antigravity: %+v", mailboxDocs)
	}
}

func TestCrearReglaNotificaGovernanceRefreshAMailboxDeAgentesActivosDelRol(t *testing.T) {
	prepararDBTemporal(t)
	SetGovernanceTransitionCoordinator(testGovernanceTransitionCoordinator{})
	t.Cleanup(func() {
		SetGovernanceTransitionCoordinator(nil)
	})
	registrarAgentesBaseDBTest(t)
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion Codex1: %v", err)
	}
	if _, err := IniciarSesion("antigravity"); err != nil {
		t.Fatalf("IniciarSesion antigravity: %v", err)
	}

	id, err := CrearRegla("Codex2", &Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "API first mailbox",
		Descripcion: "Toda operacion normal usa API",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearRegla: %v", err)
	}

	toCodex1 := "Codex1"
	mailboxCodex1, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &toCodex1})
	if err != nil {
		t.Fatalf("ListarRuntimeMailbox Codex1: %v", err)
	}
	if len(mailboxCodex1) == 0 {
		t.Fatalf("se esperaba governance refresh en mailbox de Codex1")
	}
	msg := mailboxCodex1[0]
	if msg.Kind != MailboxKindGovernanceRefresh {
		t.Fatalf("kind inesperado: %+v", msg)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &payload); err != nil {
		t.Fatalf("payload governance refresh invalido: %v", err)
	}
	if int64(payload["regla_id"].(float64)) != id || payload["motivo"] != "crear_regla" {
		t.Fatalf("payload governance refresh inesperado: %+v", payload)
	}
	if payload["tipo_agente"] != "programador" || payload["hash"] == "" {
		t.Fatalf("payload governance refresh incompleto: %+v", payload)
	}

	toDocs := "antigravity"
	mailboxDocs, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &toDocs})
	if err != nil {
		t.Fatalf("ListarRuntimeMailbox antigravity: %v", err)
	}
	if len(mailboxDocs) != 0 {
		t.Fatalf("no deberia haber governance refresh programador para antigravity: %+v", mailboxDocs)
	}
}

func TestGuardarYListarWorkflows(t *testing.T) {
	prepararDBTemporal(t)

	id, err := GuardarWorkflow(&Workflow{
		TipoAgente:  "programador",
		Nombre:      "workflow-prueba",
		Descripcion: "Flujo de prueba",
		Pasos:       "[\"uno\",\"dos\"]",
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("GuardarWorkflow: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarWorkflows("programador", nil)
	if err != nil {
		t.Fatalf("ListarWorkflows: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin workflows")
	}
}
