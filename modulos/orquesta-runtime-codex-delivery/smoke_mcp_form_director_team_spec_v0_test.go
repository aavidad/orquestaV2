package orquestaruntimecodexdelivery

import (
	"context"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaweb "orquesta/modulos/orquesta-web"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

type mcpFormDirectorTeamCodexSpecResolverV0 struct {
	Config codexRealSmokeConfigV0
	Form   orquestaweb.WebNuevaAppFormV0
}

func (resolver mcpFormDirectorTeamCodexSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	_ context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	taskRef := strings.TrimSpace(inbound.Payload.TaskRef)
	area := mcpFormDirectorTeamAreaV0(inbound.Payload.Role, taskRef)
	runtimeDir, err := CodexReceiptAgentRuntimeDirV0(resolver.Config.RuntimeWorkDir, inbound.Payload.RunID, agentRef)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	profile := codexRealSmokeProfileV0(resolver.Config)
	profile.RuntimeWorkDir = runtimeDir
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	spec.CorrelationID = strings.TrimSpace(inbound.CorrelationID)
	spec.ProfileRef = "profile-ref-" + area
	spec.ConnectorRef = "connector-ref-" + area
	spec.Command.CommandRef = "command-ref-" + area
	spec.Command.ExecutableRef = "exec-ref-" + area
	spec.Command.WorkingDirRef = "workdir-ref-" + area
	spec.AgentPacket = mcpFormDirectorTeamPacketV0(agentRef, taskRef, area, spec.CorrelationID, resolver.Form)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
	}, nil
}

func mcpFormDirectorTeamPacketV0(
	agentRef string,
	taskRef string,
	area string,
	correlationID string,
	form orquestaweb.WebNuevaAppFormV0,
) orquestaruntime.AgentStartPacketV0 {
	task := mcpFormDirectorTeamTaskV0(taskRef, area, form)
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: correlationID,
		WorkOrderRef:  taskRef,
		TargetModule:  "agenda-" + area,
		Phase:         string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		CapacityLevel: "high",
		Locale:        "es-ES",
		Task:          task,
		Context: orquestacontext.ContextMaterializedBundleV0{
			SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
			BundleRef:     "bundle-ref-agenda-" + area,
			WorkOrderRef:  taskRef,
			TargetModule:  "agenda-" + area,
			Entries: []orquestacontext.ContextMaterializedEntryV0{{
				EntryRef:  "entry-ref-agenda-" + area,
				Layer:     orquestacontext.ContextLayerTaskContextV0,
				Kind:      orquestacontext.ContextEntryDocRefV0,
				SourceRef: "source-ref-agenda-form-" + area,
				Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
				Required:  true,
			}},
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-agenda-" + area,
			AckRef:       "ack-ref-agenda-" + area,
			ReadinessRef: "readiness-ref-agenda-" + area,
		},
		Policies: []string{"write_set_closed", "ack_required"},
	}
}

func mcpFormDirectorTeamTaskV0(
	taskRef string,
	area string,
	form orquestaweb.WebNuevaAppFormV0,
) orquestaruntime.AgentStartTaskV0 {
	task := orquestaruntime.AgentStartTaskV0{
		TaskRef:   taskRef,
		Priority:  "alta",
		Title:     "Definir " + area + " de " + form.Nombre,
		Objective: "Analiza solo tu area para una agenda con Go, web, i18n y persistencia por puerto. Entrega documentacion breve, accionable y profesional.",
		WriteSet:  []string{"docs/" + area + ".md"},
		DoneCriteria: []string{
			"Documento del area creado dentro del write-set.",
			"El documento incluye decisiones, riesgos y consultas al director si faltan datos.",
			"agent_ack.json escrito con status completed.",
		},
	}
	switch area {
	case "director":
		task.WriteSet = []string{"docs/arquitectura.md", "docs/plan_microtareas.md"}
		task.Title = "Dirigir arquitectura inicial de " + form.Nombre
		task.Objective = mcpFormDirectorSmokeObjectiveV0(form)
	case "api":
		task.Objective = "Define la interfaz publica y contratos de uso sin elegir persistencia concreta ni proveedor de ejecucion."
	case "persistencia":
		task.WriteSet = []string{"docs/persistencia.md"}
		task.Objective = "Define persistencia solo como puerto y contrato de conector, sin elegir base concreta por defecto."
	}
	return task
}

func mcpFormDirectorTeamAreaV0(role string, taskRef string) string {
	value := strings.TrimSpace(role + "-" + taskRef)
	switch {
	case strings.Contains(value, "director_web"):
		return "web"
	case strings.Contains(value, "director_api"):
		return "api"
	case strings.Contains(value, "director_persistencia"):
		return "persistencia"
	default:
		return "director"
	}
}
