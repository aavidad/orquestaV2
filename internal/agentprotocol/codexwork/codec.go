package codexwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

type wireFrame struct {
	id          string
	method      string
	params      json.RawMessage
	result      json.RawMessage
	remoteError json.RawMessage
}

func decodeFrame(raw []byte) (wireFrame, [32]byte, error) {
	if len(raw) == 0 || len(raw) > MaxPacketBytesV1 {
		code := CodeFrameMalformed
		if len(raw) > MaxPacketBytesV1 {
			code = CodeFrameTooLarge
		}
		return wireFrame{}, [32]byte{}, protocolError(code)
	}
	if !utf8.Valid(raw) || !uniqueJSONObject(raw) {
		return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if decoder.Decode(&fields) != nil || fields == nil {
		return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
	}
	for key := range fields {
		switch key {
		case "id", "method", "params", "result", "error", "emittedAtMs":
		default:
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
	}
	var frame wireFrame
	if idRaw, exists := fields["id"]; exists {
		if json.Unmarshal(idRaw, &frame.id) != nil || frame.id == "" {
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
	}
	if methodRaw, exists := fields["method"]; exists {
		if json.Unmarshal(methodRaw, &frame.method) != nil || frame.method == "" {
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
		frame.params = fields["params"]
		if len(frame.params) == 0 || len(fields["result"]) != 0 || len(fields["error"]) != 0 {
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
	} else {
		frame.result, frame.remoteError = fields["result"], fields["error"]
		if frame.id == "" || (len(frame.result) == 0) == (len(frame.remoteError) == 0) || len(fields["params"]) != 0 {
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
		if len(frame.remoteError) != 0 {
			var detail struct {
				Code    *int64  `json:"code"`
				Message *string `json:"message"`
			}
			if json.Unmarshal(frame.remoteError, &detail) != nil || detail.Code == nil || detail.Message == nil {
				return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
			}
		}
	}
	if emittedAtRaw, exists := fields["emittedAtMs"]; exists {
		var emittedAtMS *uint64
		if json.Unmarshal(emittedAtRaw, &emittedAtMS) != nil || emittedAtMS == nil ||
			frame.method == "" || frame.id != "" {
			return wireFrame{}, [32]byte{}, protocolError(CodeFrameMalformed)
		}
	}
	return frame, sha256.Sum256(bytes.TrimSpace(raw)), nil
}

type threadItem struct {
	id    string
	kind  string
	text  string
	phase *string
}

func decodeThreadItem(raw json.RawMessage) (threadItem, error) {
	var header struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &header) != nil || header.ID == "" || header.Type == "" {
		return threadItem{}, protocolError(CodeFrameMalformed)
	}
	item := threadItem{id: header.ID, kind: header.Type}
	if header.Type != "agentMessage" {
		return item, nil
	}
	var message struct {
		Text  *string `json:"text"`
		Phase *string `json:"phase"`
	}
	if json.Unmarshal(raw, &message) != nil || message.Text == nil {
		return threadItem{}, protocolError(CodeFrameMalformed)
	}
	if message.Phase != nil && *message.Phase != "commentary" && *message.Phase != "final_answer" {
		return threadItem{}, protocolError(CodeFrameMalformed)
	}
	item.text, item.phase = *message.Text, message.Phase
	return item, nil
}

func notificationThreadObjectID(raw json.RawMessage) (string, error) {
	var params struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if json.Unmarshal(raw, &params) != nil || params.Thread.ID == "" {
		return "", protocolError(CodeFrameMalformed)
	}
	return params.Thread.ID, nil
}

func notificationTurnObjectIDs(raw json.RawMessage) (string, string, error) {
	var params struct {
		ThreadID string `json:"threadId"`
		Turn     struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if json.Unmarshal(raw, &params) != nil || params.ThreadID == "" || params.Turn.ID == "" {
		return "", "", protocolError(CodeFrameMalformed)
	}
	return params.ThreadID, params.Turn.ID, nil
}

func notificationCorrelation(raw json.RawMessage) (string, string, error) {
	var params struct {
		ThreadID string `json:"threadId"`
		TurnID   string `json:"turnId"`
	}
	if json.Unmarshal(raw, &params) != nil {
		return "", "", protocolError(CodeFrameMalformed)
	}
	return params.ThreadID, params.TurnID, nil
}

func decodeErrorNotification(raw json.RawMessage) (bool, string, string, error) {
	var params struct {
		Error     json.RawMessage `json:"error"`
		WillRetry *bool           `json:"willRetry"`
		ThreadID  *string         `json:"threadId"`
		TurnID    *string         `json:"turnId"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || len(params.Error) == 0 ||
		params.WillRetry == nil || params.ThreadID == nil || params.TurnID == nil ||
		!validOpaqueRef(*params.ThreadID) || !validOpaqueRef(*params.TurnID) {
		return false, "", "", protocolError(CodeFrameMalformed)
	}
	var remoteError struct {
		Message           *string         `json:"message"`
		CodexErrorInfo    json.RawMessage `json:"codexErrorInfo"`
		AdditionalDetails *string         `json:"additionalDetails"`
	}
	if err := decodeStrictObject(params.Error, &remoteError); err != nil || remoteError.Message == nil {
		return false, "", "", protocolError(CodeFrameMalformed)
	}
	return *params.WillRetry, *params.ThreadID, *params.TurnID, nil
}

func validateConfigWarning(raw json.RawMessage) error {
	var params struct {
		Summary *string         `json:"summary"`
		Details *string         `json:"details"`
		Path    *string         `json:"path"`
		Range   json.RawMessage `json:"range"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.Summary == nil {
		return protocolError(CodeFrameMalformed)
	}
	if len(params.Range) == 0 || bytes.Equal(bytes.TrimSpace(params.Range), []byte("null")) {
		return nil
	}
	var textRange struct {
		Start *textPosition `json:"start"`
		End   *textPosition `json:"end"`
	}
	if err := decodeStrictObject(params.Range, &textRange); err != nil || textRange.Start == nil ||
		textRange.End == nil || textRange.Start.Line == nil || textRange.Start.Column == nil ||
		textRange.End.Line == nil || textRange.End.Column == nil {
		return protocolError(CodeFrameMalformed)
	}
	return nil
}

type textPosition struct {
	Line   *uint64 `json:"line"`
	Column *uint64 `json:"column"`
}

func validateRemoteControlStatus(raw json.RawMessage) error {
	var params struct {
		EnvironmentID  *string `json:"environmentId"`
		InstallationID *string `json:"installationId"`
		ServerName     *string `json:"serverName"`
		Status         *string `json:"status"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.InstallationID == nil ||
		params.ServerName == nil || params.Status == nil || !validOpaqueRef(*params.InstallationID) ||
		!validOpaqueRef(*params.ServerName) {
		return protocolError(CodeFrameMalformed)
	}
	if params.EnvironmentID != nil && !validOpaqueRef(*params.EnvironmentID) {
		return protocolError(CodeFrameMalformed)
	}
	switch *params.Status {
	case "disabled", "connecting", "connected", "errored":
		return nil
	default:
		return protocolError(CodeFrameMalformed)
	}
}

func validateMCPServerStartupStatus(raw json.RawMessage) (*string, error) {
	var params struct {
		Name          *string `json:"name"`
		Status        *string `json:"status"`
		Error         *string `json:"error"`
		FailureReason *string `json:"failureReason"`
		ThreadID      *string `json:"threadId"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.Name == nil || params.Status == nil ||
		!validOpaqueRef(*params.Name) {
		return nil, protocolError(CodeFrameMalformed)
	}
	switch *params.Status {
	case "starting", "ready", "failed", "cancelled":
	default:
		return nil, protocolError(CodeFrameMalformed)
	}
	if params.FailureReason != nil && *params.FailureReason != "reauthenticationRequired" {
		return nil, protocolError(CodeFrameMalformed)
	}
	if params.ThreadID != nil && !validOpaqueRef(*params.ThreadID) {
		return nil, protocolError(CodeFrameMalformed)
	}
	return params.ThreadID, nil
}

type rateLimitSnapshot struct {
	Credits              *creditsSnapshot           `json:"credits"`
	IndividualLimit      *spendControlLimitSnapshot `json:"individualLimit"`
	LimitID              *string                    `json:"limitId"`
	LimitName            *string                    `json:"limitName"`
	PlanType             *string                    `json:"planType"`
	Primary              *rateLimitWindow           `json:"primary"`
	RateLimitReachedType *string                    `json:"rateLimitReachedType"`
	Secondary            *rateLimitWindow           `json:"secondary"`
	SpendControlReached  *bool                      `json:"spendControlReached"`
}

type creditsSnapshot struct {
	Balance    *string `json:"balance"`
	HasCredits *bool   `json:"hasCredits"`
	Unlimited  *bool   `json:"unlimited"`
}

type spendControlLimitSnapshot struct {
	Limit            *string `json:"limit"`
	RemainingPercent *int32  `json:"remainingPercent"`
	ResetsAt         *int64  `json:"resetsAt"`
	Used             *string `json:"used"`
}

type rateLimitWindow struct {
	ResetsAt           *int64 `json:"resetsAt"`
	UsedPercent        *int32 `json:"usedPercent"`
	WindowDurationMins *int64 `json:"windowDurationMins"`
}

func validateAccountRateLimitsUpdated(raw json.RawMessage) error {
	var params struct {
		RateLimits *rateLimitSnapshot `json:"rateLimits"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.RateLimits == nil {
		return protocolError(CodeFrameMalformed)
	}
	snapshot := params.RateLimits
	if snapshot.Credits != nil &&
		(snapshot.Credits.HasCredits == nil || snapshot.Credits.Unlimited == nil) {
		return protocolError(CodeFrameMalformed)
	}
	if snapshot.IndividualLimit != nil &&
		(snapshot.IndividualLimit.Limit == nil || snapshot.IndividualLimit.RemainingPercent == nil ||
			snapshot.IndividualLimit.ResetsAt == nil || snapshot.IndividualLimit.Used == nil) {
		return protocolError(CodeFrameMalformed)
	}
	for _, window := range []*rateLimitWindow{snapshot.Primary, snapshot.Secondary} {
		if window != nil && window.UsedPercent == nil {
			return protocolError(CodeFrameMalformed)
		}
	}
	if snapshot.PlanType != nil && !validPlanType(*snapshot.PlanType) {
		return protocolError(CodeFrameMalformed)
	}
	if snapshot.RateLimitReachedType != nil && !validRateLimitReachedType(*snapshot.RateLimitReachedType) {
		return protocolError(CodeFrameMalformed)
	}
	return nil
}

func validateThreadSettingsUpdated(raw json.RawMessage, threadID, model, effort string) error {
	var params struct {
		ThreadID       *string         `json:"threadId"`
		ThreadSettings json.RawMessage `json:"threadSettings"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.ThreadID == nil ||
		*params.ThreadID != threadID || len(params.ThreadSettings) == 0 ||
		!uniqueJSONObject(params.ThreadSettings) {
		return protocolError(CodeFrameMalformed)
	}
	var settings struct {
		ApprovalPolicy json.RawMessage `json:"approvalPolicy"`
		CWD            *string         `json:"cwd"`
		Effort         *string         `json:"effort"`
		Model          *string         `json:"model"`
		SandboxPolicy  *struct {
			Type *string `json:"type"`
		} `json:"sandboxPolicy"`
	}
	var approvalPolicy string
	if json.Unmarshal(params.ThreadSettings, &settings) != nil || settings.CWD == nil ||
		settings.Effort == nil || settings.Model == nil || settings.SandboxPolicy == nil ||
		settings.SandboxPolicy.Type == nil ||
		json.Unmarshal(settings.ApprovalPolicy, &approvalPolicy) != nil || approvalPolicy != "never" ||
		*settings.CWD != sealedWorkspaceCWD || *settings.Effort != effort || *settings.Model != model ||
		*settings.SandboxPolicy.Type != sealedSandboxPolicyType {
		return protocolError(CodeFrameMalformed)
	}
	return nil
}

func validateEnvironmentConnection(raw json.RawMessage, threadID string) error {
	var params struct {
		EnvironmentID *string `json:"environmentId"`
		ThreadID      *string `json:"threadId"`
	}
	if err := decodeStrictObject(raw, &params); err != nil || params.EnvironmentID == nil ||
		params.ThreadID == nil || *params.EnvironmentID != sealedEnvironmentID || *params.ThreadID != threadID {
		return protocolError(CodeFrameMalformed)
	}
	return nil
}

func validPlanType(value string) bool {
	switch value {
	case "free", "go", "plus", "pro", "prolite", "team", "self_serve_business_usage_based",
		"business", "ent26", "enterprise_cbp_usage_based", "enterprise", "edu", "unknown":
		return true
	default:
		return false
	}
}

func validRateLimitReachedType(value string) bool {
	switch value {
	case "rate_limit_reached", "workspace_owner_credits_depleted", "workspace_member_credits_depleted",
		"workspace_owner_usage_limit_reached", "workspace_member_usage_limit_reached":
		return true
	default:
		return false
	}
}

func decodeStrictObject(raw json.RawMessage, target any) error {
	if !uniqueJSONObject(raw) {
		return protocolError(CodeFrameMalformed)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return protocolError(CodeFrameMalformed)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return protocolError(CodeFrameMalformed)
	}
	return nil
}

func decodeWorkResult(text string, maxOutput uint64) (WorkResultV1, error) {
	if !utf8.ValidString(text) || len(text) > MaxPacketBytesV1 || !uniqueJSONObject([]byte(text)) {
		return WorkResultV1{}, protocolError(CodeOutputInvalid)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.DisallowUnknownFields()
	var output struct {
		Artifact *string `json:"artifact"`
	}
	if decoder.Decode(&output) != nil || output.Artifact == nil {
		return WorkResultV1{}, protocolError(CodeOutputInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) || uint64(len(*output.Artifact)) > maxOutput {
		return WorkResultV1{}, protocolError(CodeOutputInvalid)
	}
	return WorkResultV1{Artifact: *output.Artifact}, nil
}

func marshalLine(value any) ([]byte, error) {
	frame, err := json.Marshal(value)
	if err != nil || len(frame) > MaxPacketBytesV1 {
		return nil, protocolError(CodeFrameMalformed)
	}
	return append(frame, '\n'), nil
}

type turnStartRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params turnStartParams `json:"params"`
}

type initializeRequest struct {
	ID     string           `json:"id"`
	Method string           `json:"method"`
	Params initializeParams `json:"params"`
}

type initializeParams struct {
	ClientInfo   initializeClientInfo   `json:"clientInfo"`
	Capabilities initializeCapabilities `json:"capabilities"`
}

type initializeCapabilities struct {
	ExperimentalAPI bool `json:"experimentalApi"`
}

type initializeClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title"`
}

type turnStartParams struct {
	ThreadID              string                  `json:"threadId"`
	Input                 []textInput             `json:"input"`
	ClientUserMessageID   string                  `json:"clientUserMessageId"`
	Model                 string                  `json:"model"`
	Effort                string                  `json:"effort"`
	Environments          []turnEnvironmentParams `json:"environments"`
	RuntimeWorkspaceRoots []string                `json:"runtimeWorkspaceRoots"`
	OutputSchema          artifactOutputSchema    `json:"outputSchema"`
}

type turnEnvironmentParams struct {
	EnvironmentID         string   `json:"environmentId"`
	CWD                   string   `json:"cwd"`
	RuntimeWorkspaceRoots []string `json:"runtimeWorkspaceRoots"`
}

type textInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type artifactOutputSchema struct {
	Type                 string                            `json:"type"`
	AdditionalProperties bool                              `json:"additionalProperties"`
	Properties           map[string]artifactSchemaProperty `json:"properties"`
	Required             []string                          `json:"required"`
}

type artifactSchemaProperty struct {
	Type string `json:"type"`
}
