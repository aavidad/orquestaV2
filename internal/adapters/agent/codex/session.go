package codex

import (
	"context"
	"net"
	"net/url"
	"reflect"
	"strings"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

// SessionResolver is adapter-local on purpose. It receives the exact launch
// identity and resolves a per-execution opaque ref into ephemeral transport
// material immediately before a child process starts.
// Neither the endpoint nor token is written to a goal, journal, receipt, prompt
// or provider-neutral port.
type SessionResolver interface {
	ResolveCodexSession(context.Context, ports.AgentLaunchRequest) (Session, error)
}

type Session struct {
	Ref         ports.ExecutionSessionRef
	Endpoint    string
	BearerToken credentials.Secret
}

type resolvedSession struct {
	endpoint string
	token    credentials.Secret
	guard    *credentials.LeakGuard
}

// BindSessionResolver attaches the composition-owned resolver after its
// durable backing state is open. It may not replace authority once configured
// or after any execution has entered adapter state.
func (adapter *Adapter) BindSessionResolver(resolver SessionResolver) error {
	if adapter == nil || resolver == nil {
		return &Error{Code: CodeSessionInvalid}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed || len(adapter.executions) != 0 {
		return &Error{Code: CodeSessionInvalid}
	}
	if adapter.config.SessionResolver != nil &&
		!sameSessionResolver(adapter.config.SessionResolver, resolver) {
		return &Error{Code: CodeSessionInvalid}
	}
	adapter.config.SessionResolver = resolver
	return nil
}

func sameSessionResolver(left, right SessionResolver) bool {
	leftValue, rightValue := reflect.ValueOf(left), reflect.ValueOf(right)
	return leftValue.IsValid() && rightValue.IsValid() &&
		leftValue.Type() == rightValue.Type() && leftValue.Type().Comparable() &&
		leftValue.Interface() == rightValue.Interface()
}

func (session *resolvedSession) destroy() {
	if session == nil {
		return
	}
	if session.guard != nil {
		session.guard.Destroy()
		session.guard = nil
	}
	session.token.Destroy()
	session.endpoint = ""
}

func (adapter *Adapter) resolveSession(ctx context.Context, request ports.AgentLaunchRequest) (*resolvedSession, error) {
	if adapter.config.SessionResolver == nil {
		if request.SessionRef.String() != "" {
			return nil, &Error{Code: CodeSessionUnavailable}
		}
		return nil, nil
	}
	if request.SessionRef.String() == "" {
		return nil, &Error{Code: CodeSessionInvalid}
	}
	resolved, err := adapter.config.SessionResolver.ResolveCodexSession(ctx, request)
	if err != nil {
		return nil, &Error{Code: CodeSessionUnavailable, Cause: err}
	}
	canonicalRef, refErr := ports.NewExecutionSessionRef(resolved.Ref.String())
	if refErr != nil || canonicalRef != resolved.Ref || resolved.Ref != request.SessionRef || !validSessionEndpoint(resolved.Endpoint) {
		resolved.BearerToken.Destroy()
		return nil, &Error{Code: CodeSessionInvalid}
	}
	token := resolved.BearerToken
	material := token.Bytes()
	empty := len(material) == 0
	clearBytes(material)
	if empty {
		token.Destroy()
		return nil, &Error{Code: CodeSessionInvalid}
	}
	guard, err := credentials.NewLeakGuard(token)
	if err != nil {
		token.Destroy()
		return nil, &Error{Code: CodeSessionInvalid, Cause: err}
	}
	return &resolvedSession{endpoint: resolved.Endpoint, token: token, guard: guard}, nil
}

func validSessionEndpoint(raw string) bool {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsRune(raw, '\x00') {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" ||
		parsed.ForceQuery || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return parsed.Scheme == "http" && net.ParseIP(parsed.Hostname()) != nil && net.ParseIP(parsed.Hostname()).IsLoopback()
}

func (adapter *Adapter) environmentWithSession(base []string, session *resolvedSession) []string {
	environment := append([]string(nil), base...)
	if session == nil {
		return environment
	}
	material := session.token.Bytes()
	defer clearBytes(material)
	return append(environment, adapter.config.MCPBearerTokenEnvVar+"="+string(material))
}

func (adapter *Adapter) preflightSessionLaunch(session *resolvedSession, request ports.AgentLaunchRequest) error {
	if session == nil {
		return nil
	}
	prompt, err := adapter.renderAgentPrompt(request)
	if err != nil {
		return err
	}
	surfaces := []credentials.LeakSurface{
		{Name: "prompt", Content: []byte(prompt)},
		{Name: "command", Content: []byte(adapter.command)},
		{Name: "arguments", Content: []byte(strings.Join(adapter.commandArgumentsWithSession("run:session", false, false, session), "\x00"))},
	}
	defer func() {
		for index := range surfaces {
			clearBytes(surfaces[index].Content)
		}
	}()
	if err := session.guard.Scan(surfaces); err != nil {
		if credentials.HasErrorCode(err, credentials.ErrorSecretLeak) {
			return &Error{Code: CodeSecretLeak}
		}
		return &Error{Code: CodeSessionUnavailable, Cause: err}
	}
	return nil
}

func sessionArguments(session *resolvedSession, bearerTokenEnvVar string) []string {
	if session == nil {
		return nil
	}
	return []string{
		"--config", `mcp_servers.orquesta.url="` + session.endpoint + `"`,
		"--config", `mcp_servers.orquesta.bearer_token_env_var="` + bearerTokenEnvVar + `"`,
	}
}
