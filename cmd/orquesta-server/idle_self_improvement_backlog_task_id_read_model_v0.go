package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

var exactBacklogTaskIDRequestV0 = regexp.MustCompile(`(?i)^T[0-9]+$`)

func idleSelfImprovementAmbiguousTaskIDCollisionV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	sections []backlogExecutableTaskIDSectionV0,
) (orquestaserver.BacklogScanCollisionV0, bool) {
	byID := backlogTaskIDSectionsByIDV0(sections)
	for _, value := range append(append([]string(nil), request.KnownRequestRefs...), request.KnownRunRefs...) {
		taskID := idleSelfImprovementRequestedBacklogTaskIDV0(value)
		if taskID == "" || len(byID[taskID]) < 2 {
			continue
		}
		values := byID[taskID]
		return orquestaserver.BacklogScanCollisionV0{
			Code:         "backlog_duplicate_task_id_ambiguous",
			RequestRef:   strings.TrimSpace(value),
			SectionRef:   values[0].SectionRef,
			TaskID:       taskID,
			InstanceRefs: backlogTaskInstanceRefsV0(values),
			Message:      "task_id=" + taskID + ";instances=" + strings.Join(backlogTaskInstanceRefsV0(values), ","),
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-backlog-duplicate-task-id-read-model",
				"evidence-ref-autoprogramming-backlog-task-number-collision",
			},
		}, true
	}
	return orquestaserver.BacklogScanCollisionV0{}, false
}

func idleSelfImprovementAmbiguousTaskIDPlanResultV0(
	ackEvidenceRefs []string,
	collisions []orquestaserver.BacklogScanCollisionV0,
	collision orquestaserver.BacklogScanCollisionV0,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests:     []orquestaserver.IdleSelfImprovementRequestV0{},
		EvidenceRefs: compactServerStackStringsV0(append([]string{"evidence-ref-autoprogramming-backlog-duplicate-task-id-read-model"}, ackEvidenceRefs...)),
		Collisions:   compactBacklogScanCollisionsV0(append(collisions, collision)),
		Message:      "backlog_duplicate_task_id_ambiguous",
	}
}

func backlogTaskIDSectionsByIDV0(
	sections []backlogExecutableTaskIDSectionV0,
) map[string][]backlogExecutableTaskIDSectionV0 {
	byID := map[string][]backlogExecutableTaskIDSectionV0{}
	for _, section := range sections {
		if section.TaskID == "" {
			continue
		}
		byID[section.TaskID] = append(byID[section.TaskID], section)
	}
	for id := range byID {
		sort.SliceStable(byID[id], func(i, j int) bool {
			if byID[id][i].SourcePath != byID[id][j].SourcePath {
				return byID[id][i].SourcePath < byID[id][j].SourcePath
			}
			return byID[id][i].SourceLine < byID[id][j].SourceLine
		})
	}
	return byID
}

func idleSelfImprovementRequestedBacklogTaskIDV0(value string) string {
	value = strings.TrimSpace(idleSelfImprovementNormalizeQueuedRequestRefV0(value))
	value = strings.TrimPrefix(value, "task_id:")
	value = strings.TrimPrefix(value, "backlog_task_id:")
	value = strings.TrimPrefix(value, "request-ref-autoprogramming-backlog-")
	if !exactBacklogTaskIDRequestV0.MatchString(value) {
		return ""
	}
	number := backlogTaskIDNumberV0(value)
	if number <= 0 {
		return ""
	}
	return fmt.Sprintf("T%03d", number)
}

func idleSelfImprovementBacklogTaskInstanceCriteriaV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	if strings.TrimSpace(section.TaskInstanceRef) == "" {
		return nil
	}
	return []string{
		"backlog_task_instance_ref:" + section.TaskInstanceRef,
		"si el caller referencia solo " + backlogTaskIDFromSectionRefV0(section.Ref) +
			" y ese Txx tiene duplicados, bloquear con backlog_duplicate_task_id_ambiguous",
	}
}

func idleSelfImprovementBacklogTaskInstanceContextRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	taskID := backlogTaskIDFromSectionRefV0(section.Ref)
	return compactServerStackStringsV0([]string{
		"backlog_task_id:" + taskID,
		"backlog_task_instance_ref:" + section.TaskInstanceRef,
		"backlog_task_instance_source:" + firstNonEmptyServerStackV0(section.SourcePath, idleSelfImprovementBacklogDocRelV0) +
			":" + strconv.Itoa(section.SourceLine),
	})
}
