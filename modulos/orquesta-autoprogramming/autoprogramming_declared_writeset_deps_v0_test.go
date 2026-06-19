package orquestaautoprogramming

import "testing"

// Extension 2026-06-18: las tareas pueden declarar su propio write_set y depends_on
// explicito; ganan sobre la inferencia por area. Permite enviar por JSON un spec de
// app con dependencias finas (p. ej. hexagonal: dominio -> puertos -> adaptadores)
// sin codigo Go nuevo. Ver docs/orquesta_director_y_entregas_2026-06-18.md.
func TestBuildAutoprogrammingProgrammableWorkV0HonraWriteSetYDependsOnPorTareaV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{
				TaskRef:  "task-dominio",
				Area:     "dominio",
				WriteSet: []string{"internal/candidate/domain/candidate.go"},
			},
			{
				TaskRef:   "task-adaptador",
				Area:      "adaptador",
				WriteSet:  []string{"internal/candidate/adapters/repository/memory.go"},
				DependsOn: []string{"task-dominio"},
			},
		}
		// write-set global de respaldo (no deberia usarse al haber declarados).
		request.WriteSet = []string{
			"internal/candidate/domain/candidate.go",
			"internal/candidate/adapters/repository/memory.go",
		}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Tasks) != 2 {
		t.Fatalf("tasks=%d want 2", len(result.Work.Tasks))
	}

	byArea := map[string]int{}
	for i, group := range result.Work.Groups {
		byArea[group.Area] = i
	}
	dominio := result.Work.Tasks[byArea["dominio"]]
	adaptador := result.Work.Tasks[byArea["adaptador"]]

	if !stringSliceEqualForDeclaredTestV0(dominio.WriteSet, []string{"internal/candidate/domain/candidate.go"}) {
		t.Fatalf("dominio write_set=%v want solo candidate.go", dominio.WriteSet)
	}
	if !stringSliceEqualForDeclaredTestV0(adaptador.WriteSet, []string{"internal/candidate/adapters/repository/memory.go"}) {
		t.Fatalf("adaptador write_set=%v want solo memory.go", adaptador.WriteSet)
	}
	expectedDependency := autoprogrammingProgrammableTaskRefV0(request.RequestRef, byArea["dominio"])
	if !stringSliceEqualForDeclaredTestV0(adaptador.DependsOn, []string{expectedDependency}) {
		t.Fatalf("adaptador depends_on=%v want %s", adaptador.DependsOn, expectedDependency)
	}
}

func stringSliceEqualForDeclaredTestV0(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
