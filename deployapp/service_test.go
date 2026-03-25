package deployapp

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	commands []string
	failOn   string
}

func (f *fakeRunner) Run(_ context.Context, shell string) error {
	f.commands = append(f.commands, shell)
	if f.failOn != "" && strings.Contains(shell, f.failOn) {
		return errors.New("boom")
	}
	return nil
}

func TestPlanDockerSaveIncludesBuildTransferDeployHealthcheckAndRollback(t *testing.T) {
	plan, err := NewService().Plan(DockerRemoteSpec{
		ProyectoSlug: "demo",
		Servicio:     "web",
		SSHHost:      "srv.example.com",
		SSHUser:      "deploy",
		Image:        "registry/demo-web",
		Tag:          "2026.03.25",
		RollbackTag:  "2026.03.24",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Strategy != "docker_save" {
		t.Fatalf("strategy=%q", plan.Strategy)
	}
	if len(plan.Steps) != 8 {
		t.Fatalf("steps=%d, want 8", len(plan.Steps))
	}
	if plan.ReleaseDir != "/opt/orquesta/demo/releases/2026.03.25" {
		t.Fatalf("releaseDir=%q", plan.ReleaseDir)
	}
	if plan.ArtifactName == "" {
		t.Fatalf("artifact name vacío")
	}
	if !strings.Contains(plan.Steps[0].Shell, "docker build") {
		t.Fatalf("build step=%q", plan.Steps[0].Shell)
	}
	if !strings.Contains(plan.Steps[3].Shell, "scp") {
		t.Fatalf("transfer step=%q", plan.Steps[3].Shell)
	}
	if !plan.Steps[len(plan.Steps)-1].Rollback {
		t.Fatalf("last step should be rollback")
	}
}

func TestExecuteDryRunDoesNotRunShell(t *testing.T) {
	runner := &fakeRunner{}
	service := NewServiceWithRunner(runner)
	plan := &DockerRemotePlan{
		Steps: []Step{
			{Name: "build", Shell: "docker build ."},
			{Name: "healthcheck", Shell: "ssh host true"},
		},
	}
	result, err := service.Execute(context.Background(), plan, ExecuteOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("runner commands=%v, want none", runner.commands)
	}
	if len(result.Executed) != 2 || result.Executed[0] != "build" || result.Executed[1] != "healthcheck" {
		t.Fatalf("executed=%v", result.Executed)
	}
}

func TestExecuteTriggersRollbackOnFailureWhenConfigured(t *testing.T) {
	runner := &fakeRunner{failOn: "health"}
	service := NewServiceWithRunner(runner)
	plan := &DockerRemotePlan{
		Steps: []Step{
			{Name: "deploy_remote", Shell: "ssh host deploy"},
			{Name: "healthcheck", Shell: "ssh host health"},
			{Name: "rollback", Shell: "ssh host rollback", Rollback: true},
		},
	}
	result, err := service.Execute(context.Background(), plan, ExecuteOptions{AutoRollback: true})
	if err == nil {
		t.Fatalf("expected failure")
	}
	if result.FailedStep != "healthcheck" {
		t.Fatalf("failedStep=%q", result.FailedStep)
	}
	if !result.RollbackTriggered {
		t.Fatalf("rollback not triggered")
	}
	if got := strings.Join(runner.commands, "\n"); !strings.Contains(got, "rollback") {
		t.Fatalf("commands=%s", got)
	}
}
