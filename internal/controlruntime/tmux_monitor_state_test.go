package controlruntime

import (
	"strings"
	"testing"
)

func TestTMUXPaneLooksReadyIgnoraSpinnerANSI(t *testing.T) {
	captured := "\x1b[49m\x1b[K\x1b[11;2H\x1b[0m\x1b[49m\x1b[K\x1b[?25l\x1b[?2026l\x1b[?2026h\x1b[1;55H\x1b[0m"
	if tmuxPaneLooksReady(captured) {
		t.Fatal("una pantalla de spinner ANSI no deberia marcarse como ready")
	}
	if got := tmuxNormalizeCapture(captured); got != "" {
		t.Fatalf("la captura ANSI deberia quedar vacia tras sanearse, got=%q", got)
	}
}

func TestTMUXPaneLooksReadySoloAceptaPromptEnTailReciente(t *testing.T) {
	var spinner strings.Builder
	for i := 0; i < 400; i++ {
		spinner.WriteString("\x1b[49m\x1b[K\x1b[11;2H\x1b[0m")
	}
	captured := "Welcome to Codex\nHow can I help?\n" + spinner.String()
	if tmuxPaneLooksReady(captured) {
		t.Fatal("un prompt viejo fuera del tail no deberia marcar ready")
	}
}

func TestTMUXPaneLooksReadyAceptaPromptRecienteLimpio(t *testing.T) {
	captured := "" +
		"estado previo\n" +
		"otra linea\n" +
		"> \n"
	if !tmuxPaneLooksReady(captured) {
		t.Fatal("un prompt reciente y limpio deberia marcar ready")
	}
}

func TestTMUXPaneHasCodexAuthPromptToleraTailSinWelcome(t *testing.T) {
	captured := "" +
		"How can I help?\n" +
		"> 1. Sign in with ChatGPT\n" +
		"Usage included with Plus, Pro, Business, and Enterprise plans\n" +
		"2. Sign in with Device Code\n" +
		"3. Provide your own API key\n" +
		"Press Enter to continue\n"
	if !tmuxPaneHasCodexAuthPrompt(captured) {
		t.Fatal("las opciones de login de Codex deben bloquear el worker aunque falte la cabecera")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("una pantalla de login de Codex no debe marcar ready aunque contenga ruido viejo")
	}
}

func TestTMUXPaneHasCodexAuthPromptDetectaPantallaDeLogin(t *testing.T) {
	captured := "" +
		"Welcome to Codex\n" +
		"Sign in with ChatGPT to use Codex as part of your paid plan\n" +
		"> 1. Sign in with ChatGPT\n" +
		"2. Sign in with Device Code\n" +
		"3. Provide your own API key\n" +
		"Press Enter to continue\n"
	if !tmuxPaneHasCodexAuthPrompt(captured) {
		t.Fatal("la pantalla de login de Codex deberia detectarse como autenticacion requerida")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla de login de Codex no deberia marcarse como ready")
	}
}

func TestTMUXPaneHasUsageLimitPromptDetectaPantallaDeCuota(t *testing.T) {
	captured := "" +
		"Retoma el trabajo actual desde Orquesta.\n" +
		"■ You've hit your usage limit. To get more access now, send a request to your\n" +
		"admin or try again at 6:35 PM.\n" +
		"› Improve documentation in @filename\n"
	if !tmuxPaneHasUsageLimitPrompt(captured) {
		t.Fatal("la pantalla de cuota de Codex deberia detectarse como bloqueada por cuota")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla de cuota de Codex no deberia marcarse como ready")
	}
}

func TestTMUXPaneHasUsageLimitPromptDetectaCapturaRealCodex2(t *testing.T) {
	captured := "" +
		"  mailbox actual, ignóralo.\n" +
		"  No reabras frentes viejos ni reescribas módulos fuera del alcance inmediato.\n" +
		"  Tarea activa: #2 [asignada] Revisar catálogo y referencias de Orquestador.\n" +
		"  Empieza por la tarea asignada y evita tocar BD local salvo diagnóstico o\n" +
		"  recuperación.\n" +
		"  Si la acción es destructiva, irreversible o de riesgo alto, consulta antes.\n" +
		"\n" +
		"  Retoma el trabajo actual desde Orquesta. Proyecto: orquestador. Continuidad:\n" +
		"  Worktree activa en orq-orquestador-codex2/t2. checkpoint#1(stop), mailbox=1,\n" +
		"  project_context: Worktree activa en orq-orquestador-codex2/t2. 1 tarea(s)\n" +
		"  activas del agente, governance_catalog: Catálogo efectivo\n" +
		"  7df60d1723dfd3ead9bbfd75 (29 reglas, 7 skills, 4 workflows). Ignora cualquier\n" +
		"  conversación vieja que no coincida con la tarea activa o el mailbox actual.\n" +
		"  No abras frentes nuevos ni reescribas código fuera del alcance inmediato.\n" +
		"  Empieza por la tarea asignada y consulta Orquesta antes de desviarte.\n" +
		"\n" +
		"\n" +
		"■ You've hit your usage limit. To get more access now, send a request to your\n" +
		"admin or try again at 6:35 PM.\n" +
		"\n" +
		"\n" +
		"› Use /skills to list available skills\n" +
		"\n" +
		"  gpt-5.4 xhigh · 100% left · /home/berserk/Trabajo/orquesta\n"
	if !tmuxPaneHasUsageLimitPrompt(captured) {
		t.Fatal("la captura real de usage limit de Codex2 deberia detectarse como cuota")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la captura real de usage limit de Codex2 no deberia marcarse como ready")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedQuota {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedQuota)
	}
}

func TestTMUXPaneLooksReadyNoAceptaComposerPendienteDeEnvio(t *testing.T) {
	captured := "" +
		">>> SI_BLOQUEO=BLOQUEO: <motivo concreto>\n" +
		"\"SI_BLOQUEO=BLOQUEO: <motivo concreto>\" parece ser una declaración o indicador.\n" +
		">>> PROTOCOLO_ORQUESTA_MICRO NO_INTERPRETAR_COMO_PREGUNTA SI_NO_HAY_MICROTAREA=ACK-ESPERA\n" +
		"... Press Enter to send\n"
	if tmuxPaneHasPendingSubmit(captured) != true {
		t.Fatal("una captura con 'Press Enter to send' debe marcarse como pending submit")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("un pane con texto pendiente de envio no debe marcarse como ready")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusStarting {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusStarting)
	}
}

func TestTMUXClassifyPaneStatePriorizaAuthSobrePromptViejo(t *testing.T) {
	captured := "" +
		"How can I help?\n" +
		"> \n" +
		"Welcome to Codex\n" +
		"Sign in with ChatGPT to use Codex as part of your paid plan\n" +
		"> 1. Sign in with ChatGPT\n" +
		"2. Sign in with Device Code\n" +
		"3. Provide your own API key\n" +
		"Press Enter to continue\n"
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedAuth {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedAuth)
	}
}

func TestTMUXLineLooksPromptNoConfundeMenuNumerado(t *testing.T) {
	if tmuxLineLooksPrompt("> 1. Sign in with ChatGPT") {
		t.Fatal("una opcion numerada de menu no deberia detectarse como prompt listo")
	}
}
