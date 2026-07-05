package orquestaoperatortelegram

import "testing"

func TestAdapterV0AutorizaChatYDespachaComandosSeguros(t *testing.T) {
	ports := &fakePortsV0{}
	adapter, issues := NewAdapterV0(ConfigV0{
		Enabled:             true,
		BotLinkRef:          "inodo-bot-link-ref",
		TokenConfigured:     true,
		AuthorizedChatRefs:  []string{"chat-ref-alberto"},
		RequireConfirmation: true,
	}, ports)
	if len(issues) > 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	response := adapter.HandleUpdateV0(UpdateV0{
		UpdateRef: "update-ref-1",
		ChatRef:   "chat ref Alberto",
		Text:      "/observe_goal goal-ref-123",
	})
	if response.Status != "accepted" || response.CommandKind != CommandObserveV0 || ports.observed != "goal-ref-123" {
		t.Fatalf("respuesta inesperada: response=%+v ports=%+v", response, ports)
	}
}

func TestAdapterV0DespachaMensajeAlCanalDirector(t *testing.T) {
	ports := &fakePortsV0{}
	adapter, issues := NewAdapterV0(ConfigV0{
		Enabled:            true,
		BotLinkRef:         "inodo-bot-link-ref",
		TokenConfigured:    true,
		AuthorizedChatRefs: []string{"chat-ref-alberto"},
	}, ports)
	if len(issues) > 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	response := adapter.HandleUpdateV0(UpdateV0{
		ChatRef: "chat-ref-alberto",
		Text:    "/msg run-ref-1 prioriza este goal",
	})
	if response.Status != "accepted" ||
		response.CommandKind != CommandMessageV0 ||
		ports.directorTarget != "run-ref-1" ||
		ports.directorBody != "prioriza este goal" {
		t.Fatalf("respuesta=%+v ports=%+v", response, ports)
	}
}

func TestAdapterV0BloqueaChatNoAutorizadoYStopSinConfirmacion(t *testing.T) {
	adapter, issues := NewAdapterV0(ConfigV0{
		Enabled:             true,
		BotLinkRef:          "inodo-bot-link-ref",
		TokenConfigured:     true,
		AuthorizedChatRefs:  []string{"chat-ref-alberto"},
		RequireConfirmation: true,
	}, &fakePortsV0{})
	if len(issues) > 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	unauthorized := adapter.HandleUpdateV0(UpdateV0{ChatRef: "otro", Text: "/status"})
	if unauthorized.Status != "blocked" || unauthorized.Issues[0].Code != ErrTelegramUnauthorizedChatV0 {
		t.Fatalf("unauthorized inesperado: %+v", unauthorized)
	}
	stop := adapter.HandleUpdateV0(UpdateV0{ChatRef: "chat-ref-alberto", Text: "/stop run-ref-1"})
	if stop.Status != "blocked" || stop.Issues[0].Code != ErrTelegramConfirmationV0 {
		t.Fatalf("stop sin confirmacion inesperado: %+v", stop)
	}
}

func TestConfigV0FaltaEnlaceInodoTokenOChatBloquea(t *testing.T) {
	_, issues := NewAdapterV0(ConfigV0{Enabled: true}, &fakePortsV0{})
	if len(issues) != 3 {
		t.Fatalf("esperaba enlace/token/chat faltantes: %+v", issues)
	}
	if RedactTelegramSecretV0("123456:secret") != "telegram-secret-configured" {
		t.Fatalf("token no redactado")
	}
}

func TestAdapterV0NoFiltraTokenEnResumen(t *testing.T) {
	adapter, issues := NewAdapterV0(ConfigV0{
		Enabled:            true,
		BotLinkRef:         "inodo-bot-link-ref",
		TokenConfigured:    true,
		AuthorizedChatRefs: []string{"chat-ref-alberto"},
	}, &fakePortsV0{summary: "estado operativo"})
	if len(issues) > 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	response := adapter.HandleUpdateV0(UpdateV0{ChatRef: "chat-ref-alberto", Text: "/status run-ref-1"})
	if response.Summary != "estado operativo" {
		t.Fatalf("summary inesperado: %+v", response)
	}
}

type fakePortsV0 struct {
	observed string
	summary  string
	err      error

	directorTarget string
	directorBody   string
}

func (ports *fakePortsV0) QueryStatusV0(string) (string, []string, error) {
	return firstNonEmptyTestV0(ports.summary, "status ok"), []string{"evidence-ref-status"}, ports.err
}

func (ports *fakePortsV0) QueryQueueV0(string) (string, []string, error) {
	return "queue ok", []string{"evidence-ref-queue"}, ports.err
}

func (ports *fakePortsV0) LaunchTaskV0(string, []string) (string, []string, error) {
	return "launch queued", []string{"evidence-ref-launch"}, ports.err
}

func (ports *fakePortsV0) ObserveGoalV0(goalRef string) (string, []string, error) {
	ports.observed = goalRef
	return "goal observed", []string{"evidence-ref-goal"}, ports.err
}

func (ports *fakePortsV0) SendDirectorMessageV0(targetRef string, body string, _ []string) (string, []string, error) {
	ports.directorTarget = targetRef
	ports.directorBody = body
	return "director message queued", []string{"evidence-ref-director-message"}, ports.err
}

func (ports *fakePortsV0) StopV0(string, []string) (string, []string, error) {
	if ports.err != nil {
		return "", nil, ports.err
	}
	return "stop accepted", []string{"evidence-ref-stop"}, nil
}

func (ports *fakePortsV0) HandoffV0(string) (string, []string, error) {
	return "handoff compact", []string{"evidence-ref-handoff"}, nil
}

func firstNonEmptyTestV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
