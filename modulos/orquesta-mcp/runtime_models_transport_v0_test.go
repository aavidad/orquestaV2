package orquestamcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestMCPTransportV0RuntimeModelsQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	if _, ok := transport.tools[MCPRuntimeModelsToolNameV0]; ok {
		t.Fatalf("runtime models sin puerto no debe anunciarse")
	}
}

func TestMCPTransportV0RuntimeModelsInvocaPuerto(t *testing.T) {
	port := &fakeMCPRuntimeModelsPortV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RuntimeModels: port,
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRuntimeModelsToolNameV0,
		MCPRuntimeModelsToolInputV0{
			Action:       "serve",
			Model:        "qwen2.5:7b",
			EvidenceRefs: []string{"evidence-ref-runtime-model-serve-001"},
		},
	)
	if err != nil {
		t.Fatalf("call runtime models: %v", err)
	}
	var result MCPRuntimeModelsToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRuntimeModelsEstadoOKV0 ||
		result.ActionResult == nil ||
		result.ActionResult.Model != "qwen2.5:7b" ||
		len(result.ActionResult.Evidence) != 1 ||
		result.ActionResult.Evidence[0].Ref != "evidence-ref-runtime-model-serve-001" ||
		port.serve.Model != "qwen2.5:7b" {
		t.Fatalf("result=%+v port=%+v", result, port)
	}
}

func TestRuntimeModelsInputPublicoNoAceptaBaseURLV0(t *testing.T) {
	for _, sample := range []any{
		MCPRuntimeModelsToolInputV0{},
		orquestaruntime.RuntimeModelListRequestV0{},
		orquestaruntime.RuntimeModelActionRequestV0{},
	} {
		typ := reflect.TypeOf(sample)
		if _, ok := typ.FieldByName("BaseURL"); ok {
			t.Fatalf("%s no debe aceptar base_url publico", typ.Name())
		}
	}
	if strings.Contains(MCPRuntimeModelsDescriptorV0().InputSchema, "base_url") {
		t.Fatalf("descriptor no debe publicar base_url: %s", MCPRuntimeModelsDescriptorV0().InputSchema)
	}
}

func TestMCPRuntimeModelsDescriptorV0DeclaraEvidenciaOperativa(t *testing.T) {
	descriptor := MCPRuntimeModelsDescriptorV0()

	if !strings.Contains(descriptor.Output, "evidence?") ||
		!strings.Contains(descriptor.Output, "action_result?") ||
		!strings.Contains(descriptor.Output, "list_result?") {
		t.Fatalf("descriptor output sin evidencia runtime: %s", descriptor.Output)
	}
}

type fakeMCPRuntimeModelsPortV0 struct {
	list   orquestaruntime.RuntimeModelListRequestV0
	status orquestaruntime.RuntimeModelListRequestV0
	pull   orquestaruntime.RuntimeModelActionRequestV0
	serve  orquestaruntime.RuntimeModelActionRequestV0
	stop   orquestaruntime.RuntimeModelActionRequestV0
}

func (port *fakeMCPRuntimeModelsPortV0) ListRuntimeModelsV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	port.list = request
	return orquestaruntime.RuntimeModelListResultV0{
		ProviderRef: "ollama",
		Models: []orquestaruntime.RuntimeModelInfoV0{{
			Name:   "qwen2.5:7b",
			Status: "available",
		}},
	}, nil
}

func (port *fakeMCPRuntimeModelsPortV0) RuntimeModelStatusV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	port.status = request
	return orquestaruntime.RuntimeModelListResultV0{
		ProviderRef: "ollama",
		Models: []orquestaruntime.RuntimeModelInfoV0{{
			Name:   "qwen2.5:7b",
			Status: "running",
		}},
	}, nil
}

func (port *fakeMCPRuntimeModelsPortV0) PullRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.pull = request
	return runtimeModelActionResultForTestV0(request, "pulled"), nil
}

func (port *fakeMCPRuntimeModelsPortV0) ServeRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.serve = request
	return runtimeModelActionResultForTestV0(request, "served"), nil
}

func (port *fakeMCPRuntimeModelsPortV0) StopRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.stop = request
	return runtimeModelActionResultForTestV0(request, "stopped"), nil
}

func runtimeModelActionResultForTestV0(
	request orquestaruntime.RuntimeModelActionRequestV0,
	status string,
) orquestaruntime.RuntimeModelActionResultV0 {
	result := orquestaruntime.RuntimeModelActionResultV0{
		ProviderRef: "ollama",
		Model:       request.Model,
		Accepted:    true,
		Status:      status,
	}
	if len(request.Evidence) > 0 {
		result.Evidence = []orquestaruntime.RuntimeModelEvidence{{
			Kind: "request",
			Ref:  request.Evidence[0],
		}}
	}
	return result
}
