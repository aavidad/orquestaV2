package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

const helperSessionBearer = "v22<session>ephemeral"

type sessionResolverFunc func(context.Context, ports.AgentLaunchRequest) (Session, error)

func (function sessionResolverFunc) ResolveCodexSession(ctx context.Context, request ports.AgentLaunchRequest) (Session, error) {
	return function(ctx, request)
}

type sessionResolverStub struct{ id int }

func (*sessionResolverStub) ResolveCodexSession(context.Context, ports.AgentLaunchRequest) (Session, error) {
	return Session{}, errors.New("unused")
}

func TestBindSessionResolverIsIdempotentAndCannotReplaceAuthority(t *testing.T) {
	first, second := &sessionResolverStub{id: 1}, &sessionResolverStub{id: 2}
	adapter := openTestAdapter(t, testConfig(t))
	if err := adapter.BindSessionResolver(nil); ErrorCode(err) != CodeSessionInvalid {
		t.Fatalf("nil resolver error=%v code=%q", err, ErrorCode(err))
	}
	if err := adapter.BindSessionResolver(first); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	if err := adapter.BindSessionResolver(first); err != nil {
		t.Fatalf("idempotent bind: %v", err)
	}
	if err := adapter.BindSessionResolver(second); ErrorCode(err) != CodeSessionInvalid {
		t.Fatalf("replacement error=%v code=%q", err, ErrorCode(err))
	}

	configured := testConfig(t)
	configured.SessionResolver = first
	configuredAdapter := openTestAdapter(t, configured)
	if err := configuredAdapter.BindSessionResolver(first); err != nil {
		t.Fatalf("configured idempotent bind: %v", err)
	}
	if err := configuredAdapter.BindSessionResolver(second); ErrorCode(err) != CodeSessionInvalid {
		t.Fatalf("configured replacement error=%v code=%q", err, ErrorCode(err))
	}
}

func TestBindSessionResolverRejectsClosedOrUsedAdapter(t *testing.T) {
	resolver := &sessionResolverStub{id: 1}
	closed, err := New(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := closed.BindSessionResolver(resolver); ErrorCode(err) != CodeSessionInvalid {
		t.Fatalf("closed bind error=%v code=%q", err, ErrorCode(err))
	}

	used := openTestAdapter(t, testConfig(t))
	used.mu.Lock()
	used.executions["execution:already-used"] = &executionState{}
	used.mu.Unlock()
	if err := used.BindSessionResolver(resolver); ErrorCode(err) != CodeSessionInvalid {
		t.Fatalf("used bind error=%v code=%q", err, ErrorCode(err))
	}
}

func TestSessionRefFailsClosedWithoutResolver(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	request := testRequest(t, "session-resolver-missing", "helper:success", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:resolver-missing")
	if session, err := adapter.resolveSession(context.Background(), request); session != nil ||
		ErrorCode(err) != CodeSessionUnavailable {
		t.Fatalf("resolveSession() session=%v error=%v code=%q", session, err, ErrorCode(err))
	}
	if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeSessionUnavailable {
		t.Fatalf("Launch() error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Lstat(filepath.Join(
		adapter.rootPath, filepath.FromSlash(executionPath(request.ExecutionRef)),
	)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing resolver created durable/process state: %v", err)
	}
}

func TestSessionProjectionIsPerExecutionAndKeepsBearerOutOfArguments(t *testing.T) {
	config := testConfig(t)
	var seen []string
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
		seen = append(seen, request.ExecutionRef.String())
		token, err := credentials.NewSecret([]byte("session-token-" + request.ExecutionRef.String()))
		if err != nil {
			return Session{}, err
		}
		ref, _ := ports.NewExecutionSessionRef("execution-session:" + request.ExecutionRef.String())
		return Session{Ref: ref, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: token}, nil
	})
	adapter := openTestAdapter(t, config)
	first := testRequest(t, "session-one", "helper:success", 1024)
	second := testRequest(t, "session-two", "helper:success", 1024)
	for _, request := range []ports.AgentLaunchRequest{first, second} {
		request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:" + request.ExecutionRef.String())
		session, err := adapter.resolveSession(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		arguments := adapter.commandArgumentsWithSession("run:session", false, false, session)
		joined := strings.Join(arguments, "\x00")
		if !strings.Contains(joined, `mcp_servers.orquesta.url="http://127.0.0.1:7777/mcp"`) ||
			!strings.Contains(joined, `mcp_servers.orquesta.bearer_token_env_var="ORQUESTA_MCP_BEARER_TOKEN"`) ||
			strings.Contains(joined, "session-token-") {
			t.Fatalf("unsafe or incomplete session arguments: %q", joined)
		}
		environment := adapter.environmentWithSession(adapter.environment, session)
		if !containsExactEnvironment(environment, codexMCPBearerTokenEnvironment+"=session-token-"+request.ExecutionRef.String()) {
			t.Fatalf("session token missing from exact child environment: %q", environment)
		}
		if strings.Contains(shellEnvironmentIncludeOnly(adapter.config.Environment), codexMCPBearerTokenEnvironment) {
			t.Fatal("session bearer reached the Codex shell/tool include-only projection")
		}
		clearEnvironment(environment)
		session.destroy()
	}
	if strings.Join(seen, ",") != first.ExecutionRef.String()+","+second.ExecutionRef.String() {
		t.Fatalf("session resolver did not receive exact launches: %v", seen)
	}
}

func TestSessionResolutionRejectsInvalidOrLeakingMaterial(t *testing.T) {
	request := testRequest(t, "session-invalid", "objective session-secret", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:invalid")
	for name, resolver := range map[string]SessionResolver{
		"invalid endpoint": sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
			secret, _ := credentials.NewSecret([]byte("safe-token"))
			ref, _ := ports.NewExecutionSessionRef("execution-session:invalid")
			return Session{Ref: ref, Endpoint: "http://example.invalid/mcp", BearerToken: secret}, nil
		}),
		"resolver unavailable": sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
			return Session{}, errors.New("unavailable")
		}),
		"token in prompt": sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
			secret, _ := credentials.NewSecret([]byte("session-secret"))
			ref, _ := ports.NewExecutionSessionRef("execution-session:invalid")
			return Session{Ref: ref, Endpoint: "https://mcp.example.test/mcp", BearerToken: secret}, nil
		}),
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig(t)
			config.SessionResolver = resolver
			adapter := openTestAdapter(t, config)
			session, err := adapter.resolveSession(context.Background(), request)
			if name == "resolver unavailable" {
				if ErrorCode(err) != CodeSessionUnavailable {
					t.Fatalf("error=%v code=%q", err, ErrorCode(err))
				}
				return
			}
			if name == "invalid endpoint" {
				if ErrorCode(err) != CodeSessionInvalid {
					t.Fatalf("error=%v code=%q", err, ErrorCode(err))
				}
				return
			}
			defer session.destroy()
			if err := adapter.preflightSessionLaunch(session, request); ErrorCode(err) != CodeSecretLeak {
				t.Fatalf("leak preflight error=%v code=%q", err, ErrorCode(err))
			}
		})
	}
}

func TestSessionBearerCannotBeConfiguredAsPublicEnvironment(t *testing.T) {
	config := testConfig(t)
	config.Environment[codexMCPBearerTokenEnvironment] = "not-a-session"
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeEnvironmentInvalid {
		t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
	}
}

func TestSessionEndpointRejectsEmptyQueryMarker(t *testing.T) {
	if validSessionEndpoint("http://127.0.0.1:7777/mcp?") {
		t.Fatal("session endpoint with force-query marker was accepted")
	}
}

func TestSessionLaunchKeepsBearerOutOfJournalAndShellProjection(t *testing.T) {
	config := testConfig(t)
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
		secret, err := credentials.NewSecret([]byte(helperSessionBearer))
		if err != nil {
			return Session{}, err
		}
		return Session{
			Ref:         request.SessionRef,
			Endpoint:    "http://127.0.0.1:7777/mcp",
			BearerToken: secret,
		}, nil
	})
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "session-launch", "helper:success helper:session", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:session-launch")

	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if observation := awaitTerminal(t, adapter, request.ExecutionRef); observation.Status != ports.AgentCompleted {
		t.Fatalf("session launch observation = %+v", observation)
	}

	runDirectory := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
	for _, fileName := range []string{requestFileName, terminalFileName, lastMessageFileName} {
		payload, err := os.ReadFile(filepath.Join(runDirectory, fileName))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", fileName, err)
		}
		if strings.Contains(string(payload), helperSessionBearer) || strings.Contains(string(payload), "127.0.0.1:7777/mcp") {
			t.Fatalf("session material persisted in %s", fileName)
		}
	}
	requestPayload, err := os.ReadFile(filepath.Join(runDirectory, requestFileName))
	if err != nil {
		t.Fatalf("ReadFile(request) error = %v", err)
	}
	if !strings.Contains(string(requestPayload), request.SessionRef.String()) {
		t.Fatal("opaque execution session reference missing from launch journal")
	}
}

func TestSessionPrestartJournalReplayResolvesExactSession(t *testing.T) {
	config := testConfig(t)
	resolverCalls := 0
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
		resolverCalls++
		secret, err := credentials.NewSecret([]byte(helperSessionBearer))
		if err != nil {
			return Session{}, err
		}
		return Session{Ref: request.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
	})
	request := testRequest(t, "session-prestart-replay", "helper:success", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:session-prestart-replay")
	otherSession := request
	otherSession.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:session-prestart-replay-other")
	if mustRequestHash(t, request) == mustRequestHash(t, otherSession) {
		t.Fatal("opaque execution session reference did not bind the launch identity")
	}

	seed, err := New(config)
	if err != nil {
		t.Fatalf("New(seed) error = %v", err)
	}
	requestHash := mustRequestHash(t, request)
	_, runPath, created, err := seed.ensureLaunchRecord(request, requestHash)
	if err != nil || !created {
		t.Fatalf("ensureLaunchRecord() created=%v error=%v", created, err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("Close(seed) error = %v", err)
	}

	reopened := openTestAdapter(t, config)
	if _, err := reopened.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch(replay) error = %v", err)
	}
	if resolverCalls != 1 {
		t.Fatalf("session resolver calls = %d, want 1 for non-terminal journal replay", resolverCalls)
	}
	payload, err := os.ReadFile(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), requestFileName))
	if err != nil {
		t.Fatalf("ReadFile(request journal) error = %v", err)
	}
	if strings.Contains(string(payload), helperSessionBearer) || !strings.Contains(string(payload), request.SessionRef.String()) {
		t.Fatal("pre-start journal lost opaque ref or persisted bearer material")
	}
}

func containsExactEnvironment(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
