package mcpinterface

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

const (
	publicInvalidRequest = "invalid_request"
	publicNotFound       = "not_found"
	publicConflict       = "conflict"
	publicInternal       = "internal"
)

type ToolError struct {
	Code    string `json:"code" jsonschema:"stable machine-readable error code"`
	Message string `json:"message" jsonschema:"localized human-readable error message"`
}

func (server *Interface) toolError(code string) (*sdkmcp.CallToolResult, *ToolError) {
	message := server.catalog.Text(server.locale, "error."+code)
	public := &ToolError{Code: code, Message: message}
	return &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: message}},
	}, public
}

func publicCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case application.IsStateError(err, application.StateNotFound):
		return publicNotFound
	case application.IsStateError(err, application.StateConflict),
		application.IsStateError(err, application.StateAlreadyClaimed):
		return publicConflict
	case application.IsStateError(err, application.StateInvalid):
		return publicInternal
	}

	switch goal.ErrorCodeOf(err) {
	case goal.ErrorInvalidArgument, goal.ErrorInvalidRef:
		return publicInvalidRequest
	case goal.ErrorRevisionConflict, goal.ErrorInvalidTransition, goal.ErrorScopeConflict:
		return publicConflict
	case "":
	default:
		return publicInternal
	}

	switch err.Error() {
	case "application.request_ref_invalid",
		"application.actor_ref_required",
		"application.project_ref_required",
		"application.statement_required",
		"application.query_invalid",
		"artifact.ref_invalid":
		return publicInvalidRequest
	case "artifact.not_found":
		return publicNotFound
	default:
		return publicInternal
	}
}
