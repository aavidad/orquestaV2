package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/i18n"
	sdkcommands "orquesta/sdk/commands"
)

func TestLocalTokenManifestTwoCLIsKeepExplicitRBACAndCriticalSeparationAfterRestart(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	secretsRoot := filepath.Join(root, "secrets")
	if err := os.Mkdir(secretsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(secretsRoot, "local-principals.json")
	approverToken := filepath.Join(secretsRoot, "local-approver.token")
	if _, err := localtoken.Open(approverToken); err != nil {
		t.Fatalf("provision secondary token: %v", err)
	}
	manifest := `{"schema_version":1,"document_type":"orquesta.local_token_principals","principals":[{"principal_ref":"actor:local-approver","actor_ref":"actor:local-approver","token_path":"local-approver.token"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath,
		"local_token_path = "+strconv.Quote(filepath.Join(secretsRoot, "local-owner.token")),
		"local_token_path = "+strconv.Quote(filepath.Join(secretsRoot, "local-owner.token"))+"\n"+
			"local_principals_manifest_path = "+strconv.Quote(manifestPath)+"\n"+
			"local_principals_manifest_max_bytes = 4096\n"+
			"local_principals_manifest_max_entries = 8",
	)

	var launches atomic.Int64
	first := buildMultiPrincipalRuntime(t, configPath, &launches)
	ownerToken := filepath.Join(secretsRoot, "local-owner.token")
	assertReservedExecutionTokenDenied(t, first)

	denied := invokeLocalTokenCLI(t, first, approverToken, "request:multi-before-grant", `{}`,
		"system", "status")
	if denied.Failure == nil || denied.Failure.Code != sdkcommands.CodeForbidden {
		t.Fatalf("secondary gained ambient RBAC before grant: %+v", denied)
	}

	ownerAccess := testRuntimeAccess(t, first)
	submitted, err := first.Orchestrator().Submit(context.Background(), ownerAccess, application.SubmitRequest{
		RequestRef: "request:multi-critical-goal", Statement: "critical local multiprincipal proof", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:multi-critical", Key: "phase:multi-critical",
				TemplateRef: "phase-template:multi-critical",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "critical-work", Objective: "produce critical proof",
				Phase: "phase:multi-critical", Role: "role:worker",
				OutputContract:      goal.OutputContractEvidenceBundle,
				SecurityCriticality: governance.SecurityCriticalityCritical,
				ReasoningEffort:     governance.ReasoningEffortMedium,
			}},
		},
	})
	if err != nil {
		t.Fatalf("submit critical goal: %v", err)
	}
	if len(submitted.Record.EffectIntents) != 1 ||
		submitted.Record.EffectIntents[0].SecurityCriticality != governance.SecurityCriticalityCritical ||
		len(submitted.Record.EffectApprovals) != 0 {
		t.Fatalf("critical effect was weakened or auto-approved: %+v", submitted.Record)
	}
	intent := submitted.Record.EffectIntents[0]

	grantPayload := `{"target_principal_ref":"actor:local-approver","target_actor_ref":"actor:local-approver","target_kind":"human","target_method":"local_token","role":"project_owner","expected_revision":0}`
	grant := invokeLocalTokenCLI(t, first, ownerToken, "request:multi-grant-owner", grantPayload,
		"projects", "memberships", "grant")
	var grantData struct {
		Receipt struct {
			AuditRef string `json:"audit_ref"`
			Role     string `json:"role"`
			Target   string `json:"target_ref"`
		} `json:"receipt"`
	}
	if grant.Failure != nil || grant.AuditRef == "" || json.Unmarshal(grant.Data, &grantData) != nil ||
		grantData.Receipt.AuditRef == "" || grantData.Receipt.Role != "project_owner" ||
		grantData.Receipt.Target != "actor:local-approver" {
		t.Fatalf("explicit project-owner grant lacks receipt: %+v data=%s", grant, grant.Data)
	}

	decisionPayload := `{"goal_ref":` + strconv.Quote(submitted.Record.Goal.Ref().String()) +
		`,"intent_ref":` + strconv.Quote(intent.Ref) +
		`,"expected_intent_digest":` + strconv.Quote(intent.Digest) +
		`,"decision":"approved","reason":"independent critical local approval"}`
	self := invokeLocalTokenCLI(t, first, ownerToken, "request:multi-self-approval", decisionPayload,
		"effects", "decide")
	if self.Failure == nil || self.Failure.Code != sdkcommands.CodeForbidden {
		t.Fatalf("critical proposer self-approved: %+v", self)
	}

	approved := invokeLocalTokenCLI(t, first, approverToken, "request:multi-independent-approval",
		decisionPayload, "effects", "decide")
	if approved.Failure != nil || approved.AuditRef == "" {
		t.Fatalf("independent critical approval failed: %+v", approved)
	}
	var approvalData struct {
		ApprovalRef string `json:"approval_ref"`
		Decision    string `json:"decision"`
	}
	if err := json.Unmarshal(approved.Data, &approvalData); err != nil ||
		approvalData.ApprovalRef == "" || approvalData.Decision != "approved" {
		t.Fatalf("critical approval receipt invalid: %s err=%v", approved.Data, err)
	}
	terminal := waitTerminalGoal(t, first, submitted.Record.Goal.Ref())
	if terminal.Goal.State() != goal.GoalStateSucceeded || launches.Load() != 1 ||
		len(terminal.EffectApprovals) != 1 ||
		terminal.EffectApprovals[0].SecurityCriticality != governance.SecurityCriticalityCritical ||
		terminal.EffectApprovals[0].DecidedBy.String() != "actor:local-approver" {
		t.Fatalf("critical effect closure lost separation: state=%s launches=%d approvals=%+v",
			terminal.Goal.State(), launches.Load(), terminal.EffectApprovals)
	}
	assertSecretsAbsentFromEffectiveConfig(t, first, ownerToken, approverToken)
	shutdownRuntime(t, first)

	second := buildMultiPrincipalRuntime(t, configPath, &launches)
	t.Cleanup(func() { shutdownRuntime(t, second) })
	replayedGrant := invokeLocalTokenCLI(t, second, ownerToken, "request:multi-grant-owner", grantPayload,
		"projects", "memberships", "grant")
	replayedApproval := invokeLocalTokenCLI(t, second, approverToken,
		"request:multi-independent-approval", decisionPayload, "effects", "decide")
	if replayedGrant.AuditRef != grant.AuditRef || !bytes.Equal(replayedGrant.Data, grant.Data) ||
		replayedApproval.AuditRef != approved.AuditRef || !bytes.Equal(replayedApproval.Data, approved.Data) ||
		launches.Load() != 1 {
		t.Fatalf("restart replay drift: grant=%+v approval=%+v launches=%d",
			replayedGrant, replayedApproval, launches.Load())
	}
}

func buildMultiPrincipalRuntime(t *testing.T, configPath string, launches *atomic.Int64) *Runtime {
	t.Helper()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "local-token-multiprincipal-e2e",
		AgentFactory: countingFactory(launches),
	})
	if err != nil {
		t.Fatalf("build multiprincipal runtime: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start multiprincipal runtime: %v", err)
	}
	return runtime
}

func invokeLocalTokenCLI(
	t *testing.T,
	runtime *Runtime,
	tokenPath, requestRef, payload string,
	path ...string,
) sdkcommands.Result {
	t.Helper()
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--url", "http://" + runtime.Address(),
		"--credential-file", tokenPath,
		"--max-credential-bytes", "4096",
		"--max-response-bytes", "65536",
		"--timeout", "5s",
		"--request-ref", requestRef,
		"--project-ref", "project:default",
		"--payload", payload,
		"--",
	}
	arguments = append(arguments, path...)
	var stdout, stderr bytes.Buffer
	code := RunCommand(arguments, catalog, &stdout, &stderr)
	var result sdkcommands.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("CLI decode: code=%d stdout=%q stderr=%q err=%v", code, stdout.String(), stderr.String(), err)
	}
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stdout.Bytes(), token) || bytes.Contains(stderr.Bytes(), token) {
		t.Fatal("CLI output leaked bearer credential")
	}
	if (code == 0) != (result.Failure == nil) {
		t.Fatalf("CLI code/result mismatch: code=%d result=%+v stderr=%q", code, result, stderr.String())
	}
	return result
}

func assertReservedExecutionTokenDenied(t *testing.T, runtime *Runtime) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, runtime.MCPURL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization",
		"Bearer orqex1.ZXhlY3V0aW9uOmZha2U.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reserved execution-token namespace reached human providers: %d", response.StatusCode)
	}
}

func assertSecretsAbsentFromEffectiveConfig(t *testing.T, runtime *Runtime, tokenPaths ...string) {
	t.Helper()
	effective, err := os.ReadFile(runtime.config.ConfigEffectivePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range tokenPaths {
		token, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(effective, token) {
			t.Fatal("effective config contains local token material")
		}
	}
}
