package gobernanzaapp

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	rule     *db.Regla
	skill    *db.Skill
	workflow *db.Workflow
}

func (f *fakeStore) ListOverrides(scopeTipo, scopeRef, tipoAgente, entidad string) ([]*db.GovernanceOverride, error) {
	return nil, nil
}

func (f *fakeStore) SaveOverride(actor string, item *db.GovernanceOverride) (int64, error) {
	return 40, nil
}

func (f *fakeStore) ResolveCatalogForContext(tipoAgente string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return &db.GovernanceCatalog{TipoAgente: tipoAgente}, nil
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	return &db.Proyecto{ID: 1, Slug: ref, Nombre: ref}, nil
}

func (f *fakeStore) GetAgent(nombre string) (*db.Agente, error) {
	return &db.Agente{Nombre: nombre, Rol: "programador"}, nil
}

func (f *fakeStore) ListRules(tipoAgente string, activa *bool) ([]*db.Regla, error) {
	return []*db.Regla{{TipoAgente: tipoAgente}}, nil
}

func (f *fakeStore) SaveRule(r *db.Regla) (int64, error) {
	f.rule = r
	return 10, nil
}

func (f *fakeStore) ListSkills(tipoAgente string, activa *bool) ([]*db.Skill, error) {
	return []*db.Skill{{TipoAgente: tipoAgente}}, nil
}

func (f *fakeStore) SaveSkill(s *db.Skill) (int64, error) {
	f.skill = s
	return 20, nil
}

func (f *fakeStore) ListWorkflows(tipoAgente string, activo *bool) ([]*db.Workflow, error) {
	return []*db.Workflow{{TipoAgente: tipoAgente}}, nil
}

func (f *fakeStore) SaveWorkflow(w *db.Workflow) (int64, error) {
	f.workflow = w
	return 30, nil
}

func TestSaveRuleNormalizaEntrada(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	id, err := svc.SaveRule(SaveRuleInput{
		TipoAgente:  " programador ",
		Categoria:   " arquitectura ",
		Titulo:      " Puerto ",
		Descripcion: " Desacoplar BD ",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("SaveRule: %v", err)
	}
	if id != 10 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.rule == nil || store.rule.TipoAgente != "programador" || store.rule.Titulo != "Puerto" {
		t.Fatalf("regla no normalizada: %+v", store.rule)
	}
}

func TestSaveWorkflowSerializaPasos(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	id, err := svc.SaveWorkflow(SaveWorkflowInput{
		TipoAgente:  "programador",
		Nombre:      "inicio",
		Descripcion: "flujo",
		Pasos:       []string{"uno", "dos"},
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("SaveWorkflow: %v", err)
	}
	if id != 30 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.workflow == nil || store.workflow.Pasos != "[\"uno\",\"dos\"]" {
		t.Fatalf("workflow no serializado: %+v", store.workflow)
	}
}

func TestSaveWorkflowRechazaPasosVacios(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	if _, err := svc.SaveWorkflow(SaveWorkflowInput{
		TipoAgente: "programador",
		Nombre:     "inicio",
	}); err == nil {
		t.Fatalf("esperaba error por pasos vacios")
	}
}

func TestSaveSkillNormalizaContratoCompleto(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	id, err := svc.SaveSkill(SaveSkillInput{
		TipoAgente:         " programador ",
		Nombre:             " goimports ",
		Descripcion:        " formatea go ",
		CuandoUsar:         " corregir imports ",
		Escenario:          " codigo ",
		Prioridad:          25,
		AliasesJSON:        `["fmt"]`,
		HerramientasJSON:   `["goimports"]`,
		Origen:             " builtin ",
		NivelRiesgo:        " bajo ",
		RequiereAprobacion: true,
		Activa:             false,
	})
	if err != nil {
		t.Fatalf("SaveSkill: %v", err)
	}
	if id != 20 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.skill == nil {
		t.Fatalf("skill no guardada")
	}
	if store.skill.TipoAgente != "programador" || store.skill.Nombre != "goimports" {
		t.Fatalf("skill no normalizada: %+v", store.skill)
	}
	if store.skill.Escenario != "codigo" || store.skill.Prioridad != 25 {
		t.Fatalf("contrato incompleto: %+v", store.skill)
	}
	if store.skill.AliasesJSON != `["fmt"]` || store.skill.HerramientasJSON != `["goimports"]` {
		t.Fatalf("listas no preservadas: %+v", store.skill)
	}
	if store.skill.Origen != "builtin" || store.skill.NivelRiesgo != "bajo" || !store.skill.RequiereAprobacion || store.skill.Activa {
		t.Fatalf("flags inesperadas: %+v", store.skill)
	}
}
