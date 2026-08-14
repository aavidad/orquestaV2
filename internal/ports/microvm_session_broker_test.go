package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestMicroVMSessionOpenConsumesChallengeOnceAndReplaysOnlyExactRequest(t *testing.T) {
	request := validMicroVMOpen(t)
	if err := ValidateMicroVMSessionOpenRequest(request); err != nil {
		t.Fatalf("valid open rejected: %v", err)
	}
	broker := newContractMicroVMBroker()
	first, err := broker.Open(context.Background(), request)
	if err != nil || ValidateMicroVMSessionOpenReceipt(request, first) != nil {
		t.Fatalf("first open = %+v, err = %v", first, err)
	}
	replay, err := broker.Open(context.Background(), request)
	if err != nil || !replay.Replayed || replay.ReceiptRef != first.ReceiptRef || !replay.OpenedAt.Equal(first.OpenedAt) {
		t.Fatalf("exact retry did not replay receipt: %+v, err = %v", replay, err)
	}
	for name, mutate := range map[string]func(*MicroVMSessionOpenRequest){
		"image": func(v *MicroVMSessionOpenRequest) { v.GuestImageDigest = digestOf("9") },
		"workspace": func(v *MicroVMSessionOpenRequest) {
			v.ExecutionWorkspaceRef, _ = NewExecutionWorkspaceRef("workspace:microvm:other")
		},
		"egress": func(v *MicroVMSessionOpenRequest) { v.EgressAuthority = testMicroVMEgressAuthority("other") },
	} {
		divergent := request
		mutate(&divergent)
		if _, err = broker.Open(context.Background(), divergent); MicroVMSessionContractErrorCode(err) != "microvm_session.challenge_consumed" {
			t.Fatalf("divergent %s challenge reuse error = %v", name, err)
		}
	}
}

func TestMicroVMSessionOpenAcceptsCanonicalOpaqueRefsWithoutSecondGrammar(t *testing.T) {
	request := validMicroVMOpen(t)
	request.Session.ProjectRef, _ = goal.NewProjectRef("project:tenant/acme")
	request.Session.GoalRef, _ = goal.NewGoalRef("goal:team/acme")
	request.Session.WorkItemRef, _ = goal.NewWorkItemRef("work:tree/one")
	request.Session.ExecutionRef, _ = goal.NewExecutionRef("execution:host/run")
	request.AttestationRef, _ = goal.NewAttestationRef("attestation:policy/boot")
	request.EffectAuthority.AuthorizationReceiptRef = "authorization:ledger/one"
	request.EffectAuthority.EffectApprovalRef = "approval:ledger/one"
	request.EffectAuthority.EffectAttemptRef = "attempt:ledger/one"
	if err := ValidateMicroVMSessionOpenRequest(request); err != nil {
		t.Fatalf("canonical opaque refs rejected by a second grammar: %v", err)
	}
}

func TestMicroVMSessionOpenRejectsInvalidCausalityAuthorityDigestAndWindow(t *testing.T) {
	base := validMicroVMOpen(t)
	tests := map[string]func(*MicroVMSessionOpenRequest){
		"project":   func(v *MicroVMSessionOpenRequest) { v.Session.ProjectRef = goal.ProjectRef{} },
		"goal":      func(v *MicroVMSessionOpenRequest) { v.Session.GoalRef = goal.GoalRef{} },
		"work item": func(v *MicroVMSessionOpenRequest) { v.Session.WorkItemRef = goal.WorkItemRef{} },
		"execution": func(v *MicroVMSessionOpenRequest) { v.Session.ExecutionRef = goal.ExecutionRef{} },
		"attempt":   func(v *MicroVMSessionOpenRequest) { v.Session.ExecutionAttempt = 0 },
		"replacement": func(v *MicroVMSessionOpenRequest) {
			v.Session.ReplacesExecutionRef, _ = goal.NewExecutionRef("execution:old")
		},
		"spec":    func(v *MicroVMSessionOpenRequest) { v.Session.SpecHash = "bad" },
		"session": func(v *MicroVMSessionOpenRequest) { v.SessionRef = "execution-session:bad/path" },
		"workspace": func(v *MicroVMSessionOpenRequest) {
			v.ExecutionWorkspaceRef = ExecutionWorkspaceRef{value: " workspace:invalid"}
		},
		"access":            func(v *MicroVMSessionOpenRequest) { v.AccessAuthority.MCPAccessRef = "" },
		"egress":            func(v *MicroVMSessionOpenRequest) { v.EgressAuthority.PayloadSHA256 = digestOf("9") },
		"effect":            func(v *MicroVMSessionOpenRequest) { v.EffectAuthority.ActionFence = 0 },
		"attestation":       func(v *MicroVMSessionOpenRequest) { v.AttestationRef = goal.AttestationRef{} },
		"digest":            func(v *MicroVMSessionOpenRequest) { v.GuestImageDigest = "bad" },
		"challenge":         func(v *MicroVMSessionOpenRequest) { v.ChallengeRef = MicroVMSessionRef{} },
		"before effect":     func(v *MicroVMSessionOpenRequest) { v.RequestedAt = v.EffectAuthority.StartedAt.Add(-time.Nanosecond) },
		"at lease":          func(v *MicroVMSessionOpenRequest) { v.RequestedAt = v.EffectAuthority.ClaimLeaseUntil },
		"expired challenge": func(v *MicroVMSessionOpenRequest) { v.ChallengeExpiresAt = v.RequestedAt },
		"challenge after lease": func(v *MicroVMSessionOpenRequest) {
			v.ChallengeExpiresAt = v.EffectAuthority.ClaimLeaseUntil.Add(time.Nanosecond)
		},
		"challenge after approval": func(v *MicroVMSessionOpenRequest) {
			v.ChallengeExpiresAt = v.EffectAuthority.ApprovalExpiresAt.Add(time.Nanosecond)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if MicroVMSessionContractErrorCode(ValidateMicroVMSessionOpenRequest(candidate)) == "" {
				t.Fatalf("invalid open accepted: %+v", candidate)
			}
		})
	}
}

func TestMicroVMSessionOpenDigestBindsEveryAuthorityFamily(t *testing.T) {
	base := validMicroVMOpen(t)
	want := MicroVMSessionOpenDigest(base)
	tests := map[string]func(*MicroVMSessionOpenRequest){
		"causality": func(v *MicroVMSessionOpenRequest) { v.Session.PlanGeneration++ },
		"session":   func(v *MicroVMSessionOpenRequest) { v.SessionRef = mustExecutionSessionRef("9") },
		"workspace": func(v *MicroVMSessionOpenRequest) {
			v.ExecutionWorkspaceRef, _ = NewExecutionWorkspaceRef("workspace:microvm:other")
		},
		"access": func(v *MicroVMSessionOpenRequest) {
			v.AccessAuthority.MCPAccessRef, _ = NewExecutionMCPAccessRef("mcp-access:execution:sha256:" + digestOf("9"))
		},
		"egress": func(v *MicroVMSessionOpenRequest) {
			v.EgressAuthority = testMicroVMEgressAuthority("other")
		},
		"effect":      func(v *MicroVMSessionOpenRequest) { v.EffectAuthority.ActionFence++ },
		"attestation": func(v *MicroVMSessionOpenRequest) { v.AttestationSubjectDigest = digestOf("9") },
		"assets":      func(v *MicroVMSessionOpenRequest) { v.GuestKernelDigest = digestOf("9") },
		"challenge":   func(v *MicroVMSessionOpenRequest) { v.ChallengeExpiresAt = v.ChallengeExpiresAt.Add(time.Nanosecond) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if MicroVMSessionOpenDigest(candidate) == want {
				t.Fatalf("digest unchanged after %s mutation", name)
			}
		})
	}
}

func TestMicroVMSessionOpenPreservesExactAbsenceOfOptionalIsolationAuthorities(t *testing.T) {
	request := validMicroVMOpen(t)
	withAuthorities := MicroVMSessionOpenDigest(request)
	request.ExecutionWorkspaceRef = ExecutionWorkspaceRef{}
	request.EgressAuthority = AgentLaunchEgressAuthority{}
	if err := ValidateMicroVMSessionOpenRequest(request); err != nil {
		t.Fatalf("optional authority absence rejected: %v", err)
	}
	if MicroVMSessionOpenDigest(request) == withAuthorities {
		t.Fatal("optional authority absence did not alter the sealed open request")
	}
}

func TestMicroVMSessionExchangeUsesOnlyAllowlistedOpaqueAuthorities(t *testing.T) {
	open := validMicroVMOpen(t)
	opened := validMicroVMOpened(t, open)
	cases := []struct {
		op  MicroVMSessionOperation
		ref string
	}{
		{MicroVMSessionArtifactRead, open.AccessAuthority.ArtifactAccessRef.String()},
		{MicroVMSessionArtifactPublish, open.AccessAuthority.ArtifactAccessRef.String()},
		{MicroVMSessionMCPInvoke, open.AccessAuthority.MCPAccessRef.String()},
		{MicroVMSessionMailboxReceive, open.AccessAuthority.MailboxEndpointRef.String()},
		{MicroVMSessionMailboxSend, open.AccessAuthority.MailboxEndpointRef.String()},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.op), func(t *testing.T) {
			request := validMicroVMExchange(t, open, opened, testCase.op)
			request.RequestedAt = open.EffectAuthority.ClaimLeaseUntil.Add(time.Hour)
			if request.AuthorityRef != testCase.ref {
				t.Fatalf("authority = %q, want %q", request.AuthorityRef, testCase.ref)
			}
			if err := ValidateMicroVMSessionRequest(open, opened, request); err != nil {
				t.Fatalf("session request after launch lease rejected: %v", err)
			}
			receipt := validMicroVMExchangeReceipt(t, request)
			if err := ValidateMicroVMSessionRequestReceipt(open, opened, request, receipt); err != nil {
				t.Fatalf("valid exchange receipt rejected: %v", err)
			}
		})
	}
	unknown := validMicroVMExchange(t, open, opened, MicroVMSessionArtifactRead)
	unknown.Operation = "shell_execute"
	if code := MicroVMSessionContractErrorCode(ValidateMicroVMSessionRequest(open, opened, unknown)); code != "microvm_session.operation_not_allowed" {
		t.Fatalf("unknown operation error = %q", code)
	}
	wrong := validMicroVMExchange(t, open, opened, MicroVMSessionArtifactRead)
	wrong.AuthorityRef = open.AccessAuthority.MCPAccessRef.String()
	if code := MicroVMSessionContractErrorCode(ValidateMicroVMSessionRequest(open, opened, wrong)); code != "microvm_session.request_authority_mismatch" {
		t.Fatalf("wrong authority error = %q", code)
	}
}

func TestMicroVMSessionExchangeAndReceiptRejectCausalMutation(t *testing.T) {
	open := validMicroVMOpen(t)
	opened := validMicroVMOpened(t, open)
	base := validMicroVMExchange(t, open, opened, MicroVMSessionMCPInvoke)
	requests := map[string]func(*MicroVMSessionRequest){
		"session":      func(v *MicroVMSessionRequest) { v.SessionRef = mustExecutionSessionRef("9") },
		"open receipt": func(v *MicroVMSessionRequest) { v.OpenReceiptRef = MicroVMSessionRef{} },
		"request ref":  func(v *MicroVMSessionRequest) { v.RequestRef = MicroVMSessionRef{} },
		"resource":     func(v *MicroVMSessionRequest) { v.ResourceRef = MicroVMSessionRef{} },
		"digest":       func(v *MicroVMSessionRequest) { v.RequestDigest = "bad" },
		"before open":  func(v *MicroVMSessionRequest) { v.RequestedAt = opened.OpenedAt.Add(-time.Nanosecond) },
	}
	for name, mutate := range requests {
		t.Run("request "+name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if MicroVMSessionContractErrorCode(ValidateMicroVMSessionRequest(open, opened, candidate)) == "" {
				t.Fatalf("invalid request accepted: %+v", candidate)
			}
		})
	}
	for name, mutate := range map[string]func(*MicroVMSessionRequestReceipt){
		"operation": func(v *MicroVMSessionRequestReceipt) { v.Operation = MicroVMSessionMailboxSend },
		"digest":    func(v *MicroVMSessionRequestReceipt) { v.RequestBindingDigest = digestOf("9") },
	} {
		receipt := validMicroVMExchangeReceipt(t, base)
		mutate(&receipt)
		if MicroVMSessionContractErrorCode(ValidateMicroVMSessionRequestReceipt(open, opened, base, receipt)) == "" {
			t.Fatalf("receipt mutation %s accepted", name)
		}
	}
}

func TestMicroVMSessionExchangeRequestRefReplaysExactAndRejectsEveryDivergence(t *testing.T) {
	broker := newContractMicroVMBroker()
	open := validMicroVMOpen(t)
	opened, err := broker.Open(context.Background(), open)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	request := validMicroVMExchange(t, open, opened, MicroVMSessionArtifactRead)
	first, err := broker.Exchange(context.Background(), request)
	if err != nil || ValidateMicroVMSessionRequestReceipt(open, opened, request, first) != nil {
		t.Fatalf("first exchange: %v", err)
	}
	replay, err := broker.Exchange(context.Background(), request)
	if err != nil || !replay.Replayed || replay.ReceiptRef != first.ReceiptRef ||
		replay.ResultRef != first.ResultRef || !replay.CompletedAt.Equal(first.CompletedAt) {
		t.Fatalf("exact exchange retry did not replay stable receipt: %+v err=%v", replay, err)
	}
	for name, mutate := range map[string]func(*MicroVMSessionRequest){
		"operation": func(v *MicroVMSessionRequest) {
			v.Operation = MicroVMSessionMCPInvoke
			v.AuthorityRef = open.AccessAuthority.MCPAccessRef.String()
		},
		"resource":  func(v *MicroVMSessionRequest) { v.ResourceRef = mustMicroVMRef(t, "resource:other") },
		"digest":    func(v *MicroVMSessionRequest) { v.RequestDigest = digestOf("9") },
		"authority": func(v *MicroVMSessionRequest) { v.AuthorityRef = "authority:other" },
	} {
		candidate := request
		mutate(&candidate)
		if _, err := broker.Exchange(context.Background(), candidate); MicroVMSessionContractErrorCode(err) != "microvm_session.request_ref_conflict" {
			t.Fatalf("divergent %s reuse error = %v", name, err)
		}
	}

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 16)
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			got, callErr := broker.Exchange(context.Background(), request)
			if callErr != nil || !got.Replayed || got.ReceiptRef != first.ReceiptRef {
				errorsSeen <- callErr
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for callErr := range errorsSeen {
		t.Fatalf("concurrent exact replay failed: %v", callErr)
	}
}

func TestMicroVMSessionCloseAndRevokeRemainAvailableAfterLaunchLease(t *testing.T) {
	open := validMicroVMOpen(t)
	opened := validMicroVMOpened(t, open)
	for _, mode := range []MicroVMSessionEndMode{MicroVMSessionClose, MicroVMSessionRevoke} {
		request := MicroVMSessionEndRequest{
			SessionRef: open.SessionRef, OpenReceiptRef: opened.ReceiptRef, RequestRef: mustMicroVMRef(t, "end:"+string(mode)),
			Mode: mode, RequestedAt: open.EffectAuthority.ClaimLeaseUntil.Add(time.Hour),
		}
		if err := ValidateMicroVMSessionEndRequest(open, opened, request); err != nil {
			t.Fatalf("%s after launch lease rejected: %v", mode, err)
		}
		receipt := MicroVMSessionEndReceipt{SessionRef: request.SessionRef, RequestRef: request.RequestRef,
			Mode: mode, RequestBindingDigest: MicroVMSessionEndBindingDigest(request),
			ReceiptRef: digestRef(t, "receipt:sha256:", "8"), EndedAt: request.RequestedAt}
		if err := ValidateMicroVMSessionEndReceipt(open, opened, request, receipt); err != nil {
			t.Fatalf("valid %s receipt rejected: %v", mode, err)
		}
	}
}

func TestMicroVMSessionEndRequestRefReplaysExactAndRejectsDivergentMode(t *testing.T) {
	broker := newContractMicroVMBroker()
	open := validMicroVMOpen(t)
	opened, err := broker.Open(context.Background(), open)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	request := MicroVMSessionEndRequest{SessionRef: open.SessionRef, OpenReceiptRef: opened.ReceiptRef,
		RequestRef: mustMicroVMRef(t, "end:once"), Mode: MicroVMSessionClose,
		RequestedAt: open.EffectAuthority.ClaimLeaseUntil.Add(time.Hour)}
	first, err := broker.End(context.Background(), request)
	if err != nil || ValidateMicroVMSessionEndReceipt(open, opened, request, first) != nil {
		t.Fatalf("first end: %v", err)
	}
	replay, err := broker.End(context.Background(), request)
	if err != nil || !replay.Replayed || replay.ReceiptRef != first.ReceiptRef || !replay.EndedAt.Equal(first.EndedAt) {
		t.Fatalf("exact end retry did not replay stable receipt: %+v err=%v", replay, err)
	}
	request.Mode = MicroVMSessionRevoke
	if _, err = broker.End(context.Background(), request); MicroVMSessionContractErrorCode(err) != "microvm_session.request_ref_conflict" {
		t.Fatalf("divergent end mode error = %v", err)
	}
}

func TestMicroVMSessionRequestRefHasOneWinnerAcrossExchangeAndEnd(t *testing.T) {
	broker := newContractMicroVMBroker()
	open := validMicroVMOpen(t)
	opened, err := broker.Open(context.Background(), open)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sharedRef := mustMicroVMRef(t, "request:shared-race")
	exchange := validMicroVMExchange(t, open, opened, MicroVMSessionArtifactRead)
	exchange.RequestRef = sharedRef
	end := MicroVMSessionEndRequest{SessionRef: open.SessionRef, OpenReceiptRef: opened.ReceiptRef,
		RequestRef: sharedRef, Mode: MicroVMSessionClose, RequestedAt: opened.OpenedAt.Add(time.Second)}

	type outcome struct {
		kind string
		err  error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	go func() {
		<-start
		_, callErr := broker.Exchange(context.Background(), exchange)
		results <- outcome{kind: "exchange", err: callErr}
	}()
	go func() {
		<-start
		_, callErr := broker.End(context.Background(), end)
		results <- outcome{kind: "end", err: callErr}
	}()
	close(start)

	winners, conflicts := 0, 0
	for index := 0; index < 2; index++ {
		result := <-results
		switch code := MicroVMSessionContractErrorCode(result.err); {
		case result.err == nil:
			winners++
		case code == "microvm_session.request_ref_conflict":
			conflicts++
		default:
			t.Fatalf("%s returned unexpected error: %v", result.kind, result.err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("cross-operation race winners=%d conflicts=%d", winners, conflicts)
	}
}

func TestMicroVMSessionRequestRefsAreIndependentAcrossSessions(t *testing.T) {
	broker := newContractMicroVMBroker()
	openA := validMicroVMOpen(t)
	openedA, err := broker.Open(context.Background(), openA)
	if err != nil {
		t.Fatalf("open A: %v", err)
	}
	openB := validMicroVMOpen(t)
	openB.SessionRef = mustExecutionSessionRef("f")
	openB.Session.ExecutionRef, _ = goal.NewExecutionRef("execution:microvm:second")
	openB.ChallengeRef = digestRef(t, "challenge:sha256:", "f")
	openB.EffectAuthority.EffectAttemptRef = "attempt:microvm:2"
	openedB, err := broker.Open(context.Background(), openB)
	if err != nil {
		t.Fatalf("open B: %v", err)
	}

	requestA := validMicroVMExchange(t, openA, openedA, MicroVMSessionArtifactRead)
	requestB := validMicroVMExchange(t, openB, openedB, MicroVMSessionArtifactRead)
	if requestA.RequestRef != requestB.RequestRef {
		t.Fatal("fixture must exercise the same request ref in both sessions")
	}
	if _, err = broker.Exchange(context.Background(), requestA); err != nil {
		t.Fatalf("session A after opening B: %v", err)
	}
	if _, err = broker.Exchange(context.Background(), requestB); err != nil {
		t.Fatalf("same request ref in independent session B: %v", err)
	}
}

func TestMicroVMSessionRequestRefRejectsCrossKindInBothSequentialOrders(t *testing.T) {
	for _, firstKind := range []string{"exchange", "end"} {
		t.Run(firstKind+" first", func(t *testing.T) {
			broker := newContractMicroVMBroker()
			open := validMicroVMOpen(t)
			opened, err := broker.Open(context.Background(), open)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			sharedRef := mustMicroVMRef(t, "request:shared-sequential")
			exchange := validMicroVMExchange(t, open, opened, MicroVMSessionArtifactRead)
			exchange.RequestRef = sharedRef
			end := MicroVMSessionEndRequest{SessionRef: open.SessionRef, OpenReceiptRef: opened.ReceiptRef,
				RequestRef: sharedRef, Mode: MicroVMSessionClose, RequestedAt: opened.OpenedAt.Add(time.Second)}

			if firstKind == "exchange" {
				if _, err = broker.Exchange(context.Background(), exchange); err != nil {
					t.Fatalf("first exchange: %v", err)
				}
				_, err = broker.End(context.Background(), end)
			} else {
				if _, err = broker.End(context.Background(), end); err != nil {
					t.Fatalf("first end: %v", err)
				}
				_, err = broker.Exchange(context.Background(), exchange)
			}
			if MicroVMSessionContractErrorCode(err) != "microvm_session.request_ref_conflict" {
				t.Fatalf("second cross-kind request error = %v", err)
			}
		})
	}
}

func TestMicroVMSessionDTOsRejectPhysicalLocatorsAndRawSecretSurfaces(t *testing.T) {
	for _, value := range []string{"/tmp/ref", "../ref", "https://host/ref", "unix:///run/x.sock",
		"ref:user@example.test", "ref:value?token=x", "ref:line\nbreak", "ref:" + strings.Repeat("a", microVMSessionMaxRefBytes)} {
		if _, err := NewMicroVMSessionRef(value); err == nil {
			t.Fatalf("unsafe ref accepted: %q", value)
		}
	}
	for _, sample := range []any{MicroVMSessionOpenRequest{}, MicroVMSessionOpenReceipt{}, MicroVMSessionRequest{},
		MicroVMSessionRequestReceipt{}, MicroVMSessionEndRequest{}, MicroVMSessionEndReceipt{}} {
		typeOf := reflect.TypeOf(sample)
		for index := 0; index < typeOf.NumField(); index++ {
			field := typeOf.Field(index)
			name := strings.ToLower(field.Name)
			for _, forbidden := range []string{"endpoint", "path", "socket", "cid", "token", "secret", "credential", "payload", "content", "command"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("%s leaks field %s", typeOf.Name(), field.Name)
				}
			}
			if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
				t.Fatalf("%s exposes raw bytes in %s", typeOf.Name(), field.Name)
			}
		}
	}
}

func validMicroVMOpen(t *testing.T) MicroVMSessionOpenRequest {
	t.Helper()
	project, _ := goal.NewProjectRef("project:microvm")
	goalRef, _ := goal.NewGoalRef("goal:microvm")
	work, _ := goal.NewWorkItemRef("work:microvm")
	execution, _ := goal.NewExecutionRef("execution:microvm")
	artifact, _ := NewExecutionArtifactAccessRef("artifact-access:execution:sha256:" + digestOf("2"))
	mcp, _ := NewExecutionMCPAccessRef("mcp-access:execution:sha256:" + digestOf("3"))
	mailbox, _ := NewExecutionMailboxEndpointRef("mailbox-endpoint:execution:sha256:" + digestOf("4"))
	workspace, _ := NewExecutionWorkspaceRef("workspace:microvm:one")
	attestation, _ := goal.NewAttestationRef("attestation:microvm:boot")
	started := time.Unix(1_700_000_000, 0).UTC()
	return MicroVMSessionOpenRequest{
		Session: ExecutionSessionEnsureRequest{ProjectRef: project, GoalRef: goalRef, WorkItemRef: work,
			ExecutionRef: execution, ExecutionAttempt: 1, PlanGeneration: 2, AppSpecGeneration: 3, SpecHash: digestOf("a")},
		SessionRef:            mustExecutionSessionRef("1"),
		ExecutionWorkspaceRef: workspace,
		AccessAuthority:       AgentLaunchAccessAuthority{ArtifactAccessRef: artifact, MCPAccessRef: mcp, MailboxEndpointRef: mailbox},
		EgressAuthority:       testMicroVMEgressAuthority("one"),
		EffectAuthority: AgentLaunchEffectAuthority{AuthorizationReceiptRef: "authorization:microvm:1",
			EffectApprovalRef: "approval:microvm:1", EffectAttemptRef: "attempt:microvm:1", ActionFence: 7,
			StartedAt: started, ClaimLeaseUntil: started.Add(10 * time.Minute), ApprovalExpiresAt: started.Add(8 * time.Minute)},
		AttestationRef: attestation, AttestationSubjectDigest: digestOf("b"), GuestImageDigest: digestOf("c"),
		GuestKernelDigest: digestOf("d"), EffectiveConfigDigest: digestOf("e"),
		ChallengeRef: digestRef(t, "challenge:sha256:", "5"), ChallengeExpiresAt: started.Add(3 * time.Minute),
		RequestedAt: started.Add(time.Minute),
	}
}

func testMicroVMEgressAuthority(suffix string) AgentLaunchEgressAuthority {
	payload := []byte(`{"schema":"orquesta.egress-policy.v1","authority":"` + suffix + `"}`)
	digest := sha256.Sum256(payload)
	return AgentLaunchEgressAuthority{PolicyRef: "egress-policy:microvm:" + suffix,
		PayloadSHA256: hex.EncodeToString(digest[:]), CanonicalPayload: payload}
}

func validMicroVMOpened(t *testing.T, request MicroVMSessionOpenRequest) MicroVMSessionOpenReceipt {
	t.Helper()
	return MicroVMSessionOpenReceipt{SessionRef: request.SessionRef, OpenDigest: MicroVMSessionOpenDigest(request),
		ChallengeRef: request.ChallengeRef, ReceiptRef: digestRef(t, "receipt:sha256:", "6"),
		OpenedAt: request.RequestedAt.Add(time.Second)}
}

func validMicroVMExchange(t *testing.T, open MicroVMSessionOpenRequest, opened MicroVMSessionOpenReceipt, operation MicroVMSessionOperation) MicroVMSessionRequest {
	t.Helper()
	authority, _ := microVMSessionAuthority(open.AccessAuthority, operation)
	return MicroVMSessionRequest{SessionRef: open.SessionRef, OpenReceiptRef: opened.ReceiptRef,
		RequestRef: mustMicroVMRef(t, "request:one"), Operation: operation, AuthorityRef: authority,
		ResourceRef: mustMicroVMRef(t, "resource:one"), RequestDigest: digestOf("7"),
		RequestedAt: opened.OpenedAt.Add(time.Second)}
}

func validMicroVMExchangeReceipt(t *testing.T, request MicroVMSessionRequest) MicroVMSessionRequestReceipt {
	t.Helper()
	return MicroVMSessionRequestReceipt{SessionRef: request.SessionRef, RequestRef: request.RequestRef,
		Operation: request.Operation, RequestBindingDigest: MicroVMSessionRequestBindingDigest(request),
		ResultRef: mustMicroVMRef(t, "result:one"), ResultDigest: digestOf("8"),
		ReceiptRef: digestRef(t, "receipt:sha256:", "9"), CompletedAt: request.RequestedAt.Add(time.Second)}
}

func mustExecutionSessionRef(character string) ExecutionSessionRef {
	ref, _ := NewExecutionSessionRef("execution-session:sha256:" + digestOf(character))
	return ref
}

func mustMicroVMRef(t *testing.T, value string) MicroVMSessionRef {
	t.Helper()
	ref, err := NewMicroVMSessionRef(value)
	if err != nil {
		t.Fatalf("microVM ref %q: %v", value, err)
	}
	return ref
}

func digestRef(t *testing.T, prefix, character string) MicroVMSessionRef {
	return mustMicroVMRef(t, prefix+digestOf(character))
}

func digestOf(character string) string { return strings.Repeat(character, 64) }

type contractMicroVMOpen struct {
	digest  string
	receipt MicroVMSessionOpenReceipt
}

type contractMicroVMBroker struct {
	mu       sync.Mutex
	opened   map[string]contractMicroVMOpen
	sessions map[string]contractMicroVMSession
	requests map[string]contractMicroVMRequest
}

type contractMicroVMSession struct {
	open    MicroVMSessionOpenRequest
	receipt MicroVMSessionOpenReceipt
}

type contractMicroVMRequestKind string

const (
	contractMicroVMExchangeKind contractMicroVMRequestKind = "exchange"
	contractMicroVMEndKind      contractMicroVMRequestKind = "end"
)

type contractMicroVMRequest struct {
	kind     contractMicroVMRequestKind
	digest   string
	exchange MicroVMSessionRequestReceipt
	end      MicroVMSessionEndReceipt
}

func newContractMicroVMBroker() *contractMicroVMBroker {
	return &contractMicroVMBroker{opened: make(map[string]contractMicroVMOpen),
		sessions: make(map[string]contractMicroVMSession),
		requests: make(map[string]contractMicroVMRequest)}
}

var _ MicroVMSessionBroker = (*contractMicroVMBroker)(nil)

func (broker *contractMicroVMBroker) Open(_ context.Context, request MicroVMSessionOpenRequest) (MicroVMSessionOpenReceipt, error) {
	if err := ValidateMicroVMSessionOpenRequest(request); err != nil {
		return MicroVMSessionOpenReceipt{}, err
	}
	broker.mu.Lock()
	defer broker.mu.Unlock()
	digest := MicroVMSessionOpenDigest(request)
	if previous, exists := broker.opened[request.ChallengeRef.String()]; exists {
		if previous.digest != digest {
			return MicroVMSessionOpenReceipt{}, microVMSessionError("challenge_consumed")
		}
		receipt := previous.receipt
		receipt.Replayed = true
		return receipt, nil
	}
	receipt := MicroVMSessionOpenReceipt{SessionRef: request.SessionRef, OpenDigest: digest,
		ChallengeRef: request.ChallengeRef, ReceiptRef: mustContractMicroVMRef("receipt:sha256:" + digestOf("6")),
		OpenedAt: request.RequestedAt.Add(time.Second)}
	if previous, exists := broker.sessions[request.SessionRef.String()]; exists &&
		previous.receipt.OpenDigest != digest {
		return MicroVMSessionOpenReceipt{}, microVMSessionError("session_ref_conflict")
	}
	broker.opened[request.ChallengeRef.String()] = contractMicroVMOpen{digest: digest, receipt: receipt}
	broker.sessions[request.SessionRef.String()] = contractMicroVMSession{open: request, receipt: receipt}
	return receipt, nil
}

func (broker *contractMicroVMBroker) Exchange(_ context.Context, request MicroVMSessionRequest) (MicroVMSessionRequestReceipt, error) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	digest := MicroVMSessionRequestBindingDigest(request)
	key := contractMicroVMRequestKey(request.SessionRef, request.RequestRef)
	if previous, exists := broker.requests[key]; exists {
		if previous.kind != contractMicroVMExchangeKind || previous.digest != digest {
			return MicroVMSessionRequestReceipt{}, microVMSessionError("request_ref_conflict")
		}
		receipt := previous.exchange
		receipt.Replayed = true
		return receipt, nil
	}
	session, exists := broker.sessions[request.SessionRef.String()]
	if !exists {
		return MicroVMSessionRequestReceipt{}, microVMSessionError("request_session_mismatch")
	}
	if err := ValidateMicroVMSessionRequest(session.open, session.receipt, request); err != nil {
		return MicroVMSessionRequestReceipt{}, err
	}
	receipt := MicroVMSessionRequestReceipt{SessionRef: request.SessionRef, RequestRef: request.RequestRef,
		Operation: request.Operation, RequestBindingDigest: digest,
		ResultRef: mustContractMicroVMRef("result:one"), ResultDigest: digestOf("8"),
		ReceiptRef: mustContractMicroVMRef("receipt:sha256:" + digest), CompletedAt: request.RequestedAt.Add(time.Second)}
	broker.requests[key] = contractMicroVMRequest{kind: contractMicroVMExchangeKind, digest: digest, exchange: receipt}
	return receipt, nil
}

func (broker *contractMicroVMBroker) End(_ context.Context, request MicroVMSessionEndRequest) (MicroVMSessionEndReceipt, error) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	digest := MicroVMSessionEndBindingDigest(request)
	key := contractMicroVMRequestKey(request.SessionRef, request.RequestRef)
	if previous, exists := broker.requests[key]; exists {
		if previous.kind != contractMicroVMEndKind || previous.digest != digest {
			return MicroVMSessionEndReceipt{}, microVMSessionError("request_ref_conflict")
		}
		receipt := previous.end
		receipt.Replayed = true
		return receipt, nil
	}
	session, exists := broker.sessions[request.SessionRef.String()]
	if !exists {
		return MicroVMSessionEndReceipt{}, microVMSessionError("end_request_invalid")
	}
	if err := ValidateMicroVMSessionEndRequest(session.open, session.receipt, request); err != nil {
		return MicroVMSessionEndReceipt{}, err
	}
	receipt := MicroVMSessionEndReceipt{SessionRef: request.SessionRef, RequestRef: request.RequestRef,
		Mode: request.Mode, RequestBindingDigest: digest,
		ReceiptRef: mustContractMicroVMRef("receipt:sha256:" + digest), EndedAt: request.RequestedAt}
	broker.requests[key] = contractMicroVMRequest{kind: contractMicroVMEndKind, digest: digest, end: receipt}
	return receipt, nil
}

func contractMicroVMRequestKey(sessionRef ExecutionSessionRef, requestRef MicroVMSessionRef) string {
	return sessionRef.String() + "\x00" + requestRef.String()
}

func mustContractMicroVMRef(value string) MicroVMSessionRef {
	ref, _ := NewMicroVMSessionRef(value)
	return ref
}
