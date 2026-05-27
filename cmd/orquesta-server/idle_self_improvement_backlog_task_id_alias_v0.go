package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const backlogDuplicateTaskIDAmbiguousV0 = "backlog_duplicate_task_id_ambiguous"

type backlogTaskIDAliasV0 struct {
	Index, SourceLine   int
	TaskID, InstanceRef string
	Alias, SourcePath   string
	SectionRef          string
	CollisionCount      int
}

func idleSelfImprovementAnnotateBacklogTaskIDAliasIndexV0(
	sections []idleSelfImprovementBacklogSectionV0,
) {
	entries := backlogTaskIDAliasEntriesForSectionsV0(sections)
	for _, entry := range entries {
		sections[entry.Index].TaskID = entry.TaskID
		sections[entry.Index].TaskAlias = entry.Alias
		sections[entry.Index].TaskIDCollisionCount = entry.CollisionCount
		if entry.CollisionCount > 1 {
			sections[entry.Index].TaskIDIssue = backlogDuplicateTaskIDAmbiguousV0
		}
	}
}

func idleSelfImprovementWithBacklogTaskIDAliasV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementBacklogTaskIDAliasContextRefsV0(section)...,
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(
		request.AcceptanceCriteria,
		idleSelfImprovementBacklogTaskIDAliasCriteriaV0(section)...,
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(
		request.EvidenceRefs,
		idleSelfImprovementBacklogTaskIDAliasEvidenceRefsV0(section)...,
	))
	return request
}

func idleSelfImprovementBacklogTaskIDAliasContextRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	if section.TaskID == "" || section.TaskAlias == "" {
		return nil
	}
	refs := []string{
		"backlog_task_id:" + section.TaskID,
		"backlog_task_alias:" + section.TaskAlias,
		"backlog_task_entry_ref:" + section.TaskInstanceRef,
	}
	if section.TaskIDCollisionCount > 1 {
		refs = append(refs,
			"backlog_task_id_collision_count:"+strconv.Itoa(section.TaskIDCollisionCount),
			"backlog_task_id_collision_reason:"+section.TaskIDIssue,
		)
	}
	return refs
}

func idleSelfImprovementBacklogTaskIDAliasCriteriaV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	if section.TaskIDCollisionCount < 2 {
		return nil
	}
	return []string{
		"backlog_task_alias:" + section.TaskAlias,
		"backlog_task_entry_ref:" + section.TaskInstanceRef,
		"si cierre, ACK, stats, roadmap o MCP reciben solo " + section.TaskID + ", responder " + backlogDuplicateTaskIDAmbiguousV0 + " y exigir task_entry_ref, alias canonico o decision del director",
	}
}

func idleSelfImprovementBacklogTaskIDAliasEvidenceRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	if section.TaskID == "" {
		return nil
	}
	refs := []string{"evidence-ref-autoprogramming-backlog-task-id-alias-index"}
	if section.TaskIDCollisionCount > 1 {
		refs = append(refs, "evidence-ref-autoprogramming-backlog-task-id-ambiguous-alias")
	}
	return refs
}

func backlogTaskIDAliasEntriesForSectionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
) []backlogTaskIDAliasV0 {
	entries := make([]backlogTaskIDAliasV0, 0, len(sections))
	for index, section := range sections {
		taskID := backlogTaskIDForHeadingV0(section.Heading)
		if taskID == "" {
			continue
		}
		entries = append(entries, backlogTaskIDAliasV0{
			Index:       index,
			TaskID:      taskID,
			InstanceRef: section.TaskInstanceRef,
			SourcePath:  firstNonEmptyServerStackV0(section.SourcePath, idleSelfImprovementBacklogDocRelV0),
			SourceLine:  section.SourceLine,
			SectionRef:  section.Ref,
		})
	}
	backlogTaskIDAssignAliasesV0(entries)
	return entries
}

func backlogTaskIDAliasMessageV0(values []backlogExecutableTaskIDSectionV0) string {
	entries := make([]backlogTaskIDAliasV0, 0, len(values))
	for index, value := range values {
		entries = append(entries, backlogTaskIDAliasV0{
			Index:       index,
			TaskID:      value.TaskID,
			InstanceRef: value.TaskInstanceRef,
			SourcePath:  value.SourcePath,
			SourceLine:  value.SourceLine,
			SectionRef:  value.SectionRef,
		})
	}
	backlogTaskIDAssignAliasesV0(entries)
	taskID := ""
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		taskID = firstNonEmptyServerStackV0(taskID, entry.TaskID)
		parts = append(parts, entry.Alias+"="+entry.InstanceRef+"@"+entry.SourcePath+":"+strconv.Itoa(entry.SourceLine))
	}
	return "task_id=" + taskID + ";reason=" + backlogDuplicateTaskIDAmbiguousV0 + ";aliases=" + strings.Join(parts, ",")
}

func backlogTaskIDAssignAliasesV0(entries []backlogTaskIDAliasV0) {
	byID := map[string][]int{}
	for index, entry := range entries {
		byID[entry.TaskID] = append(byID[entry.TaskID], index)
	}
	for _, indexes := range byID {
		sort.SliceStable(indexes, func(i, j int) bool {
			left, right := entries[indexes[i]], entries[indexes[j]]
			if left.SourcePath != right.SourcePath {
				return left.SourcePath < right.SourcePath
			}
			if left.SourceLine != right.SourceLine {
				return left.SourceLine < right.SourceLine
			}
			return left.SectionRef < right.SectionRef
		})
		for ordinal, entryIndex := range indexes {
			entries[entryIndex].CollisionCount = len(indexes)
			entries[entryIndex].Alias = entries[entryIndex].TaskID
			if len(indexes) > 1 {
				entries[entryIndex].Alias = fmt.Sprintf("%s#%02d", entries[entryIndex].TaskID, ordinal+1)
			}
		}
	}
}

func backlogTaskIDForHeadingV0(heading string) string {
	matches := backlogExecutableTaskHeadingV0.FindStringSubmatch("## " + strings.TrimSpace(heading))
	if len(matches) != 2 {
		return ""
	}
	number, _ := strconv.Atoi(matches[1])
	if number <= 0 {
		return ""
	}
	return fmt.Sprintf("T%03d", number)
}
