package orquestaappdirectorservice

import (
	"fmt"
	"strings"
)

func (source *serviceExternalFakeClosureSourceForTestV0) validateV0(
	request AppDirectorOperationalClosureRequestV0,
) error {
	if len(source.Task.RequiredTests) != 0 || len(request.RequiredTestEvidenceRefs) != 0 {
		return fmt.Errorf("composicion externa fake no debe requerir tests de programacion")
	}
	if err := serviceExternalFakeRejectConnectorDetailsForTestV0(
		append(append(append(append([]string{
			source.EntityRef,
			source.ArtifactRef,
			source.Task.TaskID,
			request.Run.RunID,
		}, request.Run.Tasks...),
			request.Run.Deliveries...),
			request.Run.AcceptedReviews...),
			request.WaitAgentRefs...),
	); err != nil {
		return err
	}
	for _, check := range []struct {
		values []string
		want   string
		field  string
	}{
		{values: request.Run.Tasks, want: source.Task.TaskID, field: "run.tasks"},
		{values: request.Run.Deliveries, want: source.DeliveryRef, field: "run.deliveries"},
		{values: request.Run.AcceptedReviews, want: source.AcceptedReviewRef, field: "run.accepted_reviews"},
		{values: source.Task.ContextRefs, want: source.EntityRef, field: "task.context_refs"},
	} {
		if !serviceStringInSetV0(check.values, check.want) {
			return fmt.Errorf("%s no contiene ref opaca %q", check.field, check.want)
		}
	}
	return nil
}

func serviceExternalFakeProductAdapterImportForTestV0(path string) bool {
	for _, fragment := range []string{
		"orquesta/modulos/orquesta-runtime-codex",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-opes-",
	} {
		if strings.HasPrefix(path, fragment) {
			return true
		}
	}
	return false
}

func serviceExternalFakeRejectConnectorDetailsForTestV0(refs []string) error {
	for _, ref := range refs {
		trimmed := strings.TrimSpace(strings.ToLower(ref))
		for _, fragment := range []string{
			"://",
			"/",
			"\\",
			"postgres",
			"sqlite",
			"mysql",
			"database",
			"dsn",
			"opes",
			"codex",
			"oauth",
			"token",
			"secret",
		} {
			if strings.Contains(trimmed, fragment) {
				return fmt.Errorf("ref externa fake expone detalle interno de conector: %q", ref)
			}
		}
	}
	return nil
}
