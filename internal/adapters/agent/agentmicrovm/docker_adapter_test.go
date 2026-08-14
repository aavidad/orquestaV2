package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/agentprotocol/codexwork"
	"orquesta/internal/ports"
)

func TestDockerAdapterPersistsExactBytesBeforeEveryEffect(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	receipt, err := fixture.adapter.Launch(context.Background(), fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{
		"negotiate", "record:launch", "effect:launch", "bind:launch",
		"record:session_start", "effect:session_start",
		"record:session_input", "effect:session_input",
	}
	if !reflect.DeepEqual(fixture.events, wantEvents) {
		t.Fatalf("events=%v want=%v", fixture.events, wantEvents)
	}
	if fixture.client.launchCalls != 1 || fixture.client.queryCalls != 0 || fixture.client.startCalls != 1 ||
		fixture.client.inputCalls != 1 || len(fixture.store.calls) != 1 {
		t.Fatalf("calls launch=%d query=%d start=%d input=%d signer=%d",
			fixture.client.launchCalls, fixture.client.queryCalls, fixture.client.startCalls, fixture.client.inputCalls, len(fixture.store.calls))
	}
	launch := fixture.journal.mustRecord(t, fixture.request, ports.AgentProviderRequestLaunch)
	signed := decodeDockerSigned(t, launch.Body)
	compiled := mustCompileDocker(t, fixture.request, fixture.binding)
	if !reflect.DeepEqual(signed.Plan, compiled.Plan) || launch.IdempotencyKey != signed.Plan.OperacionRef ||
		launch.LaunchBindingRef != signed.Plan.EjecucionRef || launch.LaunchBindingRevision != 3 {
		t.Fatalf("launch journal=%+v plan=%+v", launch, signed.Plan)
	}
	start := fixture.journal.mustRecord(t, fixture.request, ports.AgentProviderRequestSessionStart)
	input := fixture.journal.mustRecord(t, fixture.request, ports.AgentProviderRequestSessionInput)
	if !bytes.Equal(launch.Body, fixture.client.launchBodies[0]) ||
		!bytes.Equal(start.Body, fixture.client.startBodies[0]) ||
		!bytes.Equal(input.Body, fixture.client.inputBodies[0]) {
		t.Fatal("client did not receive exact journaled bytes")
	}
	var inputRequest microvm.SolicitudEntradaSesionContenedorV1
	if json.Unmarshal(input.Body, &inputRequest) != nil {
		t.Fatal("session input is not JSON")
	}
	packetRaw, decodeErr := base64.StdEncoding.Strict().DecodeString(inputRequest.ContenidoBase64)
	packet, packetErr := codexwork.DecodeWorkPacketV1(packetRaw)
	if decodeErr != nil || packetErr != nil || packet.ExecutionRef != fixture.request.ExecutionRef.String() ||
		packet.EffectAttemptRef != fixture.request.EffectAuthority.EffectAttemptRef || packet.ControlledEgressProxy != "" {
		t.Fatalf("packet=%+v decode=%v packet_error=%v", packet, decodeErr, packetErr)
	}
	if receipt.ExternalRef != signed.Plan.EjecucionRef || receipt.AcceptedAt != compiled.IssuedAt ||
		receipt.ProviderRef != DockerProviderRef || receipt.RequierePreservacionEntorno {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestDockerAdapterLaunchAndReconcileReplayKeepExactBytesAndReceipt(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	first, firstErr := fixture.adapter.Launch(context.Background(), fixture.request)
	second, secondErr := fixture.adapter.Launch(context.Background(), fixture.request)
	reopened := fixture.journal.reopen()
	fixture.client.journal = reopened
	third, thirdErr := fixture.mustAdapter(t, reopened).ReconcileLaunch(context.Background(), fixture.request)
	if firstErr != nil || secondErr != nil || thirdErr != nil ||
		!reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) ||
		fixture.client.launchCalls != 2 || fixture.client.startCalls != 3 ||
		fixture.client.queryCalls != 1 || fixture.client.inputCalls != 3 || len(fixture.store.calls) != 2 ||
		!bytes.Equal(fixture.client.launchBodies[0], fixture.client.launchBodies[1]) ||
		!bytes.Equal(fixture.client.startBodies[0], fixture.client.startBodies[1]) ||
		!bytes.Equal(fixture.client.startBodies[0], fixture.client.startBodies[2]) ||
		!bytes.Equal(fixture.client.inputBodies[0], fixture.client.inputBodies[1]) ||
		!bytes.Equal(fixture.client.inputBodies[0], fixture.client.inputBodies[2]) {
		t.Fatalf("errors=%v/%v/%v receipts_equal=%t calls=%d/%d/%d signer=%d",
			firstErr, secondErr, thirdErr, reflect.DeepEqual(first, third), fixture.client.launchCalls,
			fixture.client.startCalls, fixture.client.inputCalls, len(fixture.store.calls))
	}
}

func TestDockerAdapterReconcileQueriesRecordedOperationWithoutSecondLaunch(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	fixture.client.launchErr = errors.New("lost launch acknowledgement")
	if _, err := fixture.adapter.Launch(context.Background(), fixture.request); ErrorCode(err) != CodeLaunchUnavailable || err.(*Error).DefinitelyNotApplied() {
		t.Fatalf("Launch() err=%v", err)
	}
	if fixture.client.launchCalls != 1 || len(fixture.store.calls) != 1 {
		t.Fatalf("initial calls launch=%d signer=%d", fixture.client.launchCalls, len(fixture.store.calls))
	}

	reopened := fixture.journal.reopen()
	fixture.events = nil
	fixture.client.events = &fixture.events
	fixture.client.journal = reopened
	fixture.client.launchErr = nil
	reconciler := fixture.mustAdapter(t, reopened)
	receipt, err := reconciler.ReconcileLaunch(context.Background(), fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{
		"negotiate", "resolve:launch", "query:launch", "bind:launch",
		"record:session_start", "effect:session_start",
		"record:session_input", "effect:session_input",
	}
	if !reflect.DeepEqual(fixture.events, wantEvents) || fixture.client.launchCalls != 1 ||
		fixture.client.queryCalls != 1 || len(fixture.store.calls) != 1 {
		t.Fatalf("events=%v launch=%d query=%d signer=%d receipt=%+v", fixture.events,
			fixture.client.launchCalls, fixture.client.queryCalls, len(fixture.store.calls), receipt)
	}
	resolved := reopened.mustRecord(t, fixture.request, ports.AgentProviderRequestLaunch)
	if !bytes.Equal(resolved.Body, fixture.client.launchBodies[0]) || resolved.LaunchBindingRef != receipt.ExternalRef {
		t.Fatalf("recovered launch=%+v receipt=%+v", resolved, receipt)
	}
}

func TestDockerAdapterReconcileRejectsMissingOrNonCanonicalJournalBeforeQuery(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		fixture := newDockerAdapterFixture(t)
		_, err := fixture.adapter.ReconcileLaunch(context.Background(), fixture.request)
		if ErrorCode(err) != CodeDockerRecoveryInvalid || fixture.client.queryCalls != 0 ||
			fixture.client.launchCalls != 0 || len(fixture.store.calls) != 0 {
			t.Fatalf("err=%v query=%d launch=%d signer=%d", err, fixture.client.queryCalls, fixture.client.launchCalls, len(fixture.store.calls))
		}
	})
	mutations := []struct {
		name   string
		mutate func(*ports.AgentProviderRequest)
	}{
		{"non canonical", func(value *ports.AgentProviderRequest) {
			value.Body = append(value.Body, ' ')
			value.BodySHA256 = ports.AgentProviderRequestBodySHA256(value.Body)
		}},
		{"crossed key", func(value *ports.AgentProviderRequest) { value.Key.ActionFence++ }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			fixture := lostDockerLaunchFixture(t)
			key := dockerJournalKey(fixture.request, ports.AgentProviderRequestLaunch)
			stored := fixture.journal.records[key]
			mutation.mutate(&stored)
			fixture.journal.records[key] = stored
			_, err := fixture.adapter.ReconcileLaunch(context.Background(), fixture.request)
			if ErrorCode(err) != CodeDockerRecoveryInvalid || fixture.client.queryCalls != 0 {
				t.Fatalf("err=%v query=%d", err, fixture.client.queryCalls)
			}
		})
	}
}

func TestDockerAdapterReconcileClassifiesQueryFailureAndRejectsCrossedReceipt(t *testing.T) {
	t.Run("query unavailable", func(t *testing.T) {
		fixture := lostDockerLaunchFixture(t)
		fixture.client.queryErr = errors.New("query unavailable")
		_, err := fixture.adapter.ReconcileLaunch(context.Background(), fixture.request)
		adapterErr, ok := err.(*Error)
		if !ok || ErrorCode(err) != CodeDockerRecoveryPending || !adapterErr.Temporary() ||
			fixture.client.queryCalls != 1 || fixture.client.startCalls != 0 {
			t.Fatalf("err=%v query=%d start=%d", err, fixture.client.queryCalls, fixture.client.startCalls)
		}
	})
	tests := []struct {
		name, code string
		bound      bool
		mutate     func(*microvm.RespuestaOperacionContenedorV1)
	}{
		{"operation", CodeDockerResponseInvalid, false, func(value *microvm.RespuestaOperacionContenedorV1) { value.OperacionRef = "docker-launch:other" }},
		{"plan", CodeDockerResponseInvalid, false, func(value *microvm.RespuestaOperacionContenedorV1) { value.PlanSHA256 = strings.Repeat("f", 64) }},
		{"binding", CodeDockerJournalFailed, true, func(value *microvm.RespuestaOperacionContenedorV1) { value.Contenedor.Revision++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := lostDockerLaunchFixture(t)
			if test.bound {
				fixture = newDockerAdapterFixture(t)
				if _, err := fixture.adapter.Launch(context.Background(), fixture.request); err != nil {
					t.Fatal(err)
				}
				fixture.client.startCalls, fixture.client.inputCalls = 0, 0
			}
			fixture.client.mutateQuery = test.mutate
			_, err := fixture.adapter.ReconcileLaunch(context.Background(), fixture.request)
			if ErrorCode(err) != test.code || fixture.client.queryCalls != 1 || fixture.client.startCalls != 0 {
				t.Fatalf("err=%v query=%d start=%d", err, fixture.client.queryCalls, fixture.client.startCalls)
			}
		})
	}
}

func TestDockerAdapterCancellationAfterJournalDoesNotReachLaunch(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	fixture.journal.afterRecord = func(stage ports.AgentProviderRequestStage) {
		if stage == ports.AgentProviderRequestLaunch {
			cancel()
		}
	}
	_, err := fixture.adapter.Launch(ctx, fixture.request)
	if ErrorCode(err) != CodeLaunchCanceledBeforeSubmit || !errors.Is(err, context.Canceled) ||
		fixture.client.launchCalls != 0 || len(fixture.store.calls) != 1 {
		t.Fatalf("err=%v launch=%d signer=%d", err, fixture.client.launchCalls, len(fixture.store.calls))
	}
	fixture.journal.mustRecord(t, fixture.request, ports.AgentProviderRequestLaunch)
}

func TestDockerAdapterJournalFailuresStopOnlyEffectsNotYetAttempted(t *testing.T) {
	tests := []struct {
		name         string
		recordStage  ports.AgentProviderRequestStage
		corruptStage ports.AgentProviderRequestStage
		bindFailure  bool
		wantLaunch   int
		wantStart    int
		wantInput    int
	}{
		{"launch record", ports.AgentProviderRequestLaunch, "", false, 0, 0, 0},
		{"launch bind", "", "", true, 1, 0, 0},
		{"session start record", ports.AgentProviderRequestSessionStart, "", false, 1, 0, 0},
		{"session input record", ports.AgentProviderRequestSessionInput, "", false, 1, 1, 0},
		{"session start returned binding", "", ports.AgentProviderRequestSessionStart, false, 1, 0, 0},
		{"session input returned binding", "", ports.AgentProviderRequestSessionInput, false, 1, 1, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDockerAdapterFixture(t)
			if test.recordStage != "" {
				fixture.journal.recordErrors[test.recordStage] = errors.New("journal unavailable")
			}
			if test.bindFailure {
				fixture.journal.bindErr = errors.New("binding unavailable")
			}
			fixture.journal.corruptResultStage = test.corruptStage
			_, err := fixture.adapter.Launch(context.Background(), fixture.request)
			if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.launchCalls != test.wantLaunch ||
				fixture.client.startCalls != test.wantStart || fixture.client.inputCalls != test.wantInput {
				t.Fatalf("err=%v calls=%d/%d/%d", err, fixture.client.launchCalls, fixture.client.startCalls, fixture.client.inputCalls)
			}
		})
	}
}

func TestDockerAdapterRejectsUnsupportedAuthorityBeforeCredentialOrClient(t *testing.T) {
	authority := validLaunchRequest(t).AccessAuthority
	egress := withEgressAuthority(t, validLaunchRequest(t), validEgressGrant()).EgressAuthority
	workspace, _ := ports.NewExecutionWorkspaceRef("workspace:execution:docker-rejected")
	tests := []struct {
		name, code string
		mutate     func(*ports.AgentLaunchRequest)
	}{
		{"egress", CodeDockerAuthorityUnsupported, func(value *ports.AgentLaunchRequest) { value.EgressAuthority = egress }},
		{"workspace", CodeDockerAuthorityUnsupported, func(value *ports.AgentLaunchRequest) { value.ExecutionWorkspaceRef = workspace }},
		{"access", CodeDockerAuthorityUnsupported, func(value *ports.AgentLaunchRequest) { value.AccessAuthority = authority }},
		{"artifact partial", CodeLaunchRequestInvalid, func(value *ports.AgentLaunchRequest) {
			value.AccessAuthority.ArtifactAccessRef = authority.ArtifactAccessRef
		}},
		{"mcp partial", CodeLaunchRequestInvalid, func(value *ports.AgentLaunchRequest) { value.AccessAuthority.MCPAccessRef = authority.MCPAccessRef }},
		{"mailbox partial", CodeLaunchRequestInvalid, func(value *ports.AgentLaunchRequest) {
			value.AccessAuthority.MailboxEndpointRef = authority.MailboxEndpointRef
		}},
		{"preservation", CodeCapabilityMismatch, func(value *ports.AgentLaunchRequest) { value.RequierePreservacionEntorno = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDockerAdapterFixture(t)
			request := fixture.request
			test.mutate(&request)
			_, err := fixture.adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.code || len(fixture.store.calls) != 0 ||
				fixture.client.negotiations != 0 || len(fixture.journal.records) != 0 || !err.(*Error).DefinitelyNotApplied() {
				t.Fatalf("err=%v signer=%d client=%d records=%d", err, len(fixture.store.calls),
					fixture.client.negotiations, len(fixture.journal.records))
			}
		})
	}
}

func TestDockerAdapterRejectsCrossedResponses(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*dockerClientStub)
	}{
		{"launch", func(client *dockerClientStub) {
			client.mutateLaunch = func(response *microvm.RespuestaContenedorV1) { response.Cerca++ }
		}},
		{"session start", func(client *dockerClientStub) {
			client.mutateStart = func(response *microvm.RespuestaInicioSesionContenedorV1) { response.SesionRef = "session:other" }
		}},
		{"session input", func(client *dockerClientStub) {
			client.mutateInput = func(response *microvm.RespuestaEntradaSesionContenedorV1) { response.EntradaRef = "input:other" }
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDockerAdapterFixture(t)
			test.mutate(fixture.client)
			_, err := fixture.adapter.Launch(context.Background(), fixture.request)
			want := CodeDockerResponseInvalid
			if test.name != "launch" {
				want = CodeSessionResponseInvalid
			}
			if ErrorCode(err) != want {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

type dockerAdapterFixture struct {
	t       *testing.T
	request ports.AgentLaunchRequest
	binding DockerPhysicalBinding
	events  []string
	journal *dockerJournalStub
	client  *dockerClientStub
	store   *credentialSignerStoreStub
	signer  *CredentialSigner
	adapter *DockerAdapter
}

func newDockerAdapterFixture(t *testing.T) *dockerAdapterFixture {
	t.Helper()
	request := validLaunchRequest(t)
	request.AccessAuthority = ports.AgentLaunchAccessAuthority{}
	request.EgressAuthority = ports.AgentLaunchEgressAuthority{}
	request.ExecutionWorkspaceRef = ports.ExecutionWorkspaceRef{}
	request.RequierePreservacionEntorno = false
	fixture := &dockerAdapterFixture{t: t, request: request, binding: validDockerPhysicalBinding(request)}
	fixture.journal = newDockerJournalStub(&fixture.events)
	fixture.client = &dockerClientStub{
		t: t, request: request, journal: fixture.journal, events: &fixture.events,
		capabilities: validDockerRemoteCapabilities(),
	}
	fixture.store = &credentialSignerStoreStub{
		material: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{11}, ed25519.SeedSize)),
	}
	fixture.signer = mustCredentialSigner(t, fixture.store)
	fixture.adapter = fixture.mustAdapter(t, fixture.journal)
	return fixture
}

func lostDockerLaunchFixture(t *testing.T) *dockerAdapterFixture {
	t.Helper()
	fixture := newDockerAdapterFixture(t)
	fixture.client.launchErr = errors.New("lost launch acknowledgement")
	if _, err := fixture.adapter.Launch(context.Background(), fixture.request); ErrorCode(err) != CodeLaunchUnavailable {
		t.Fatalf("Launch() = %v", err)
	}
	fixture.client.launchErr = nil
	return fixture
}

func (fixture *dockerAdapterFixture) mustAdapter(t *testing.T, journal ports.AgentProviderRequestJournal) *DockerAdapter {
	t.Helper()
	executorRef, err := microvm.ConstruirEjecutorRefPerfilV1(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	capabilities := validAdapterCapabilities()
	capabilities.ProviderRef = DockerProviderRef
	capabilities.ModelRef = "model:codex-docker"
	capabilities.AgentRef = "agent:codex-docker"
	capabilities.RequierePreservacionEntorno = false
	adapter, err := NewDockerAdapter(DockerConfig{
		Client: fixture.client, Signer: fixture.signer, Journal: journal,
		Capabilities: capabilities, PlacementRef: fixture.binding.PlacementRef,
		ProviderModel: "gpt-5.6", PromptRenderer: staticSessionPromptRenderer("execute exact work"),
		ImageRef: fixture.binding.ImageRef, ExecutorRef: executorRef,
		VCPU: fixture.binding.VCPU, MemoryMiB: fixture.binding.MemoryMiB, MaxPIDs: fixture.binding.MaxPIDs,
	})
	if err != nil {
		t.Fatalf("NewDockerAdapter() = %v", err)
	}
	return adapter
}

type dockerJournalStub struct {
	records             map[ports.AgentProviderRequestKey]ports.AgentProviderRequest
	events              *[]string
	recordErrors        map[ports.AgentProviderRequestStage]error
	bindErr, resolveErr error
	corruptResultStage  ports.AgentProviderRequestStage
	afterRecord         func(ports.AgentProviderRequestStage)
	resolveHook         func()
}

func newDockerJournalStub(events *[]string) *dockerJournalStub {
	return &dockerJournalStub{
		records: make(map[ports.AgentProviderRequestKey]ports.AgentProviderRequest),
		events:  events, recordErrors: make(map[ports.AgentProviderRequestStage]error),
	}
}

func (journal *dockerJournalStub) RecordAgentProviderRequest(_ context.Context, request ports.AgentProviderRequest) (ports.AgentProviderRequest, error) {
	*journal.events = append(*journal.events, "record:"+string(request.Key.Stage))
	if err := journal.recordErrors[request.Key.Stage]; err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if ports.ValidatePreparedAgentProviderRequest(request) != nil {
		return ports.AgentProviderRequest{}, errors.New("invalid prepared request")
	}
	if existing, found := journal.records[request.Key]; found {
		comparable := ports.CloneAgentProviderRequest(existing)
		comparable.LaunchBindingRef, comparable.LaunchBindingRevision = "", 0
		if !reflect.DeepEqual(comparable, request) {
			return ports.AgentProviderRequest{}, errors.New("divergent replay")
		}
		return ports.CloneAgentProviderRequest(existing), nil
	}
	if request.Key.Stage != ports.AgentProviderRequestLaunch {
		launchKey := request.Key
		launchKey.Stage = ports.AgentProviderRequestLaunch
		launch, found := journal.records[launchKey]
		if !found || launch.LaunchBindingRef != request.TargetRef ||
			launch.LaunchBindingRevision != request.ExpectedRevision {
			return ports.AgentProviderRequest{}, errors.New("missing launch binding")
		}
	}
	if request.Key.Stage == ports.AgentProviderRequestSessionInput {
		startKey := request.Key
		startKey.Stage = ports.AgentProviderRequestSessionStart
		if _, found := journal.records[startKey]; !found {
			return ports.AgentProviderRequest{}, errors.New("missing session start")
		}
	}
	journal.records[request.Key] = ports.CloneAgentProviderRequest(request)
	if journal.afterRecord != nil {
		journal.afterRecord(request.Key.Stage)
	}
	result := ports.CloneAgentProviderRequest(request)
	if request.Key.Stage == journal.corruptResultStage {
		result.LaunchBindingRef, result.LaunchBindingRevision = "ejecucion:crossed", 1
	}
	return result, nil
}

func (journal *dockerJournalStub) BindAgentProviderLaunch(_ context.Context, key ports.AgentProviderRequestKey, target string, revision uint64) (ports.AgentProviderRequest, error) {
	*journal.events = append(*journal.events, "bind:"+string(key.Stage))
	if journal.bindErr != nil {
		return ports.AgentProviderRequest{}, journal.bindErr
	}
	request, found := journal.records[key]
	if !found || key.Stage != ports.AgentProviderRequestLaunch {
		return ports.AgentProviderRequest{}, errors.New("launch missing")
	}
	if request.LaunchBindingRef != "" &&
		(request.LaunchBindingRef != target || request.LaunchBindingRevision != revision) {
		return ports.AgentProviderRequest{}, errors.New("binding conflict")
	}
	request.LaunchBindingRef, request.LaunchBindingRevision = target, revision
	journal.records[key] = ports.CloneAgentProviderRequest(request)
	return ports.CloneAgentProviderRequest(request), nil
}

func (journal *dockerJournalStub) ResolveAgentProviderRequest(_ context.Context, key ports.AgentProviderRequestKey) (ports.AgentProviderRequest, bool, error) {
	if journal.resolveHook != nil {
		journal.resolveHook()
	}
	*journal.events = append(*journal.events, "resolve:"+string(key.Stage))
	if journal.resolveErr != nil {
		return ports.AgentProviderRequest{}, false, journal.resolveErr
	}
	request, found := journal.records[key]
	return ports.CloneAgentProviderRequest(request), found, nil
}

func (journal *dockerJournalStub) mustRecord(t *testing.T, request ports.AgentLaunchRequest, stage ports.AgentProviderRequestStage) ports.AgentProviderRequest {
	t.Helper()
	stored, found := journal.records[dockerJournalKey(request, stage)]
	if !found || ports.ValidateAgentProviderRequest(stored) != nil {
		t.Fatalf("missing or invalid %s record: %+v", stage, stored)
	}
	return ports.CloneAgentProviderRequest(stored)
}

func (journal *dockerJournalStub) reopen() *dockerJournalStub {
	reopened := newDockerJournalStub(journal.events)
	for key, request := range journal.records {
		reopened.records[key] = ports.CloneAgentProviderRequest(request)
	}
	return reopened
}

type dockerClientStub struct {
	t                                                             *testing.T
	request                                                       ports.AgentLaunchRequest
	journal                                                       *dockerJournalStub
	events                                                        *[]string
	capabilities                                                  microvm.RespuestaCapacidades
	capabilityErr, launchErr, queryErr, startErr, inputErr        error
	mutateLaunch                                                  func(*microvm.RespuestaContenedorV1)
	mutateQuery                                                   func(*microvm.RespuestaOperacionContenedorV1)
	mutateStart                                                   func(*microvm.RespuestaInicioSesionContenedorV1)
	mutateInput                                                   func(*microvm.RespuestaEntradaSesionContenedorV1)
	negotiations, launchCalls, queryCalls, startCalls, inputCalls int
	launchBodies, startBodies, inputBodies                        [][]byte
}

func (client *dockerClientStub) NegociarDocker(context.Context) (microvm.RespuestaCapacidades, error) {
	*client.events = append(*client.events, "negotiate")
	client.negotiations++
	return client.capabilities, client.capabilityErr
}

func (client *dockerClientStub) LanzarORecuperarContenedorCodificada(_ context.Context, key string, body json.RawMessage) (microvm.RespuestaContenedorV1, error) {
	client.assertRecorded(ports.AgentProviderRequestLaunch, key, "", 0, body)
	*client.events = append(*client.events, "effect:launch")
	client.launchCalls++
	client.launchBodies = append(client.launchBodies, append([]byte(nil), body...))
	signed := decodeDockerSigned(client.t, body)
	response := validDockerContainerResponse(signed.Plan, true)
	if client.mutateLaunch != nil {
		client.mutateLaunch(&response)
	}
	if client.launchErr != nil {
		return microvm.RespuestaContenedorV1{}, client.launchErr
	}
	return response, nil
}

func (client *dockerClientStub) ConsultarOperacionContenedor(_ context.Context, key string) (microvm.RespuestaOperacionContenedorV1, error) {
	*client.events = append(*client.events, "query:launch")
	client.queryCalls++
	if client.queryErr != nil {
		return microvm.RespuestaOperacionContenedorV1{}, client.queryErr
	}
	if len(client.launchBodies) == 0 {
		client.t.Fatal("query without recorded launch body")
	}
	signed := decodeDockerSigned(client.t, client.launchBodies[0])
	if key != signed.Plan.OperacionRef {
		client.t.Fatalf("query key=%q plan=%q", key, signed.Plan.OperacionRef)
	}
	planSHA, err := microvm.CalcularSHA256PlanLanzamientoContenedorV1(signed.Plan)
	if err != nil {
		client.t.Fatal(err)
	}
	response := microvm.RespuestaOperacionContenedorV1{
		OperacionRef: key, PlanSHA256: planSHA, ConcesionRef: dockerGrantRef(client.t, signed.Concesion),
		Contenedor: validDockerContainerResponse(signed.Plan, false),
	}
	if client.mutateQuery != nil {
		client.mutateQuery(&response)
	}
	return response, nil
}

func (client *dockerClientStub) IniciarSesionContenedorCodificada(_ context.Context, key, target string, body json.RawMessage) (microvm.RespuestaInicioSesionContenedorV1, error) {
	client.assertRecorded(ports.AgentProviderRequestSessionStart, key, target, 3, body)
	*client.events = append(*client.events, "effect:session_start")
	client.startCalls++
	client.startBodies = append(client.startBodies, append([]byte(nil), body...))
	var request microvm.SolicitudInicioSesionContenedorV1
	if json.Unmarshal(body, &request) != nil {
		client.t.Fatal("invalid start body")
	}
	response := microvm.RespuestaInicioSesionContenedorV1{
		Referencia: target, SesionRef: request.SesionRef, Cerca: request.Cerca, Iniciada: true,
	}
	if client.mutateStart != nil {
		client.mutateStart(&response)
	}
	if client.startErr != nil {
		return microvm.RespuestaInicioSesionContenedorV1{}, client.startErr
	}
	return response, nil
}

func (client *dockerClientStub) EnviarEntradaSesionContenedorCodificada(_ context.Context, key, target, session string, body json.RawMessage) (microvm.RespuestaEntradaSesionContenedorV1, error) {
	client.assertRecorded(ports.AgentProviderRequestSessionInput, key, target, 3, body)
	*client.events = append(*client.events, "effect:session_input")
	client.inputCalls++
	client.inputBodies = append(client.inputBodies, append([]byte(nil), body...))
	response := microvm.RespuestaEntradaSesionContenedorV1{
		Referencia: target, SesionRef: session, EntradaRef: key,
		Cerca: client.request.EffectAuthority.ActionFence, Aceptada: true,
	}
	if client.mutateInput != nil {
		client.mutateInput(&response)
	}
	if client.inputErr != nil {
		return microvm.RespuestaEntradaSesionContenedorV1{}, client.inputErr
	}
	return response, nil
}

func (client *dockerClientStub) assertRecorded(stage ports.AgentProviderRequestStage, key, target string, revision uint64, body []byte) {
	client.t.Helper()
	stored := client.journal.mustRecord(client.t, client.request, stage)
	if stored.IdempotencyKey != key || stored.TargetRef != target || stored.ExpectedRevision != revision ||
		!bytes.Equal(stored.Body, body) {
		client.t.Fatalf("effect %s crossed journal: stored=%+v key=%q target=%q revision=%d",
			stage, stored, key, target, revision)
	}
}

func validDockerRemoteCapabilities() microvm.RespuestaCapacidades {
	return microvm.RespuestaCapacidades{
		Protocolo: microvm.ProtocoloLocal, Version: "0.1.0",
		BackendEjecuciones: microvm.BackendEjecucionesDocker,
		DockerConfigurado:  true, DockerEngineDisponible: true, MaximoEjecuciones: 4,
	}
}

func decodeDockerSigned(t *testing.T, body []byte) microvm.SolicitudLanzarORecuperarContenedorV1 {
	t.Helper()
	var signed microvm.SolicitudLanzarORecuperarContenedorV1
	if json.Unmarshal(body, &signed) != nil {
		t.Fatal("invalid signed Docker body")
	}
	canonical, err := microvm.CodificarSolicitudLanzarORecuperarContenedorV1(signed)
	if err != nil || !bytes.Equal(canonical, body) {
		t.Fatalf("non-canonical signed Docker body: %v", err)
	}
	return signed
}

func dockerGrantRef(t *testing.T, grant []byte) string {
	t.Helper()
	var decoded struct {
		Content struct {
			Ref string `json:"concesion_ref"`
		} `json:"contenido"`
	}
	if json.Unmarshal(grant, &decoded) != nil || decoded.Content.Ref == "" {
		t.Fatal("invalid Docker grant")
	}
	return decoded.Content.Ref
}

func validDockerContainerResponse(plan microvm.PlanLanzamientoContenedorV1, withLiveness bool) microvm.RespuestaContenedorV1 {
	response := microvm.RespuestaContenedorV1{
		Referencia: plan.EjecucionRef, Estado: "activo", Revision: 3, Cerca: plan.Cerca,
		VCPU: plan.VCPU, MemoriaMiB: plan.MemoriaMiB,
		Identidad: &microvm.IdentidadContenedorV1{
			ContainerID: strings.Repeat("1", 64), PIDObservado: 1234, OwnerLabel: "v1",
			ExecutionLabel: plan.EjecucionRef, RunLabel: plan.RunRef, Generation: plan.Cerca,
		},
	}
	if withLiveness {
		alive, state := true, "running"
		response.RecursoVivo, response.EstadoMotor = &alive, &state
	}
	return response
}
