package orquestaautoprogramming

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type AutoprogrammingProgrammableWorkResultV1 struct {
	Accepted bool                              `json:"accepted"`
	Work     AutoprogrammingProgrammableWorkV1 `json:"work"`
	Issues   []AutoprogrammingRequestIssueV0   `json:"issues,omitempty"`
}

type AutoprogrammingProgrammableWorkV1 struct {
	SchemaVersion   string                                 `json:"schema_version"`
	AppKind         string                                 `json:"app_kind,omitempty"`
	Base            AutoprogrammingProgrammableWorkV0      `json:"base"`
	ProfileBindings []AutoprogrammingResolvedWorkProfileV1 `json:"profile_bindings,omitempty"`
}

type autoprogrammingSelectedWorkProfileV1 struct {
	kind                 orquestacoreworkflow.WorkProfileKindV0
	source               string
	functionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0
	skillRefs            []string
}

func BuildAutoprogrammingProgrammableWorkV1(
	request AutoprogrammingRequestV1,
) AutoprogrammingProgrammableWorkResultV1 {
	validation := ValidateAutoprogrammingRequestV1(request)
	if !validation.Accepted {
		return AutoprogrammingProgrammableWorkResultV1{
			Accepted: false,
			Work:     autoprogrammingProgrammableWorkSkeletonV1(request),
			Issues:   append([]AutoprogrammingRequestIssueV0(nil), validation.Issues...),
		}
	}
	partition, partitionIssues := autoprogrammingPartitionWriteSetByAreaV0(
		request.AutoprogrammingRequestV0,
		validation.WriteSet,
		validation.Groups,
	)
	if len(partitionIssues) > 0 {
		return AutoprogrammingProgrammableWorkResultV1{
			Accepted: false,
			Work:     autoprogrammingProgrammableWorkSkeletonV1(request),
			Issues:   partitionIssues,
		}
	}
	work := autoprogrammingProgrammableWorkSkeletonV1(request)
	work.Base.Partition = partition
	for i, group := range validation.Groups {
		selected := autoprogrammingWorkProfileForGroupV1(request, group)
		profile, task, issue := autoprogrammingWorkflowTaskForGroupV1(
			request.AutoprogrammingRequestV0,
			group,
			partition.WriteSetByArea[group.Area],
			partition.DependsOnByArea[group.Area],
			validation.RequiredTests,
			i,
			selected,
		)
		if issue.Code != "" {
			return AutoprogrammingProgrammableWorkResultV1{
				Accepted: false,
				Work:     work,
				Issues:   []AutoprogrammingRequestIssueV0{issue},
			}
		}
		work.Base.Groups = append(work.Base.Groups, AutoprogrammingProgrammableGroupV0{
			Area:             group.Area,
			TaskRefs:         append([]string(nil), group.TaskRefs...),
			WriteSet:         append([]string(nil), task.WriteSet...),
			RequiredTests:    append([]string(nil), task.RequiredTests...),
			AcceptanceChecks: autoprogrammingAcceptanceChecksForGroupV0(group),
			DependsOn:        append([]string(nil), task.DependsOn...),
			BlockedBy:        append([]string(nil), partition.BlockedByArea[group.Area]...),
			Profile:          profile,
			Task:             task,
		})
		work.Base.Profiles = append(work.Base.Profiles, profile)
		work.Base.Tasks = append(work.Base.Tasks, task)
		work.ProfileBindings = append(work.ProfileBindings, AutoprogrammingResolvedWorkProfileV1{
			Area:        group.Area,
			TaskRefs:    append([]string(nil), group.TaskRefs...),
			ProfileKind: selected.kind,
			Source:      selected.source,
		})
	}
	goalSpecs, goalIssues := BuildAutoprogrammingGoalWorkSpecsV0(work.Base)
	if len(goalIssues) > 0 {
		return AutoprogrammingProgrammableWorkResultV1{
			Accepted: false,
			Work:     work,
			Issues:   goalIssues,
		}
	}
	work.Base.GoalSpecs = goalSpecs
	work.Base = autoprogrammingGoalReadyWithoutLegacyWorkflowSurfaceV0(work.Base)
	return AutoprogrammingProgrammableWorkResultV1{Accepted: true, Work: work}
}

func autoprogrammingProgrammableWorkSkeletonV1(
	request AutoprogrammingRequestV1,
) AutoprogrammingProgrammableWorkV1 {
	return AutoprogrammingProgrammableWorkV1{
		SchemaVersion: AutoprogrammingProgrammableWorkSchemaVersionV1,
		AppKind:       normalizeAutoprogrammingAppKindV1(request.AppKind),
		Base:          autoprogrammingProgrammableWorkSkeletonV0(request.AutoprogrammingRequestV0),
	}
}

func autoprogrammingWorkProfileForGroupV1(
	request AutoprogrammingRequestV1,
	group AutoprogrammingTaskGroupV0,
) autoprogrammingSelectedWorkProfileV1 {
	selected := autoprogrammingSelectedWorkProfileV1{
		kind:   orquestacoreworkflow.WorkProfileImplementationV0,
		source: "default-v0",
	}
	if taskProfile, ok := autoprogrammingTaskWorkProfileForGroupV1(request, group); ok {
		return taskProfile
	}
	if areaProfile, ok := autoprogrammingAreaWorkProfileForGroupV1(request, group); ok {
		return areaProfile
	}
	if appProfile, ok := autoprogrammingDefaultWorkProfileForAppV1(request); ok {
		return appProfile
	}
	return selected
}

func autoprogrammingTaskWorkProfileForGroupV1(
	request AutoprogrammingRequestV1,
	group AutoprogrammingTaskGroupV0,
) (autoprogrammingSelectedWorkProfileV1, bool) {
	taskRefs := map[string]bool{}
	for _, ref := range group.TaskRefs {
		taskRefs[strings.TrimSpace(ref)] = true
	}
	for _, profile := range request.WorkProfiles {
		if !autoprogrammingWorkProfileAppliesToAppV1(request, profile) {
			continue
		}
		taskRef := strings.TrimSpace(profile.TaskRef)
		if taskRef != "" && taskRefs[taskRef] {
			return autoprogrammingSelectedWorkProfileFromBindingV1(profile, "task_ref:"+taskRef), true
		}
	}
	return autoprogrammingSelectedWorkProfileV1{}, false
}

func autoprogrammingAreaWorkProfileForGroupV1(
	request AutoprogrammingRequestV1,
	group AutoprogrammingTaskGroupV0,
) (autoprogrammingSelectedWorkProfileV1, bool) {
	for _, profile := range request.WorkProfiles {
		if !autoprogrammingWorkProfileAppliesToAppV1(request, profile) {
			continue
		}
		area := normalizeAutoprogrammingTaskAreaV0(profile.Area)
		if area != "" && area == group.Area {
			return autoprogrammingSelectedWorkProfileFromBindingV1(profile, "area:"+area), true
		}
	}
	return autoprogrammingSelectedWorkProfileV1{}, false
}

func autoprogrammingDefaultWorkProfileForAppV1(
	request AutoprogrammingRequestV1,
) (autoprogrammingSelectedWorkProfileV1, bool) {
	for _, profile := range request.WorkProfiles {
		if !autoprogrammingWorkProfileAppliesToAppV1(request, profile) ||
			strings.TrimSpace(profile.TaskRef) != "" ||
			strings.TrimSpace(profile.Area) != "" {
			continue
		}
		return autoprogrammingSelectedWorkProfileFromBindingV1(profile, "app_kind"), true
	}
	return autoprogrammingSelectedWorkProfileV1{}, false
}

func autoprogrammingWorkProfileAppliesToAppV1(
	request AutoprogrammingRequestV1,
	profile AutoprogrammingWorkProfileV1,
) bool {
	profileApp := normalizeAutoprogrammingAppKindV1(profile.AppKind)
	return profileApp == "" || profileApp == normalizeAutoprogrammingAppKindV1(request.AppKind)
}

func autoprogrammingSelectedWorkProfileFromBindingV1(
	profile AutoprogrammingWorkProfileV1,
	source string,
) autoprogrammingSelectedWorkProfileV1 {
	return autoprogrammingSelectedWorkProfileV1{
		kind:   orquestacoreworkflow.NormalizeWorkProfileKindV0(profile.ProfileKind),
		source: source,
		functionContractRefs: append(
			[]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil),
			profile.FunctionContractRefs...,
		),
		skillRefs: compactStringsV0(profile.SkillRefs),
	}
}
