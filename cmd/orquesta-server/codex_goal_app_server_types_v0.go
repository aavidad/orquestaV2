package main

import "strings"

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

type serverCodexAppServerTurnStartParamsV0 struct {
	ThreadID        string
	CWD             string
	InputText       string
	ClientMessageID string
	Model           string
	Effort          string
	ApprovalPolicy  string
	ServiceTier     string
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
	ID     string                           `json:"id"`
	Status string                           `json:"status,omitempty"`
	Turns  []serverCodexAppServerReadTurnV0 `json:"turns,omitempty"`
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
