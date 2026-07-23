package mcpinterface

import "testing"

func TestMCPPlanCarriesStructuredRequiredTestsIntoGoalView(t *testing.T) {
	server, _, _ := newTestInterface(t, 64*1024)
	session := serveOfficialClient(t, server)
	created := callCreateGoal(t, session, map[string]any{
		"project_ref": "project:local", "request_ref": "request:mcp-required-tests",
		"statement": "write code with declared tests", "confirm": true,
		"plan": map[string]any{
			"phases": []any{map[string]any{
				"ref": "phase-instance:mcp-required-tests", "key": "phase:mcp-required-tests",
				"template_ref": "phase-template:mcp-required-tests",
			}},
			"work_items": []any{map[string]any{
				"key": "writer", "objective": "write exact code", "phase": "phase:mcp-required-tests",
				"role": "role:worker", "dependencies": []any{}, "write_set": []any{"internal/mcp"},
				"council_policy": "skip_by_operator",
				"required_tests": []any{map[string]any{
					"ref": "required-test:mcp", "tool_ref": "tool:go-test",
					"arguments": []any{"./internal/interfaces/mcp"}, "working_directory": ".",
				}},
				"output_contract": "evidence_bundle",
			}},
		},
	})
	if !created.Created || created.Goal == nil || len(created.Goal.WorkItems) != 1 {
		t.Fatalf("create output=%+v", created)
	}
	if created.Goal.WorkItems[0].CouncilPolicy != "skip_by_operator" {
		t.Fatalf("Council policy view=%q", created.Goal.WorkItems[0].CouncilPolicy)
	}
	tests := created.Goal.WorkItems[0].RequiredTests
	if len(tests) != 1 || tests[0].Ref != "required-test:mcp" || tests[0].ToolRef != "tool:go-test" ||
		len(tests[0].Arguments) != 1 || tests[0].Arguments[0] != "./internal/interfaces/mcp" ||
		tests[0].WorkingDirectory != "." || len(tests[0].Digest) != 64 {
		t.Fatalf("required tests view=%+v", tests)
	}
}

func TestApplicationPlanDeepCopiesRequiredTestArguments(t *testing.T) {
	input := &PlanInput{WorkItems: []WorkItemInput{{
		CouncilPolicy: "required",
		RequiredTests: []RequiredTestInput{{
			Ref: "required-test:copy", ToolRef: "tool:test",
			Arguments: []string{"./..."}, WorkingDirectory: ".",
		}},
	}}}
	plan := applicationPlan(input)
	input.WorkItems[0].RequiredTests[0].Arguments[0] = "./changed/..."
	if got := plan.WorkItems[0].RequiredTests[0].Arguments[0]; got != "./..." {
		t.Fatalf("applicationPlan retained mutable input: %q", got)
	}
	if got := plan.WorkItems[0].CouncilPolicy; got != "required" {
		t.Fatalf("applicationPlan lost Council policy: %q", got)
	}
}
