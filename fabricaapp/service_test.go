package fabricaapp

import (
	"strings"
	"testing"
)

func TestGenerateWebAPIBacklog(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre:      "Expedientes",
		Descripcion: "Gestion integral de expedientes",
		Tipo:        "web_api",
		Database:    true,
		Auth:        true,
		Docker:      true,
		I18n:        true,
		Idiomas:     []string{"es", "en", "fr"},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.Tasks) < 8 {
		t.Fatalf("se esperaban suficientes tareas base, got=%d", len(result.Tasks))
	}

	assertHasTask(t, result, "briefing")
	assertHasTask(t, result, "investigacion")
	assertHasTask(t, result, "arquitectura")
	assertHasTask(t, result, "persistencia")
	assertHasTask(t, result, "api_base")
	assertHasTask(t, result, "frontend_base")
	assertHasTask(t, result, "auth")
	assertHasTask(t, result, "i18n")
	assertHasTask(t, result, "qa_smoke")
	assertHasTask(t, result, "documentacion")
	assertHasTask(t, result, "docker")
	assertHasTask(t, result, "entrega_final")

	i18n := mustTask(t, result, "i18n")
	if len(i18n.Dependencias) == 0 {
		t.Fatalf("la tarea de i18n debe depender de tareas previas")
	}
	arquitectura := mustTask(t, result, "arquitectura")
	if len(arquitectura.Dependencias) != 2 {
		t.Fatalf("arquitectura deberia depender de briefing e investigacion: %+v", arquitectura)
	}
	if got := mustTask(t, result, "entrega_final"); len(got.Dependencias) < 2 {
		t.Fatalf("la entrega debe depender de qa/docs y opcionalmente docker: %+v", got)
	}
	if got := mustTask(t, result, "documentacion").Descripcion; !strings.Contains(got, "Orquesta de Alberto Avidad Fernandez") {
		t.Fatalf("descripcion docs sin atribucion: %s", got)
	}
	if got := mustTask(t, result, "entrega_final").Descripcion; !strings.Contains(got, "Orquesta de Alberto Avidad Fernandez") {
		t.Fatalf("descripcion entrega sin atribucion: %s", got)
	}
	arquitectura = mustTask(t, result, "arquitectura")
	if arquitectura.RolSugerido != "arquitecto" || arquitectura.Fase != "arquitectura" || arquitectura.Entregable == "" || len(arquitectura.CriteriosCierre) < 3 {
		t.Fatalf("SOP de arquitectura incompleto: %+v", arquitectura)
	}
	frontend := mustTask(t, result, "frontend_base")
	if frontend.RolSugerido != "frontend" || frontend.Entregable == "" {
		t.Fatalf("SOP de frontend incompleto: %+v", frontend)
	}
}

func TestGenerateCLIBacklogIsMinimal(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre: "cli-util",
		Tipo:   "cli",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if hasTask(result.Tasks, "frontend_base") {
		t.Fatalf("una app CLI no deberia incluir frontend")
	}
	if hasTask(result.Tasks, "api_base") {
		t.Fatalf("una app CLI no deberia incluir api por defecto")
	}
	assertHasTask(t, result, "briefing")
	assertHasTask(t, result, "investigacion")
	assertHasTask(t, result, "arquitectura")
	assertHasTask(t, result, "integracion")
	assertHasTask(t, result, "qa_smoke")
	assertHasTask(t, result, "documentacion")
	assertHasTask(t, result, "entrega_final")
}

func TestGenerateRequiresNameAndType(t *testing.T) {
	svc := NewService()
	if _, err := svc.Generate(AppSpec{Tipo: "web"}); err == nil {
		t.Fatalf("se esperaba error por nombre vacio")
	}
	if _, err := svc.Generate(AppSpec{Nombre: "SinTipo"}); err == nil {
		t.Fatalf("se esperaba error por tipo vacio")
	}
	if _, err := svc.Generate(AppSpec{Nombre: "Raro", Tipo: "desktop"}); err == nil {
		t.Fatalf("se esperaba error por tipo no soportado")
	}
}

func TestGenerateNormalizesI18nIdiomas(t *testing.T) {
	svc := NewService()
	result, err := svc.Generate(AppSpec{
		Nombre:  "Portal",
		Tipo:    "web",
		I18n:    true,
		Idiomas: []string{" ES ", "en", "es", ""},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	i18n := mustTask(t, result, "i18n")
	if want := "es, en"; !strings.Contains(i18n.Descripcion, want) {
		t.Fatalf("descripcion i18n inesperada: %s", i18n.Descripcion)
	}
}

func assertHasTask(t *testing.T, result GenerationResult, key string) {
	t.Helper()
	if !hasTask(result.Tasks, key) {
		t.Fatalf("falta la tarea %s en %+v", key, result.Tasks)
	}
}

func hasTask(tasks []BlueprintTask, key string) bool {
	for _, item := range tasks {
		if item.Key == key {
			return true
		}
	}
	return false
}

func mustTask(t *testing.T, result GenerationResult, key string) BlueprintTask {
	t.Helper()
	for _, item := range result.Tasks {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("no existe la tarea %s", key)
	return BlueprintTask{}
}
