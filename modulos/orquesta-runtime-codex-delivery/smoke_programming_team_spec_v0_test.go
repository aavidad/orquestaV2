package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type programmingTeamCodexSpecResolverV0 struct {
	Config codexRealSmokeConfigV0
	Tasks  []programmingTeamTaskV0
	Form   string
}

type programmingTeamTaskV0 struct {
	Area          string
	AgentRef      string
	TaskRef       string
	Title         string
	Objective     string
	WriteSet      []string
	RequiredTests []string
}

func (resolver programmingTeamCodexSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	task, ok := programmingTeamTaskByAgentV0(resolver.Tasks, inbound.Payload.AgentRequestID)
	if !ok {
		return resolver.resolveDirectorSpecV0(ctx, inbound)
	}
	runtimeDir, err := CodexReceiptAgentRuntimeDirV0(
		resolver.Config.RuntimeWorkDir,
		inbound.Payload.RunID,
		task.AgentRef,
	)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	profile := codexRealSmokeProfileV0(resolver.Config)
	profile.RuntimeWorkDir = runtimeDir
	spec := codexDeliveryLoopSpecForTestV0(task.AgentRef, task.TaskRef)
	spec.CorrelationID = strings.TrimSpace(inbound.CorrelationID)
	spec.ProfileRef = "profile-ref-programming-" + task.Area
	spec.ConnectorRef = "connector-ref-programming-" + task.Area
	spec.Command.CommandRef = "command-ref-programming-" + task.Area
	spec.Command.ExecutableRef = "exec-ref-programming-" + task.Area
	spec.Command.WorkingDirRef = "workdir-ref-programming-" + task.Area
	spec.AgentPacket = programmingTeamPacketV0(task, spec.CorrelationID)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
	}, nil
}

func (resolver programmingTeamCodexSpecResolverV0) resolveDirectorSpecV0(
	_ context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	if agentRef == "" {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{},
			fmt.Errorf("programming_team_spec: agent_ref requerido")
	}
	taskRef := strings.TrimSpace(inbound.Payload.TaskRef)
	if taskRef == "" {
		taskRef = "task-agenda-director"
	}
	runtimeDir, err := CodexReceiptAgentRuntimeDirV0(
		resolver.Config.RuntimeWorkDir,
		inbound.Payload.RunID,
		agentRef,
	)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	profile := codexRealSmokeProfileV0(resolver.Config)
	profile.RuntimeWorkDir = runtimeDir
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	spec.CorrelationID = strings.TrimSpace(inbound.CorrelationID)
	spec.ProfileRef = "profile-ref-programming-director"
	spec.ConnectorRef = "connector-ref-programming-director"
	spec.Command.CommandRef = "command-ref-programming-director"
	spec.Command.ExecutableRef = "exec-ref-programming-director"
	spec.Command.WorkingDirRef = "workdir-ref-programming-director"
	spec.AgentPacket = programmingTeamDirectorPacketV0(
		agentRef,
		taskRef,
		strings.TrimSpace(inbound.Payload.RunID),
		strings.TrimSpace(inbound.Payload.PhaseID),
		spec.CorrelationID,
		resolver.Tasks,
	)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
	}, nil
}

func programmingTeamTasksV0() []programmingTeamTaskV0 {
	tasks := []programmingTeamTaskV0{
		{
			Area:      "api",
			TaskRef:   "task-programming-api",
			Title:     "Crear API REST Go de agenda",
			Objective: "API REST Go pequena: health y POST/GET /events con title, starts_at y description opcional. Estado en memoria, stdlib, server/main.go. Tests prueban description.",
			WriteSet: []string{
				"internal/api/handler.go",
				"internal/api/handler_test.go",
				"server/main.go",
			},
			RequiredTests: []string{"go test ./..."},
		},
		{
			Area:      "web",
			TaskRef:   "task-programming-web",
			Title:     "Crear web estatica de agenda",
			Objective: "HTML estatico pequeno para agenda con lista, campo description/descripcion e i18n ES/EN. Sin JS separado; escribe ACK.",
			WriteSet: []string{
				"web/index.html",
			},
			RequiredTests: []string{"test -f web/index.html"},
		},
		{
			Area:      "docs",
			TaskRef:   "task-programming-docs",
			Title:     "Documentar uso minimo de agenda",
			Objective: "README breve con arranque, endpoints REST, payload title/starts_at/description y flujo web. Sin dependencias inventadas.",
			WriteSet: []string{
				"README.md",
			},
			RequiredTests: []string{"test -f README.md"},
		},
	}
	for index := range tasks {
		tasks[index].AgentRef = orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(tasks[index].TaskRef)
	}
	return tasks
}

func programmingTeamPacketV0(
	task programmingTeamTaskV0,
	correlationID string,
) orquestaruntime.AgentStartPacketV0 {
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     task.AgentRef,
		CorrelationID: correlationID,
		WorkOrderRef:  task.TaskRef,
		TargetModule:  "agenda-programming-" + task.Area,
		Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		CapacityLevel: "medium",
		Locale:        "es-ES",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:       task.TaskRef,
			Priority:      "normal",
			Title:         task.Title,
			Objective:     task.Objective,
			WriteSet:      task.WriteSet,
			RequiredTests: task.RequiredTests,
			DoneCriteria: []string{
				"El write-set se usa como alcance primario y cualquier ampliacion queda justificada.",
				"El codigo o web queda listo para revision.",
				"Cada fichero queda por debajo de 300 lineas.",
				"agent_ack.json escrito con status completed.",
			},
		},
		Context: programmingTeamContextV0(task),
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-programming-" + task.Area,
			AckRef:       "ack-ref-programming-" + task.Area,
			ReadinessRef: "readiness-ref-programming-" + task.Area,
		},
		Policies: []string{"write_set_closed", "ack_required"},
	}
}

func programmingTeamContextV0(task programmingTeamTaskV0) orquestacontext.ContextMaterializedBundleV0 {
	return orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-programming-" + task.Area,
		WorkOrderRef:  task.TaskRef,
		TargetModule:  "agenda-programming-" + task.Area,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:  "entry-ref-programming-" + task.Area,
			Layer:     orquestacontext.ContextLayerTaskContextV0,
			Kind:      orquestacontext.ContextEntryDocRefV0,
			SourceRef: "source-ref-programming-" + task.Area,
			Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:  true,
		}},
	}
}

func programmingTeamTaskByAgentV0(
	tasks []programmingTeamTaskV0,
	agentRef string,
) (programmingTeamTaskV0, bool) {
	for _, task := range tasks {
		if task.AgentRef == strings.TrimSpace(agentRef) {
			return task, true
		}
	}
	return programmingTeamTaskV0{}, false
}
