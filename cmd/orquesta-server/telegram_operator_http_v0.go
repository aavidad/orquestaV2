package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	channel "orquesta/modulos/orquesta-operator-director-channel"
	orquestatelegram "orquesta/modulos/orquesta-operator-telegram"
)

const telegramOperatorUpdateHTTPPathV0 = "/api/v0/operator/telegram/update"

type telegramOperatorUpdateHTTPHandlerV0 struct {
	wiring telegramOperatorWiringResultV0
	sender hermesTelegramSendPortV0
}

type telegramOperatorHTTPResponseV0 struct {
	Estado       string                         `json:"estado"`
	Response     orquestatelegram.ResponseV0    `json:"response,omitempty"`
	Sent         bool                           `json:"sent,omitempty"`
	SendRef      string                         `json:"send_ref,omitempty"`
	Missing      []string                       `json:"missing_fields,omitempty"`
	ErrorCode    string                         `json:"error_code,omitempty"`
	EvidenceRefs []string                       `json:"evidence_refs,omitempty"`
	Update       telegramOperatorUpdatePublicV0 `json:"update,omitempty"`
}

type telegramOperatorUpdatePublicV0 struct {
	UpdateRef string `json:"update_ref,omitempty"`
	ChatRef   string `json:"chat_ref,omitempty"`
}

type telegramOperatorIncomingUpdateV0 struct {
	UpdateRef string `json:"update_ref,omitempty"`
	ChatRef   string `json:"chat_ref,omitempty"`
	Text      string `json:"text,omitempty"`
	UpdateID  int64  `json:"update_id,omitempty"`
	Message   struct {
		Text string `json:"text,omitempty"`
		Chat struct {
			ID any `json:"id,omitempty"`
		} `json:"chat,omitempty"`
	} `json:"message,omitempty"`
}

func newTelegramOperatorUpdateHTTPHandlerV0(
	wiring telegramOperatorWiringResultV0,
	sender hermesTelegramSendPortV0,
) http.Handler {
	return telegramOperatorUpdateHTTPHandlerV0{wiring: wiring, sender: sender}
}

func (handler telegramOperatorUpdateHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTelegramOperatorHTTPJSONV0(w, http.StatusMethodNotAllowed, telegramOperatorHTTPResponseV0{
			Estado:    "error",
			ErrorCode: "method_not_allowed",
		})
		return
	}
	if !handler.wiring.Enabled {
		writeTelegramOperatorHTTPJSONV0(w, http.StatusNotFound, telegramOperatorHTTPResponseV0{
			Estado:    "error",
			ErrorCode: "telegram_operator_disabled",
		})
		return
	}
	if !handler.wiring.Ready {
		writeTelegramOperatorHTTPJSONV0(w, http.StatusOK, telegramOperatorHTTPResponseV0{
			Estado:    "blocked",
			ErrorCode: handler.wiring.BlockedCode,
			Missing:   handler.wiring.MissingFields,
		})
		return
	}
	update, err := decodeTelegramOperatorUpdateV0(r)
	if err != nil {
		writeTelegramOperatorHTTPJSONV0(w, http.StatusBadRequest, telegramOperatorHTTPResponseV0{
			Estado:    "error",
			ErrorCode: err.Error(),
		})
		return
	}
	response := handler.wiring.Adapter.HandleUpdateV0(update)
	result := telegramOperatorHTTPResponseV0{
		Estado:       response.Status,
		Response:     response,
		EvidenceRefs: response.EvidenceRefs,
		Update: telegramOperatorUpdatePublicV0{
			UpdateRef: update.UpdateRef,
			ChatRef:   update.ChatRef,
		},
	}
	if handler.sender != nil && strings.TrimSpace(update.ChatRef) != "" {
		sendRef, err := handler.sender.SendTelegramMessageV0(r.Context(), hermesTelegramMessageV0{
			TargetRef:    update.ChatRef,
			Text:         response.Summary,
			DedupeKey:    telegramOperatorDedupeKeyV0(update, response),
			EvidenceRefs: response.EvidenceRefs,
		})
		if err != nil {
			result.Estado = "blocked"
			result.ErrorCode = "telegram_send_failed"
		} else {
			result.Sent = true
			result.SendRef = sendRef
		}
	}
	writeTelegramOperatorHTTPJSONV0(w, http.StatusOK, result)
}

func decodeTelegramOperatorUpdateV0(r *http.Request) (orquestatelegram.UpdateV0, error) {
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return orquestatelegram.UpdateV0{}, errors.New("telegram_update_read_failed")
	}
	var input telegramOperatorIncomingUpdateV0
	if err := json.Unmarshal(data, &input); err != nil {
		return orquestatelegram.UpdateV0{}, errors.New("telegram_update_invalid_json")
	}
	updateRef := strings.TrimSpace(input.UpdateRef)
	if updateRef == "" && input.UpdateID != 0 {
		updateRef = "telegram-update-" + strconv.FormatInt(input.UpdateID, 10)
	}
	chatRef := strings.TrimSpace(input.ChatRef)
	if chatRef == "" {
		chatRef = telegramOperatorChatRefFromAnyV0(input.Message.Chat.ID)
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		text = strings.TrimSpace(input.Message.Text)
	}
	if chatRef == "" || text == "" {
		return orquestatelegram.UpdateV0{}, errors.New("telegram_update_missing_chat_or_text")
	}
	return orquestatelegram.UpdateV0{
		UpdateRef: updateRef,
		ChatRef:   chatRef,
		Text:      text,
	}, nil
}

func telegramOperatorChatRefFromAnyV0(value any) string {
	switch typed := value.(type) {
	case float64:
		return "telegram:" + strconv.FormatInt(int64(typed), 10)
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return ""
		}
		if strings.HasPrefix(typed, "telegram:") {
			return typed
		}
		return "telegram:" + typed
	default:
		return ""
	}
}

func telegramOperatorDedupeKeyV0(
	update orquestatelegram.UpdateV0,
	response orquestatelegram.ResponseV0,
) string {
	parts := []string{
		firstNonEmptyOperatorNotificationV0(update.UpdateRef, update.ChatRef),
		response.CommandKind,
		response.Status,
	}
	return strings.Join(parts, ":")
}

type telegramOperatorHTTPPortsV0 struct {
	Handler         http.Handler
	DirectorMessage channel.OperatorDirectorChannelServiceV0
}

func (ports telegramOperatorHTTPPortsV0) QueryStatusV0(targetRef string) (string, []string, error) {
	payload := map[string]any{"run_ref": strings.TrimSpace(targetRef)}
	return ports.postJSONV0("/api/v0/autoprogramming/status", payload)
}

func (ports telegramOperatorHTTPPortsV0) QueryQueueV0(targetRef string) (string, []string, error) {
	payload := map[string]any{"run_ref": strings.TrimSpace(targetRef)}
	return ports.postJSONV0("/api/v0/runs/queue/priority", payload)
}

func (ports telegramOperatorHTTPPortsV0) LaunchTaskV0(taskSpec string, _ []string) (string, []string, error) {
	taskSpec = strings.TrimSpace(taskSpec)
	if !strings.HasPrefix(taskSpec, "{") {
		return "", nil, errors.New("telegram_launch_requires_prepare_run_json")
	}
	return ports.postRawJSONV0("/api/v0/autoprogramming/prepare-run", []byte(taskSpec))
}

func (ports telegramOperatorHTTPPortsV0) ObserveGoalV0(goalRef string) (string, []string, error) {
	payload := map[string]any{"goal_ref": strings.TrimSpace(goalRef)}
	return ports.postJSONV0("/api/v0/autoprogramming/goal/observe", payload)
}

func (ports telegramOperatorHTTPPortsV0) SendDirectorMessageV0(
	targetRef string,
	body string,
	evidenceRefs []string,
) (string, []string, error) {
	response, issues, err := channel.DispatchOperatorDirectorMessageV0(context.Background(), ports.DirectorMessage, channel.OperatorMessageV0{
		RequestRef:   "request-ref-telegram-operator-director-message",
		AdapterRef:   "telegram_operator",
		SenderRef:    "telegram_operator",
		TargetRef:    strings.TrimSpace(targetRef),
		Intent:       channel.OperatorMessageIntentInstructionV0,
		Body:         strings.TrimSpace(body),
		EvidenceRefs: evidenceRefs,
	})
	if err != nil {
		return "", nil, err
	}
	if len(issues) > 0 {
		return "", nil, errors.New(issues[0].Code)
	}
	return response.Summary, response.EvidenceRefs, nil
}

func (ports telegramOperatorHTTPPortsV0) StopV0(targetRef string, evidenceRefs []string) (string, []string, error) {
	payload := map[string]any{
		"run_ref":        strings.TrimSpace(targetRef),
		"action":         "stop",
		"forced":         false,
		"reason":         "telegram_operator_stop",
		"evidence_refs":  evidenceRefs,
		"requested_from": "telegram_operator",
	}
	return ports.postJSONV0("/api/v0/runs/control", payload)
}

func (ports telegramOperatorHTTPPortsV0) HandoffV0(targetRef string) (string, []string, error) {
	payload := map[string]any{"run_ref": strings.TrimSpace(targetRef)}
	return ports.postJSONV0("/api/v0/autoprogramming/status", payload)
}

func (ports telegramOperatorHTTPPortsV0) postJSONV0(path string, payload any) (string, []string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	return ports.postRawJSONV0(path, data)
}

func (ports telegramOperatorHTTPPortsV0) postRawJSONV0(path string, data []byte) (string, []string, error) {
	if ports.Handler == nil {
		return "", nil, errors.New("telegram_operator_http_port_unavailable")
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ports.Handler.ServeHTTP(rec, req)
	body := strings.TrimSpace(rec.Body.String())
	if rec.Code >= 400 {
		return "", nil, fmt.Errorf("telegram_operator_http_%d", rec.Code)
	}
	return telegramOperatorCompactHTTPBodyV0(body), []string{"evidence-ref-telegram-operator-http-" + strings.Trim(strings.ReplaceAll(path, "/", "-"), "-")}, nil
}

func telegramOperatorCompactHTTPBodyV0(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "ok"
	}
	runes := []rune(body)
	if len(runes) <= 240 {
		return body
	}
	return strings.TrimSpace(string(runes[:240])) + "..."
}

func writeTelegramOperatorHTTPJSONV0(w http.ResponseWriter, status int, payload telegramOperatorHTTPResponseV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
