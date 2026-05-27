package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

var backlogExecutableTaskHeadingV0 = regexp.MustCompile(`^##\s+T([0-9]+)(?:\s|$)`)

type backlogTaskIDAllocationV0 struct {
	TaskIDRef      string
	ReservationRef string
	RangeStart     int
	RangeEnd       int
	ExistingID     string
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogTaskIDAllocationV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	sectionRef string,
) backlogTaskIDAllocationV0 {
	existing := backlogTaskIDFromSectionRefV0(sectionRef)
	start, end := planner.backlogTaskIDAllocationRangeV0(existing)
	seed := strings.Join([]string{
		request.RequestRef,
		request.CorrelationID,
		request.BacklogScanEpoch,
		sectionRef,
		strconv.Itoa(start),
		strconv.Itoa(end),
	}, "|")
	ref := "task-id-ref-backlog-" + backlogScanHashV0(seed)[:12]
	return backlogTaskIDAllocationV0{
		TaskIDRef:      ref,
		ReservationRef: "reservation-ref-backlog-task-id-" + backlogScanHashV0(ref + "|" + seed)[:12],
		RangeStart:     start,
		RangeEnd:       end,
		ExistingID:     existing,
	}
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogTaskIDAllocationRangeV0(existing string) (int, int) {
	if id := backlogTaskIDNumberV0(existing); id > 0 {
		return id, id
	}
	next := planner.nextBacklogTaskIDNumberV0()
	return next, next
}

func (planner idleSelfImprovementBacklogPlannerV0) nextBacklogTaskIDNumberV0() int {
	maxID := 0
	for _, section := range planner.backlogExecutableTaskIDSectionsV0() {
		if section.TaskNumber > maxID {
			maxID = section.TaskNumber
		}
	}
	if maxID <= 0 {
		return 1
	}
	return maxID + 1
}

func (allocation backlogTaskIDAllocationV0) ContextRefs() []string {
	return compactServerStackStringsV0([]string{
		"backlog_task_id_ref:" + allocation.TaskIDRef,
		"backlog_task_id_range:" + allocation.rangeTextV0(),
		"backlog_task_id_reservation_ref:" + allocation.ReservationRef,
	})
}

func (allocation backlogTaskIDAllocationV0) AcceptanceCriteria() []string {
	return []string{
		"backlog_task_id_ref:" + allocation.TaskIDRef,
		"backlog_task_id_range:" + allocation.rangeTextV0(),
		"antes de insertar nuevas tareas ## Txx, usar solo el rango reservado por task_id_ref; si falta rango, ACK failed con CONSULTA AL DIRECTOR",
		"si aparece backlog_task_number_collision, backlog_task_number_reservation_required, backlog_task_id_collision o backlog_task_id_allocation_gap durante el merge, bloquear y pedir rebase/CONSULTA AL DIRECTOR",
	}
}

func (allocation backlogTaskIDAllocationV0) rangeTextV0() string {
	if allocation.RangeStart == allocation.RangeEnd {
		return fmt.Sprintf("T%03d", allocation.RangeStart)
	}
	return fmt.Sprintf("T%03d-T%03d", allocation.RangeStart, allocation.RangeEnd)
}

type backlogExecutableTaskIDSectionV0 struct {
	TaskID          string
	TaskNumber      int
	TaskInstanceRef string
	Fingerprint     string
	SectionRef      string
	SourcePath      string
	SourceLine      int
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogExecutableTaskIDSectionsV0() []backlogExecutableTaskIDSectionV0 {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil
	}
	docs, err := planner.backlogSectionDocumentRefsV0()
	if err != nil {
		return nil
	}
	var out []backlogExecutableTaskIDSectionV0
	for _, rel := range docs {
		body, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			continue
		}
		out = append(out, backlogExecutableTaskIDSectionsFromDocumentV0(string(body), rel)...)
	}
	return out
}

func backlogExecutableTaskIDSectionsFromDocumentV0(content string, sourcePath string) []backlogExecutableTaskIDSectionV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []backlogExecutableTaskIDSectionV0
	for index, line := range lines {
		matches := backlogExecutableTaskHeadingV0.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) != 2 {
			continue
		}
		number, _ := strconv.Atoi(matches[1])
		sectionRef := idleSelfImprovementHeadingRefV0(line)
		fingerprint := backlogScanHashV0(sourcePath + "|" + strconv.Itoa(index+1) + "|" + strings.TrimSpace(line))[:12]
		out = append(out, backlogExecutableTaskIDSectionV0{
			TaskID:          fmt.Sprintf("T%03d", number),
			TaskNumber:      number,
			TaskInstanceRef: "task-instance-ref-backlog-" + fingerprint,
			Fingerprint:     fingerprint,
			SectionRef:      sectionRef,
			SourcePath:      sourcePath,
			SourceLine:      index + 1,
		})
	}
	return out
}

func backlogTaskIDFromSectionRefV0(sectionRef string) string {
	sectionRef = strings.TrimSpace(strings.ToLower(sectionRef))
	if !strings.HasPrefix(sectionRef, "t") {
		return ""
	}
	digits := strings.Builder{}
	for _, r := range strings.TrimPrefix(sectionRef, "t") {
		if r < '0' || r > '9' {
			break
		}
		digits.WriteRune(r)
	}
	if digits.Len() == 0 {
		return ""
	}
	return "T" + digits.String()
}

func backlogTaskIDNumberV0(taskID string) int {
	taskID = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(taskID), "T"))
	number, _ := strconv.Atoi(taskID)
	return number
}

func idleSelfImprovementBacklogTaskIDCollisionsV0(
	sections []backlogExecutableTaskIDSectionV0,
) []orquestaserver.BacklogScanCollisionV0 {
	byID := map[string][]backlogExecutableTaskIDSectionV0{}
	for _, section := range sections {
		if section.TaskID != "" {
			byID[section.TaskID] = append(byID[section.TaskID], section)
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var collisions []orquestaserver.BacklogScanCollisionV0
	for _, id := range ids {
		values := byID[id]
		if len(values) < 2 {
			continue
		}
		refs := make([]string, 0, len(values))
		for _, value := range values {
			refs = append(refs, value.TaskInstanceRef+"@"+value.SourcePath+":"+strconv.Itoa(value.SourceLine)+":hash:"+value.Fingerprint)
		}
		collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
			Code:         "backlog_task_number_collision",
			SectionRef:   values[0].SectionRef,
			TaskID:       id,
			InstanceRefs: backlogTaskInstanceRefsV0(values),
			Message:      backlogTaskIDAliasMessageV0(values) + ";sections=" + strings.Join(refs, ","),
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-backlog-task-id-allocation-lease",
				"evidence-ref-autoprogramming-backlog-task-number-collision",
				"evidence-ref-autoprogramming-backlog-task-id-alias-index",
			},
		})
	}
	return compactBacklogScanCollisionsV0(collisions)
}

func backlogTaskInstanceRefsV0(sections []backlogExecutableTaskIDSectionV0) []string {
	refs := make([]string, 0, len(sections))
	for _, section := range sections {
		refs = append(refs, section.TaskInstanceRef)
	}
	return compactServerStackStringsV0(refs)
}
