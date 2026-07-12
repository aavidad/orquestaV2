package main

import (
	"context"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func backlogRequestRefForTestV0(content string, index int) string {
	sections := parseIdleSelfImprovementBacklogSectionsV0(content)
	if index < 0 || index >= len(sections) {
		return ""
	}
	return idleSelfImprovementRequestRefForBacklogSectionV0(sections[index])
}

type fakeIdleSelfRunQueueV0 struct {
	candidates       []orquestarunqueue.RunSchedulingCandidateV0
	lastRead         orquestarunqueue.RunQueueReadRequestV0
	priorityCommands []orquestarunqueue.RunQueuePriorityCommandV0
}

func (fake *fakeIdleSelfRunQueueV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.lastRead = request
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

func (fake *fakeIdleSelfRunQueueV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.priorityCommands = append(fake.priorityCommands, command)
	candidate := orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        command.RunRef,
		AppRef:        command.AppRef,
		Status:        command.Status,
		PriorityScore: command.PriorityScore,
		UpdatedAt:     command.UpdatedAt,
		EvidenceRefs:  append([]string(nil), command.EvidenceRefs...),
	}
	for index := range fake.candidates {
		if fake.candidates[index].RunRef == command.RunRef {
			if candidate.AppRef == "" {
				candidate.AppRef = fake.candidates[index].AppRef
			}
			fake.candidates[index] = candidate
			return candidate, nil
		}
	}
	fake.candidates = append(fake.candidates, candidate)
	return candidate, nil
}
