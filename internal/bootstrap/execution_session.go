package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/adapters/auth/executiontoken"
	"orquesta/internal/application"
	commandcore "orquesta/internal/commands"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type codexExecutionSessionResolver struct {
	authority application.ExecutionSessionAuthoritySource
	broker    *executiontoken.Broker
	endpoint  string
}

func newCodexExecutionSessionResolver(
	authority application.ExecutionSessionAuthoritySource,
	broker *executiontoken.Broker,
	endpoint string,
) (codex.SessionResolver, error) {
	if authority == nil || broker == nil || !validLoopbackMCPEndpoint(endpoint) {
		return nil, errors.New("bootstrap.execution_session_resolver_invalid")
	}
	return &codexExecutionSessionResolver{
		authority: authority,
		broker:    broker,
		endpoint:  endpoint,
	}, nil
}

func (resolver *codexExecutionSessionResolver) ResolveCodexSession(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (codex.Session, error) {
	if resolver == nil || resolver.authority == nil || resolver.broker == nil ||
		request.SessionRef.String() == "" {
		return codex.Session{}, errors.New("bootstrap.execution_session_unavailable")
	}
	authority, err := resolver.authority.ExecutionSessionAuthority(
		ctx,
		request.ExecutionRef,
		executiontoken.AuthenticationMethod,
	)
	if err != nil || authority.SessionRef != request.SessionRef ||
		!executionAuthorityMatchesLaunch(authority, request) {
		return codex.Session{}, errors.New("bootstrap.execution_session_unavailable")
	}
	var token credentials.Secret
	var secretErr error
	err = resolver.broker.UseToken(ctx, authority.Request, func(material []byte) error {
		token, secretErr = credentials.NewSecret(material)
		return secretErr
	})
	if err != nil || secretErr != nil {
		token.Destroy()
		return codex.Session{}, errors.New("bootstrap.execution_session_unavailable")
	}
	return codex.Session{
		Ref:         authority.SessionRef,
		Endpoint:    resolver.endpoint,
		BearerToken: token,
	}, nil
}

func executionAuthorityMatchesLaunch(
	authority ports.ExecutionSessionAuthority,
	request ports.AgentLaunchRequest,
) bool {
	binding := authority.Request
	return binding.ProjectRef == request.ProjectRef &&
		binding.GoalRef == request.GoalRef &&
		binding.WorkItemRef == request.WorkItemRef &&
		binding.ExecutionRef == request.ExecutionRef &&
		binding.ExecutionAttempt == request.ExecutionAttempt &&
		binding.PlanGeneration == request.PlanGeneration &&
		binding.AppSpecGeneration == request.AppSpecGeneration &&
		binding.SpecHash == request.SpecHash
}

type loopbackPostArtifactMailboxAdmitter struct {
	broker   *executiontoken.Broker
	endpoint string
}

func newLoopbackPostArtifactMailboxAdmitter(
	broker *executiontoken.Broker,
	endpoint string,
) (application.PostArtifactMailboxAdmitter, error) {
	if broker == nil || !validLoopbackMCPEndpoint(endpoint) {
		return nil, errors.New("bootstrap.post_artifact_mailbox_admitter_invalid")
	}
	return &loopbackPostArtifactMailboxAdmitter{broker: broker, endpoint: endpoint}, nil
}

func (admitter *loopbackPostArtifactMailboxAdmitter) AdmitPostArtifactMailbox(
	ctx context.Context,
	request application.PostArtifactMailboxAdmissionRequest,
) (application.PostArtifactMailboxAdmissionReceipt, error) {
	if admitter == nil || admitter.broker == nil || ctx == nil ||
		strings.TrimSpace(request.RequestRef) != request.RequestRef || request.RequestRef == "" {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_request_invalid")
	}
	var receipt application.PostArtifactMailboxAdmissionReceipt
	err := admitter.broker.UseToken(ctx, request.Session, func(token []byte) error {
		var callErr error
		receipt, callErr = callPostArtifactMailboxMCP(ctx, admitter.endpoint, token, request)
		return callErr
	})
	if err != nil {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_unavailable")
	}
	return receipt, nil
}

func callPostArtifactMailboxMCP(
	ctx context.Context,
	endpoint string,
	token []byte,
	request application.PostArtifactMailboxAdmissionRequest,
) (application.PostArtifactMailboxAdmissionReceipt, error) {
	transport, err := newExecutionBearerTransport(endpoint, token)
	if err != nil {
		return application.PostArtifactMailboxAdmissionReceipt{}, err
	}
	defer transport.destroy()
	httpClient := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("bootstrap.execution_session_redirect_forbidden")
		},
	}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "orquesta-post-artifact-mailbox", Version: "1"},
		nil,
	)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: endpoint, HTTPClient: httpClient,
	}, nil)
	if err != nil {
		return application.PostArtifactMailboxAdmissionReceipt{}, err
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "orquesta.mailbox.admit",
		Arguments: map[string]any{
			"version":               "1",
			"request_ref":           request.RequestRef,
			"project_ref":           request.Session.ProjectRef.String(),
			"claimed_execution_ref": request.Session.ExecutionRef.String(),
			"payload": map[string]any{
				"goal_ref":                 request.GoalRef.String(),
				"expected_plan_generation": uint64(request.ExpectedPlanGeneration),
				"kind":                     "child_delivery",
				"parent_work_item_ref":     request.ParentWorkItemRef.String(),
				"child_work_item_ref":      request.ChildWorkItemRef.String(),
				"recipient_principal_ref":  request.RecipientPrincipalRef.String(),
				"recipient_execution_ref":  request.RecipientExecutionRef.String(),
				"summary":                  request.Summary,
				"artifact_refs":            artifactRefStrings(request.ArtifactRefs),
			},
		},
	})
	if err != nil || result == nil || result.IsError {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_call_failed")
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return application.PostArtifactMailboxAdmissionReceipt{}, err
	}
	var output struct {
		Result commandcore.Result `json:"result"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil || output.Result.Failure != nil {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_result_invalid")
	}
	var data struct {
		Receipt struct {
			GoalRef      string `json:"goal_ref"`
			MessageRef   string `json:"message_ref"`
			AdmissionRef string `json:"admission_ref"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal(output.Result.Data, &data); err != nil {
		return application.PostArtifactMailboxAdmissionReceipt{}, err
	}
	goalRef, err := goal.NewGoalRef(data.Receipt.GoalRef)
	if err != nil || goalRef != request.GoalRef {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_receipt_invalid")
	}
	messageRef, err := application.NewMailboxMessageRef(data.Receipt.MessageRef)
	if err != nil || data.Receipt.AdmissionRef == "" {
		return application.PostArtifactMailboxAdmissionReceipt{},
			errors.New("bootstrap.post_artifact_mailbox_receipt_invalid")
	}
	return application.PostArtifactMailboxAdmissionReceipt{
		GoalRef:      goalRef,
		MessageRef:   messageRef,
		AdmissionRef: data.Receipt.AdmissionRef,
	}, nil
}

func artifactRefStrings(refs []goal.ArtifactRef) []string {
	result := make([]string, len(refs))
	for index := range refs {
		result[index] = refs[index].String()
	}
	return result
}

type executionBearerTransport struct {
	base     http.RoundTripper
	scheme   string
	host     string
	path     string
	material []byte
}

func newExecutionBearerTransport(endpoint string, material []byte) (*executionBearerTransport, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || len(material) == 0 || !validLoopbackMCPEndpoint(endpoint) {
		return nil, errors.New("bootstrap.execution_session_transport_invalid")
	}
	return &executionBearerTransport{
		base: http.DefaultTransport, scheme: parsed.Scheme, host: parsed.Host,
		path: parsed.EscapedPath(), material: append([]byte(nil), material...),
	}, nil
}

func (transport *executionBearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || request == nil || request.URL == nil ||
		request.URL.Scheme != transport.scheme || request.URL.Host != transport.host ||
		request.URL.EscapedPath() != transport.path || request.URL.User != nil ||
		request.URL.RawQuery != "" || request.URL.ForceQuery || request.URL.Fragment != "" ||
		len(transport.material) == 0 {
		return nil, errors.New("bootstrap.execution_session_target_forbidden")
	}
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+string(transport.material))
	return transport.base.RoundTrip(cloned)
}

func (transport *executionBearerTransport) destroy() {
	if transport == nil {
		return
	}
	clear(transport.material)
	transport.material = nil
}

func validLoopbackMCPEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil ||
		parsed.Host == "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		parsed.Path == "" {
		return false
	}
	address := net.ParseIP(parsed.Hostname())
	return address != nil && address.IsLoopback()
}
