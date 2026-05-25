package orquestaruntime

import "testing"

func TestBuildAgentStartPacketV0RechazaContradiccionWriteSetClosed(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	request.FunctionContract.Objetivo = "Usa el write-set como alcance primario; si debes tocar otros ficheros del repo, hazlo y dejalo justificado en el ACK."
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)

	packet := BuildAgentStartPacketV0(request, materialized)

	requireRuntimeLaunchCodeV0(t, packet.Issues, AgentStartPacketInvalidoV0)
}

func TestAgentStartPacketWriteSetClosedV0DetectaPolicy(t *testing.T) {
	packet := AgentStartPacketV0{Policies: []string{"ack_required", "write_set_closed"}}

	if !AgentStartPacketWriteSetClosedV0(packet) {
		t.Fatalf("write_set_closed no detectado: %+v", packet.Policies)
	}
}
