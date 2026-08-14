package agentmicrovm

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strconv"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeDockerPlanInvalid              = "agentmicrovm.docker_plan_invalid"
	dockerCompilationFingerprintDomain = "orquesta.agentmicrovm.docker-compilation-source.v1\x00"
)

// DockerPhysicalBinding is the immutable composition choice consumed by one
// Docker plan. The compiler never discovers or selects these physical facts.
type DockerPhysicalBinding struct {
	PlacementRef ports.AgentPlacementRef
	ImageRef     string
	VCPU         uint8
	MemoryMiB    uint32
	MaxPIDs      uint32
}

// DockerCompilation seals the complete request and physical binding behind
// the exact public plan that may later consume a signing credential.
type DockerCompilation struct {
	Plan     microvm.PlanLanzamientoContenedorV1
	IssuedAt time.Time
	Validity time.Duration

	sourceFingerprint [sha256.Size]byte
}

func CompileDockerLaunch(
	request ports.AgentLaunchRequest,
	binding DockerPhysicalBinding,
) (DockerCompilation, error) {
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return DockerCompilation{}, fail(CodeLaunchRequestInvalid, err)
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return DockerCompilation{}, fail(CodeEffectAuthorityInvalid, err)
	}
	if binding.PlacementRef.String() == "" || binding.PlacementRef != request.ReferenciaColocacion {
		return DockerCompilation{}, fail(CodeProfileBindingInvalid, nil)
	}
	activeTimeNS := request.BudgetDemand.Resources.ActiveTimeNS
	if activeTimeNS <= 0 || activeTimeNS%int64(time.Millisecond) != 0 ||
		request.BudgetDemand.Resources.DiskBytes <= 0 || request.BudgetDemand.Resources.Tokens <= 0 {
		return DockerCompilation{}, fail(CodeLimitsInvalid, nil)
	}
	issuedAt, validity, ok := grantWindow(request.EffectAuthority)
	if !ok {
		return DockerCompilation{}, fail(CodeGrantWindowInvalid, nil)
	}
	plan := microvm.PlanLanzamientoContenedorV1{
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
		LimiteTiempoMS:       uint64(activeTimeNS / int64(time.Millisecond)),
		LimiteDiscoPicoBytes: uint64(request.BudgetDemand.Resources.DiskBytes),
		LimiteTokensAgente:   uint64(request.BudgetDemand.Resources.Tokens),
	}
	if _, err := microvm.CodificarPlanLanzamientoContenedorV1(plan); err != nil {
		return DockerCompilation{}, fail(CodeDockerPlanInvalid, err)
	}
	return DockerCompilation{
		Plan: plan, IssuedAt: issuedAt, Validity: validity,
		sourceFingerprint: dockerCompilationSourceFingerprint(request, binding),
	}, nil
}

func validDockerCompilation(
	request ports.AgentLaunchRequest,
	binding DockerPhysicalBinding,
	compiled DockerCompilation,
) bool {
	expected, err := CompileDockerLaunch(request, binding)
	return err == nil && reflect.DeepEqual(compiled, expected)
}

func dockerCompilationSourceFingerprint(
	request ports.AgentLaunchRequest,
	binding DockerPhysicalBinding,
) [sha256.Size]byte {
	// The existing compilation seal already covers every request field. A zero
	// ProfileBinding contributes only fixed empty fields and no physical choice.
	requestFingerprint := compilationSourceFingerprint(request, ProfileBinding{})
	digest := sha256.New()
	fingerprintString(digest, dockerCompilationFingerprintDomain)
	_, _ = digest.Write(requestFingerprint[:])
	fingerprintString(digest, binding.PlacementRef.String())
	fingerprintString(digest, binding.ImageRef)
	fingerprintUint64(digest, uint64(binding.VCPU))
	fingerprintUint64(digest, uint64(binding.MemoryMiB))
	fingerprintUint64(digest, uint64(binding.MaxPIDs))
	var result [sha256.Size]byte
	copy(result[:], digest.Sum(nil))
	return result
}

func dockerLaunchOperationRef(request ports.AgentLaunchRequest) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("orquesta.agentmicrovm.docker-launch-operation.v1\x00"))
	for _, value := range []string{request.ExecutionRef.String(), request.EffectAuthority.EffectAttemptRef,
		request.IdempotencyKey, strconv.FormatUint(request.EffectAuthority.ActionFence, 10)} {
		appendDigestField(digest, []byte(value))
	}
	return "orquesta-docker-launch:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func dockerPhysicalExecutionRef(request ports.AgentLaunchRequest) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("orquesta.agentmicrovm.docker-execution.v1\x00"))
	appendDigestField(digest, []byte(request.ExecutionRef.String()))
	return "ejecucion:orquesta_" + hex.EncodeToString(digest.Sum(nil))
}
