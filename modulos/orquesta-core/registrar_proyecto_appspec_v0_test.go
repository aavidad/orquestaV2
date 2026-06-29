package orquestacore

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRegistrarProyectoDesdeAppSpecV0ProduceBorrador(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)

	result, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan := result.ProyectoPlanBorrador
	if result.RegistroID == "" {
		t.Fatalf("registro_id required")
	}
	if plan.ProyectoIDPropuesto == "" || plan.Estado != EstadoProyectoPlanBorradorV0 {
		t.Fatalf("invalid project draft: %+v", plan)
	}
	if plan.AppSpecRef != cmd.AppSpec.SpecID {
		t.Fatalf("app_spec_ref=%q, want %q", plan.AppSpecRef, cmd.AppSpec.SpecID)
	}
	if len(plan.FasesIniciales) == 0 {
		t.Fatalf("expected phases")
	}
	if len(plan.Microtareas) != len(cmd.Backlog.Microtareas) {
		t.Fatalf("microtareas=%d, want %d", len(plan.Microtareas), len(cmd.Backlog.Microtareas))
	}
	if len(plan.BacklogNormalizado) != len(plan.Microtareas) {
		t.Fatalf("backlog_normalizado should mirror microtareas")
	}
	if !containsCoreStringV0(plan.ContratosRequeridos, "AppSpecV0") {
		t.Fatalf("missing required AppSpecV0 contract: %+v", plan.ContratosRequeridos)
	}
	if !containsCoreStringV0(plan.ContratosRequeridos, "FunctionContract v0") {
		t.Fatalf("missing required FunctionContract v0: %+v", plan.ContratosRequeridos)
	}
	if len(result.EventosDominio) != 1 || result.EventosDominio[0].Tipo != EventoProyectoRegistradoEnBorradorV0 {
		t.Fatalf("unexpected events: %+v", result.EventosDominio)
	}
}

func TestRegistrarProyectoDesdeAppSpecV0RechazaAppSpecNoValidada(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)
	cmd.AppSpec.Validation.Estado = "provisional"

	_, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	assertRegistrarErrorCodeV0(t, err, ErrAppSpecNoValidadaV0)
}

func TestRegistrarProyectoDesdeAppSpecV0RechazaBacklogVacio(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)
	cmd.Backlog.Microtareas = nil

	_, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	assertRegistrarErrorCodeV0(t, err, ErrBacklogVacioV0)
}

func TestRegistrarProyectoDesdeAppSpecV0RechazaSpecIDIncompatible(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)
	cmd.Backlog.SpecID = "spec-distinta"

	_, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	assertRegistrarErrorCodeV0(t, err, ErrBacklogIncompatibleConAppSpecV0)
}

func TestRegistrarProyectoDesdeAppSpecV0RechazaIdempotencyKeyVacia(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)
	cmd.IdempotencyKey = " "

	_, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	assertRegistrarErrorCodeV0(t, err, ErrIdempotencyKeyRequeridaV0)
}

func TestRegistrarProyectoDesdeAppSpecV0RechazaContratoFactoryNoSoportado(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)
	cmd.Backlog.SchemaVersion = "backlog_inicial_propuesto.v1"

	_, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	assertRegistrarErrorCodeV0(t, err, ErrContratoFactoryNoSoportadoV0)
}

func TestRegistrarProyectoDesdeAppSpecV0IdempotenciaDeterminista(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)

	first, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first.RegistroID != second.RegistroID {
		t.Fatalf("registro_id mismatch: %q != %q", first.RegistroID, second.RegistroID)
	}
	if first.ProyectoPlanBorrador.ProyectoIDPropuesto != second.ProyectoPlanBorrador.ProyectoIDPropuesto {
		t.Fatalf("proyecto_id mismatch: %q != %q", first.ProyectoPlanBorrador.ProyectoIDPropuesto, second.ProyectoPlanBorrador.ProyectoIDPropuesto)
	}
}

func TestRegistrarProyectoDesdeAppSpecV0SalidaSinDetallesAdaptador(t *testing.T) {
	cmd := validRegistrarProyectoCommandV0(t)

	result, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("registrar proyecto: %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	serialized := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"dsn",
		"postgres",
		"table",
		"tmux",
		"docker",
		"token",
		"oauth",
		"/home/",
		"mcp",
		"http://",
		"https://",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized project registration contains adapter detail %q: %s", forbidden, string(payload))
		}
	}
}

func validRegistrarProyectoCommandV0(t *testing.T) RegistrarProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	specID := "spec-core-001"
	return RegistrarProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: "idem-core-001",
		AppSpec: RegistrarAppSpecV0{
			SchemaVersion: RegistrarAppSpecSchemaV0,
			SpecID:        specID,
			RequestID:     "req-core-001",
			CreatedAt:     "2026-05-04T12:30:00Z",
			App: RegistrarAppInfoV0{
				Nombre: "Panel de reservas",
			},
			Validation: RegistrarValidationSummaryV0{
				Estado: "valida",
			},
		},
		AppSpecVersion: RegistrarProyectoDesdeAppSpecVersionV0,
		Backlog: RegistrarBacklogInicialV0{
			SchemaVersion: RegistrarBacklogInicialSchemaV0,
			SpecID:        specID,
			Fases: []RegistrarFaseInicialV0{
				{
					ID:       "discovery",
					Nombre:   "Descubrimiento",
					Objetivo: "Cerrar alcance y supuestos.",
					Orden:    10,
				},
				{
					ID:       "implementacion",
					Nombre:   "Implementacion",
					Objetivo: "Construir el primer slice.",
					Orden:    20,
				},
			},
			Microtareas: []RegistrarMicrotareaV0{
				{
					ID:               "BLG-001",
					Fase:             "discovery",
					ModuloSugerido:   "producto",
					Objetivo:         "Cerrar alcance inicial y registrar supuestos.",
					WriteSetPrevisto: []string{"docs/app_spec.md"},
					Contrato:         "AppSpecV0",
					Validacion:       "AppSpec revisada.",
					Bloqueos:         []string{"decision de producto"},
				},
				{
					ID:               "BLG-002",
					Fase:             "implementacion",
					ModuloSugerido:   "core",
					Objetivo:         "Implementar primer caso de uso vertical.",
					WriteSetPrevisto: []string{"core/"},
					Contrato:         "FunctionContract v0",
					Validacion:       "Tests del slice en verde.",
					Bloqueos:         []string{"BLG-001"},
				},
			},
			ContratosRequeridos: []string{"AppSpecV0", "FunctionContract v0"},
			Riesgos:             []string{"riesgo documentado"},
			PreguntasAbiertas:   []string{"pregunta abierta"},
		},
		BacklogVersion: RegistrarProyectoDesdeAppSpecVersionV0,
		Origen:         "orquesta-factory",
		CorrelationID:  "corr-core-001",
		RequestID:      "req-core-001",
		SolicitadoEn:   "2026-05-04T12:35:00Z",
	}
}

func assertRegistrarErrorCodeV0(t *testing.T, err error, want string) {
	t.Helper()
	var registrarErr RegistrarProyectoDesdeAppSpecErrorV0
	if !errors.As(err, &registrarErr) {
		t.Fatalf("error type=%T, want RegistrarProyectoDesdeAppSpecErrorV0", err)
	}
	if registrarErr.Code != want {
		t.Fatalf("code=%q, want %q", registrarErr.Code, want)
	}
}

func containsCoreStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
