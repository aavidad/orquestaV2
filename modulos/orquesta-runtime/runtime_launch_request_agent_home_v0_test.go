package orquestaruntime

import "testing"

func TestValidateRuntimeLaunchRequestV0RechazaBindingAgentHomeV0ConDetallesReales(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RuntimeLaunchRequestV0)
		code   RuntimeLaunchErrorCodeV0
	}{
		{
			name: "home real",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.RuntimeBinding.HomeRef = "/home/alberto/.codex"
			},
			code: RutaHomeRealDetectadaV0,
		},
		{
			name: "account email",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.RuntimeBinding.CredentialRef = "agent@example.com"
			},
			code: ReferenciaNoOpacaV0,
		},
		{
			name: "credential token",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.RuntimeBinding.CredentialRef = "bearer abc123"
			},
			code: SecretoDetectadoV0,
		},
		{
			name: "provider hardcodeado",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.RuntimeBinding.ProviderRef = "openai"
			},
			code: ReferenciaNoOpacaV0,
		},
		{
			name: "model hardcodeado",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.CapacityDecision.ModelRef = "gpt-4o"
				req.RuntimeBinding.ModelRef = "gpt-4o"
			},
			code: ReferenciaNoOpacaV0,
		},
		{
			name: "home no opaco",
			mutate: func(req *RuntimeLaunchRequestV0) {
				req.RuntimeBinding.HomeRef = "home ref real"
			},
			code: ReferenciaNoOpacaV0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := runtimeLaunchRequestValidaV0()
			tt.mutate(&req)

			requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), tt.code)
		})
	}
}
