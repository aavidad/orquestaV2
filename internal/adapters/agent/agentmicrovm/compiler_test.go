package agentmicrovm

import (
	"crypto/ed25519"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestCompileBindsAuthorizedContextAndPhysicalPlan(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)

	compiled, err := Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	wantContext := microvm.ContextoAutorizado{
		ProyectoRef:          request.ProjectRef.String(),
		GoalRef:              request.GoalRef.String(),
		WorkItemRef:          request.WorkItemRef.String(),
		EjecucionRef:         request.ExecutionRef.String(),
		AutorizacionRef:      request.EffectAuthority.AuthorizationReceiptRef,
		AprobacionEfectoRef:  request.EffectAuthority.EffectApprovalRef,
		IntentoEfectoRef:     request.EffectAuthority.EffectAttemptRef,
		SesionRef:            request.SessionRef.String(),
		AccesoArtefactosRef:  request.AccessAuthority.ArtifactAccessRef.String(),
		AccesoMCPRef:         request.AccessAuthority.MCPAccessRef.String(),
		EndpointBuzonRef:     request.AccessAuthority.MailboxEndpointRef.String(),
		DescriptorPerfilRef:  descriptor.DescriptorRef,
		EspecificacionSHA256: request.SpecHash,
	}
	if compiled.Context != wantContext {
		t.Fatalf("context = %+v, want %+v", compiled.Context, wantContext)
	}
	if !compiled.IssuedAt.Equal(request.EffectAuthority.StartedAt) || compiled.Validity != 10*time.Second {
		t.Fatalf("grant window = %s/%s", compiled.IssuedAt, compiled.Validity)
	}
	if compiled.Plan.Esquema != microvm.EsquemaPlanLanzamiento ||
		compiled.Plan.PlanRef != "plan:"+request.SpecHash ||
		compiled.Plan.RunRef != request.ExecutionRef.String() ||
		compiled.Plan.Cerca != request.EffectAuthority.ActionFence ||
		compiled.Plan.VCPU != descriptor.VCPU || compiled.Plan.MemoriaMiB != descriptor.MemoriaMiB ||
		compiled.Plan.KernelSHA256 != descriptor.KernelSHA256 ||
		compiled.Plan.InitramfsSHA256 != descriptor.InitramfsSHA256 ||
		compiled.Plan.PerfilSHA256 == nil || *compiled.Plan.PerfilSHA256 != descriptor.PerfilSHA256 ||
		compiled.Plan.LimiteTiempoMS != 90_000 || compiled.Plan.LimiteRAMPicoBytes != 512*bytesPerMiB ||
		compiled.Plan.LimiteDiscoPicoBytes != 8_192 || compiled.Plan.LimiteTokensAgente != 4_096 ||
		!reflect.DeepEqual(compiled.Plan.Servicios, descriptor.ServiciosDisponibles) {
		t.Fatalf("plan = %+v", compiled.Plan)
	}
	if err := microvm.ValidarPlanConDescriptorPerfilLanzamientoV1(descriptor, compiled.Plan); err != nil {
		t.Fatalf("public plan validation = %v", err)
	}

	// Preparar is the public connector's only exported validation of the full
	// context. Signing here proves that no private approximation accepted it.
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	signer, err := microvm.NuevoFirmanteConcesiones("clave-publica:test", privateKey)
	if err != nil {
		t.Fatalf("NuevoFirmanteConcesiones() = %v", err)
	}
	if _, err := signer.Preparar(compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity); err != nil {
		t.Fatalf("public context validation = %v", err)
	}

	compiled.Plan.Servicios[0].Puerto++
	if descriptor.ServiciosDisponibles[0].Puerto == compiled.Plan.Servicios[0].Puerto {
		t.Fatal("compiled services alias descriptor storage")
	}
}

func TestCompileMapsEveryContextBinding(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	compiled := mustCompile(t, request, descriptor)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"project", compiled.Context.ProyectoRef, request.ProjectRef.String()},
		{"goal", compiled.Context.GoalRef, request.GoalRef.String()},
		{"work item", compiled.Context.WorkItemRef, request.WorkItemRef.String()},
		{"execution", compiled.Context.EjecucionRef, request.ExecutionRef.String()},
		{"authorization", compiled.Context.AutorizacionRef, request.EffectAuthority.AuthorizationReceiptRef},
		{"approval", compiled.Context.AprobacionEfectoRef, request.EffectAuthority.EffectApprovalRef},
		{"attempt", compiled.Context.IntentoEfectoRef, request.EffectAuthority.EffectAttemptRef},
		{"session", compiled.Context.SesionRef, request.SessionRef.String()},
		{"artifacts", compiled.Context.AccesoArtefactosRef, request.AccessAuthority.ArtifactAccessRef.String()},
		{"mcp", compiled.Context.AccesoMCPRef, request.AccessAuthority.MCPAccessRef.String()},
		{"mailbox", compiled.Context.EndpointBuzonRef, request.AccessAuthority.MailboxEndpointRef.String()},
		{"descriptor", compiled.Context.DescriptorPerfilRef, descriptor.DescriptorRef},
		{"spec", compiled.Context.EspecificacionSHA256, request.SpecHash},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("binding = %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestCompileSourceFingerprintIsDeterministicAndBindsPromptFacts(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	first := mustCompile(t, request, descriptor)
	second := mustCompile(t, request, descriptor)
	if first.sourceFingerprint == ([32]byte{}) || first.sourceFingerprint != second.sourceFingerprint {
		t.Fatalf("source fingerprint is zero or nondeterministic: %x/%x", first.sourceFingerprint, second.sourceFingerprint)
	}

	mutations := []struct {
		name   string
		mutate func(*ports.AgentLaunchRequest)
	}{
		{"objective", func(value *ports.AgentLaunchRequest) { value.Objective = "other objective" }},
		{"generation", func(value *ports.AgentLaunchRequest) { value.PlanGeneration++ }},
		{"role", func(value *ports.AgentLaunchRequest) { value.RoleKey = "role:other" }},
		{"write set", func(value *ports.AgentLaunchRequest) { value.WriteSet = []string{"other/path"} }},
		{"output", func(value *ports.AgentLaunchRequest) { value.ArtifactMediaType = "text/plain" }},
		{"effort", func(value *ports.AgentLaunchRequest) { value.ReasoningEffort = governance.ReasoningEffortHigh }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changedRequest := request
			mutation.mutate(&changedRequest)
			changed := mustCompile(t, changedRequest, descriptor)
			if changed.sourceFingerprint == first.sourceFingerprint {
				t.Fatal("source mutation preserved fingerprint")
			}
		})
	}
}

func TestCompileRejectsMissingAuthorityFenceAndLimits(t *testing.T) {
	validRequest := validLaunchRequest(t)
	validProfile := validDescriptor(t, false)
	tests := []struct {
		name   string
		mutate func(*ports.AgentLaunchRequest, *uint64)
		code   string
	}{
		{"session", func(r *ports.AgentLaunchRequest, _ *uint64) { r.SessionRef = "" }, CodeSessionRequired},
		{"access authority", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.AccessAuthority.MailboxEndpointRef = ""
		}, CodeAccessAuthorityRequired},
		{"request", func(r *ports.AgentLaunchRequest, _ *uint64) { r.Objective = "" }, CodeLaunchRequestInvalid},
		{"context control character", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.ProjectRef, _ = goal.NewProjectRef("project:microvm\nother")
		}, CodeContextInvalid},
		{"effect authority", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.EffectAuthority.AuthorizationReceiptRef = ""
		}, CodeEffectAuthorityInvalid},
		{"zero fence", func(_ *ports.AgentLaunchRequest, fence *uint64) { *fence = 0 }, CodeFenceInvalid},
		{"stale fence", func(_ *ports.AgentLaunchRequest, fence *uint64) { (*fence)++ }, CodeFenceMismatch},
		{"time zero", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.BudgetDemand.Resources.ActiveTimeNS = 0
		}, CodeLimitsInvalid},
		{"time below millisecond", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.BudgetDemand.Resources.ActiveTimeNS = int64(time.Millisecond) - 1
		}, CodeLimitsInvalid},
		{"tokens zero", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.BudgetDemand.Resources.Tokens = 0
		}, CodeLimitsInvalid},
		{"disk zero", func(r *ports.AgentLaunchRequest, _ *uint64) {
			r.BudgetDemand.Resources.DiskBytes = 0
		}, CodeLimitsInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validRequest
			fence := request.EffectAuthority.ActionFence
			test.mutate(&request, &fence)
			_, err := Compile(request, profileBinding(request, validProfile), fence)
			if code := ErrorCode(err); code != test.code {
				t.Fatalf("ErrorCode() = %q, want %q; err=%v", code, test.code, err)
			}
		})
	}
}

func TestCompileBindsPlacementAndDerivesBoundedGrantWindow(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	otherPlacement, _ := ports.NewAgentPlacementRef("placement:other")
	_, err := Compile(request, ProfileBinding{PlacementRef: otherPlacement, Descriptor: descriptor}, request.EffectAuthority.ActionFence)
	if code := ErrorCode(err); code != CodeProfileBindingInvalid {
		t.Fatalf("wrong placement code = %q; err=%v", code, err)
	}

	request.EffectAuthority.ClaimLeaseUntil = request.EffectAuthority.StartedAt.Add(20 * time.Minute)
	request.EffectAuthority.ApprovalExpiresAt = request.EffectAuthority.StartedAt.Add(15 * time.Minute)
	compiled, err := Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if err != nil || !compiled.IssuedAt.Equal(request.EffectAuthority.StartedAt) ||
		compiled.Validity != microvm.VigenciaMaximaConcesion {
		t.Fatalf("bounded grant = %+v; err=%v", compiled, err)
	}

	request.EffectAuthority.StartedAt = time.Unix(-1, 0).UTC()
	request.EffectAuthority.ClaimLeaseUntil = request.EffectAuthority.StartedAt.Add(time.Minute)
	request.EffectAuthority.ApprovalExpiresAt = request.EffectAuthority.StartedAt.Add(time.Minute)
	_, err = Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if code := ErrorCode(err); code != CodeGrantWindowInvalid {
		t.Fatalf("pre-epoch grant code = %q; err=%v", code, err)
	}

	request.EffectAuthority.StartedAt = time.Unix(10, 0).UTC()
	request.EffectAuthority.ClaimLeaseUntil = request.EffectAuthority.StartedAt.Add(500 * time.Microsecond)
	request.EffectAuthority.ApprovalExpiresAt = request.EffectAuthority.ClaimLeaseUntil
	_, err = Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if code := ErrorCode(err); code != CodeGrantWindowInvalid {
		t.Fatalf("sub-millisecond grant code = %q; err=%v", code, err)
	}
}

func TestCompileRejectsDescriptorMutationAndUnauthorizedEgress(t *testing.T) {
	request := validLaunchRequest(t)
	validProfile := validDescriptor(t, false)
	tests := []struct {
		name       string
		descriptor func() microvm.DescriptorPerfilLanzamientoV1
		code       string
	}{
		{"digest mutation", func() microvm.DescriptorPerfilLanzamientoV1 {
			descriptor := validProfile
			descriptor.KernelSHA256 = strings.Repeat("9", 64)
			return descriptor
		}, CodeDescriptorInvalid},
		{"invalid service", func() microvm.DescriptorPerfilLanzamientoV1 {
			descriptor := validProfile
			descriptor.ServiciosDisponibles = append([]microvm.ServicioVsock(nil), descriptor.ServiciosDisponibles...)
			descriptor.ServiciosDisponibles[0].Puerto = 0
			return descriptor
		}, CodeDescriptorInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Compile(request, profileBinding(request, test.descriptor()), request.EffectAuthority.ActionFence)
			if code := ErrorCode(err); code != test.code {
				t.Fatalf("ErrorCode() = %q, want %q; err=%v", code, test.code, err)
			}
		})
	}
}

func TestCompileUsesOnlyControlBrokerUntilEgressIsAuthorized(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, true)
	compiled, err := Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if err != nil {
		t.Fatalf("Compile() with available proxy = %v", err)
	}
	if len(compiled.Plan.Servicios) != 1 || compiled.Plan.Servicios[0].Papel != "control_broker" ||
		compiled.Plan.Egreso != nil {
		t.Fatalf("unauthorized egress entered plan: %+v", compiled.Plan)
	}

	descriptor.ServiciosDisponibles = descriptor.ServiciosDisponibles[1:]
	descriptor = sealDescriptor(t, descriptor)
	_, err = Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if code := ErrorCode(err); code != CodeControlBrokerRequired {
		t.Fatalf("missing broker code = %q; err=%v", code, err)
	}
}

func TestCheckedMultiplyRejectsOverflow(t *testing.T) {
	if _, ok := checkedMultiply(math.MaxUint64, 2); ok {
		t.Fatal("overflow accepted")
	}
	if got, ok := checkedMultiply(512, bytesPerMiB); !ok || got != 512*bytesPerMiB {
		t.Fatalf("checkedMultiply() = %d/%v", got, ok)
	}
}

func mustCompile(t *testing.T, request ports.AgentLaunchRequest, descriptor microvm.DescriptorPerfilLanzamientoV1) Compilation {
	t.Helper()
	compiled, err := Compile(request, profileBinding(request, descriptor), request.EffectAuthority.ActionFence)
	if err != nil {
		t.Fatalf("Compile() = %v", err)
	}
	return compiled
}

func profileBinding(
	request ports.AgentLaunchRequest,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) ProfileBinding {
	return ProfileBinding{PlacementRef: request.ReferenciaColocacion, Descriptor: descriptor}
}

func validLaunchRequest(t *testing.T) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:microvm-1")
	goalRef, _ := goal.NewGoalRef("goal:microvm-1")
	workItemRef, _ := goal.NewWorkItemRef("work:microvm-1")
	actorRef, _ := goal.NewActorRef("actor:owner")
	projectRef, _ := goal.NewProjectRef("project:microvm")
	placementRef, _ := ports.NewAgentPlacementRef("placement:microvm-1")
	sessionRef, _ := ports.NewExecutionSessionRef("execution-session:microvm-1")
	artifactsRef, _ := ports.NewExecutionArtifactAccessRef("artifact-access:execution:sha256:" + strings.Repeat("a", 64))
	mcpRef, _ := ports.NewExecutionMCPAccessRef("mcp-access:execution:sha256:" + strings.Repeat("b", 64))
	mailboxRef, _ := ports.NewExecutionMailboxEndpointRef("mailbox-endpoint:execution:sha256:" + strings.Repeat("c", 64))
	return ports.AgentLaunchRequest{
		ExecutionRef:         executionRef,
		ReferenciaColocacion: placementRef,
		SessionRef:           sessionRef,
		AccessAuthority: ports.AgentLaunchAccessAuthority{
			ArtifactAccessRef: artifactsRef, MCPAccessRef: mcpRef, MailboxEndpointRef: mailboxRef,
		},
		GoalRef:            goalRef,
		WorkItemRef:        workItemRef,
		PlanGeneration:     2,
		AppSpecGeneration:  3,
		ExecutionAttempt:   1,
		SpecHash:           strings.Repeat("d", 64),
		ActorRef:           actorRef,
		ProjectRef:         projectRef,
		Objective:          "execute exact work",
		PhaseRef:           "phase-instance:build",
		PhaseKey:           "phase:build",
		PhaseTemplateRef:   "phase-template:program",
		PhaseInputRefs:     []string{"input:spec"},
		PhaseCriterionRefs: []string{"criterion:green"},
		RoleKey:            "role:worker",
		SkillRefs:          []string{"skill:go"},
		ToolRefs:           []string{"tool:test"},
		CapabilityRefs:     []string{"capability:code"},
		WriteSet:           []string{"internal/adapters/agent/agentmicrovm"},
		OutputContract:     string(goal.OutputContractEvidenceBundle),
		ArtifactMediaType:  "application/json",
		IdempotencyKey:     "launch:microvm-1",
		MaxOutputBytes:     1 << 20,
		BudgetDemand: governance.BudgetDemand{Ref: "budget-demand:microvm-1", Resources: governance.ResourceVector{
			Tokens: 4_096, ActiveTimeNS: int64(90 * time.Second), ProcessSlots: 1, DiskBytes: 8_192,
		}},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortMedium,
		EffectAuthority: ports.AgentLaunchEffectAuthority{
			AuthorizationReceiptRef: "authorization:microvm-1",
			EffectApprovalRef:       "effect-approval:microvm-1",
			EffectAttemptRef:        "effect-attempt:microvm-1",
			ActionFence:             7,
			StartedAt:               time.Unix(10, 0).UTC(),
			ClaimLeaseUntil:         time.Unix(30, 0).UTC(),
			ApprovalExpiresAt:       time.Unix(20, 0).UTC(),
		},
	}
}

func validDescriptor(t *testing.T, withEgress bool) microvm.DescriptorPerfilLanzamientoV1 {
	t.Helper()
	profileSHA := strings.Repeat("3", 64)
	executorRef, err := microvm.ConstruirEjecutorRefPerfilV1(profileSHA)
	if err != nil {
		t.Fatalf("ConstruirEjecutorRefPerfilV1() = %v", err)
	}
	services := []microvm.ServicioVsock{{
		Papel: "control_broker", ServicioRef: "servicio:control", Puerto: 10_001,
		IdentidadRef: "identidad-servicio:control", IdentidadSHA256: strings.Repeat("4", 64),
	}}
	if withEgress {
		services = append(services, microvm.ServicioVsock{
			Papel: "controlled_egress_proxy", ServicioRef: "servicio:egreso", Puerto: 10_002,
			IdentidadRef: "identidad-servicio:egreso", IdentidadSHA256: strings.Repeat("5", 64),
		})
	}
	descriptor := microvm.DescriptorPerfilLanzamientoV1{
		Esquema:              microvm.EsquemaDescriptorPerfilLanzamientoV1,
		EjecutorRef:          executorRef,
		VCPU:                 2,
		MemoriaMiB:           512,
		KernelSHA256:         strings.Repeat("1", 64),
		InitramfsSHA256:      strings.Repeat("2", 64),
		PerfilSHA256:         profileSHA,
		ServiciosDisponibles: services,
	}
	return sealDescriptor(t, descriptor)
}

func sealDescriptor(
	t *testing.T,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) microvm.DescriptorPerfilLanzamientoV1 {
	t.Helper()
	descriptor.DescriptorRef = ""
	digest, err := microvm.CalcularSHA256DescriptorPerfilLanzamientoV1(descriptor)
	if err != nil {
		t.Fatalf("CalcularSHA256DescriptorPerfilLanzamientoV1() = %v", err)
	}
	descriptor.DescriptorRef = "perfil-lanzamiento:sha256:" + digest
	return descriptor
}
