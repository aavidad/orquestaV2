package orquestaappdirectorintake

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAdvanceAppDirectorIntakeWizardV0PideSiguienteCampo(t *testing.T) {
	result, err := AdvanceAppDirectorIntakeWizardV0(AppDirectorIntakeWizardRequestV0{
		RequestRef: "request-ref-wizard-001",
		Source:     "orquesta-web",
		Locale:     "es-ES",
		OccurredAt: "2026-05-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("AdvanceAppDirectorIntakeWizardV0: %v", err)
	}
	if result.Status != AppDirectorIntakeWizardStatusNeedsInputV0 ||
		result.NextQuestion == nil ||
		result.NextQuestion.Summary != "app_director_intake.question.nombre" {
		t.Fatalf("result=%+v", result)
	}
	if result.NextQuestion.TargetGroup != orquestacoreworkflow.DirectorQuestionTargetDirectorV0 ||
		!result.NextQuestion.Blocking {
		t.Fatalf("question=%+v", result.NextQuestion)
	}
}

func TestAdvanceAppDirectorIntakeWizardV0CreaAppSpecYRunPreparado(t *testing.T) {
	result, err := AdvanceAppDirectorIntakeWizardV0(AppDirectorIntakeWizardRequestV0{
		RequestRef: "request-ref-wizard-ready-001",
		Source:     "orquesta-mcp",
		Locale:     "es-ES",
		OccurredAt: "2026-05-10T10:05:00Z",
		Answers: []AppDirectorIntakeWizardAnswerV0{
			{Field: "nombre", Value: "Agenda"},
			{Field: "objetivo", Value: "Gestionar contactos y citas desde una API y una web."},
			{Field: "tipo_app", Value: "mixed"},
			{Field: "preferencias_tecnicas.arquitectura", Value: "hexagonal"},
			{Field: "agentes.autonomia", Value: "alta"},
		},
	})
	if err != nil {
		t.Fatalf("AdvanceAppDirectorIntakeWizardV0: %v", err)
	}
	if result.Status != AppDirectorIntakeWizardStatusReadyV0 ||
		result.AppSpec == nil ||
		result.Prepared == nil {
		t.Fatalf("result=%+v", result)
	}
	if result.AppSpec.Validation.Estado != "valida" ||
		result.Prepared.Run.AppSpecRef != result.AppSpec.SpecID {
		t.Fatalf("app_spec=%+v prepared=%+v", result.AppSpec, result.Prepared)
	}
	if len(result.Prepared.DirectorTasks) < 2 {
		t.Fatalf("director_tasks=%+v", result.Prepared.DirectorTasks)
	}
}

func TestAdvanceAppDirectorIntakeWizardV0UsaFactoryParaInvalidarEnums(t *testing.T) {
	result, err := AdvanceAppDirectorIntakeWizardV0(AppDirectorIntakeWizardRequestV0{
		RequestRef: "request-ref-wizard-invalid-001",
		Source:     "orquesta-web",
		Locale:     "es-ES",
		OccurredAt: "2026-05-10T10:10:00Z",
		Answers: []AppDirectorIntakeWizardAnswerV0{
			{Field: "nombre", Value: "Agenda"},
			{Field: "objetivo", Value: "Gestionar contactos."},
			{Field: "tipo_app", Value: "mainframe"},
		},
	})
	if err != nil {
		t.Fatalf("AdvanceAppDirectorIntakeWizardV0: %v", err)
	}
	if result.Status != AppDirectorIntakeWizardStatusInvalidV0 ||
		len(result.Issues) == 0 ||
		result.Issues[0].Field != "tipo_app" {
		t.Fatalf("result=%+v", result)
	}
}
