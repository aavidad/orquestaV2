package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	mcpinterface "orquesta/internal/interfaces/mcp"
	"orquesta/internal/wizard/gaps"
)

func TestV23WizardGapsNoOpReplayIsExactAfterLaterMutationAndRestart(
	t *testing.T,
) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	first := buildV23IntakeRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(first.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()
	const intakeRef = "intake:v23-wizard-gaps-noop-replay"
	dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.create",
		"request:v23-wizard-gaps-noop-create",
		map[string]any{"intake_ref": intakeRef},
	)
	payload := wizardGapsPublicPayload(intakeRef, 1)
	seed := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.wizard.gaps.apply",
		"request:v23-wizard-gaps-noop-seed",
		payload,
	)
	seedView := decodeV23WizardGapsReplayView(t, seed)
	if seedView.Intake.Revision != 2 || !seedView.RequestRefReserved ||
		seedView.RequestOutcome.Kind != "intake_mutation" ||
		seedView.RequestOutcome.ReceiptRef != seedView.Intake.ReceiptRef {
		t.Fatalf("seed=%+v", seedView)
	}

	payload = wizardGapsPublicPayload(intakeRef, 2)
	const requestRef = "request:v23-wizard-gaps-noop-replay"
	noOp := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.wizard.gaps.apply",
		requestRef,
		payload,
	)
	noOpView := decodeV23WizardGapsReplayView(t, noOp)
	if noOpView.Intake.Revision != 2 ||
		noOpView.Intake.ReceiptRef != seedView.Intake.ReceiptRef ||
		!noOpView.RequestRefReserved ||
		noOpView.RequestOutcome.Kind != "wizard_gaps_noop" ||
		noOpView.RequestOutcome.ReceiptRef == "" ||
		noOpView.RequestOutcome.ReceiptRef == noOpView.Intake.ReceiptRef ||
		!noOpView.EvaluationReplayExact {
		t.Fatalf("no-op=%+v seed=%+v", noOpView, seedView)
	}
	later := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.apply",
		"request:v23-wizard-gaps-noop-later-mutation",
		map[string]any{
			"intake_ref": intakeRef, "expected_revision": 2,
			"origin": "form",
			"issues": []any{}, "questions": []any{},
			"choices": []any{map[string]any{
				"question_ref": "intake-question:wizard.u1",
				"option_ref":   "intake-option:wizard.u1.team",
			}},
		},
	)
	if laterView := decodeV23IntakeMutation(t, later); laterView.Revision != 3 {
		t.Fatalf("later=%+v", laterView)
	}
	shutdownRuntime(t, first)

	second := buildV23IntakeRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, second) })
	replayed := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.wizard.gaps.apply",
		requestRef,
		payload,
	)
	replayedView := decodeV23WizardGapsReplayView(t, replayed)
	if replayed.Failure != nil ||
		replayed.AuditRef != noOp.AuditRef ||
		!bytes.Equal(replayed.Data, noOp.Data) ||
		replayedView.Intake.Revision != 2 ||
		replayedView.Intake.ReceiptRef != noOpView.Intake.ReceiptRef ||
		!replayedView.RequestRefReserved ||
		replayedView.RequestOutcome != noOpView.RequestOutcome ||
		!replayedView.EvaluationReplayExact {
		t.Fatalf("replayed=%+v no_op=%+v", replayed, noOp)
	}
}

type v23WizardGapsReplayView struct {
	Intake struct {
		Revision   uint64 `json:"revision"`
		ReceiptRef string `json:"receipt_ref"`
	} `json:"intake"`
	EvaluationReplayExact bool `json:"evaluation_replay_exact"`
	RequestRefReserved    bool `json:"request_ref_reserved"`
	RequestOutcome        struct {
		Kind       string `json:"kind"`
		ReceiptRef string `json:"receipt_ref"`
	} `json:"request_outcome"`
}

func decodeV23WizardGapsReplayView(
	t *testing.T,
	result commandcore.Result,
) v23WizardGapsReplayView {
	t.Helper()
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("wizard gaps result=%+v", result)
	}
	var view v23WizardGapsReplayView
	if err := json.Unmarshal(result.Data, &view); err != nil {
		t.Fatalf("decode wizard gaps result=%s err=%v", result.Data, err)
	}
	return view
}

func wizardGapsPublicPayload(intakeRef string, revision uint64) map[string]any {
	return map[string]any{
		"intake_ref": intakeRef, "expected_revision": revision,
		"origin": "form",
		"facts": map[string]any{
			"surface":                 "server_service",
			"sharing_intent":          "shared",
			"corporate_identity":      "declared",
			"target_users":            "declared",
			"integration_auth":        "declared",
			"integration_criticality": "declared",
		},
		"pack_refs": []any{},
	}
}

func TestV23WizardGapsCLIAndMCPUseSharedSQLiteIntakeWriter(t *testing.T) {
	root := t.TempDir()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root), Version: "v23-wizard-gaps-public",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err = runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { shutdownRuntime(t, runtime) })
	principal, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()
	for _, surface := range []string{"cli", "mcp"} {
		created := dispatchV23IntakeCommand(
			t,
			runtime,
			principal,
			projectRef,
			"orquesta.intakes.create",
			"request:v23-wizard-gaps-public-create-"+surface,
			map[string]any{
				"intake_ref": "intake:v23-wizard-gaps-public-" + surface,
			},
		)
		if view := decodeV23IntakeMutation(t, created); view.Revision != 1 {
			t.Fatalf("%s created=%+v", surface, view)
		}
	}

	payload := map[string]any{
		"intake_ref":        "intake:v23-wizard-gaps-public-cli",
		"expected_revision": 1,
		"origin":            "form",
		"facts": map[string]any{
			"surface":                 "server_service",
			"sharing_intent":          "shared",
			"corporate_identity":      "declared",
			"target_users":            "declared",
			"integration_auth":        "declared",
			"integration_criticality": "declared",
		},
		"pack_refs": []any{},
	}
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	const cliRequestRef = "request:v23-wizard-gaps-public-cli-apply"
	cliResult := invokeRealCommandCLIRequest(
		t,
		runtime,
		root,
		"es",
		cliRequestRef,
		string(encodedPayload),
		"intakes",
		"wizard",
		"gaps",
		"apply",
	)
	cliCore := decodeSDKResult(t, cliResult)
	assertV23WizardGapsPublicResult(
		t,
		cliCore,
		"intake:v23-wizard-gaps-public-cli",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{
			Name:    "v23-wizard-gaps-public",
			Version: "1",
		},
		nil,
	)
	session, err := client.Connect(
		ctx,
		&sdkmcp.StreamableClientTransport{
			Endpoint: runtime.MCPURL(),
			HTTPClient: authorizedHTTPClient(
				t,
				root+"/secrets/local-owner.token",
			),
		},
		nil,
	)
	if err != nil {
		t.Fatalf("MCP connect: %v", err)
	}
	defer session.Close()
	mcpPayload := cloneWizardGapsPublicPayload(t, payload)
	mcpPayload["intake_ref"] = "intake:v23-wizard-gaps-public-mcp"
	called := callMCPTool(
		t,
		ctx,
		session,
		"orquesta.intakes.wizard.gaps.apply",
		map[string]any{
			"version":     "1",
			"request_ref": "request:v23-wizard-gaps-public-mcp-apply",
			"project_ref": projectRef,
			"payload":     mcpPayload,
		},
	)
	var mcpOutput mcpinterface.CommandToolOutput
	decodeMCPOutput(t, called, &mcpOutput)
	assertV23WizardGapsPublicResult(
		t,
		mcpOutput.Result,
		"intake:v23-wizard-gaps-public-mcp",
	)
	if called.IsError ||
		!reflect.DeepEqual(
			wizardGapsPublicEvaluation(t, cliCore),
			wizardGapsPublicEvaluation(t, mcpOutput.Result),
		) {
		t.Fatalf(
			"CLI/MCP evaluation differs; cli_failure=%+v mcp_failure=%+v is_error=%t",
			cliCore.Failure,
			mcpOutput.Result.Failure,
			called.IsError,
		)
	}
}

func assertV23WizardGapsPublicResult(
	t *testing.T,
	result commandcore.Result,
	intakeRef string,
) {
	t.Helper()
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("result=%+v", result)
	}
	var output struct {
		Intake struct {
			IntakeRef  string `json:"intake_ref"`
			ProjectRef string `json:"project_ref"`
			Revision   uint64 `json:"revision"`
			ReceiptRef string `json:"receipt_ref"`
		} `json:"intake"`
		Evaluation struct {
			SchemaVersion string `json:"schema_version"`
			Issues        []any  `json:"issues"`
			Questions     []any  `json:"questions"`
		} `json:"evaluation"`
		EvaluationReplayExact bool `json:"evaluation_replay_exact"`
		RequestRefReserved    bool `json:"request_ref_reserved"`
		RequestOutcome        struct {
			Kind       string `json:"kind"`
			ReceiptRef string `json:"receipt_ref"`
		} `json:"request_outcome"`
		InputDurability struct {
			ReceiptRef             string `json:"receipt_ref"`
			SourceIntakeReceiptRef string `json:"source_intake_receipt_ref"`
			Selections             string `json:"selections"`
			SelectionsDigest       string `json:"selections_digest"`
			Facts                  string `json:"facts"`
			FactsDigest            string `json:"facts_digest"`
			PackRefs               string `json:"pack_refs"`
			PackRefsDigest         string `json:"pack_refs_digest"`
		} `json:"input_durability"`
		EvaluatorIdentity struct {
			Schema         string `json:"schema"`
			Version        string `json:"version"`
			SemanticDigest string `json:"semantic_digest"`
		} `json:"evaluator_identity"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatalf("decode result=%s err=%v", result.Data, err)
	}
	wantIdentity := gaps.EvaluatorV1Identity()
	if output.Intake.IntakeRef != intakeRef ||
		output.Intake.ProjectRef != "project:default" ||
		output.Intake.Revision != 2 ||
		output.Intake.ReceiptRef == "" ||
		output.Evaluation.SchemaVersion != gaps.SchemaVersion ||
		len(output.Evaluation.Issues) == 0 ||
		len(output.Evaluation.Questions) == 0 ||
		!output.EvaluationReplayExact ||
		!output.RequestRefReserved ||
		output.RequestOutcome.Kind != "intake_mutation" ||
		output.RequestOutcome.ReceiptRef != output.Intake.ReceiptRef ||
		output.InputDurability.ReceiptRef == "" ||
		output.InputDurability.SourceIntakeReceiptRef == "" ||
		output.InputDurability.Selections != "wizard_gaps_input_receipt" ||
		output.InputDurability.SelectionsDigest == "" ||
		output.InputDurability.Facts != "wizard_gaps_input_receipt" ||
		output.InputDurability.FactsDigest == "" ||
		output.InputDurability.PackRefs != "wizard_gaps_input_receipt" ||
		output.InputDurability.PackRefsDigest == "" ||
		output.EvaluatorIdentity.Schema != wantIdentity.Schema ||
		output.EvaluatorIdentity.Version != wantIdentity.Version ||
		output.EvaluatorIdentity.SemanticDigest != wantIdentity.SemanticDigest {
		t.Fatalf("output=%+v", output)
	}
}

func cloneWizardGapsPublicPayload(
	t *testing.T,
	source map[string]any,
) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func wizardGapsPublicEvaluation(
	t *testing.T,
	result commandcore.Result,
) map[string]any {
	t.Helper()
	var output map[string]any
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatal(err)
	}
	delete(output, "intake")
	delete(output, "request_outcome")
	if durability, ok := output["input_durability"].(map[string]any); ok {
		delete(durability, "receipt_ref")
		delete(durability, "source_intake_receipt_ref")
	}
	return output
}
