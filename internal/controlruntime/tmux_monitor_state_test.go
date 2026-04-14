package controlruntime

import (
	"strings"
	"testing"
	"time"
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

func TestTMUXPaneHasTrustPromptDetectaPantallaRealGemini(t *testing.T) {
	captured := "" +
		"Gemini CLI v0.37.1\n" +
		"Do you trust the files in this folder?\n" +
		"> 1. Trust folder (geminiproj)\n" +
		"2. Trust parent folder (/tmp)\n" +
		"3. Don't trust\n"
	if !tmuxPaneHasTrustPrompt(captured) {
		t.Fatal("la pantalla real de trust de Gemini deberia detectarse como trust prompt")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedTrust {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedTrust)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla de trust de Gemini no deberia marcarse como ready")
	}
}

func TestTMUXPaneHasTrustPromptDetectaPantallaRealClaude(t *testing.T) {
	captured := "" +
		"Accessing workspace:\n" +
		"/var/lib/snapd/void\n" +
		"Quick safety check: Is this a project you created or one you trust?\n" +
		"Claude Code'll be able to read, edit, and execute files here.\n" +
		"❯ 1. Yes, I trust this folder\n" +
		"2. No, exit\n" +
		"Enter to confirm · Esc to cancel\n"
	if !tmuxPaneHasTrustPrompt(captured) {
		t.Fatal("la pantalla real de trust de Claude deberia detectarse como trust prompt")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedTrust {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedTrust)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla de trust de Claude no deberia marcarse como ready")
	}
}

func TestTMUXPaneHasUntrustedFolderWarningDetectaGeminiPostBootstrap(t *testing.T) {
	captured := "" +
		"We're making changes to Gemini CLI that may impact your workflow.\n" +
		"Skipping project agents due to untrusted folder. To enable\n" +
		"ensure that the project root is trusted.\n"
	if !tmuxPaneHasUntrustedFolderWarning(captured) {
		t.Fatal("la advertencia de carpeta no confiable de Gemini deberia detectarse")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedTrust {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedTrust)
	}
}

func TestTMUXTrustPromptDismissKeysGeminiYClaude(t *testing.T) {
	gemini := "" +
		"Do you trust the files in this folder?\n" +
		"1. Trust folder (gproj)\n" +
		"2. Trust parent folder (/tmp)\n"
	if got := tmuxTrustPromptDismissKeys(gemini); len(got) != 2 || got[0] != "1" || got[1] != "C-m" {
		t.Fatalf("dismiss keys Gemini inesperadas: %+v", got)
	}

	claude := "" +
		"Quick safety check: Is this a project you created or one you trust?\n" +
		"❯ 1. Yes, I trust this folder\n" +
		"2. No, exit\n" +
		"Enter to confirm\n"
	if got := tmuxTrustPromptDismissKeys(claude); len(got) != 1 || got[0] != "C-m" {
		t.Fatalf("dismiss keys Claude inesperadas: %+v", got)
	}
}

func TestTMUXPaneHasBypassPermissionsPromptDetectaClaude(t *testing.T) {
	captured := "" +
		"WARNING: Claude Code running in Bypass Permissions mode\n" +
		"1. No, exit\n" +
		"2. Yes, I accept\n" +
		"Enter to confirm\n"
	if !tmuxPaneHasBypassPermissionsPrompt(captured) {
		t.Fatal("la advertencia de bypass permissions de Claude deberia detectarse")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la advertencia de bypass permissions de Claude no deberia marcar ready")
	}
	if keys, ok := tmuxBootstrapPromptDismissKeys(captured); !ok || len(keys) != 2 || keys[0] != "2" || keys[1] != "C-m" {
		t.Fatalf("dismiss keys Claude bypass inesperadas: ok=%v keys=%+v", ok, keys)
	}
}

func TestTMUXPaneHasActionRequiredPromptDetectaGemini(t *testing.T) {
	captured := "" +
		"Action Required\n" +
		"Shell make build && ./orquesta tarea listar\n" +
		"Allow execution of: 'make, ./orquesta'?\n" +
		"1. Allow once\n" +
		"2. Allow for this session\n" +
		"3. No, suggest changes (esc)\n"
	if !tmuxPaneHasActionRequiredPrompt(captured) {
		t.Fatal("la pantalla Action Required de Gemini deberia detectarse")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla Action Required de Gemini no deberia marcarse como ready")
	}
	if keys, ok := tmuxBootstrapPromptDismissKeys(captured); !ok || len(keys) != 2 || keys[0] != "2" || keys[1] != "C-m" {
		t.Fatalf("dismiss keys Gemini action required inesperadas: ok=%v keys=%+v", ok, keys)
	}
}

func TestTMUXPaneHasActionRequiredPromptDetectaGeminiWriteFile(t *testing.T) {
	captured := "" +
		"Action Required\n" +
		"WriteFile Writing to internal/.../smoke_helper.go\n" +
		"Apply this change?\n" +
		"1. Allow once\n" +
		"2. Allow for this session\n" +
		"3. Modify with external editor\n" +
		"4. No, suggest changes (esc)\n"
	if !tmuxPaneHasActionRequiredPrompt(captured) {
		t.Fatal("la pantalla Action Required de Gemini con WriteFile deberia detectarse")
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla Action Required de Gemini con WriteFile no deberia marcarse como ready")
	}
	if keys, ok := tmuxBootstrapPromptDismissKeys(captured); !ok || len(keys) != 2 || keys[0] != "2" || keys[1] != "C-m" {
		t.Fatalf("dismiss keys Gemini WriteFile inesperadas: ok=%v keys=%+v", ok, keys)
	}
}

func TestTMUXPaneHasActiveTaskDetectaGeminiAcceptEdits(t *testing.T) {
	captured := "" +
		"Refactoring Control Plane Logic\n" +
		"Consolidation\n" +
		"> Type your message or @path/to/file\n" +
		"Shift+Tab to accept edits\n"
	if tmuxPaneHasActiveTask(captured) {
		t.Fatal("Gemini con accept edits y prompt activo no deberia contar como tarea ocupada")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusReady {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusReady)
	}
}

func TestTMUXPaneHasActiveTaskDetectaClaudeTransfiguring(t *testing.T) {
	captured := "" +
		"●\n" +
		"Transfiguring…\n" +
		"Micro-refactor control plane cyclically\n"
	if !tmuxPaneHasActiveTask(captured) {
		t.Fatal("Claude con estado Transfiguring deberia detectarse como tarea activa")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusRunning {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusRunning)
	}
}

func TestTMUXPaneHasActiveTaskDetectaGeminiThinkingConBusquedaReal(t *testing.T) {
	captured := "" +
		"│ Found 94 matches                                                         │\n" +
		"✦ tmux_cli_session is a staple in my tests. I'm moving on to\n" +
		"  cmd/controlplane_support.go to examine construirInstruccionMailboxInteractivo.\n" +
		"⠼ Thinking... (esc to cancel, 38s)                             ? for shortcuts\n" +
		"workspace (/directory)         branch                        sandbox\n" +
		"~/.../orquestador-gemini1      orq-orquestador-gemini1       no sandbox\n" +
		"*   Type your message or @path/to/file\n"
	if !tmuxPaneHasActiveTask(captured) {
		t.Fatal("Gemini pensando y ejecutando búsqueda real debería detectarse como tarea activa")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusRunning {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusRunning)
	}
}

func TestTMUXPaneHasUsageLimitPromptDetectaClaudeResetHorario(t *testing.T) {
	captured := "" +
		"You've hit your limit · resets 2am (Europe/Madrid)\n" +
		"/rate-limit-options\n" +
		"1. Stop and wait for limit to reset\n" +
		"2. Upgrade your plan\n"
	if !tmuxPaneHasUsageLimitPrompt(captured) {
		t.Fatal("la pantalla de cuota de Claude deberia detectarse")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedQuota {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedQuota)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("la pantalla de cuota de Claude no deberia marcar ready")
	}
}

func TestTMUXPaneHasClaudeAuthPromptDetectaLoginRequerido(t *testing.T) {
	captured := "" +
		"Orquesta: continua el trabajo acotado\n" +
		"Not logged in. Please run /login\n" +
		"⏵⏵ bypass permissions on\n"
	if !tmuxPaneHasClaudeAuthPrompt(captured) {
		t.Fatal("Claude deberia detectarse como auth requerida")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedAuth {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedAuth)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("una sesion de Claude pidiendo login no deberia marcar ready")
	}
}

func TestTMUXPaneHasGeminiAuthPromptDetectaEsperaDeAutenticacion(t *testing.T) {
	captured := "" +
		"Gemini CLI v0.37.1\n" +
		"Signed in with Google /auth\n" +
		"Plan: Gemini Code Assist in Google One AI Pro /upgrade\n" +
		"Waiting for authentication... (Press Esc or Ctrl+C to cancel)\n"
	if !tmuxPaneHasGeminiAuthPrompt(captured) {
		t.Fatal("Gemini deberia detectarse como auth requerida")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedAuth {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedAuth)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("una sesion de Gemini esperando autenticacion no deberia marcarse como ready")
	}
}

func TestTMUXPaneHasGeminiAuthPromptIgnoraSpinnerViejoSiYaHayPromptActivo(t *testing.T) {
	captured := "" +
		"Waiting for authentication... (Press Esc or Ctrl+C to cancel)\n" +
		"Shift+Tab to accept edits\n" +
		"> Type your message or @path/to/file\n"
	if tmuxPaneHasGeminiAuthPrompt(captured) {
		t.Fatal("Gemini no deberia seguir en auth si ya hay prompt activo")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusReady {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusReady)
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

func TestTMUXTrackOutputMomentActualizaCuandoCambiaElTail(t *testing.T) {
	now := time.Date(2026, 4, 13, 14, 30, 0, 0, time.UTC)
	prev := now.Add(-10 * time.Minute)

	sig, at := tmuxTrackOutputMoment("", "estado inicial\n> \n", now, prev)
	if sig == "" {
		t.Fatal("deberia generar firma semantica no vacia")
	}
	if !at.Equal(now) {
		t.Fatalf("last output inesperado tras primer cambio: got=%s want=%s", at, now)
	}

	sig2, at2 := tmuxTrackOutputMoment(sig, "estado inicial\n> \n", now.Add(2*time.Minute), at)
	if sig2 != sig {
		t.Fatalf("firma semantica inesperada: got=%q want=%q", sig2, sig)
	}
	if !at2.Equal(at) {
		t.Fatalf("no deberia mover last output sin cambio de tail: got=%s want=%s", at2, at)
	}

	_, at3 := tmuxTrackOutputMoment(sig, "estado inicial\npatch aplicado\n> \n", now.Add(3*time.Minute), at)
	if !at3.Equal(now.Add(3 * time.Minute)) {
		t.Fatalf("deberia mover last output al cambiar el tail: got=%s want=%s", at3, now.Add(3*time.Minute))
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

func TestTMUXPaneHasCodexAuthPromptDetectaTokenExpiradoEnMCP(t *testing.T) {
	captured := "" +
		"Bootstrap de Orquesta para Codex1.\n" +
		"Booting MCP server: codex_apps\n" +
		"MCP startup incomplete (failed: codex_apps)\n" +
		"Provided authentication token is expired. Please try signing in again.\n" +
		"token_expired\n" +
		"status 401\n"
	if !tmuxPaneHasCodexAuthPrompt(captured) {
		t.Fatal("Codex deberia detectar auth expirada en MCP como bloqueo de autenticacion")
	}
	if got := tmuxClassifyPaneState(captured); got != workerStatusBlockedAuth {
		t.Fatalf("estado de pane inesperado: got=%q want=%q", got, workerStatusBlockedAuth)
	}
	if tmuxPaneLooksReady(captured) {
		t.Fatal("una sesion de Codex con token expirado en MCP no deberia marcarse como ready")
	}
}

func TestTMUXLineLooksPromptNoConfundeMenuNumerado(t *testing.T) {
	if tmuxLineLooksPrompt("> 1. Sign in with ChatGPT") {
		t.Fatal("una opcion numerada de menu no deberia detectarse como prompt listo")
	}
}
