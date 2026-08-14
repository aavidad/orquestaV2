package agentmicrovm

import (
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

func TestCompileDockerLaunchBindsCompletePhysicalPlanAndWindow(t *testing.T) {
	request := validLaunchRequest(t)
	binding := validDockerPhysicalBinding(request)
	compiled, err := CompileDockerLaunch(request, binding)
	if err != nil {
		t.Fatal(err)
	}
	want := microvm.PlanLanzamientoContenedorV1{
		Esquema:              microvm.EsquemaPlanLanzamientoContenedorV1,
		OperacionRef:         dockerLaunchOperationRef(request),
		EjecucionRef:         dockerPhysicalExecutionRef(request),
		RunRef:               request.ExecutionRef.String(),
		Cerca:                request.EffectAuthority.ActionFence,
		EspecificacionRef:    "app-spec:sha256:" + request.SpecHash,
		BultosRef:            []string{},
		ImagenRef:            binding.ImageRef,
		VCPU:                 binding.VCPU,
		MemoriaMiB:           binding.MemoryMiB,
		MaximoPIDs:           binding.MaxPIDs,
		LimiteTiempoMS:       90_000,
		LimiteDiscoPicoBytes: 8_192,
		LimiteTokensAgente:   4_096,
	}
	if !reflect.DeepEqual(compiled.Plan, want) || !compiled.IssuedAt.Equal(time.Unix(10, 0)) ||
		compiled.Validity != 10*time.Second || compiled.sourceFingerprint == ([32]byte{}) {
		t.Fatalf("compiled = %+v", compiled)
	}
	if _, err := microvm.CodificarPlanLanzamientoContenedorV1(compiled.Plan); err != nil {
		t.Fatalf("public plan encoding = %v", err)
	}
	if replay := mustCompileDocker(t, request, binding); !reflect.DeepEqual(compiled, replay) {
		t.Fatalf("replay differs: first=%+v replay=%+v", compiled, replay)
	}
}

func TestCompileDockerLaunchFingerprintBindsRequestAndPhysicalFacts(t *testing.T) {
	request := validLaunchRequest(t)
	binding := validDockerPhysicalBinding(request)
	first := mustCompileDocker(t, request, binding)

	changedRequest := request
	changedRequest.Objective = "different authorized work"
	if changed := mustCompileDocker(t, changedRequest, binding); changed.sourceFingerprint == first.sourceFingerprint {
		t.Fatal("request mutation preserved source fingerprint")
	}
	changedBinding := binding
	changedBinding.MaxPIDs++
	if changed := mustCompileDocker(t, request, changedBinding); changed.sourceFingerprint == first.sourceFingerprint {
		t.Fatal("physical mutation preserved source fingerprint")
	}
}

func TestCompileDockerLaunchRejectsInvalidSourceBeforePlan(t *testing.T) {
	request := validLaunchRequest(t)
	binding := validDockerPhysicalBinding(request)
	tests := []struct {
		name    string
		request ports.AgentLaunchRequest
		binding DockerPhysicalBinding
		code    string
	}{
		{"request", func() ports.AgentLaunchRequest { value := request; value.Objective = ""; return value }(), binding, CodeLaunchRequestInvalid},
		{"effect", func() ports.AgentLaunchRequest { value := request; value.EffectAuthority.ActionFence = 0; return value }(), binding, CodeEffectAuthorityInvalid},
		{"placement empty", request, func() DockerPhysicalBinding {
			value := binding
			value.PlacementRef = ports.AgentPlacementRef{}
			return value
		}(), CodeProfileBindingInvalid},
		{"placement crossed", request, func() DockerPhysicalBinding {
			value := binding
			value.PlacementRef, _ = ports.NewAgentPlacementRef("placement:other")
			return value
		}(), CodeProfileBindingInvalid},
		{"sub millisecond", func() ports.AgentLaunchRequest {
			value := request
			value.BudgetDemand.Resources.ActiveTimeNS++
			return value
		}(), binding, CodeLimitsInvalid},
		{"disk", func() ports.AgentLaunchRequest {
			value := request
			value.BudgetDemand.Resources.DiskBytes = 0
			return value
		}(), binding, CodeLimitsInvalid},
		{"image", request, func() DockerPhysicalBinding {
			value := binding
			value.ImageRef = "registry.invalid/unpinned:latest"
			return value
		}(), CodeDockerPlanInvalid},
		{"vcpu", request, func() DockerPhysicalBinding { value := binding; value.VCPU = 0; return value }(), CodeDockerPlanInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiled, err := CompileDockerLaunch(test.request, test.binding)
			if ErrorCode(err) != test.code || !reflect.DeepEqual(compiled, DockerCompilation{}) {
				t.Fatalf("compiled=%+v code=%q err=%v", compiled, ErrorCode(err), err)
			}
		})
	}
}

func TestCompileDockerLaunchRejectsNonExactMillisecondEvenWhenPositive(t *testing.T) {
	request := validLaunchRequest(t)
	request.BudgetDemand.Resources.ActiveTimeNS = int64(time.Second) + 1
	compiled, err := CompileDockerLaunch(request, validDockerPhysicalBinding(request))
	if ErrorCode(err) != CodeLimitsInvalid || !reflect.DeepEqual(compiled, DockerCompilation{}) ||
		strings.Contains(err.Error(), request.Objective) {
		t.Fatalf("compiled=%+v err=%v", compiled, err)
	}
}
