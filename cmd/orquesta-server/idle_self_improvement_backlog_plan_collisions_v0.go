package main

import orquestaserver "orquesta/modulos/orquesta-server"

func (planner idleSelfImprovementBacklogPlannerV0) backlogPlanCollisionsV0(
	ackCollisions []orquestaserver.BacklogScanCollisionV0,
	canonicalCollisions []orquestaserver.BacklogScanCollisionV0,
	preflightCollisions []orquestaserver.BacklogScanCollisionV0,
	proposalCollisions []orquestaserver.BacklogScanCollisionV0,
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
	taskIDSections []backlogExecutableTaskIDSectionV0,
) []orquestaserver.BacklogScanCollisionV0 {
	return compactBacklogScanCollisionsV0(append(
		ackCollisions,
		append(canonicalCollisions, append(
			idleSelfImprovementBacklogSectionCollisionsV0(sections, excluded),
			append(planner.localDocIntegrityCollisionsV0(), append(
				idleSelfImprovementBacklogTaskIDCollisionsV0(taskIDSections),
				append(preflightCollisions, proposalCollisions...)...,
			)...)...,
		)...)...,
	))
}
