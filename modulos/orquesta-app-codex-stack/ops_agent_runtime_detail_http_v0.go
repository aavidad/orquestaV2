package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexStackAgentRuntimeDetailHTTPHandlerV0 struct {
	config CodexStackAgentRuntimeDetailConfigV0
}

func NewCodexStackAgentRuntimeDetailHTTPHandlerV0(
	config CodexStackAgentRuntimeDetailConfigV0,
) http.Handler {
	return codexStackAgentRuntimeDetailHTTPHandlerV0{config: config}
}

func (handler codexStackAgentRuntimeDetailHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	if handler.config.ReceiptStore == nil {
		http.Error(w, "receipt_store no configurado", http.StatusServiceUnavailable)
		return
	}
	var request CodexStackAgentRuntimeDetailRequestV0
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "json invalido", http.StatusBadRequest)
		return
	}
	response, status, err := handler.buildResponseV0(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (handler codexStackAgentRuntimeDetailHTTPHandlerV0) buildResponseV0(
	ctx context.Context,
	request CodexStackAgentRuntimeDetailRequestV0,
) (CodexStackAgentRuntimeDetailResponseV0, int, error) {
	runRef := strings.TrimSpace(request.RunRef)
	if runRef == "" {
		return CodexStackAgentRuntimeDetailResponseV0{}, http.StatusBadRequest, errors.New("run_ref requerido")
	}
	descriptors, err := handler.config.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		return CodexStackAgentRuntimeDetailResponseV0{}, http.StatusInternalServerError, err
	}
	agentRef := strings.TrimSpace(request.AgentRef)
	maxBytes := codexStackAgentRuntimeDetailMaxBytesV0(request.MaxFileBytes)
	response := CodexStackAgentRuntimeDetailResponseV0{Estado: "ok", RunRef: runRef}
	for _, descriptor := range descriptors {
		descriptorAgentRef := codexStackAgentRuntimeDetailDescriptorAgentRefV0(descriptor)
		if agentRef != "" && descriptorAgentRef != agentRef {
			continue
		}
		detail := handler.agentDetailV0(descriptor, descriptorAgentRef, request.IncludeLogs, maxBytes)
		response.Agents = append(response.Agents, detail)
	}
	if len(response.Agents) == 0 {
		response.Issues = append(response.Issues, CodexStackAgentRuntimeDetailIssueV0{
			Field:   "agent_ref",
			Message: "sin agentes encontrados para el filtro solicitado",
		})
	}
	return response, http.StatusOK, nil
}

func (handler codexStackAgentRuntimeDetailHTTPHandlerV0) agentDetailV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	agentRef string,
	includeLogs bool,
	maxBytes int64,
) CodexStackAgentRuntimeDetailAgentV0 {
	dirs := codexStackAgentRuntimeDetailRuntimeDirsV0(handler.config.RuntimeWorkDir, descriptor, agentRef)
	runtimeDir := ""
	for _, dir := range dirs {
		if codexStackAgentRuntimeDetailDirExistsV0(dir) {
			runtimeDir = dir
			break
		}
	}
	files := codexStackAgentRuntimeDetailFilesV0(dirs, includeLogs, maxBytes)
	prompt := codexStackAgentRuntimeDetailContentByNameV0(files, orquestaruntimecodex.CodexAgentPromptFileNameV0)
	ack := codexStackAgentRuntimeDetailJSONByNameV0(files, orquestaruntimecodex.CodexAgentAckFileNameV0)
	packet := descriptor.Spec.AgentPacket
	return CodexStackAgentRuntimeDetailAgentV0{
		RunRef:      descriptor.RunID,
		AgentRef:    agentRef,
		RuntimeDir:  runtimeDir,
		Descriptor:  descriptor,
		AgentPacket: packet,
		Ack:         ack,
		PromptText:  prompt,
		Skills:      codexStackAgentRuntimeDetailSkillHintsV0(packet, prompt),
		Files:       files,
	}
}
