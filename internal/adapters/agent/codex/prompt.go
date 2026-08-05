package codex

import (
	"fmt"
	"strings"

	"orquesta/internal/ports"
)

// AgentPrompt remains an alias for compatibility with existing compositions.
// Ownership of the provider-neutral allowlist lives in ports.
type AgentPrompt = ports.AgentPrompt

// PromptRenderer keeps locale and catalog ownership in composition.
type PromptRenderer interface {
	RenderAgentPrompt(AgentPrompt) (string, error)
}

func (adapter *Adapter) renderAgentPrompt(request ports.AgentLaunchRequest) (string, error) {
	if adapter == nil || adapter.config.PromptRenderer == nil {
		return "", &Error{Code: CodePromptRendererInvalid}
	}
	rendered, err := adapter.config.PromptRenderer.RenderAgentPrompt(ports.AgentPromptFromLaunchRequest(request))
	if err != nil {
		return "", &Error{Code: CodePromptRenderFailed, Cause: err}
	}
	if strings.TrimSpace(rendered) == "" {
		return "", &Error{Code: CodePromptRenderFailed, Cause: fmt.Errorf("empty rendered prompt")}
	}
	return rendered, nil
}
