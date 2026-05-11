package orquestaappdirectorintake

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

type appDirectorIntakeWizardQuestionSpecV0 struct {
	Field   string
	Summary string
	Options []string
}

var appDirectorIntakeWizardQuestionSpecsV0 = []appDirectorIntakeWizardQuestionSpecV0{
	{
		Field:   "source",
		Summary: "app_director_intake.question.source",
		Options: []string{"orquesta-web", "orquesta-mcp", "orquesta-cli"},
	},
	{Field: "locale", Summary: "app_director_intake.question.locale"},
	{Field: "nombre", Summary: "app_director_intake.question.nombre"},
	{Field: "objetivo", Summary: "app_director_intake.question.objetivo"},
	{
		Field:   "tipo_app",
		Summary: "app_director_intake.question.tipo_app",
		Options: []string{"web", "api", "cli", "automation", "mixed"},
	},
	{
		Field:   "datos.necesidad_funcional",
		Summary: "app_director_intake.question.datos.necesidad_funcional",
	},
}

func nextAppDirectorIntakeWizardFieldV0(
	draft orquestafactory.AppSpecRequestV0,
) string {
	required := []struct {
		field string
		value string
	}{
		{"request_id", draft.RequestID},
		{"source", draft.Source},
		{"locale", draft.Locale},
		{"nombre", draft.Nombre},
		{"objetivo", draft.Objetivo},
		{"tipo_app", draft.TipoApp},
	}
	for _, candidate := range required {
		if strings.TrimSpace(candidate.value) == "" {
			return candidate.field
		}
	}
	if draft.Datos.DBRequired && strings.TrimSpace(draft.Datos.NecesidadFuncional) == "" {
		return "datos.necesidad_funcional"
	}
	return ""
}

func appDirectorIntakeWizardQuestionV0(
	request AppDirectorIntakeWizardRequestV0,
	field string,
) (orquestacoreworkflow.DirectorQuestionV0, error) {
	spec := appDirectorIntakeWizardQuestionSpecForFieldV0(field)
	return orquestacoreworkflow.NewDirectorQuestionV0(orquestacoreworkflow.DirectorQuestionV0{
		QuestionID:   "question-ref-app-intake-" + safeDirectorIntakeRefPartV0(request.RequestRef+"-"+field),
		RunID:        request.RunRef,
		SourceGroup:  "app_director_intake_wizard",
		TargetGroup:  orquestacoreworkflow.DirectorQuestionTargetDirectorV0,
		Summary:      spec.Summary,
		Options:      append([]string(nil), spec.Options...),
		EvidenceRefs: []string{"evidence-ref-app-director-intake-wizard-v0", "field-ref-" + safeDirectorIntakeRefPartV0(field)},
		Blocking:     true,
		RequestedAt:  request.OccurredAt,
	})
}

func appDirectorIntakeWizardQuestionSpecForFieldV0(
	field string,
) appDirectorIntakeWizardQuestionSpecV0 {
	for _, spec := range appDirectorIntakeWizardQuestionSpecsV0 {
		if spec.Field == field {
			return spec
		}
	}
	return appDirectorIntakeWizardQuestionSpecV0{
		Field:   field,
		Summary: "app_director_intake.question." + safeDirectorIntakeRefPartV0(field),
	}
}
