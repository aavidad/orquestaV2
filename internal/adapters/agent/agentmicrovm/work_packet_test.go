package agentmicrovm

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/agentprotocol/codexwork"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type recordingWorkPacketRenderer struct {
	output string
	err    error
	calls  int
	prompt ports.AgentPrompt
}

func (renderer *recordingWorkPacketRenderer) RenderAgentPrompt(prompt ports.AgentPrompt) (string, error) {
	renderer.calls++
	renderer.prompt = prompt
	return renderer.output, renderer.err
}

func TestBuildWorkPacketV1ExactDeterministicAndAllowlisted(t *testing.T) {
	request := validLaunchRequest(t)
	compiled := mustCompile(t, request, validDescriptor(t, false))
	renderer := &recordingWorkPacketRenderer{output: "Implementa el cambio y entrega el artefacto."}

	packet, raw, err := BuildWorkPacketV1(request, workPacketBinding(t, request), compiled, "gpt-5.6", renderer)
	if err != nil {
		t.Fatalf("BuildWorkPacketV1() = %v", err)
	}
	want := codexwork.WorkPacketV1{
		Schema: codexwork.WorkPacketSchemaV1, Prompt: renderer.output, Model: "gpt-5.6",
		Effort: string(request.ReasoningEffort), TokenBudget: 4_096,
		MaxOutputBytes: uint64(request.MaxOutputBytes), TimeBudgetMS: 90_000,
		GoalRef: request.GoalRef.String(), WorkItemRef: request.WorkItemRef.String(),
		ExecutionRef: request.ExecutionRef.String(), EffectAttemptRef: request.EffectAuthority.EffectAttemptRef,
	}
	if packet != want {
		t.Fatalf("packet = %+v, want %+v", packet, want)
	}
	if strings.Contains(string(raw), "controlled_egress_proxy") {
		t.Fatalf("packet without grant exposed egress: %s", raw)
	}
	if renderer.calls != 1 || !reflect.DeepEqual(renderer.prompt, ports.AgentPromptFromLaunchRequest(request)) {
		t.Fatalf("renderer calls/prompt = %d/%+v", renderer.calls, renderer.prompt)
	}
	decoded, decodeErr := codexwork.DecodeWorkPacketV1(raw)
	if decodeErr != nil || decoded != packet {
		t.Fatalf("round trip = %+v/%v", decoded, decodeErr)
	}
	canonical, _ := json.Marshal(packet)
	if !bytes.Equal(raw, canonical) {
		t.Fatalf("raw is not canonical deterministic JSON: %q", raw)
	}

	secondRenderer := &recordingWorkPacketRenderer{output: renderer.output}
	secondPacket, secondRaw, err := BuildWorkPacketV1(
		request, workPacketBinding(t, request), compiled, "gpt-5.6", secondRenderer,
	)
	if err != nil || secondPacket != packet || !bytes.Equal(secondRaw, raw) {
		t.Fatalf("second build = %+v/%q/%v", secondPacket, secondRaw, err)
	}
	secondRaw[0] = '!'
	if raw[0] == '!' {
		t.Fatal("returned encodings alias storage")
	}
}

func TestBuildWorkPacketV1ProjectsOnlySignedControlledEgress(t *testing.T) {
	request := withEgressAuthority(t, validLaunchRequest(t), validEgressGrant())
	binding := profileBinding(request, validDescriptor(t, true))
	compiled, err := Compile(request, binding, request.EffectAuthority.ActionFence)
	if err != nil {
		t.Fatal(err)
	}

	packet, raw, err := BuildWorkPacketV1(
		request, binding, compiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.ControlledEgressProxy != codexwork.ControlledEgressProxyURLV1 ||
		!bytes.Contains(raw, []byte(`"controlled_egress_proxy":"`+codexwork.ControlledEgressProxyURLV1+`"`)) {
		t.Fatalf("controlled egress was not projected exactly: packet=%+v raw=%s", packet, raw)
	}
	for _, forbidden := range []string{
		request.EgressAuthority.PolicyRef,
		request.EgressAuthority.PayloadSHA256,
		string(request.EgressAuthority.CanonicalPayload),
		"api.openai.com",
		"chatgpt.com",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("egress authority leaked into work packet: %q", forbidden)
		}
	}

	crossedGrant := validEgressGrant()
	crossedGrant.Destinos[0].Puertos[0] = 8443
	crossed := withEgressAuthority(t, request, crossedGrant)
	_, _, err = BuildWorkPacketV1(
		crossed, binding, compiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("crossed egress authority code=%q error=%v", code, err)
	}
}

func TestBuildWorkPacketV1RejectsForeignOrMutatedCompilation(t *testing.T) {
	request := validLaunchRequest(t)
	base := mustCompile(t, request, validDescriptor(t, false))
	mutations := []struct {
		name   string
		mutate func(*Compilation)
	}{
		{"project", func(value *Compilation) { value.Context.ProyectoRef = "project:other" }},
		{"goal", func(value *Compilation) { value.Context.GoalRef = "goal:other" }},
		{"work item", func(value *Compilation) { value.Context.WorkItemRef = "work:other" }},
		{"execution context", func(value *Compilation) { value.Context.EjecucionRef = "execution:other" }},
		{"authorization", func(value *Compilation) { value.Context.AutorizacionRef = "authorization:other" }},
		{"approval", func(value *Compilation) { value.Context.AprobacionEfectoRef = "effect-approval:other" }},
		{"effect attempt", func(value *Compilation) { value.Context.IntentoEfectoRef = "effect-attempt:other" }},
		{"session", func(value *Compilation) { value.Context.SesionRef = "execution-session:other" }},
		{"artifact access", func(value *Compilation) { value.Context.AccesoArtefactosRef = "artifact-access:other" }},
		{"mcp access", func(value *Compilation) { value.Context.AccesoMCPRef = "mcp-access:other" }},
		{"mailbox", func(value *Compilation) { value.Context.EndpointBuzonRef = "mailbox-endpoint:other" }},
		{"descriptor", func(value *Compilation) { value.Context.DescriptorPerfilRef = "" }},
		{"spec", func(value *Compilation) { value.Context.EspecificacionSHA256 = strings.Repeat("e", 64) }},
		{"plan ref", func(value *Compilation) { value.Plan.PlanRef = "plan:" + strings.Repeat("e", 64) }},
		{"run", func(value *Compilation) { value.Plan.RunRef = "execution:other" }},
		{"fence", func(value *Compilation) { value.Plan.Cerca++ }},
		{"tokens", func(value *Compilation) { value.Plan.LimiteTokensAgente++ }},
		{"time", func(value *Compilation) { value.Plan.LimiteTiempoMS++ }},
		{"disk", func(value *Compilation) { value.Plan.LimiteDiscoPicoBytes++ }},
		{"issued at", func(value *Compilation) { value.IssuedAt = value.IssuedAt.Add(time.Millisecond) }},
		{"validity", func(value *Compilation) { value.Validity += time.Millisecond }},
		{"egress", func(value *Compilation) { value.Plan.Egreso = &microvm.ConcesionEgreso{} }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			compiled := cloneCompilationForPacketTest(base)
			mutation.mutate(&compiled)
			renderer := &recordingWorkPacketRenderer{output: "work"}
			_, _, err := BuildWorkPacketV1(
				request, workPacketBinding(t, request), compiled, "gpt-5.6", renderer,
			)
			if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
				t.Fatalf("ErrorCode() = %q; err=%v", code, err)
			}
			if renderer.calls != 0 {
				t.Fatal("foreign compilation reached renderer")
			}
		})
	}

	foreignRequest := request
	foreignRequest.EffectAuthority.EffectAttemptRef = "effect-attempt:foreign"
	foreign := mustCompile(t, foreignRequest, validDescriptor(t, false))
	_, _, err := BuildWorkPacketV1(
		request, workPacketBinding(t, request), foreign, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("crossed request ErrorCode() = %q; err=%v", code, err)
	}
}

func TestBuildWorkPacketV1SealRejectsCrossedRequestFacts(t *testing.T) {
	baseRequest := validLaunchRequest(t)
	baseBinding := workPacketBinding(t, baseRequest)
	baseCompiled := mustCompile(t, baseRequest, baseBinding.Descriptor)
	mutations := []struct {
		name   string
		mutate func(*ports.AgentLaunchRequest)
	}{
		{"objective", func(value *ports.AgentLaunchRequest) { value.Objective = "different objective" }},
		{"plan generation", func(value *ports.AgentLaunchRequest) { value.PlanGeneration++ }},
		{"app spec generation", func(value *ports.AgentLaunchRequest) { value.AppSpecGeneration++ }},
		{"execution attempt", func(value *ports.AgentLaunchRequest) { value.ExecutionAttempt++ }},
		{"phase ref", func(value *ports.AgentLaunchRequest) { value.PhaseRef = "phase-instance:review" }},
		{"phase key", func(value *ports.AgentLaunchRequest) { value.PhaseKey = "phase:review" }},
		{"phase template", func(value *ports.AgentLaunchRequest) { value.PhaseTemplateRef = "phase-template:review" }},
		{"phase inputs", func(value *ports.AgentLaunchRequest) { value.PhaseInputRefs = []string{"input:other"} }},
		{"phase criteria", func(value *ports.AgentLaunchRequest) { value.PhaseCriterionRefs = []string{"criterion:other"} }},
		{"role", func(value *ports.AgentLaunchRequest) { value.RoleKey = "role:reviewer" }},
		{"skills", func(value *ports.AgentLaunchRequest) { value.SkillRefs = []string{"skill:review"} }},
		{"tools", func(value *ports.AgentLaunchRequest) { value.ToolRefs = []string{"tool:review"} }},
		{"capabilities", func(value *ports.AgentLaunchRequest) {
			value.CapabilityRefs = []string{"capability:review"}
		}},
		{"write set", func(value *ports.AgentLaunchRequest) { value.WriteSet = []string{"internal/review"} }},
		{"output contract", func(value *ports.AgentLaunchRequest) {
			value.OutputContract = string(goal.OutputContractArtifact)
		}},
		{"media type", func(value *ports.AgentLaunchRequest) { value.ArtifactMediaType = "text/plain" }},
		{"effort", func(value *ports.AgentLaunchRequest) { value.ReasoningEffort = governance.ReasoningEffortHigh }},
		{"max output", func(value *ports.AgentLaunchRequest) { value.MaxOutputBytes-- }},
		{"budget ref", func(value *ports.AgentLaunchRequest) { value.BudgetDemand.Ref = "budget-demand:other" }},
		{"budget tokens", func(value *ports.AgentLaunchRequest) { value.BudgetDemand.Resources.Tokens++ }},
		{"budget money", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.MoneyMicros = 1
			value.BudgetDemand.Resources.Currency = "USD"
		}},
		{"budget process slots", func(value *ports.AgentLaunchRequest) { value.BudgetDemand.Resources.ProcessSlots++ }},
		{"criticality", func(value *ports.AgentLaunchRequest) {
			value.SecurityCriticality = governance.SecurityCriticalitySensitive
		}},
		{"preserve", func(value *ports.AgentLaunchRequest) { value.RequierePreservacionEntorno = true }},
		{"idempotency", func(value *ports.AgentLaunchRequest) { value.IdempotencyKey = "launch:other" }},
		{"actor", func(value *ports.AgentLaunchRequest) {
			value.ActorRef, _ = goal.NewActorRef("actor:other")
		}},
		{"workspace", func(value *ports.AgentLaunchRequest) {
			value.ExecutionWorkspaceRef, _ = ports.NewExecutionWorkspaceRef("workspace:execution:other")
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			request := baseRequest
			mutation.mutate(&request)
			if err := ports.ValidateAgentLaunchRequest(request); err != nil {
				t.Fatalf("mutation must remain a valid request: %v", err)
			}
			renderer := &recordingWorkPacketRenderer{output: "work"}
			_, _, err := BuildWorkPacketV1(request, baseBinding, baseCompiled, "gpt-5.6", renderer)
			if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
				t.Fatalf("ErrorCode() = %q; err=%v", code, err)
			}
			if renderer.calls != 0 {
				t.Fatal("crossed request reached renderer")
			}
		})
	}
}

func TestBuildWorkPacketV1SealPreservesExactProfileServiceOrder(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, true)
	originalBinding := profileBinding(request, descriptor)
	compiled := mustCompile(t, request, descriptor)

	reorderedDescriptor := descriptor
	reorderedDescriptor.ServiciosDisponibles = append([]microvm.ServicioVsock(nil), descriptor.ServiciosDisponibles...)
	reorderedDescriptor.ServiciosDisponibles[0], reorderedDescriptor.ServiciosDisponibles[1] =
		reorderedDescriptor.ServiciosDisponibles[1], reorderedDescriptor.ServiciosDisponibles[0]
	if err := microvm.ValidarDescriptorPerfilLanzamientoV1(reorderedDescriptor); err != nil {
		t.Fatalf("reordered descriptor must remain valid: %v", err)
	}
	reorderedBinding := profileBinding(request, reorderedDescriptor)
	_, _, err := BuildWorkPacketV1(
		request, reorderedBinding, compiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("reordered service ErrorCode() = %q; err=%v", code, err)
	}

	otherDescriptor := descriptor
	otherDescriptor.MemoriaMiB++
	digest, digestErr := microvm.CalcularSHA256DescriptorPerfilLanzamientoV1(otherDescriptor)
	if digestErr != nil {
		t.Fatalf("CalcularSHA256DescriptorPerfilLanzamientoV1() = %v", digestErr)
	}
	otherDescriptor.DescriptorRef = "perfil-lanzamiento:sha256:" + digest
	if err := microvm.ValidarDescriptorPerfilLanzamientoV1(otherDescriptor); err != nil {
		t.Fatalf("other descriptor must remain valid: %v", err)
	}
	otherCompiled := mustCompile(t, request, otherDescriptor)
	_, _, err = BuildWorkPacketV1(
		request, originalBinding, otherCompiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("crossed profile ErrorCode() = %q; err=%v", code, err)
	}
}

func TestBuildWorkPacketV1SealRejectsAliasedSliceMutation(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, true)
	binding := profileBinding(request, descriptor)
	compiled := mustCompile(t, request, descriptor)
	originalSkill := request.SkillRefs[0]
	request.SkillRefs[0] = "skill:other"
	_, _, err := BuildWorkPacketV1(
		request, binding, compiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("aliased request slice ErrorCode() = %q; err=%v", code, err)
	}
	request.SkillRefs[0] = originalSkill

	controlService := compiled.Plan.Servicios[0]
	binding.Descriptor.ServiciosDisponibles[0], binding.Descriptor.ServiciosDisponibles[1] =
		binding.Descriptor.ServiciosDisponibles[1], binding.Descriptor.ServiciosDisponibles[0]
	if compiled.Plan.Servicios[0] != controlService {
		t.Fatal("compiled plan aliases profile service storage")
	}
	_, _, err = BuildWorkPacketV1(
		request, binding, compiled, "gpt-5.6", &recordingWorkPacketRenderer{output: "work"},
	)
	if code := ErrorCode(err); code != CodeWorkPacketCompilationMismatch {
		t.Fatalf("aliased profile slice ErrorCode() = %q; err=%v", code, err)
	}
}

func TestBuildWorkPacketV1RejectsInvalidModelEffortBudgetsAndOutput(t *testing.T) {
	validRequest := validLaunchRequest(t)
	validCompiled := mustCompile(t, validRequest, validDescriptor(t, false))
	tests := []struct {
		name   string
		model  string
		mutate func(*ports.AgentLaunchRequest)
		code   string
	}{
		{"empty model", "", nil, CodeWorkPacketModelInvalid},
		{"spaced model", " gpt-5.6", nil, CodeWorkPacketModelInvalid},
		{"control model", "gpt-5.6\n", nil, CodeWorkPacketModelInvalid},
		{"long model", strings.Repeat("m", 129), nil, CodeWorkPacketModelInvalid},
		{"effort", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.ReasoningEffort = governance.ReasoningEffort("ultra")
		}, CodeWorkPacketEffortInvalid},
		{"tokens zero", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.Tokens = 0
		}, CodeWorkPacketBudgetInvalid},
		{"tokens high", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.Tokens = int64(codexwork.MaxTokenBudgetV1) + 1
		}, CodeWorkPacketBudgetInvalid},
		{"sub millisecond time", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.ActiveTimeNS = int64(time.Millisecond) + 1
		}, CodeWorkPacketBudgetInvalid},
		{"time high", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.ActiveTimeNS = int64(codexwork.MaxTimeBudgetMSV1+1) * int64(time.Millisecond)
		}, CodeWorkPacketBudgetInvalid},
		{"disk zero", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.BudgetDemand.Resources.DiskBytes = 0
		}, CodeWorkPacketBudgetInvalid},
		{"output zero", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.MaxOutputBytes = 0
		}, CodeWorkPacketOutputInvalid},
		{"output high", "gpt-5.6", func(value *ports.AgentLaunchRequest) {
			value.MaxOutputBytes = int64(codexwork.MaxOutputBytesV1) + 1
		}, CodeWorkPacketOutputInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validRequest
			if test.mutate != nil {
				test.mutate(&request)
			}
			renderer := &recordingWorkPacketRenderer{output: "work"}
			_, _, err := BuildWorkPacketV1(
				request, workPacketBinding(t, request), validCompiled, test.model, renderer,
			)
			if code := ErrorCode(err); code != test.code {
				t.Fatalf("ErrorCode() = %q, want %q; err=%v", code, test.code, err)
			}
			if renderer.calls != 0 {
				t.Fatal("invalid packet reached renderer")
			}
		})
	}
}

func TestBuildWorkPacketV1RejectsRendererPromptAndOversizeWithoutLeakingErrors(t *testing.T) {
	request := validLaunchRequest(t)
	compiled := mustCompile(t, request, validDescriptor(t, false))

	var typedNil *recordingWorkPacketRenderer
	for name, renderer := range map[string]PromptRenderer{"nil": nil, "typed nil": typedNil} {
		t.Run(name, func(t *testing.T) {
			_, _, err := BuildWorkPacketV1(
				request, workPacketBinding(t, request), compiled, "gpt-5.6", renderer,
			)
			if code := ErrorCode(err); code != CodeWorkPacketRendererInvalid {
				t.Fatalf("ErrorCode() = %q; err=%v", code, err)
			}
		})
	}

	secret := "oauth-secret-must-not-escape"
	failed := &recordingWorkPacketRenderer{err: errors.New(secret)}
	_, _, err := BuildWorkPacketV1(request, workPacketBinding(t, request), compiled, "gpt-5.6", failed)
	if code := ErrorCode(err); code != CodeWorkPacketRenderFailed || strings.Contains(err.Error(), secret) || errors.Is(err, failed.err) {
		t.Fatalf("renderer error was not redacted: %v", err)
	}

	for name, prompt := range map[string]string{
		"empty": "", "blank": " \n\t", "invalid utf8": string([]byte{0xff}),
	} {
		t.Run(name, func(t *testing.T) {
			if name == "invalid utf8" && utf8.ValidString(prompt) {
				t.Fatal("test fixture unexpectedly valid")
			}
			_, _, err := BuildWorkPacketV1(
				request, workPacketBinding(t, request), compiled, "gpt-5.6",
				&recordingWorkPacketRenderer{output: prompt},
			)
			if code := ErrorCode(err); code != CodeWorkPacketPromptInvalid {
				t.Fatalf("ErrorCode() = %q; err=%v", code, err)
			}
		})
	}

	oversized := &recordingWorkPacketRenderer{output: strings.Repeat("x", 16*codexwork.MaxPacketBytesV1)}
	_, raw, err := BuildWorkPacketV1(request, workPacketBinding(t, request), compiled, "gpt-5.6", oversized)
	if code := ErrorCode(err); code != CodeWorkPacketTooLarge || raw != nil {
		t.Fatalf("oversize = %q/%d; err=%v", code, len(raw), err)
	}
}

func TestBuildWorkPacketV1KeepsSecretAuthorityOutOfPromptAndPacket(t *testing.T) {
	request := validLaunchRequest(t)
	request.IdempotencyKey = "idempotency:secret-marker"
	request.ActorRef, _ = goal.NewActorRef("actor:secret-marker")
	request.ReferenciaColocacion, _ = ports.NewAgentPlacementRef("placement:secret-marker")
	request.ExecutionWorkspaceRef, _ = ports.NewExecutionWorkspaceRef("workspace:execution:secret-marker")
	request.EffectAuthority.AuthorizationReceiptRef = "authorization:secret-marker"
	request.EffectAuthority.EffectApprovalRef = "effect-approval:secret-marker"
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:secret-marker")
	request.AccessAuthority.ArtifactAccessRef, _ = ports.NewExecutionArtifactAccessRef(
		"artifact-access:execution:sha256:" + strings.Repeat("7", 64),
	)
	request.AccessAuthority.MCPAccessRef, _ = ports.NewExecutionMCPAccessRef(
		"mcp-access:execution:sha256:" + strings.Repeat("8", 64),
	)
	request.AccessAuthority.MailboxEndpointRef, _ = ports.NewExecutionMailboxEndpointRef(
		"mailbox-endpoint:execution:sha256:" + strings.Repeat("9", 64),
	)
	compiled := mustCompile(t, request, validDescriptor(t, false))
	renderer := &recordingWorkPacketRenderer{output: "work"}

	packet, raw, err := BuildWorkPacketV1(request, workPacketBinding(t, request), compiled, "gpt-5.6", renderer)
	if err != nil {
		t.Fatalf("BuildWorkPacketV1() = %v", err)
	}
	promptJSON, marshalErr := json.Marshal(renderer.prompt)
	if marshalErr != nil {
		t.Fatalf("json.Marshal(renderer.prompt) = %v", marshalErr)
	}
	if renderer.prompt.Objective != request.Objective || renderer.prompt.ProjectRef != request.ProjectRef.String() ||
		renderer.prompt.GoalRef != request.GoalRef.String() || renderer.prompt.WorkItemRef != request.WorkItemRef.String() ||
		renderer.prompt.ExecutionRef != request.ExecutionRef.String() {
		t.Fatalf("neutral prompt lost declared work facts: %+v", renderer.prompt)
	}
	joined := string(promptJSON) + "\n" + packet.Prompt + "\n" + string(raw)
	for _, forbidden := range []string{
		request.IdempotencyKey,
		request.ActorRef.String(),
		request.ReferenciaColocacion.String(),
		request.ExecutionWorkspaceRef.String(),
		request.EffectAuthority.AuthorizationReceiptRef,
		request.EffectAuthority.EffectApprovalRef,
		request.SessionRef.String(),
		request.AccessAuthority.ArtifactAccessRef.String(),
		request.AccessAuthority.MCPAccessRef.String(),
		request.AccessAuthority.MailboxEndpointRef.String(),
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("secret authority leaked: %q", forbidden)
		}
	}
	if !strings.Contains(string(raw), request.EffectAuthority.EffectAttemptRef) {
		t.Fatal("schema-declared effect_attempt_ref missing")
	}
}

func cloneCompilationForPacketTest(value Compilation) Compilation {
	value.Plan.Servicios = append([]microvm.ServicioVsock(nil), value.Plan.Servicios...)
	if value.Plan.PerfilSHA256 != nil {
		profileSHA := *value.Plan.PerfilSHA256
		value.Plan.PerfilSHA256 = &profileSHA
	}
	if value.Plan.Egreso != nil {
		egress := *value.Plan.Egreso
		egress.Destinos = append([]microvm.DestinoEgreso(nil), value.Plan.Egreso.Destinos...)
		for index := range egress.Destinos {
			egress.Destinos[index].Puertos = append(
				[]uint16(nil), value.Plan.Egreso.Destinos[index].Puertos...,
			)
		}
		value.Plan.Egreso = &egress
	}
	return value
}

func workPacketBinding(t *testing.T, request ports.AgentLaunchRequest) ProfileBinding {
	t.Helper()
	return profileBinding(request, validDescriptor(t, false))
}
