package controlruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

type remoteConfig struct {
	Endpoint              string `json:"endpoint"`
	AuthHeader            string `json:"auth_header"`
	AuthTokenEnv          string `json:"auth_token_env"`
	AuthMode              string `json:"auth_mode"`
	PreserveExternalStop  bool   `json:"preserve_external_session_on_stop"`
	RequiresHumanReauth   bool   `json:"requires_human_reauth"`
	TimeoutMS             int    `json:"timeout_ms"`
	LaunchPath            string `json:"launch_path"`
	ResumePath            string `json:"resume_path"`
	StatusPath            string `json:"status_path"`
	PausePath             string `json:"pause_path"`
	ContinuePath          string `json:"continue_path"`
	StopPath              string `json:"stop_path"`
	InputPath             string `json:"input_path"`
	SessionIDField        string `json:"session_id_field"`
	HandleRefField        string `json:"handle_ref_field"`
	CanSendInput          *bool  `json:"can_send_input"`
	CanPause              *bool  `json:"can_pause"`
	CanStop               *bool  `json:"can_stop"`
	ControlRetryCount     int    `json:"control_retry_count"`
	ControlRetryBackoffMS int    `json:"control_retry_backoff_ms"`
	ControlRetryStatuses  []int  `json:"control_retry_statuses"`
}

type remoteRequestPolicy struct {
	RetryCount    int
	RetryBackoff  time.Duration
	RetryStatuses map[int]struct{}
}

type EstadoRemoto struct {
	HandleEstado     string
	LogicalState     string
	ProcessState     string
	MetadataJSON     string
	CapabilitiesJSON string
	RawJSON          string
}

func isRemoteTransport(transport string) bool {
	switch strings.TrimSpace(transport) {
	case "mcp_http", "api", "otro":
		return true
	default:
		return false
	}
}

func arrancarPlanRemoto(req SolicitudArranque) (*ProcesoArrancado, error) {
	cfg := remoteConfigFromPlan(req.Plan)
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("el plan remoto no define endpoint")
	}
	path := cfg.LaunchPath
	if strings.TrimSpace(req.Plan.Modo) == "resume" {
		path = cfg.ResumePath
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("el plan remoto no define ruta para %s", strings.TrimSpace(req.Plan.Modo))
	}

	payload := map[string]any{
		"agente":   strings.TrimSpace(req.Agente),
		"proyecto": strings.TrimSpace(req.Proyecto),
		"plan":     req.Plan,
	}
	respuesta, err := remoteJSONRequest(cfg, path, payload, remoteNoRetryPolicy())
	if err != nil {
		return nil, err
	}

	externalSessionID := extractRemoteString(respuesta, cfg.SessionIDField, "external_session_id", "session_id")
	handleRef := extractRemoteString(respuesta, cfg.HandleRefField, "handle_ref", "session_id", "external_session_id")
	handleKind := extractRemoteString(respuesta, "handle_kind", "handle_kind")
	if handleKind == "" {
		handleKind = "session"
	}
	pid := extractRemoteInt(respuesta, "pid")

	meta := map[string]any{
		"driver":                            "remote_http",
		"endpoint":                          cfg.Endpoint,
		"auth_header":                       cfg.AuthHeader,
		"auth_token_env":                    cfg.AuthTokenEnv,
		"auth_mode":                         cfg.AuthMode,
		"preserve_external_session_on_stop": cfg.PreserveExternalStop,
		"requires_human_reauth":             cfg.RequiresHumanReauth,
		"timeout_ms":                        cfg.TimeoutMS,
		"launch_path":                       cfg.LaunchPath,
		"resume_path":                       cfg.ResumePath,
		"status_path":                       cfg.StatusPath,
		"pause_path":                        cfg.PausePath,
		"continue_path":                     cfg.ContinuePath,
		"stop_path":                         cfg.StopPath,
		"input_path":                        cfg.InputPath,
		"session_id_field":                  cfg.SessionIDField,
		"handle_ref_field":                  cfg.HandleRefField,
		"external_session_id":               externalSessionID,
		"handle_ref":                        handleRef,
	}
	if nested, ok := respuesta["metadata"].(map[string]any); ok {
		for k, v := range nested {
			meta[k] = v
		}
	}
	if pid > 0 {
		meta["pid"] = pid
	}
	metaJSON, _ := json.Marshal(meta)

	caps := map[string]any{
		"can_send_input":          remoteCapabilityEnabled(cfg.CanSendInput, cfg.InputPath),
		"can_checkpoint":          true,
		"can_resume":              true,
		"can_capture_pid":         pid > 0,
		"can_track_continuity":    true,
		"can_pause":               remoteCapabilityEnabled(cfg.CanPause, cfg.PausePath),
		"can_stop":                remoteCapabilityEnabled(cfg.CanStop, cfg.StopPath),
		"can_stop_without_reauth": remoteCapabilityEnabled(cfg.CanStop, cfg.StopPath) && !cfg.PreserveExternalStop && !cfg.RequiresHumanReauth && !strings.EqualFold(strings.TrimSpace(cfg.AuthMode), "oauth"),
	}
	if nested, ok := respuesta["capabilities"].(map[string]any); ok {
		for k, v := range nested {
			caps[k] = v
		}
	}
	capsJSON, _ := json.Marshal(caps)

	rendered := strings.TrimSpace(cfg.Endpoint + path)
	return &ProcesoArrancado{
		PID:               pid,
		HandleKind:        handleKind,
		HandleRef:         handleRef,
		ExternalSessionID: externalSessionID,
		WorkingDir:        strings.TrimSpace(req.Plan.WorkingDir),
		RenderedCommand:   rendered,
		WrappedCommand:    rendered,
		MetadataJSON:      string(metaJSON),
		CapabilitiesJSON:  string(capsJSON),
	}, nil
}

func enviarInstruccionRemota(obj ObjetivoProceso, instruccion string) (bool, int, error) {
	cfg := remoteConfigFromMetadata(obj.MetadataJSON)
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.InputPath) == "" {
		return false, 0, nil
	}
	payload := map[string]any{
		"handle_ref":          strings.TrimSpace(obj.HandleRef),
		"external_session_id": externalSessionIDFromMetadata(obj.MetadataJSON),
		"texto":               strings.TrimRight(instruccion, "\n"),
	}
	if runtimeOrderID := extractRemoteInt(metadataMap(obj.MetadataJSON), "runtime_order_id"); runtimeOrderID > 0 {
		payload["runtime_order_id"] = runtimeOrderID
	}
	_, err := remoteJSONRequest(cfg, cfg.InputPath, payload, remoteNoRetryPolicy())
	if err != nil {
		return true, 0, err
	}
	return true, 0, nil
}

func controlRemoto(obj ObjetivoProceso, path string) (bool, int, error) {
	cfg := remoteConfigFromMetadata(obj.MetadataJSON)
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(path) == "" {
		return false, 0, nil
	}
	payload := map[string]any{
		"handle_ref":          strings.TrimSpace(obj.HandleRef),
		"external_session_id": externalSessionIDFromMetadata(obj.MetadataJSON),
	}
	_, err := remoteJSONRequest(cfg, path, payload, remoteControlPolicy(cfg))
	if err != nil {
		return true, 0, err
	}
	return true, 0, nil
}

func ConsultarEstadoRemoto(obj ObjetivoProceso) (*EstadoRemoto, bool, error) {
	cfg := remoteConfigFromMetadata(obj.MetadataJSON)
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.StatusPath) == "" {
		return nil, false, nil
	}
	payload := map[string]any{
		"handle_ref":          strings.TrimSpace(obj.HandleRef),
		"external_session_id": externalSessionIDFromMetadata(obj.MetadataJSON),
	}
	respuesta, err := remoteJSONRequest(cfg, cfg.StatusPath, payload, remoteControlPolicy(cfg))
	if err != nil {
		return nil, true, err
	}
	estado := &EstadoRemoto{
		HandleEstado: extractRemoteStringFromMaps(respuesta, []string{"handle.estado", "handle_state", "estado_handle", "estado"}, ""),
		LogicalState: extractRemoteStringFromMaps(respuesta, []string{"runtime.logical_state", "logical_state", "estado_logico"}, ""),
		ProcessState: extractRemoteStringFromMaps(respuesta, []string{"runtime.process_state", "process_state", "estado_proceso"}, ""),
	}
	if metadata := extractRemoteMap(respuesta, "metadata", "handle.metadata", "runtime.metadata"); len(metadata) > 0 {
		data, _ := json.Marshal(metadata)
		estado.MetadataJSON = string(data)
	}
	if capabilities := extractRemoteMap(respuesta, "capabilities", "handle.capabilities"); len(capabilities) > 0 {
		data, _ := json.Marshal(capabilities)
		estado.CapabilitiesJSON = string(data)
	}
	data, _ := json.Marshal(respuesta)
	estado.RawJSON = string(data)
	return estado, true, nil
}

func remoteConfigFromPlan(plan *runtimeagente.LaunchPlan) remoteConfig {
	if plan == nil {
		return remoteConfig{}
	}
	return remoteConfigFromJSON(plan.RemoteConfigJSON)
}

func remoteConfigFromMetadata(raw string) remoteConfig {
	return remoteConfigFromJSON(raw)
}

func remoteConfigFromJSON(raw string) remoteConfig {
	cfg := remoteConfig{
		TimeoutMS:             15000,
		LaunchPath:            "/launch",
		ResumePath:            "/resume",
		StatusPath:            "",
		PausePath:             "/pause",
		ContinuePath:          "/continue",
		StopPath:              "/stop",
		InputPath:             "/input",
		SessionIDField:        "external_session_id",
		HandleRefField:        "handle_ref",
		ControlRetryCount:     2,
		ControlRetryBackoffMS: 200,
		ControlRetryStatuses:  []int{408, 425, 429, 500, 502, 503, 504},
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

func remoteCapabilityEnabled(override *bool, path string) bool {
	if override != nil {
		return *override
	}
	return strings.TrimSpace(path) != ""
}

func remoteJSONRequest(cfg remoteConfig, path string, payload map[string]any, policy remoteRequestPolicy) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	client := remoteHTTPClient(cfg)
	url := joinRemoteURL(cfg.Endpoint, path)
	attempts := policy.RetryCount + 1
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if token := remoteAuthToken(cfg); token != "" && strings.TrimSpace(cfg.AuthHeader) != "" {
			req.Header.Set(strings.TrimSpace(cfg.AuthHeader), token)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = remoteNetworkError(url, err, attempt, attempts)
			if attempt < attempts && remoteShouldRetryError(err) {
				time.Sleep(policy.RetryBackoff)
				continue
			}
			return nil, lastErr
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < attempts {
				time.Sleep(policy.RetryBackoff)
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = remoteStatusError(url, resp.StatusCode, data, attempt, attempts)
			if attempt < attempts && remoteShouldRetryStatus(policy, resp.StatusCode) {
				time.Sleep(policy.RetryBackoff)
				continue
			}
			return nil, lastErr
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return map[string]any{}, nil
		}
		out := map[string]any{}
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("fallo remoto desconocido en %s", url)
	}
	return nil, lastErr
}

func remoteHTTPClient(cfg remoteConfig) *http.Client {
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func joinRemoteURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	return base + path
}

func remoteAuthToken(cfg remoteConfig) string {
	if strings.TrimSpace(cfg.AuthTokenEnv) == "" {
		return ""
	}
	return strings.TrimSpace(os.Getenv(strings.TrimSpace(cfg.AuthTokenEnv)))
}

func remoteNoRetryPolicy() remoteRequestPolicy {
	return remoteRequestPolicy{}
}

func remoteControlPolicy(cfg remoteConfig) remoteRequestPolicy {
	statuses := map[int]struct{}{}
	for _, code := range cfg.ControlRetryStatuses {
		if code > 0 {
			statuses[code] = struct{}{}
		}
	}
	backoff := time.Duration(cfg.ControlRetryBackoffMS) * time.Millisecond
	if backoff < 0 {
		backoff = 0
	}
	return remoteRequestPolicy{
		RetryCount:    cfg.ControlRetryCount,
		RetryBackoff:  backoff,
		RetryStatuses: statuses,
	}
}

func remoteShouldRetryStatus(policy remoteRequestPolicy, status int) bool {
	_, ok := policy.RetryStatuses[status]
	return ok
}

func remoteShouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "temporary")
}

func remoteNetworkError(url string, err error, attempt, attempts int) error {
	if remoteShouldRetryError(err) && attempt < attempts {
		return fmt.Errorf("error temporal llamando al adaptador remoto %s (intento %d/%d): %w", url, attempt, attempts, err)
	}
	if remoteShouldRetryError(err) {
		return fmt.Errorf("error temporal agotando reintentos del adaptador remoto %s tras %d intento(s): %w", url, attempts, err)
	}
	return fmt.Errorf("error llamando al adaptador remoto %s: %w", url, err)
}

func remoteStatusError(url string, status int, body []byte, attempt, attempts int) error {
	detail := strings.TrimSpace(string(body))
	if remoteDefaultRetryStatus(status) && attempt < attempts {
		return fmt.Errorf("adaptador remoto devolvió %d en %s (intento %d/%d): %s", status, url, attempt, attempts, detail)
	}
	return fmt.Errorf("adaptador remoto devolvió %d en %s: %s", status, url, detail)
}

func remoteDefaultRetryStatus(status int) bool {
	switch status {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func extractRemoteString(payload map[string]any, preferred string, aliases ...string) string {
	keys := []string{}
	if strings.TrimSpace(preferred) != "" {
		keys = append(keys, strings.TrimSpace(preferred))
	}
	keys = append(keys, aliases...)
	for _, key := range keys {
		if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extractRemoteStringFromMaps(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts := strings.Split(key, ".")
		var current any = payload
		for _, part := range parts {
			m, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = m[part]
		}
		if v, ok := current.(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return strings.TrimSpace(fallback)
}

func extractRemoteMap(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts := strings.Split(key, ".")
		var current any = payload
		for _, part := range parts {
			m, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = m[part]
		}
		if out, ok := current.(map[string]any); ok {
			return out
		}
	}
	return nil
}

func extractRemoteInt(payload map[string]any, key string) int {
	switch v := payload[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func metadataMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func externalSessionIDFromMetadata(raw string) string {
	payload := metadataMap(raw)
	if v, ok := payload["external_session_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
