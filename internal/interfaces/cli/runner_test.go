package cliinterface

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
	sdkcommands "orquesta/sdk/commands"
)

type captureInvoker struct {
	request sdkcommands.Request
	result  sdkcommands.Result
}

func (invoker *captureInvoker) Invoke(_ context.Context, request sdkcommands.Request) (sdkcommands.Result, error) {
	invoker.request = request
	return invoker.result, nil
}

func TestCLIResolvesOnlyGeneratedPathAndPreservesCanonicalEnvelope(t *testing.T) {
	invoker := &captureInvoker{result: sdkcommands.Result{Failure: &sdkcommands.Failure{Code: sdkcommands.CodeConflict, MessageKey: "error.conflict"}, AuditRef: "audit:cli"}}
	runner, err := New(invoker)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(context.Background(), []string{"system", "status"}, "request:cli", "project:cli", "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if invoker.request.CommandID != "orquesta.system.status" || invoker.request.Version != "1" || result.Failure == nil || result.Failure.Code != sdkcommands.CodeConflict || result.AuditRef != "audit:cli" {
		t.Fatalf("request=%+v result=%+v", invoker.request, result)
	}
}

func TestCLIBindingsExactlyMatchCanonicalDefinitions(t *testing.T) {
	runner, err := New(&captureInvoker{})
	if err != nil {
		t.Fatal(err)
	}
	definitions := commandcore.CanonicalDefinitions()
	if len(runner.paths) != len(definitions) {
		t.Fatalf("paths=%d definitions=%d", len(runner.paths), len(definitions))
	}
	for _, definition := range definitions {
		path := strings.Join(definition.CLI.Path, " ")
		binding, ok := runner.paths[path]
		if !ok || binding.CommandID != definition.ID || binding.Version != definition.Version {
			t.Fatalf("path=%q binding=%+v definition=%+v", path, binding, definition)
		}
	}
}

func TestCommandBindingsUseOnlyCanonicalConfigAndRequireExplicitCLIConnection(t *testing.T) {
	if _, err := New(nil); err == nil || err.Error() != "cli.command_invoker_required" {
		t.Fatalf("cli err=%v", err)
	}
	for _, config := range []sdkcommands.Config{
		{HTTPClient: http.DefaultClient, MaxResponseBytes: 4096},
		{BaseURL: "http://127.0.0.1", MaxResponseBytes: 4096},
		{BaseURL: "http://127.0.0.1", HTTPClient: http.DefaultClient},
	} {
		if _, err := sdkcommands.New(config); err == nil || err.Error() != "commandsdk.config_invalid" {
			t.Fatalf("sdk config=%+v err=%v", config, err)
		}
	}
}
