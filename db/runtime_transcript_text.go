package db

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func extraerLineasTranscript(raw string) (string, []string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	if raw == "" {
		return "", nil
	}
	parts := strings.Split(raw, "\n")
	if strings.HasSuffix(raw, "\n") {
		return "", parts[:len(parts)-1]
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return parts[len(parts)-1], parts[:len(parts)-1]
}

func extraerLineasTranscriptConOffsets(raw string, startOffset int64) (string, []transcriptRawLine) {
	if raw == "" {
		return "", nil
	}
	out := make([]transcriptRawLine, 0, 8)
	lineStart := 0
	for i := 0; i < len(raw); {
		switch raw[i] {
		case '\r':
			out = append(out, transcriptRawLine{
				ByteOffset: startOffset + int64(lineStart),
				Raw:        raw[lineStart:i],
			})
			if i+1 < len(raw) && raw[i+1] == '\n' {
				i += 2
			} else {
				i++
			}
			lineStart = i
		case '\n':
			out = append(out, transcriptRawLine{
				ByteOffset: startOffset + int64(lineStart),
				Raw:        raw[lineStart:i],
			})
			i++
			lineStart = i
		default:
			i++
		}
	}
	if lineStart >= len(raw) {
		return "", out
	}
	return raw[lineStart:], out
}

func limpiarLineaTranscript(raw string) string {
	raw = oscTranscriptRegexp.ReplaceAllString(raw, "")
	raw = ansiTranscriptRegexp.ReplaceAllString(raw, "")
	raw = strings.ReplaceAll(raw, "\x00", "")
	raw = strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, raw)
	raw = recortarPreambuloSistemaHastaFallo(raw)
	raw = strings.TrimSpace(raw)
	return raw
}

func SanitizarTextoObservabilidadRuntime(raw string) string {
	return limpiarPrefijoRuidoFalloRuntime(limpiarLineaTranscript(raw))
}

func recortarPreambuloSistemaHastaFallo(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "Script started on ") {
		return raw
	}
	panicMarkers := []string{
		"The application panicked (crashed).",
		"thread 'main' panicked",
		"panic:",
		"fatal error:",
	}
	best := -1
	for _, marker := range panicMarkers {
		idx := strings.Index(raw, marker)
		if idx <= 0 {
			continue
		}
		if best < 0 || idx < best {
			best = idx
		}
	}
	if best < 0 {
		return raw
	}
	return raw[best:]
}

func limpiarPrefijoRuidoFalloRuntime(raw string) string {
	raw = strings.TrimSpace(raw)
	failureMarkers := []string{
		"The application panicked (crashed).",
		"thread 'main' panicked",
		"panic:",
		"fatal error:",
	}
	for _, marker := range failureMarkers {
		idx := strings.Index(raw, marker)
		if idx <= 0 {
			continue
		}
		prefix := strings.TrimSpace(raw[:idx])
		if prefix == "." || prefix == ":" || prefix == "|" {
			return raw[idx:]
		}
		if !strings.Contains(prefix, " ") && utf8.RuneCountInString(prefix) <= 4 {
			return raw[idx:]
		}
	}
	return raw
}

func normalizarTextoTranscript(raw string) string {
	raw = strings.ToLower(limpiarLineaTranscript(raw))
	if raw == "" {
		return ""
	}
	return strings.Join(strings.Fields(raw), " ")
}

func descartarRuidoTranscript(stream, texto, normalized, classification string) bool {
	if strings.TrimSpace(stream) != "pty_out" {
		return false
	}
	if strings.TrimSpace(classification) != "" {
		return false
	}
	texto = strings.TrimSpace(texto)
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return true
	}
	if esBannerRuidoCodex(normalized) {
		return true
	}
	if esRuidoProgresoUITranscript(texto, normalized) {
		return true
	}
	alnum := contarRunasSemanticas(normalized)
	if alnum == 0 {
		return true
	}
	if alnum < 3 && len(strings.Fields(normalized)) <= 1 {
		return true
	}
	total := contarRunasVisibles(texto)
	if total == 0 {
		return true
	}
	if float64(alnum)/float64(total) < 0.25 {
		return true
	}
	return false
}

func esRuidoProgresoUITranscript(texto, normalized string) bool {
	texto = strings.TrimSpace(texto)
	normalized = strings.TrimSpace(normalized)
	if texto == "" || normalized == "" {
		return false
	}
	markers := []string{
		"esc to interrupt",
		"use /skills to list available skills",
		"use /skill",
		"waiting for background terminal",
		"waited for background terminal",
		"background terminal running",
		"background terminals running",
		"/ps to",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if strings.Contains(normalized, "working(") &&
		(strings.Contains(texto, "◦") || strings.Contains(texto, "•") || strings.Contains(texto, "─")) {
		return true
	}
	if strings.Contains(normalized, "explored") &&
		(strings.Contains(normalized, "└ search ") || strings.Contains(normalized, "└ read ")) {
		return true
	}
	if strings.Count(texto, "◦")+strings.Count(texto, "•") >= 4 && strings.Contains(normalized, "working") {
		return true
	}
	if transcriptPareceRuidoSpinnerCorrupto(texto) {
		return true
	}
	return false
}

func transcriptPareceRuidoSpinnerCorrupto(texto string) bool {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return false
	}
	duplicados := contarDuplicadosAdyacentes(texto)
	if duplicados < 8 {
		return false
	}
	ratioSpinner := ratioRunasSpinner(texto)
	if ratioSpinner >= 0.75 {
		return true
	}
	colapsado := strings.ToLower(colapsarRunasRepetidas(texto))
	if strings.Contains(colapsado, "working") && strings.Count(texto, "◦")+strings.Count(texto, "•") >= 2 {
		return true
	}
	semanticas := contarRunasSemanticas(texto)
	if semanticas <= 30 {
		return false
	}
	if contarSeparadoresTranscript(texto) >= 5 {
		return false
	}
	return ratioSpinner >= 0.55
}

func runtimeTranscriptClassificationCarriesSemanticProgress(classification string) bool {
	switch strings.ToLower(strings.TrimSpace(classification)) {
	case "progress_update",
		"patch_or_code_evidence",
		"task_completed",
		"ready_for_review",
		"tool_result_ok":
		return true
	default:
		return false
	}
}

func colapsarRunasRepetidas(raw string) string {
	var out strings.Builder
	var prev rune
	hasPrev := false
	for _, r := range raw {
		if hasPrev && r == prev {
			continue
		}
		out.WriteRune(r)
		prev = r
		hasPrev = true
	}
	return out.String()
}

func contarDuplicadosAdyacentes(raw string) int {
	count := 0
	var prev rune
	hasPrev := false
	for _, r := range raw {
		if hasPrev && r == prev && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			count++
		}
		prev = r
		hasPrev = true
	}
	return count
}

func contarSeparadoresTranscript(raw string) int {
	count := 0
	for _, r := range raw {
		if unicode.IsSpace(r) || r == '•' || r == '◦' || r == '|' || r == '└' || r == '-' {
			count++
		}
	}
	return count
}

func ratioRunasSpinner(raw string) float64 {
	semanticas := 0
	spinner := 0
	for _, r := range strings.ToLower(raw) {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			continue
		}
		semanticas++
		switch r {
		case 'w', 'o', 'r', 'k', 'i', 'n', 'g':
			spinner++
		}
	}
	if semanticas == 0 {
		return 0
	}
	return float64(spinner) / float64(semanticas)
}

func esBannerRuidoCodex(normalized string) bool {
	if normalized == "" {
		return false
	}
	banners := []string{
		"perfil activo:",
		"codex_home:",
		"credenciales:",
		"consejo: si es el primer arranque, ejecuta",
		"https://github.com/openai/codex/releases/latest",
		"https://chatgpt.com/codex/settings/usage",
		"https://chatgpt.com/explore/pro",
		"(http://localhost:8080).",
	}
	for _, marker := range banners {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func contarRunasSemanticas(raw string) int {
	count := 0
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			count++
		}
	}
	return count
}

func contarRunasVisibles(raw string) int {
	count := 0
	for _, r := range raw {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func ClasificarTextoTranscript(normalized string) string {
	return clasificarTextoTranscript(normalized)
}

func clasificarTextoTranscript(normalized string) string {
	if normalized == "" {
		return ""
	}
	if transcriptNormalizedPareceRuidoUI(normalized) {
		return "ui_noise"
	}
	panicPhrases := []string{
		"panic:",
		"fatal error:",
		"the application panicked",
		"thread 'main' panicked",
		"stack backtrace:",
		"segmentation fault",
		"sigsegv",
		"traceback (most recent call last):",
		"uncaught exception",
		"unexpected error",
	}
	for _, phrase := range panicPhrases {
		if strings.Contains(normalized, phrase) {
			return "runtime_panic"
		}
	}
	donePhrases := []string{
		"<done/>",
		"<done>",
		"<completado/>",
		"<terminado/>",
		"tarea terminada",
		"tarea completada",
		"tarea finalizada",
		"proceso terminado",
		"pipeline step finished",
		"he terminado la tarea",
		"he terminado mi trabajo",
		"acabo de terminar",
	}
	for _, phrase := range donePhrases {
		if strings.Contains(normalized, phrase) {
			return "task_completed"
		}
	}
	crashPhrases := []string{
		"aborted",
		"core dumped",
		"broken pipe",
		"unexpected eof",
		"connection reset by peer",
		"process exited",
	}
	for _, phrase := range crashPhrases {
		if strings.Contains(normalized, phrase) {
			return "runtime_crash"
		}
	}
	if strings.Contains(normalized, "goroutine ") && strings.Contains(normalized, "[") {
		return "runtime_panic"
	}
	reviewReadyPhrases := []string{
		"listo para review",
		"lista para review",
		"listo para revisión",
		"lista para revisión",
		"ready for review",
		"ready to review",
		"puedes revisar",
		"ya puede revisarse",
		"he terminado y está listo para review",
		"he terminado y esta listo para review",
	}
	for _, phrase := range reviewReadyPhrases {
		if strings.Contains(normalized, phrase) {
			return "ready_for_review"
		}
	}
	reviewApprovedPhrases := []string{
		"review aprobada",
		"review aprobado",
		"revisión aprobada",
		"revision aprobada",
		"review: approved",
		"approved after review",
		"lgtm",
		"looks good to me",
	}
	for _, phrase := range reviewApprovedPhrases {
		if strings.Contains(normalized, phrase) {
			return "review_approved"
		}
	}
	reviewChangesPhrases := []string{
		"cambios solicitados",
		"cambios pedidos",
		"review con cambios",
		"revisión con cambios",
		"revision con cambios",
		"changes requested",
		"requesting changes",
	}
	for _, phrase := range reviewChangesPhrases {
		if strings.Contains(normalized, phrase) {
			return "review_changes_requested"
		}
	}
	reviewBlockedPhrases := []string{
		"review bloqueada",
		"review bloqueado",
		"revisión bloqueada",
		"revision bloqueada",
		"review blocked",
		"review on hold",
	}
	for _, phrase := range reviewBlockedPhrases {
		if strings.Contains(normalized, phrase) {
			return "review_blocked"
		}
	}
	replanPhrases := []string{
		"qué hago ahora",
		"que hago ahora",
		"cuál es el siguiente paso",
		"cual es el siguiente paso",
		"no tengo siguiente paso",
		"no tengo claro el siguiente paso",
		"no veo el siguiente frente",
		"no encuentro el siguiente frente",
		"what should i do next",
		"what next",
	}
	for _, phrase := range replanPhrases {
		if strings.Contains(normalized, phrase) {
			return "needs_replan"
		}
	}
	bootstrapGuidancePhrases := []string{
		"retoma el trabajo actual desde orquesta",
		"write-set preferente",
		"salida minima",
		"salida minima:",
		"si la accion es destructiva",
		"si la acción es destructiva",
		"usa caveman",
		"guidance durable escrita en inbox",
	}
	for _, phrase := range bootstrapGuidancePhrases {
		if strings.Contains(normalized, phrase) {
			return "bootstrap_guidance"
		}
	}
	if pareceErrorServidorLocal(normalized) {
		return "server_url_error"
	}
	if transcriptNormalizedPareceResultadoToolError(normalized) {
		return "tool_result_error"
	}
	if transcriptNormalizedPareceResultadoToolOK(normalized) {
		return "tool_result_ok"
	}
	if transcriptNormalizedParecePatchOCodigo(normalized) {
		return "patch_or_code_evidence"
	}
	toolExecutionPhrases := []string{
		"go test",
		"go build",
		"git status",
		"git diff",
		"rg ",
		"npm test",
		"pnpm test",
		"pytest",
		"cargo test",
	}
	for _, phrase := range toolExecutionPhrases {
		if strings.Contains(normalized, phrase) {
			return "tool_execution"
		}
	}
	toolExplorationPhrases := []string{
		"explored",
		"search ",
		"read ",
	}
	for _, phrase := range toolExplorationPhrases {
		if strings.Contains(normalized, phrase) {
			return "tool_exploration"
		}
	}
	if transcriptNormalizedPareceExploracionGrep(normalized) {
		return "tool_exploration"
	}
	if transcriptNormalizedPareceProgreso(normalized) {
		return "progress_update"
	}
	approvalPhrases := []string{
		"si me dejas",
		"me dejas",
		"si me permites",
		"me permites",
		"quieres que",
		"te parece bien si",
		"puedo hacerlo",
		"puedo seguir",
		"me autorizas",
		"do you want me to",
	}
	for _, phrase := range approvalPhrases {
		if strings.Contains(normalized, phrase) {
			return "approval_request"
		}
	}
	if parecePreguntaAprobacion(normalized) {
		return "approval_request"
	}
	waitingPhrases := []string{
		"espero tu respuesta",
		"quedo a la espera",
		"esperando tu confirmacion",
		"esperando tu confirmación",
		"esperando aprobacion",
		"esperando aprobación",
		"awaiting approval",
		"waiting for approval",
		"waiting for your answer",
	}
	for _, phrase := range waitingPhrases {
		if strings.Contains(normalized, phrase) {
			return "waiting_human"
		}
	}
	cliQueryPhrases := []string{
		"que comando",
		"qué comando",
		"cual es el comando",
		"cuál es el comando",
		"como ejecuto",
		"cómo ejecuto",
		"como lanzo",
		"cómo lanzo",
		"how do i run",
		"which command should i run",
		"what command should i run",
	}
	for _, phrase := range cliQueryPhrases {
		if strings.Contains(normalized, phrase) {
			return "cli_query"
		}
	}
	credentialsPhrases := []string{
		"faltan credenciales",
		"falta credencial",
		"necesito credenciales",
		"necesito acceso",
		"me falta acceso",
		"missing credentials",
		"need credentials",
		"need access",
		"missing access",
		"missing api key",
		"missing oauth",
		"token missing",
	}
	for _, phrase := range credentialsPhrases {
		if strings.Contains(normalized, phrase) {
			return "credentials_request"
		}
	}
	blockedPhrases := []string{
		"bloqueado",
		"no puedo continuar sin",
		"no puedo seguir sin",
		"cannot continue without",
		"can't continue without",
		"blocked on",
		"pendiente de credenciales",
		"faltan credenciales",
		"falta acceso",
	}
	for _, phrase := range blockedPhrases {
		if strings.Contains(normalized, phrase) {
			return "blocked"
		}
	}
	return ""
}

func transcriptNormalizedPareceRuidoUI(normalized string) bool {
	if transcriptNormalizedParecePromptShell(normalized) {
		return true
	}
	markers := []string{
		"esc to interrupt",
		"use /skills to list available skills",
		"use /skill",
		"waiting for background terminal",
		"waited for background terminal",
		"background terminal running",
		"background terminals running",
		"/ps to",
		"working(",
		"summarize recent commits",
		"+1 lines",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if !strings.Contains(normalized, " ") && strings.ContainsAny(normalized, "◦•") {
		return true
	}
	return false
}

func transcriptNormalizedParecePromptShell(normalized string) bool {
	fields := strings.Fields(strings.TrimSpace(normalized))
	if len(fields) == 0 || len(fields) > 2 {
		return false
	}
	last := fields[len(fields)-1]
	switch last {
	case "$", "#", ">":
		return true
	}
	if !(strings.HasSuffix(last, "$") || strings.HasSuffix(last, "#") || strings.HasSuffix(last, ">")) {
		return false
	}
	if strings.Contains(last, "@") || strings.Contains(last, ":") || strings.Contains(last, "~/") || strings.Contains(last, "/") {
		return true
	}
	for _, shell := range []string{"bash", "zsh", "sh", "fish", "pwsh", "powershell"} {
		if strings.Contains(last, shell) {
			return true
		}
	}
	return false
}

func transcriptNormalizedParecePatchOCodigo(normalized string) bool {
	if transcriptNormalizedPareceResultadoToolOK(normalized) || transcriptNormalizedPareceExploracionGrep(normalized) {
		return false
	}
	markers := []string{
		"diff --git",
		"*** begin patch",
		"@@",
		"```go",
		"```ts",
		"```tsx",
		"```js",
		"```jsx",
		"```py",
		"```rs",
		"package ",
		"func ",
		"type ",
		"struct {",
		"interface {",
		"const ",
		"var ",
		"import (",
		"class ",
		"def ",
		"edited ",
		"updated ",
		"aplique un slice",
		"apliqué un slice",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if strings.Contains(normalized, "ok.") && strings.Contains(normalized, "test") {
		return true
	}
	return false
}

func transcriptNormalizedPareceResultadoToolError(normalized string) bool {
	markers := []string{
		"exit status 1",
		"exit code 1",
		"exit code ",
		"command not found",
		"permission denied",
		"no such file or directory",
		"build failed",
		"test failed",
		"tests failed",
		"failed in ",
		"error:",
		" errors",
		" fail ",
		" fail:",
		" failed",
		"undefined:",
		"timed out",
		"timeout",
		"context deadline exceeded",
		"connection refused",
		"address already in use",
		"syntax error",
		"unexpected token",
		"unknown flag:",
		"flag provided but not defined",
		"no matches found",
		" stderr",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if strings.HasPrefix(normalized, "fail ") || strings.HasPrefix(normalized, "--- fail:") {
		return true
	}
	return false
}

func transcriptNormalizedPareceResultadoToolOK(normalized string) bool {
	markers := []string{
		"tests passed",
		"all checks passed",
		"build succeeded",
		"compiled successfully",
		"lint passed",
		"nothing to commit, working tree clean",
		"already up to date",
		"up to date",
		"working tree clean",
		"no tests to run",
		"0 failed",
		"0 failures",
		"0 errors",
		"done.",
		"pass ok ",
		"ok\t",
		"ok github.com/",
		" ok ",
		" passed",
		"successfully",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func transcriptNormalizedPareceProgreso(normalized string) bool {
	markers := []string{
		"he actualizado",
		"he corregido",
		"he añadido",
		"he anadido",
		"he cambiado",
		"he implementado",
		"he dejado",
		"he movido",
		"he sacado",
		"he alineado",
		"ya esta",
		"ya está",
		"ya responde",
		"ya devuelve",
		"ya converge",
		"avance:",
		"progress:",
		"sigo con",
		"continuo con",
		"continúo con",
		"siguiente corte",
		"hecho:",
		"ajustado ",
		"agregado test",
		"agregada regresion",
		"regresion en ",
		"implemented ",
		"updated ",
		"fixed ",
		"refactored ",
		"moved ",
		"split ",
		"done with ",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func transcriptNormalizedPareceExploracionGrep(normalized string) bool {
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "diff --git") {
		return false
	}
	if strings.HasSuffix(normalized, ".go") || strings.HasSuffix(normalized, "_test.go") {
		return true
	}
	if strings.Contains(normalized, ".go.") {
		return true
	}
	if strings.Contains(normalized, "|") && strings.Contains(normalized, " in") {
		return true
	}
	if strings.Contains(normalized, "session_resume in") || strings.Contains(normalized, "runtime.*") {
		return true
	}
	return false
}

func pareceErrorServidorLocal(normalized string) bool {
	if normalized == "" {
		return false
	}
	if !strings.Contains(normalized, "http://127.0.0.1:") && !strings.Contains(normalized, "http://localhost:") {
		return false
	}
	markers := []string{
		"/api/status",
		"/api/agentes/",
		"/api/tareas",
		"/api/sesiones/",
		"/api/runtime",
		"consultando servidor",
		"timeout awaiting response headers",
		"net/http: timeout",
		"operation timed out",
		"curl: (28)",
		"failed to connect",
		"connection refused",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func parecePreguntaAprobacion(normalized string) bool {
	if normalized == "" {
		return false
	}
	if !strings.ContainsAny(normalized, "?¿") {
		return false
	}
	questionPhrases := []string{
		"puedo ",
		"can i ",
		"may i ",
		"should i",
		"quieres que",
		"te parece bien si",
	}
	for _, phrase := range questionPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func esLineaSistemaTranscript(texto string) bool {
	return strings.HasPrefix(texto, "Script started on ") || strings.HasPrefix(texto, "Script done on ")
}
