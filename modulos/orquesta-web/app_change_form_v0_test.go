package orquestaweb

import "testing"

func TestWebAppChangeFormV0MapeaContratoMCP(t *testing.T) {
	input := WebAppChangeFormV0{
		RequestID:          "req-change-web-001",
		CorrelationID:      "corr-change-web-001",
		Locale:             "es",
		RunRef:             "run-ref-web-change-001",
		AppRef:             "app-ref-agenda",
		ChangeRef:          "change-ref-web-001",
		UserIntent:         "Cambiar la web para mostrar vista semanal.",
		TargetArea:         "web",
		AcceptanceCriteria: []string{"vista semanal", "vista semanal"},
		AllowedWriteSet:    []string{"web/agenda"},
		ExternalProjectRef: "project-ref-opes",
		ExternalInterfaces: []string{"mcp-contract-ref-opes-v0"},
		ExternalWorkKind:   "documentation",
		ExternalWorkRefs:   []string{"domain-work-ref-opes-topic-001"},
	}.ToMCPRequestAppChangeInputV0()

	if input.AppChangeRequest.RunRef != "run-ref-web-change-001" ||
		input.AppChangeRequest.ChangeRef != "change-ref-web-001" ||
		input.AppChangeRequest.UserIntent != "Cambiar la web para mostrar vista semanal." ||
		len(input.AppChangeRequest.AcceptanceCriteria) != 1 ||
		input.AppChangeRequest.ExternalWork == nil ||
		input.AppChangeRequest.ExternalWork.ProjectRef != "project-ref-opes" {
		t.Fatalf("input=%+v", input)
	}
}
