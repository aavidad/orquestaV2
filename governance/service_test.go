package governance

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	rule     *db.Regla
	skill    *db.Skill
	workflow *db.Workflow
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
