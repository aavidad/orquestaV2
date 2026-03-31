package controlruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type codexSessionMetaLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		CWD       string `json:"cwd"`
	} `json:"payload"`
}

func DetectExternalSessionID(obj ObjetivoProceso) (string, error) {
	if estado, observed, err := ConsultarEstadoLocal(obj); err != nil {
		return "", err
	} else if observed && estado != nil {
		if strings.TrimSpace(estado.MetadataJSON) != "" {
			obj.MetadataJSON = estado.MetadataJSON
		}
		if ext := strings.TrimSpace(stringValueFromMetadata(metadataMap(estado.MetadataJSON), "external_session_id")); ext != "" {
			return ext, nil
		}
	}
	meta := metadataMap(obj.MetadataJSON)
	if ext := strings.TrimSpace(stringValueFromMetadata(meta, "external_session_id")); ext != "" {
		return ext, nil
	}
	if !esRuntimeCodexLocal(obj) {
		return "", nil
	}
	startedAt := metadataTime(meta, "started_at")
	workingDir := strings.TrimSpace(stringValueFromMetadata(meta, "working_dir"))
	rendered := firstNonEmptyString(
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
	)
	return detectCodexSessionID(rendered, workingDir, startedAt, time.Now().UTC())
}

func EnviarInstruccionSesionResume(obj ObjetivoProceso, externalSessionID, instruccion string) (bool, int, error) {
	if estado, observed, err := ConsultarEstadoLocal(obj); err != nil {
		return false, 0, err
	} else if observed && estado != nil && strings.TrimSpace(estado.MetadataJSON) != "" {
		obj.MetadataJSON = estado.MetadataJSON
	}
	meta := metadataMap(obj.MetadataJSON)
	if !esRuntimeCodexLocal(obj) {
		return false, 0, nil
	}
	externalSessionID = strings.TrimSpace(externalSessionID)
	if externalSessionID == "" {
		detected, err := DetectExternalSessionID(obj)
		if err != nil {
			return false, 0, err
		}
		externalSessionID = strings.TrimSpace(detected)
	}
	texto := strings.TrimSpace(instruccion)
	if externalSessionID == "" || texto == "" {
		return false, 0, nil
	}
	argv, err := codexResumeExecArgv(meta, externalSessionID, texto)
	if err != nil {
		return false, 0, err
	}
	if len(argv) == 0 {
		return false, 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), codexResumeTimeout(meta))
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if dir := strings.TrimSpace(stringValueFromMetadata(meta, "working_dir")); dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return true, 0, fmt.Errorf("session_resume timeout: %s", trimmedCommandOutput(out))
	}
	if err != nil {
		if detalle := trimmedCommandOutput(out); detalle != "" {
			return true, 0, fmt.Errorf("session_resume fallo: %w: %s", err, detalle)
		}
		return true, 0, fmt.Errorf("session_resume fallo: %w", err)
	}
	return true, 0, nil
}

func codexResumeExecArgv(meta map[string]any, externalSessionID, instruccion string) ([]string, error) {
	rendered := firstNonEmptyString(
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
	)
	tokens, err := splitShellQuotedCommand(rendered)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}
	switch filepath.Base(tokens[0]) {
	case "codex-perfil":
		if len(tokens) < 2 {
			return nil, nil
		}
		return []string{tokens[0], tokens[1], "exec", "resume", strings.TrimSpace(externalSessionID), strings.TrimSpace(instruccion)}, nil
	case "codex":
		return []string{tokens[0], "exec", "resume", strings.TrimSpace(externalSessionID), strings.TrimSpace(instruccion)}, nil
	default:
		return nil, nil
	}
}

func detectCodexSessionID(renderedCommand, workingDir string, startedAt, now time.Time) (string, error) {
	workingDir = strings.TrimSpace(workingDir)
	if strings.TrimSpace(renderedCommand) == "" || workingDir == "" {
		return "", nil
	}
	type candidate struct {
		id        string
		startedAt time.Time
		modTime   time.Time
	}
	var best candidate
	for _, root := range codexSessionRoots(renderedCommand) {
		for _, dir := range codexSessionDayDirs(root, startedAt, now) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if !startedAt.IsZero() && info.ModTime().Before(startedAt.Add(-2*time.Minute)) {
					continue
				}
				meta, err := readCodexSessionMeta(path)
				if err != nil || meta == nil {
					continue
				}
				if strings.TrimSpace(meta.Payload.CWD) != workingDir {
					continue
				}
				sessionID := strings.TrimSpace(meta.Payload.ID)
				if sessionID == "" {
					continue
				}
				sessionStartedAt := firstParsedTime(meta.Payload.Timestamp, meta.Timestamp, info.ModTime())
				if !startedAt.IsZero() && sessionStartedAt.Before(startedAt.Add(-2*time.Minute)) {
					continue
				}
				if best.id == "" || sessionStartedAt.After(best.startedAt) || (sessionStartedAt.Equal(best.startedAt) && info.ModTime().After(best.modTime)) {
					best = candidate{id: sessionID, startedAt: sessionStartedAt, modTime: info.ModTime()}
				}
			}
		}
	}
	return best.id, nil
}

func readCodexSessionMeta(path string) (*codexSessionMetaLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return nil, nil
	}
	var meta codexSessionMetaLine
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(meta.Type) != "session_meta" {
		return nil, nil
	}
	return &meta, nil
}

func codexSessionRoots(renderedCommand string) []string {
	tokens, err := splitShellQuotedCommand(renderedCommand)
	if err != nil || len(tokens) == 0 {
		return nil
	}
	rootSet := map[string]struct{}{}
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		rootSet[path] = struct{}{}
	}
	switch filepath.Base(tokens[0]) {
	case "codex-perfil":
		if len(tokens) < 2 {
			break
		}
		perfil := strings.TrimSpace(tokens[1])
		if perfil == "" {
			break
		}
		base := filepath.Join(userHomeDir(), "Trabajo", "codex-perfiles")
		if strings.Contains(tokens[0], string(os.PathSeparator)) {
			base = filepath.Dir(filepath.Dir(tokens[0]))
		}
		add(filepath.Join(base, "homes", perfil, "sessions"))
	case "codex":
		codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
		if codexHome == "" {
			codexHome = filepath.Join(userHomeDir(), ".codex")
		}
		add(filepath.Join(codexHome, "sessions"))
	}
	out := make([]string, 0, len(rootSet))
	for root := range rootSet {
		out = append(out, root)
	}
	return out
}

func codexSessionDayDirs(root string, startedAt, now time.Time) []string {
	keys := map[string]struct{}{}
	add := func(ts time.Time) {
		if ts.IsZero() {
			return
		}
		local := ts.In(time.Local)
		dir := filepath.Join(root, local.Format("2006"), local.Format("01"), local.Format("02"))
		keys[dir] = struct{}{}
	}
	add(now)
	add(startedAt)
	add(now.Add(-24 * time.Hour))
	add(startedAt.Add(-24 * time.Hour))
	add(now.Add(24 * time.Hour))
	add(startedAt.Add(24 * time.Hour))
	out := make([]string, 0, len(keys))
	for dir := range keys {
		out = append(out, dir)
	}
	return out
}

func splitShellQuotedCommand(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	var token strings.Builder
	inSingle := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case inSingle && ch == '\'':
			inSingle = false
		case !inSingle && ch == '\'':
			inSingle = true
		case !inSingle && (ch == ' ' || ch == '\t' || ch == '\n'):
			if token.Len() > 0 {
				out = append(out, token.String())
				token.Reset()
			}
		case ch == '\\' && i+1 < len(raw):
			i++
			token.WriteByte(raw[i])
		default:
			token.WriteByte(ch)
		}
	}
	if inSingle {
		return nil, fmt.Errorf("rendered_command con comillas sin cerrar")
	}
	if token.Len() > 0 {
		out = append(out, token.String())
	}
	return out, nil
}

func metadataTime(meta map[string]any, key string) time.Time {
	raw := strings.TrimSpace(stringValueFromMetadata(meta, key))
	if raw == "" {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return ts
}

func firstParsedTime(raw ...any) time.Time {
	for _, item := range raw {
		switch v := item.(type) {
		case time.Time:
			if !v.IsZero() {
				return v
			}
		case string:
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
				return ts
			}
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				return ts
			}
		}
	}
	return time.Time{}
}

func codexResumeTimeout(meta map[string]any) time.Duration {
	if timeoutMS, ok := meta["session_resume_timeout_ms"].(float64); ok && timeoutMS > 0 {
		return time.Duration(timeoutMS) * time.Millisecond
	}
	return 20 * time.Second
}

func trimmedCommandOutput(raw []byte) string {
	texto := strings.TrimSpace(string(raw))
	if texto == "" {
		return ""
	}
	lower := strings.ToLower(texto)
	for _, marker := range []string{
		"hit your usage limit",
		"usage limit",
		"rate limit",
		"purchase more credits",
		"try again at",
		"error:",
	} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			texto = strings.TrimSpace(texto[idx:])
			break
		}
	}
	const maxChars = 240
	if len(texto) <= maxChars {
		return texto
	}
	return strings.TrimSpace(texto[:maxChars]) + "..."
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func userHomeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}
