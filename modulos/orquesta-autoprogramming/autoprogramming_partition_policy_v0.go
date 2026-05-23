package orquestaautoprogramming

import (
	"fmt"
	"strings"
	"unicode"
)

type AutoprogrammingAreaAliasV0 struct {
	Alias string `json:"alias"`
	Area  string `json:"area"`
}

type AutoprogrammingLiveWorkV0 struct {
	WorkRef  string   `json:"work_ref,omitempty"`
	TaskRef  string   `json:"task_ref,omitempty"`
	AgentRef string   `json:"agent_ref,omitempty"`
	Status   string   `json:"status,omitempty"`
	WriteSet []string `json:"write_set,omitempty"`
}

type AutoprogrammingPartitionPlanV0 struct {
	WriteSetByArea  map[string][]string                `json:"write_set_by_area,omitempty"`
	DependsOnByArea map[string][]string                `json:"depends_on_by_area,omitempty"`
	BlockedByArea   map[string][]string                `json:"blocked_by_area,omitempty"`
	Steps           []AutoprogrammingPartitionStepV0   `json:"steps,omitempty"`
	Repairs         []AutoprogrammingPartitionRepairV0 `json:"repairs,omitempty"`
}

type AutoprogrammingPartitionStepV0 struct {
	StepRef              string   `json:"step_ref"`
	Area                 string   `json:"area"`
	TaskRef              string   `json:"task_ref"`
	SourceTaskRefs       []string `json:"source_task_refs,omitempty"`
	WriteSet             []string `json:"write_set,omitempty"`
	DependsOn            []string `json:"depends_on,omitempty"`
	BlockedByLiveWorkRef []string `json:"blocked_by_live_work_ref,omitempty"`
	Status               string   `json:"status"`
}

type AutoprogrammingPartitionRepairV0 struct {
	Code          string   `json:"code"`
	Path          string   `json:"path,omitempty"`
	Areas         []string `json:"areas,omitempty"`
	SuggestedArea string   `json:"suggested_area,omitempty"`
	Message       string   `json:"message"`
}

func autoprogrammingPartitionWriteSetByAreaV0(
	request AutoprogrammingRequestV0,
	writeSet []string,
	groups []AutoprogrammingTaskGroupV0,
) (AutoprogrammingPartitionPlanV0, []AutoprogrammingRequestIssueV0) {
	plan := AutoprogrammingPartitionPlanV0{
		WriteSetByArea:  map[string][]string{},
		DependsOnByArea: map[string][]string{},
		BlockedByArea:   map[string][]string{},
	}
	if len(groups) == 1 {
		plan.WriteSetByArea[groups[0].Area] = append([]string(nil), writeSet...)
		return autoprogrammingCompletePartitionPlanV0(request, plan, groups), nil
	}

	sequenced := map[string][]string{}
	var issues []AutoprogrammingRequestIssueV0
	for _, path := range writeSet {
		matches := autoprogrammingWriteSetMatchingAreasV0(path, groups, request.AreaAliases)
		switch len(matches) {
		case 0:
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_unassigned",
				"write_set",
				"ruta sin area reparable: "+path,
			))
		case 1:
			plan.WriteSetByArea[matches[0]] = appendUniqueStringV0(plan.WriteSetByArea[matches[0]], path)
		default:
			for _, area := range matches {
				plan.WriteSetByArea[area] = appendUniqueStringV0(plan.WriteSetByArea[area], path)
			}
			sequenced[path] = matches
			plan.Repairs = append(plan.Repairs, AutoprogrammingPartitionRepairV0{
				Code:    "write_set_overlap_sequenced",
				Path:    path,
				Areas:   append([]string(nil), matches...),
				Message: "ruta compartida secuenciada entre areas compatibles",
			})
		}
	}
	for _, group := range groups {
		if len(plan.WriteSetByArea[group.Area]) == 0 {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_group_empty",
				"write_set",
				"area sin write-set: "+group.Area,
			))
		}
	}
	if len(issues) > 0 {
		return AutoprogrammingPartitionPlanV0{}, issues
	}
	plan = autoprogrammingApplySequencedPathDepsV0(request, plan, groups, sequenced)
	return autoprogrammingCompletePartitionPlanV0(request, plan, groups), nil
}

func autoprogrammingApplySequencedPathDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
	sequenced map[string][]string,
) AutoprogrammingPartitionPlanV0 {
	groupIndex := map[string]int{}
	for i, group := range groups {
		groupIndex[group.Area] = i
	}
	for _, areas := range sequenced {
		for i := 1; i < len(areas); i++ {
			prev := autoprogrammingProgrammableTaskRefV0(request.RequestRef, groupIndex[areas[i-1]])
			plan.DependsOnByArea[areas[i]] = appendUniqueStringV0(plan.DependsOnByArea[areas[i]], prev)
		}
	}
	return plan
}

func autoprogrammingCompletePartitionPlanV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	plan = autoprogrammingApplyLiveWorkDepsV0(request, plan, groups)
	for i, group := range groups {
		deps := plan.DependsOnByArea[group.Area]
		blocked := plan.BlockedByArea[group.Area]
		status := "ready"
		if len(blocked) > 0 {
			status = "postponed"
		} else if len(deps) > 0 {
			status = "sequenced"
		}
		plan.Steps = append(plan.Steps, AutoprogrammingPartitionStepV0{
			StepRef:              fmt.Sprintf("partition-step-%02d", i+1),
			Area:                 group.Area,
			TaskRef:              autoprogrammingProgrammableTaskRefV0(request.RequestRef, i),
			SourceTaskRefs:       append([]string(nil), group.TaskRefs...),
			WriteSet:             append([]string(nil), plan.WriteSetByArea[group.Area]...),
			DependsOn:            append([]string(nil), deps...),
			BlockedByLiveWorkRef: append([]string(nil), blocked...),
			Status:               status,
		})
	}
	return plan
}

func autoprogrammingApplyLiveWorkDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	for _, live := range request.LiveWorks {
		if !autoprogrammingLiveWorkIsActiveV0(live) {
			continue
		}
		liveRef := autoprogrammingLiveWorkDependencyRefV0(live)
		for _, group := range groups {
			if !autoprogrammingWriteSetsOverlapV0(plan.WriteSetByArea[group.Area], live.WriteSet) {
				continue
			}
			plan.DependsOnByArea[group.Area] = appendUniqueStringV0(plan.DependsOnByArea[group.Area], liveRef)
			plan.BlockedByArea[group.Area] = appendUniqueStringV0(plan.BlockedByArea[group.Area], liveRef)
		}
	}
	return plan
}

func autoprogrammingWriteSetMatchingAreasV0(
	path string,
	groups []AutoprogrammingTaskGroupV0,
	aliases []AutoprogrammingAreaAliasV0,
) []string {
	pathTokens := autoprogrammingCanonicalPathTokensV0(path)
	pathSegments := autoprogrammingCanonicalPathSegmentKeysV0(path)
	var matches []string
	for _, group := range groups {
		for _, areaTokens := range autoprogrammingAreaTokenSetsV0(group.Area, aliases) {
			if autoprogrammingPathTokensContainAreaV0(pathTokens, areaTokens) ||
				autoprogrammingPathSegmentsContainAreaV0(pathSegments, areaTokens) {
				matches = append(matches, group.Area)
				break
			}
		}
	}
	return matches
}

func autoprogrammingAreaTokenSetsV0(area string, aliases []AutoprogrammingAreaAliasV0) [][]string {
	normalizedArea := normalizeAutoprogrammingTaskAreaV0(area)
	out := [][]string{autoprogrammingCanonicalPathTokensV0(normalizedArea)}
	for _, alias := range aliases {
		if normalizeAutoprogrammingTaskAreaV0(alias.Area) != normalizedArea {
			continue
		}
		out = append(out, autoprogrammingCanonicalPathTokensV0(alias.Alias))
	}
	return out
}

func autoprogrammingCanonicalPathTokensV0(value string) []string {
	tokens := autoprogrammingNormalizedPathTokensV0(normalizeAutoprogrammingPathForMatchV0(value))
	for i, token := range tokens {
		tokens[i] = autoprogrammingCanonicalPathTokenV0(token)
	}
	return compactPathTokensV0(tokens)
}

func autoprogrammingCanonicalPathTokenV0(token string) string {
	switch token {
	case "application", "applications", "apps":
		return "app"
	case "services", "svc":
		return "service"
	}
	if len(token) > 3 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
		return strings.TrimSuffix(token, "s")
	}
	return token
}

func autoprogrammingNormalizedPathTokensV0(value string) []string {
	return compactPathTokensV0(strings.Split(value, "-"))
}

func compactPathTokensV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func autoprogrammingPathTokensContainAreaV0(pathTokens []string, areaTokens []string) bool {
	if len(pathTokens) == 0 || len(areaTokens) == 0 || len(areaTokens) > len(pathTokens) {
		return false
	}
	for start := 0; start <= len(pathTokens)-len(areaTokens); start++ {
		matched := true
		for offset := range areaTokens {
			if pathTokens[start+offset] != areaTokens[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func autoprogrammingPathSegmentsContainAreaV0(pathSegments []string, areaTokens []string) bool {
	areaKey := strings.Join(areaTokens, "")
	if areaKey == "" {
		return false
	}
	for _, segment := range pathSegments {
		if segment == areaKey {
			return true
		}
	}
	return false
}

func autoprogrammingCanonicalPathSegmentKeysV0(path string) []string {
	rawSegments := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(path)), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		key := strings.Join(autoprogrammingCanonicalPathTokensV0(segment), "")
		if key != "" {
			segments = appendUniqueStringV0(segments, key)
		}
	}
	return segments
}

func normalizeAutoprogrammingPathForMatchV0(value string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return unicode.ToLower(r)
		default:
			return '-'
		}
	}, value), "-")
}
