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
	}.ToMCPRequestAppChangeInputV0()

	if input.AppChangeRequest.RunRef != "run-ref-web-change-001" ||
		input.AppChangeRequest.ChangeRef != "change-ref-web-001" ||
		input.AppChangeRequest.UserIntent != "Cambiar la web para mostrar vista semanal." ||
		len(input.AppChangeRequest.AcceptanceCriteria) != 1 {
		t.Fatalf("input=%+v", input)
	}
}
