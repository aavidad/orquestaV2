package orquestaweb

import (
	"encoding/json"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppViewModelV0ResumeSpecYBacklogPropuesto(t *testing.T) {
	spec := validSpecForViewModelV0()
	spec.DefaultsApplied = []orquestafactory.DefaultAppliedV0{
		{Campo: "i18n.enabled", Valor: true, Motivo: "default"},
		{Campo: "docs.locales", Valor: []string{"es", "en"}, Motivo: "hereda i18n"},
	}
	spec.Validation.Warnings = []orquestafactory.ValidationIssue{{
		Code: "opcion_incompatible", Field: "deploy.target", Message: "warning publico",
	}}
	backlog := backlogForViewModelV0()

	vm := NewWebNuevaAppViewModelV0(spec, backlog)

	if vm.Estado != WebNuevaAppEstadoValida {
		t.Fatalf("estado=%q", vm.Estado)
	}
	if vm.ResumenApp.Nombre != "Agenda" || vm.ResumenApp.TipoApp != "web" || vm.ResumenApp.Locale != "es" || vm.ResumenApp.DefaultLocale != "es" {
		t.Fatalf("resumen_app=%+v", vm.ResumenApp)
	}
	if len(vm.DefaultsAplicados) != 2 || vm.DefaultsAplicados[0].Valor != "true" || vm.DefaultsAplicados[1].Valor != "es,en" {
		t.Fatalf("defaults=%+v", vm.DefaultsAplicados)
	}
	if len(vm.Fases) != 1 || vm.Fases[0].ID != "discovery" {
		t.Fatalf("fases=%+v", vm.Fases)
	}
	if len(vm.Microtareas) != 1 || vm.Microtareas[0].WriteSetPrevisto[0] != "docs/app_spec.md" || vm.Microtareas[0].Bloqueos[0] != "pregunta legal" {
		t.Fatalf("microtareas=%+v", vm.Microtareas)
	}
	if len(vm.Riesgos) != 1 || vm.Riesgos[0] != "riesgo compliance" || len(vm.Warnings) != 1 {
		t.Fatalf("riesgos/warnings: riesgos=%+v warnings=%+v", vm.Riesgos, vm.Warnings)
	}
}

func TestWebNuevaAppBacklogPreviewV0CompactaFasesYMicrotareas(t *testing.T) {
	vm := NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0())

	if vm.BacklogPreview.SchemaVersion != NuevaAppBacklogPreviewSchemaV0 ||
		vm.BacklogPreview.TotalFases != 1 ||
		vm.BacklogPreview.TotalMicrotareas != 1 ||
		!vm.BacklogPreview.TieneBloqueos {
		t.Fatalf("preview inesperado: %+v", vm.BacklogPreview)
	}
	phase := vm.BacklogPreview.Fases[0]
	if phase.Key != "discovery" || phase.Titulo != "Descubrimiento" || phase.Microtareas != 1 {
		t.Fatalf("fase preview sin semantica basica: %+v", phase)
	}
	task := vm.BacklogPreview.Microtareas[0]
	if task.Key != "BLG-001" ||
		task.Fase != "discovery" ||
		task.Titulo != "Cerrar alcance" ||
		task.ModuloFrontera != "producto / AppSpecV0" ||
		task.WriteSetPrevisto[0] != "docs/app_spec.md" ||
		task.Bloqueos[0] != "pregunta legal" ||
		task.CriterioCierre != "revision completa" {
		t.Fatalf("microtarea preview incompleta: %+v", task)
	}

	raw, err := json.MarshalIndent(vm.BacklogPreview, "", "  ")
	if err != nil {
		t.Fatalf("marshal preview: %v", err)
	}
	got := string(raw)
	want := `{
  "schema_version": "nueva_app_backlog_preview.v0",
  "fases": [
    {
      "key": "discovery",
      "titulo": "Descubrimiento",
      "objetivo": "Cerrar alcance",
      "orden": 10,
      "microtareas": 1
    }
  ],
  "microtareas": [
    {
      "key": "BLG-001",
      "fase": "discovery",
      "titulo": "Cerrar alcance",
      "modulo_frontera": "producto / AppSpecV0",
      "write_set_previsto": [
        "docs/app_spec.md"
      ],
      "bloqueos": [
        "pregunta legal"
      ],
      "criterio_cierre": "revision completa"
    }
  ],
  "total_fases": 1,
  "total_microtareas": 1,
  "tiene_bloqueos": true
}`
	if got != want {
		t.Fatalf("snapshot preview:\nwant:\n%s\n\ngot:\n%s", want, got)
	}
	forbidden := []string{"MicrotareaPropuestaV0", "FaseInicialV0", "modulo_sugerido", "validacion"}
	for _, value := range forbidden {
		if strings.Contains(got, value) {
			t.Fatalf("preview expone detalle no permitido %q: %s", value, got)
		}
	}
}

func TestWebNuevaAppViewModelV0RepresentaPreguntasAbiertasComoRequiereDatos(t *testing.T) {
	spec := validSpecForViewModelV0()
	spec.Scope.PreguntasAbiertas = []string{" definir audiencia "}
	backlog := backlogForViewModelV0()
	backlog.PreguntasAbiertas = []string{"definir audiencia", "confirmar SLA"}

	vm := NewWebNuevaAppViewModelV0(spec, backlog)

	if vm.Estado != WebNuevaAppEstadoRequiereDatos {
		t.Fatalf("estado=%q", vm.Estado)
	}
	if len(vm.PreguntasAbiertas) != 2 || vm.PreguntasAbiertas[0] != "definir audiencia" || vm.PreguntasAbiertas[1] != "confirmar SLA" {
		t.Fatalf("preguntas=%+v", vm.PreguntasAbiertas)
	}
}

func TestWebNuevaAppViewModelV0RepresentaErroresPublicos(t *testing.T) {
	issues := []orquestafactory.ValidationIssue{{
		Code:    orquestafactory.ErrAppSpecInvalida,
		Field:   "nombre",
		Message: "campo obligatorio",
	}}

	vm := NewWebNuevaAppErrorViewModelV0(" req-err ", " es ", issues)

	if vm.Estado != WebNuevaAppEstadoInvalida || vm.RequestID != "req-err" || vm.Locale != "es" {
		t.Fatalf("vm error=%+v", vm)
	}
	if len(vm.ErroresPublicos) != 1 || vm.ErroresPublicos[0].Code != orquestafactory.ErrAppSpecInvalida {
		t.Fatalf("errores_publicos=%+v", vm.ErroresPublicos)
	}
	if len(vm.Fases) != 0 || len(vm.Microtareas) != 0 {
		t.Fatalf("un error publico no debe inventar backlog: fases=%+v microtareas=%+v", vm.Fases, vm.Microtareas)
	}
}

func validSpecForViewModelV0() orquestafactory.AppSpecV0 {
	return orquestafactory.AppSpecV0{
		SchemaVersion: orquestafactory.AppSpecSchemaV0,
		SpecID:        "spec-agenda-req-1",
		RequestID:     "req-1",
		Locale:        "es",
		App: orquestafactory.AppInfoV0{
			Nombre:      "Agenda",
			Slug:        "agenda",
			Objetivo:    "Coordinar ensayos",
			Descripcion: "Gestion operativa",
			TipoApp:     "web",
		},
		Scope: orquestafactory.ScopeV0{},
		I18N: orquestafactory.I18NSpecV0{
			Enabled:       true,
			DefaultLocale: "es",
			Locales:       []string{"es", "en"},
		},
		Platforms: []string{"web"},
		Deploy:    orquestafactory.DeploySpecV0{Target: "sin_preferencia"},
		Validation: orquestafactory.ValidationSummaryV0{
			Estado: "valida",
		},
	}
}

func backlogForViewModelV0() orquestafactory.BacklogInicialPropuestoV0 {
	return orquestafactory.BacklogInicialPropuestoV0{
		SchemaVersion: orquestafactory.BacklogInicialPropuestoSchemaV0,
		SpecID:        "spec-agenda-req-1",
		Fases: []orquestafactory.FaseInicialV0{{
			ID:       "discovery",
			Nombre:   "Descubrimiento",
			Objetivo: "Cerrar alcance",
			Orden:    10,
		}},
		Microtareas: []orquestafactory.MicrotareaPropuestaV0{{
			ID:               "BLG-001",
			Fase:             "discovery",
			ModuloSugerido:   "producto",
			Objetivo:         "Cerrar alcance",
			WriteSetPrevisto: []string{"docs/app_spec.md"},
			Contrato:         "AppSpecV0",
			Validacion:       "revision completa",
			Bloqueos:         []string{"pregunta legal"},
		}},
		ContratosRequeridos: []string{"AppSpecV0"},
		Riesgos:             []string{"riesgo compliance"},
	}
}
