// Package cliinterface projects canonical command paths onto the HTTP SDK.
package cliinterface

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commandcore "orquesta/internal/commands"
	sdkcommands "orquesta/sdk/commands"
)

var (
	ErrCommandRunnerUnavailable = errors.New("cli.command_runner_unavailable")
	ErrCommandUnknown           = errors.New("cli.command_unknown")
)

type commandBinding struct{ CommandID, Version, Path string }
type Invoker interface {
	Invoke(context.Context, sdkcommands.Request) (sdkcommands.Result, error)
}
type Runner struct {
	invoker Invoker
	paths   map[string]commandBinding
}

func New(invoker Invoker) (*Runner, error) {
	if invoker == nil {
		return nil, errors.New("cli.command_invoker_required")
	}
	definitions := commandcore.CanonicalDefinitions()
	paths := make(map[string]commandBinding, len(definitions))
	for _, definition := range definitions {
		binding := commandBinding{
			CommandID: definition.ID, Version: definition.Version, Path: strings.Join(definition.CLI.Path, " "),
		}
		if binding.CommandID == "" || binding.Version == "" || binding.Path == "" {
			return nil, errors.New("cli.command_binding_registry_invalid")
		}
		if _, duplicate := paths[binding.Path]; duplicate {
			return nil, errors.New("cli.command_binding_duplicate")
		}
		paths[binding.Path] = binding
	}
	return &Runner{invoker: invoker, paths: paths}, nil
}

func (runner *Runner) Run(ctx context.Context, path []string, requestRef, projectRef, claimedExecutionRef string, payload json.RawMessage) (sdkcommands.Result, error) {
	if runner == nil {
		return sdkcommands.Result{}, ErrCommandRunnerUnavailable
	}
	binding, ok := runner.paths[strings.Join(path, " ")]
	if !ok {
		return sdkcommands.Result{}, ErrCommandUnknown
	}
	return runner.invoker.Invoke(ctx, sdkcommands.Request{CommandID: binding.CommandID, Version: binding.Version, RequestRef: requestRef, ProjectRef: projectRef, ClaimedExecutionRef: claimedExecutionRef, Payload: payload})
}
