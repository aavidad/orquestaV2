package orquestaruntimecodexappserver

import (
	"encoding/json"
	"strings"
)

type serverCodexAppServerThreadStartParamsV0 struct {
	CWD            string
	Ephemeral      bool
	Model          string
	Sandbox        string
	ApprovalPolicy string
	ServiceTier    string
}

func (params serverCodexAppServerThreadStartParamsV0) toJSONV0() map[string]interface{} {
	out := map[string]interface{}{"ephemeral": params.Ephemeral}
	setNonEmptyJSONFieldV0(out, "cwd", params.CWD)
	setNonEmptyJSONFieldV0(out, "model", params.Model)
	setNonEmptyJSONFieldV0(out, "sandbox", params.Sandbox)
	setNonEmptyJSONFieldV0(out, "approvalPolicy", params.ApprovalPolicy)
	setNonEmptyJSONFieldV0(out, "serviceTier", params.ServiceTier)
	return out
}

type serverCodexAppServerThreadGoalSetParamsV0 struct {
	ThreadID    string
	Objective   string
	Status      string
	TokenBudget int
}

func (params serverCodexAppServerThreadGoalSetParamsV0) toJSONV0() map[string]interface{} {
	out := map[string]interface{}{"threadId": strings.TrimSpace(params.ThreadID)}
	setNonEmptyJSONFieldV0(out, "objective", params.Objective)
	setNonEmptyJSONFieldV0(out, "status", params.Status)
	if params.TokenBudget > 0 {
		out["tokenBudget"] = params.TokenBudget
	}
	return out
}

type serverCodexAppServerThreadSettingsUpdateParamsV0 struct {
	ThreadID string
	Effort   string
}

func (params serverCodexAppServerThreadSettingsUpdateParamsV0) toJSONV0() map[string]interface{} {
	return map[string]interface{}{
		"threadId": strings.TrimSpace(params.ThreadID),
		"effort":   strings.TrimSpace(params.Effort),
	}
}

type serverCodexAppServerTurnStartParamsV0 struct {
	ThreadID          string
	CWD               string
	InputText         string
	ClientMessageID   string
	Model             string
	Effort            string
	ApprovalPolicy    string
	ServiceTier       string
	ToolOutputPolicy  serverCodexAppServerTurnStartToolOutputPolicyV0
	DisablePolicyJSON bool
}

func (params serverCodexAppServerTurnStartParamsV0) toJSONV0() map[string]interface{} {
	out := map[string]interface{}{
		"threadId": strings.TrimSpace(params.ThreadID),
		"input": []map[string]interface{}{{
			"type": "text",
			"text": strings.TrimSpace(params.InputText),
		}},
	}
	setNonEmptyJSONFieldV0(out, "cwd", params.CWD)
	setNonEmptyJSONFieldV0(out, "clientUserMessageId", params.ClientMessageID)
	setNonEmptyJSONFieldV0(out, "model", params.Model)
	setNonEmptyJSONFieldV0(out, "effort", params.Effort)
	setNonEmptyJSONFieldV0(out, "approvalPolicy", params.ApprovalPolicy)
	setNonEmptyJSONFieldV0(out, "serviceTier", params.ServiceTier)
	if !params.DisablePolicyJSON && !params.ToolOutputPolicy.emptyV0() {
		out["toolOutputPolicy"] = params.ToolOutputPolicy.toJSONV0()
	}
	return out
}

type serverCodexAppServerTurnStartToolOutputPolicyV0 struct {
	MaxTextBytes            int
	ThreadReadMaxBytes      int
	RequireBoundedCommands  bool
	BoundedCommandHints     []string
	DurableEvidenceRequired bool
}

func (policy serverCodexAppServerTurnStartToolOutputPolicyV0) emptyV0() bool {
	return policy.MaxTextBytes <= 0 &&
		policy.ThreadReadMaxBytes <= 0 &&
		!policy.RequireBoundedCommands &&
		len(compactServerStackStringsV0(policy.BoundedCommandHints)) == 0 &&
		!policy.DurableEvidenceRequired
}

func (policy serverCodexAppServerTurnStartToolOutputPolicyV0) toJSONV0() map[string]interface{} {
	out := map[string]interface{}{}
	if policy.MaxTextBytes > 0 {
		out["maxTextBytes"] = policy.MaxTextBytes
	}
	if policy.ThreadReadMaxBytes > 0 {
		out["threadReadMaxBytes"] = policy.ThreadReadMaxBytes
	}
	if policy.RequireBoundedCommands {
		out["requireBoundedCommands"] = true
	}
	if hints := compactServerStackStringsV0(policy.BoundedCommandHints); len(hints) > 0 {
		out["boundedCommandHints"] = hints
	}
	if policy.DurableEvidenceRequired {
		out["durableEvidenceRequired"] = true
	}
	return out
}

func setNonEmptyJSONFieldV0(out map[string]interface{}, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		out[key] = trimmed
	}
}

type serverCodexAppServerThreadStartResponseV0 struct {
	Thread serverCodexAppServerThreadV0 `json:"thread"`
}

type serverCodexAppServerThreadV0 struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId,omitempty"`
}

type serverCodexAppServerThreadGoalSetResponseV0 struct {
	Goal serverCodexAppServerThreadGoalV0 `json:"goal"`
}

type serverCodexAppServerThreadGoalGetResponseV0 struct {
	Goal *serverCodexAppServerThreadGoalV0 `json:"goal"`
}

type serverCodexAppServerThreadLoadedListResponseV0 struct {
	Data       []string `json:"data,omitempty"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

type serverCodexAppServerThreadGoalV0 struct {
	ThreadID        string                          `json:"threadId"`
	Objective       string                          `json:"objective"`
	Status          string                          `json:"status"`
	TokenBudget     *int                            `json:"tokenBudget,omitempty"`
	TokensUsed      int                             `json:"tokensUsed,omitempty"`
	TimeUsedSeconds int                             `json:"timeUsedSeconds,omitempty"`
	CreatedAt       serverCodexAppServerTimestampV0 `json:"createdAt,omitempty"`
	UpdatedAt       serverCodexAppServerTimestampV0 `json:"updatedAt,omitempty"`
}

type serverCodexAppServerThreadReadResponseV0 struct {
	Thread serverCodexAppServerThreadReadV0 `json:"thread"`
}

type serverCodexAppServerThreadReadV0 struct {
	ID     string                             `json:"id"`
	Status serverCodexAppServerThreadStatusV0 `json:"status,omitempty"`
	Path   string                             `json:"path,omitempty"`
	Turns  []serverCodexAppServerReadTurnV0   `json:"turns,omitempty"`
}

type serverCodexAppServerThreadStatusV0 string

func (status *serverCodexAppServerThreadStatusV0) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		*status = serverCodexAppServerThreadStatusV0(strings.TrimSpace(text))
		return nil
	}
	var object struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	*status = serverCodexAppServerThreadStatusV0(strings.TrimSpace(object.Type))
	return nil
}

type serverCodexAppServerReadTurnV0 struct {
	ID        string                           `json:"id"`
	Status    string                           `json:"status,omitempty"`
	ItemsView string                           `json:"itemsView,omitempty"`
	Items     []serverCodexAppServerReadItemV0 `json:"items,omitempty"`
}

type serverCodexAppServerReadItemV0 struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Phase string `json:"phase,omitempty"`
	Text  string `json:"text,omitempty"`
}

type serverCodexAppServerTurnStartResponseV0 struct {
	Turn serverCodexAppServerTurnV0 `json:"turn"`
}

type serverCodexAppServerTurnV0 struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
}
