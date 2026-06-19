package orquestaautoprogramming

import "testing"

func TestBuildAutoprogrammingProgrammableWorkV0NormalizesAreaAlias(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-web", Area: "Web Application"},
			{TaskRef: "task-ref-api", Area: "API"},
		}
		request.AreaAliases = []AutoprogrammingAreaAliasV0{
			{Alias: "web_app", Area: "web-application"},
		}
		request.WriteSet = []string{
			"apps/admin/web_app/page.go",
			"apps/admin/api/handler.go",
		}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	got := result.Work.Partition.WriteSetByArea["web-application"]
	assertStringsEqualV0(t, got, []string{"apps/admin/web_app/page.go"})
}

func TestBuildAutoprogrammingProgrammableWorkV0PropagaSkillRefsDeclaradas(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:   "task-ref-web",
			Area:      "Web Application",
			SkillRefs: []string{"skill-ref-catalogo-declarado-v0"},
		}}
		request.WriteSet = []string{"apps/admin/web_app/page.go"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if !stringsSliceContainsForAutoprogrammingTestV0(
		result.Work.Groups[0].Task.SkillRefs,
		"skill-ref-catalogo-declarado-v0",
	) {
		t.Fatalf("skill_refs=%v", result.Work.Groups[0].Task.SkillRefs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0PostponesLiveWorkOverlap(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-intake", Area: "Intake"},
			{TaskRef: "task-ref-review", Area: "Review"},
		}
		request.WriteSet = []string{
			"modulos/orquesta-app-director-intake/intake_policy.go",
			"modulos/orquesta-app-director-intake/review_policy.go",
		}
		request.LiveWorks = []AutoprogrammingLiveWorkV0{{
			TaskRef:  "task-ref-live-intake",
			Status:   "running",
			WriteSet: []string{"modulos/orquesta-app-director-intake/intake_policy.go"},
		}}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	intake := result.Work.Partition.Steps[0]
	if intake.Status != "postponed" ||
		len(intake.DependsOn) != 1 ||
		intake.DependsOn[0] != "task-ref-live-intake" {
		t.Fatalf("intake step=%+v", intake)
	}
	if len(result.Work.Groups[0].Task.DependsOn) != 1 {
		t.Fatalf("task depends_on=%+v", result.Work.Groups[0].Task.DependsOn)
	}
}

func TestValidateAutoprogrammingRequestV0RejectsUnsafeLiveWork(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.LiveWorks = []AutoprogrammingLiveWorkV0{{
			Status:   "running",
			WriteSet: []string{"../outside"},
		}}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "live_work_ref_missing")
	assertAutoprogrammingRequestIssueV0(t, result, "live_work_write_set_path_invalid")
}
