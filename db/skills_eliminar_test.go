package db

import "testing"

func TestEliminarSkill(t *testing.T) {
	prepararDBTemporal(t)
	id, err := CrearSkill("Codex1", &Skill{
		TipoAgente:  "programador",
		Nombre:      "skill-borrable",
		Descripcion: "skill temporal",
		CuandoUsar:  "cuando haga falta",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}
	if err := EliminarSkill("Codex1", id); err != nil {
		t.Fatalf("EliminarSkill: %v", err)
	}
	if _, err := GetSkill(id); err == nil {
		t.Fatalf("la skill deberia haber sido borrada")
	}
}
