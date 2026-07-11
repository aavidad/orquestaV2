package orquestaautoprogramming

import (
	"context"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type MaterialProgressEvidenceRequestV0 struct {
	State  orquestagoal.GoalWorkStateV0
	Result orquestagoal.GoalWorkResultV0
}

type MaterialProgressEvidenceV0 struct {
	Verified           bool
	MaterialClass      MaterialProgressClassV0
	BaselineRef        string
	WriteSetSHA256     string
	ContextRevisionRef string
	EvidenceRefs       []string
}

type MaterialProgressEvidencePortV0 interface {
	ClassifyMaterialProgressV0(
		context.Context,
		MaterialProgressEvidenceRequestV0,
	) (MaterialProgressEvidenceV0, error)
}
