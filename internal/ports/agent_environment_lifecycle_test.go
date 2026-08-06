package ports

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func validAgentEnvironmentSubject(t *testing.T) AgentEnvironmentLifecycleSubject {
	t.Helper()
	launch := validAgentLaunchReceipt(validAgentLaunchRequest(t))
	return AgentEnvironmentLifecycleSubject{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef,
	}
}

func validAgentEnvironmentToken(t *testing.T, state AgentEnvironmentLifecycleState, revision string) AgentEnvironmentLifecycleToken {
	t.Helper()
	physical, err := NewAgentPhysicalToken("external:1")
	if err != nil {
		t.Fatal(err)
	}
	physicalRevision, err := NewAgentPhysicalRevision(revision)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := NewAgentPhysicalFence("fence:launch-7")
	if err != nil {
		t.Fatal(err)
	}
	return AgentEnvironmentLifecycleToken{PhysicalToken: physical, Revision: physicalRevision, Fence: fence, State: state}
}

func validAgentPreservationManifest(t *testing.T) AgentPhysicalPreservationManifest {
	t.Helper()
	content := []byte(`{"schema":"agent-preservation.v1","artifacts":[]}`)
	digest := sha256.Sum256(content)
	revision, err := NewAgentPhysicalRevision("work-revision:9")
	if err != nil {
		t.Fatal(err)
	}
	return AgentPhysicalPreservationManifest{Ref: "physical-content:manifest-1", SHA256: fmt.Sprintf("%x", digest),
		Content: content, ContentBytes: uint64(len(content)), WorkRevision: revision,
		Causality: AgentPhysicalPreservationCausality{
			PlanSHA256: strings.Repeat("a", 64), GrantSHA256: strings.Repeat("b", 64),
			KernelSHA256: strings.Repeat("c", 64), InitramfsSHA256: strings.Repeat("d", 64),
			ProfileSHA256: strings.Repeat("e", 64),
		}, SealedAt: time.Unix(29, 0).UTC()}
}

func TestAgentEnvironmentOpaqueTokensAreTypedAndClosed(t *testing.T) {
	constructors := map[string]func(string) error{
		"token":    func(value string) error { _, err := NewAgentPhysicalToken(value); return err },
		"revision": func(value string) error { _, err := NewAgentPhysicalRevision(value); return err },
		"fence":    func(value string) error { _, err := NewAgentPhysicalFence(value); return err },
	}
	for name, constructor := range constructors {
		t.Run(name, func(t *testing.T) {
			if err := constructor(name + ":opaque-1"); err != nil {
				t.Fatal(err)
			}
			for mutation, value := range map[string]string{
				"empty": "", "leading space": " " + name, "trailing space": name + " ",
				"nul": name + "\x00bad", "invalid utf8": string([]byte{0xff}),
				"oversize": strings.Repeat("a", maxAgentEnvironmentOpaqueValueBytes+1),
			} {
				t.Run(mutation, func(t *testing.T) {
					if err := constructor(value); AgentContractErrorCode(err) == "" {
						t.Fatalf("invalid opaque value accepted: %q", value)
					}
				})
			}
		})
	}
}

func TestAgentQuiesceContractBindsExactSuccessfulLifecycle(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	request := AgentQuiesceRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentActive, "revision:1"),
		IdempotencyKey: "quiesce:execution-1"}
	receipt := AgentQuiesceReceipt{Subject: subject, PreviousToken: request.ExpectedToken,
		NextToken: validAgentEnvironmentToken(t, AgentEnvironmentQuiesced, "revision:2"), IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:quiesce:execution-1", ConfirmedAt: time.Unix(20, 0).UTC()}
	if err := ValidateAgentQuiesceReceipt(request, receipt); err != nil {
		t.Fatal(err)
	}
	pending := receipt
	pending.NextToken.State = AgentEnvironmentQuiescing
	pending.ReceiptRef = ""
	pending.ConfirmedAt = time.Time{}
	if err := ValidateAgentQuiesceReceipt(request, pending); err != nil {
		t.Fatalf("pending quiesce rejected: %v", err)
	}

	otherGoal, _ := goal.NewGoalRef("goal:other")
	mutations := map[string]func(*AgentQuiesceReceipt){
		"subject":       func(value *AgentQuiesceReceipt) { value.Subject.GoalRef = otherGoal },
		"previous":      func(value *AgentQuiesceReceipt) { value.PreviousToken.Revision.value = "revision:other" },
		"same revision": func(value *AgentQuiesceReceipt) { value.NextToken.Revision = value.PreviousToken.Revision },
		"changed fence": func(value *AgentQuiesceReceipt) { value.NextToken.Fence.value = "fence:other" },
		"wrong state":   func(value *AgentQuiesceReceipt) { value.NextToken.State = AgentEnvironmentPreserving },
		"idempotency":   func(value *AgentQuiesceReceipt) { value.IdempotencyKey = "quiesce:other" },
		"receipt":       func(value *AgentQuiesceReceipt) { value.ReceiptRef = "" },
		"confirmed at":  func(value *AgentQuiesceReceipt) { value.ConfirmedAt = time.Time{} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := receipt
			mutate(&changed)
			if AgentContractErrorCode(ValidateAgentQuiesceReceipt(request, changed)) == "" {
				t.Fatalf("mutated quiesce receipt accepted: %+v", changed)
			}
		})
	}
}

func TestAgentEnvironmentInspectionFeedsTheFirstOpaqueToken(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	request := AgentEnvironmentInspectRequest{Subject: subject}
	receipt := AgentEnvironmentInspectReceipt{Subject: subject,
		Token: validAgentEnvironmentToken(t, AgentEnvironmentActive, "revision:1")}
	if err := ValidateAgentEnvironmentInspectReceipt(request, receipt); err != nil {
		t.Fatal(err)
	}
	receipt.Token.PhysicalToken.value = "external:other"
	if AgentContractErrorCode(ValidateAgentEnvironmentInspectReceipt(request, receipt)) == "" {
		t.Fatal("inspection accepted a physical token for another execution")
	}
}

func TestAgentPhysicalRevisionHasNoNumericOrderAndReceiptReplayIsExact(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	request := AgentQuiesceRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentActive, "revision:z"),
		IdempotencyKey: "quiesce:opaque-revision"}
	receipt := AgentQuiesceReceipt{Subject: subject, PreviousToken: request.ExpectedToken,
		NextToken:      validAgentEnvironmentToken(t, AgentEnvironmentQuiesced, "revision:a"),
		IdempotencyKey: request.IdempotencyKey, ReceiptRef: "receipt:quiesce:opaque-revision",
		ConfirmedAt: time.Unix(21, 0).UTC()}
	for replay := 0; replay < 2; replay++ {
		if err := ValidateAgentQuiesceReceipt(request, receipt); err != nil {
			t.Fatalf("opaque revision/replay %d rejected: %v", replay, err)
		}
	}
}

func TestAgentEnvironmentSubjectAndRequestMutantsFailClosed(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	base := AgentPreserveRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentQuiesced, "revision:2"),
		IdempotencyKey: "preserve:execution-1"}
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	otherGoal, _ := goal.NewGoalRef("goal:other")
	otherWork, _ := goal.NewWorkItemRef("work:other")
	mutations := map[string]func(*AgentPreserveRequest){
		"execution":   func(value *AgentPreserveRequest) { value.Subject.ExecutionRef = otherExecution },
		"goal":        func(value *AgentPreserveRequest) { value.Subject.GoalRef = otherGoal },
		"work item":   func(value *AgentPreserveRequest) { value.Subject.WorkItemRef = otherWork },
		"plan":        func(value *AgentPreserveRequest) { value.Subject.PlanGeneration = 0 },
		"app spec":    func(value *AgentPreserveRequest) { value.Subject.AppSpecGeneration = 0 },
		"attempt":     func(value *AgentPreserveRequest) { value.Subject.ExecutionAttempt = 0 },
		"spec hash":   func(value *AgentPreserveRequest) { value.Subject.SpecHash = "bad" },
		"provider":    func(value *AgentPreserveRequest) { value.Subject.ProviderRef = "" },
		"model":       func(value *AgentPreserveRequest) { value.Subject.ModelRef = "" },
		"agent":       func(value *AgentPreserveRequest) { value.Subject.AgentRef = "" },
		"external":    func(value *AgentPreserveRequest) { value.Subject.ExternalRef = "external:other" },
		"token":       func(value *AgentPreserveRequest) { value.ExpectedToken.PhysicalToken.value = "external:other" },
		"revision":    func(value *AgentPreserveRequest) { value.ExpectedToken.Revision = AgentPhysicalRevision{} },
		"fence":       func(value *AgentPreserveRequest) { value.ExpectedToken.Fence = AgentPhysicalFence{} },
		"state":       func(value *AgentPreserveRequest) { value.ExpectedToken.State = AgentEnvironmentActive },
		"idempotency": func(value *AgentPreserveRequest) { value.IdempotencyKey = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			launch := validAgentLaunchReceipt(validAgentLaunchRequest(t))
			requestErr := ValidateAgentPreserveRequest(request)
			targetErr := ValidateAgentEnvironmentLifecycleTarget(launch, request.Subject)
			if AgentContractErrorCode(requestErr) == "" && AgentContractErrorCode(targetErr) == "" {
				t.Fatalf("mutated preserve request accepted: %+v", request)
			}
		})
	}
}

func TestAgentPreserveReceiptRequiresExactSealedManifestOnlyWhenComplete(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	request := AgentPreserveRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentQuiesced, "revision:2"),
		IdempotencyKey: "preserve:execution-1"}
	pending := AgentPreserveReceipt{Subject: subject, PreviousToken: request.ExpectedToken,
		NextToken: validAgentEnvironmentToken(t, AgentEnvironmentPreserving, "revision:3"), IdempotencyKey: request.IdempotencyKey}
	if err := ValidateAgentPreserveReceipt(request, pending); err != nil {
		t.Fatal(err)
	}
	complete := pending
	complete.NextToken = validAgentEnvironmentToken(t, AgentEnvironmentPreserved, "revision:4")
	complete.Manifest = validAgentPreservationManifest(t)
	complete.ReceiptRef = "receipt:preserve:execution-1"
	complete.ConfirmedAt = time.Unix(30, 0).UTC()
	if err := ValidateAgentPreserveReceipt(request, complete); err != nil {
		t.Fatal(err)
	}
	withoutProfile := complete
	withoutProfile.Manifest.Causality.ProfileSHA256 = ""
	if err := ValidateAgentPreserveReceipt(request, withoutProfile); err != nil {
		t.Fatalf("optional physical profile rejected: %v", err)
	}

	for name, mutate := range map[string]func(*AgentPreserveReceipt){
		"pending manifest": func(value *AgentPreserveReceipt) { value.NextToken.State = AgentEnvironmentPreserving },
		"manifest ref":     func(value *AgentPreserveReceipt) { value.Manifest.Ref = "" },
		"manifest digest":  func(value *AgentPreserveReceipt) { value.Manifest.SHA256 = strings.Repeat("b", 64) },
		"manifest content": func(value *AgentPreserveReceipt) { value.Manifest.Content[0] ^= 1 },
		"manifest bytes":   func(value *AgentPreserveReceipt) { value.Manifest.ContentBytes++ },
		"work revision":    func(value *AgentPreserveReceipt) { value.Manifest.WorkRevision = AgentPhysicalRevision{} },
		"causal plan":      func(value *AgentPreserveReceipt) { value.Manifest.Causality.PlanSHA256 = "" },
		"causal grant":     func(value *AgentPreserveReceipt) { value.Manifest.Causality.GrantSHA256 = "" },
		"causal kernel":    func(value *AgentPreserveReceipt) { value.Manifest.Causality.KernelSHA256 = "" },
		"causal initramfs": func(value *AgentPreserveReceipt) { value.Manifest.Causality.InitramfsSHA256 = "" },
		"causal profile":   func(value *AgentPreserveReceipt) { value.Manifest.Causality.ProfileSHA256 = "bad" },
		"sealed at":        func(value *AgentPreserveReceipt) { value.Manifest.SealedAt = time.Time{} },
		"timestamp order":  func(value *AgentPreserveReceipt) { value.ConfirmedAt = value.Manifest.SealedAt.Add(-time.Nanosecond) },
		"receipt":          func(value *AgentPreserveReceipt) { value.ReceiptRef = "" },
		"confirmed at":     func(value *AgentPreserveReceipt) { value.ConfirmedAt = time.Time{} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := complete
			changed.Manifest.Content = append([]byte(nil), complete.Manifest.Content...)
			mutate(&changed)
			if AgentContractErrorCode(ValidateAgentPreserveReceipt(request, changed)) == "" {
				t.Fatalf("mutated manifest accepted: %+v", changed.Manifest)
			}
		})
	}

}

func TestAgentCloseBindsPreservedManifestAndNeverMeansControlStop(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	manifest := validAgentPreservationManifest(t)
	request := AgentCloseRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentPreserved, "revision:4"),
		Preservation:   AgentPhysicalPreservationBinding{ManifestRef: manifest.Ref, ManifestSHA256: manifest.SHA256},
		IdempotencyKey: "close:execution-1"}
	receipt := AgentCloseReceipt{Subject: subject, PreviousToken: request.ExpectedToken,
		NextToken:    validAgentEnvironmentToken(t, AgentEnvironmentClosed, "revision:5"),
		Preservation: request.Preservation, IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:close:execution-1", ConfirmedAt: time.Unix(40, 0).UTC()}
	if err := ValidateAgentCloseReceipt(request, receipt); err != nil {
		t.Fatal(err)
	}
	pending := receipt
	pending.NextToken.State = AgentEnvironmentClosing
	pending.ReceiptRef = ""
	pending.ConfirmedAt = time.Time{}
	if err := ValidateAgentCloseReceipt(request, pending); err != nil {
		t.Fatalf("pending close rejected: %v", err)
	}

	for name, mutate := range map[string]func(*AgentCloseReceipt){
		"manifest ref":    func(value *AgentCloseReceipt) { value.Preservation.ManifestRef = "physical-content:other" },
		"manifest digest": func(value *AgentCloseReceipt) { value.Preservation.ManifestSHA256 = strings.Repeat("b", 64) },
		"state":           func(value *AgentCloseReceipt) { value.NextToken.State = AgentEnvironmentQuiesced },
		"receipt":         func(value *AgentCloseReceipt) { value.ReceiptRef = "" },
		"confirmed at":    func(value *AgentCloseReceipt) { value.ConfirmedAt = time.Time{} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := receipt
			mutate(&changed)
			if AgentContractErrorCode(ValidateAgentCloseReceipt(request, changed)) == "" {
				t.Fatalf("mutated close receipt accepted: %+v", changed)
			}
		})
	}

	if reflect.TypeOf(AgentCloseRequest{}).AssignableTo(reflect.TypeOf(AgentStopRequest{})) ||
		reflect.TypeOf(AgentCloseReceipt{}).AssignableTo(reflect.TypeOf(AgentStopReceipt{})) {
		t.Fatal("successful close aliases the control/cancellation stop contract")
	}
}

func TestPendingPhysicalLifecycleReceiptCannotInventConfirmation(t *testing.T) {
	subject := validAgentEnvironmentSubject(t)
	request := AgentPreserveRequest{Subject: subject,
		ExpectedToken:  validAgentEnvironmentToken(t, AgentEnvironmentQuiesced, "revision:2"),
		IdempotencyKey: "preserve:pending-confirmation"}
	base := AgentPreserveReceipt{Subject: subject, PreviousToken: request.ExpectedToken,
		NextToken: validAgentEnvironmentToken(t, AgentEnvironmentPreserving, "revision:3"), IdempotencyKey: request.IdempotencyKey}
	for name, mutate := range map[string]func(*AgentPreserveReceipt){
		"ref":  func(value *AgentPreserveReceipt) { value.ReceiptRef = "receipt:invented" },
		"time": func(value *AgentPreserveReceipt) { value.ConfirmedAt = time.Unix(30, 0).UTC() },
	} {
		t.Run(name, func(t *testing.T) {
			pending := base
			mutate(&pending)
			if code := AgentContractErrorCode(ValidateAgentPreserveReceipt(request, pending)); code != "agent.environment_lifecycle_pending_has_confirmation" {
				t.Fatalf("invented pending confirmation code = %q", code)
			}
		})
	}
}
