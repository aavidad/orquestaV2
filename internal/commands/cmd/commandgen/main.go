// Command commandgen compiles the canonical command registry into transport
// bindings. Generated files contain data only; runtime policy stays in commands.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
)

type generationMode string

const (
	modeCheck generationMode = "check"
	modeWrite generationMode = "write"
)

type document struct {
	SchemaVersion int          `json:"schema_version"`
	Revision      string       `json:"revision"`
	Commands      []definition `json:"commands"`
}

type definition struct {
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	Kind           string          `json:"kind"`
	Handler        string          `json:"handler"`
	Permission     string          `json:"permission"`
	Audience       string          `json:"audience"`
	ExecutionBound bool            `json:"execution_bound"`
	ReplayMode     string          `json:"replay_mode"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	DescriptionKey string          `json:"description_key"`
	ErrorCodes     []string        `json:"error_codes"`
	HTTP           struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"http"`
	MCP struct {
		Tool        string          `json:"tool"`
		Annotations *mcpAnnotations `json:"annotations"`
	} `json:"mcp"`
	CLI struct {
		Path []string `json:"path"`
	} `json:"cli"`
}

type mcpAnnotations struct {
	ReadOnly    *bool `json:"read_only"`
	Destructive *bool `json:"destructive"`
	Idempotent  *bool `json:"idempotent"`
	OpenWorld   *bool `json:"open_world"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	registry := flag.String("registry", "internal/commands/registry.json", "registry path relative to root")
	check := flag.Bool("check", false, "verify generated files without writing")
	write := flag.Bool("write", false, "atomically replace generated files")
	flag.Parse()
	if *check == *write {
		_, _ = fmt.Fprintln(os.Stderr, "commandgen.exactly_one_mode_required")
		os.Exit(2)
	}
	mode := modeCheck
	if *write {
		mode = modeWrite
	}
	if err := run(*root, *registry, mode); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, registryPath string, mode generationMode) error {
	if mode != modeCheck && mode != modeWrite {
		return errors.New("commandgen.mode_invalid")
	}
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(registryPath)))
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var registry document
	if err := decoder.Decode(&registry); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("commandgen.registry_invalid")
	}
	if registry.SchemaVersion != 1 || strings.TrimSpace(registry.Revision) == "" || len(registry.Commands) == 0 {
		return errors.New("commandgen.registry_invalid")
	}
	if err := validate(registry.Commands); err != nil {
		return err
	}
	semantic, err := canonicalRegistry(registry)
	if err != nil {
		return err
	}
	digestBytes := sha256.Sum256(semantic)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	outputs := map[string][]byte{
		"internal/commands/definitions_generated.go": generateDefinitions(registry, digest),
	}
	paths := make([]string, 0, len(outputs))
	formattedOutputs := make(map[string][]byte, len(outputs))
	for path := range outputs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		formatted, err := format.Source(outputs[relative])
		if err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		formattedOutputs[relative] = formatted
	}
	if mode == modeWrite {
		return writeGenerated(root, paths, formattedOutputs)
	}
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		current, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(current, formattedOutputs[relative]) {
			return fmt.Errorf("commandgen.generated_drift:%s", relative)
		}
	}
	return nil
}

func validate(definitions []definition) error {
	ids, handlers, bindings := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, item := range definitions {
		annotations := item.MCP.Annotations
		wantReplay := "application_receipt"
		if item.Kind == "query" {
			wantReplay = "read_reexecute"
		}
		if !strings.HasPrefix(item.ID, "orquesta.") || item.Version != "1" ||
			(item.Kind != "command" && item.Kind != "query") ||
			(item.Audience != "principal" && item.Audience != "execution") ||
			item.ExecutionBound != (item.Audience == "execution") || item.ReplayMode != wantReplay ||
			item.Handler == "" || handlerPermissions[item.Handler] != item.Permission ||
			identity.ValidatePermission(identity.Permission(item.Permission)) != nil ||
			commandcore.ValidateSchemaContract(item.InputSchema) != nil ||
			commandcore.ValidateSchemaContract(item.OutputSchema) != nil ||
			item.DescriptionKey != "command."+strings.TrimPrefix(item.ID, "orquesta.")+".description" ||
			!equalStrings(item.ErrorCodes, errorCodes) ||
			item.HTTP.Method != "POST" || item.HTTP.Path != "/api/v1/commands/"+item.ID ||
			annotations == nil || annotations.ReadOnly == nil ||
			annotations.Destructive == nil || annotations.Idempotent == nil ||
			annotations.OpenWorld == nil ||
			*annotations.ReadOnly != (item.Kind == "query") ||
			(*annotations.ReadOnly && *annotations.Destructive) ||
			!*annotations.Idempotent ||
			*annotations.OpenWorld ||
			item.MCP.Tool != item.ID || !equalStrings(item.CLI.Path, strings.Split(strings.TrimPrefix(item.ID, "orquesta."), ".")) {
			return errors.New("commandgen.definition_invalid")
		}
		if ids[item.ID] || handlers[item.Handler] {
			return errors.New("commandgen.definition_duplicate")
		}
		ids[item.ID], handlers[item.Handler] = true, true
		for _, binding := range []string{item.HTTP.Method + " " + item.HTTP.Path, item.MCP.Tool, strings.Join(item.CLI.Path, "\x00")} {
			if binding == "" || bindings[binding] {
				return errors.New("commandgen.binding_duplicate")
			}
			bindings[binding] = true
		}
	}
	return nil
}

var errorCodes = []string{"invalid_request", "unauthenticated", "forbidden", "not_found", "conflict", "unavailable", "internal"}

var handlerPermissions = map[string]string{
	"Submit": "goals.create", "Amend": "goals.amend", "GetGoal": "goals.get", "ListGoals": "goals.list",
	"GetArtifact": "artifacts.read", "Status": "project.status", "GrantMembership": "project.membership.manage",
	"RevokeMembership": "project.membership.manage", "ClaimDirector": "goals.direct", "RenewDirector": "goals.direct",
	"ProposeDirectorPlan": "goals.direct", "Control": "goals.direct", "DecideEffect": "effects.approve",
	"ReconcileTerminalAgentLaunch":            "effects.approve",
	"PreflightExpiredAgentLaunchContinuation": "effects.approve",
	"ConfirmExpiredAgentLaunchContinuation":   "effects.approve",
	"ListPendingChanges":                      "goals.list", "IntegrateChange": "changes.integrate", "AdmitMailbox": "goals.direct",
	"ClaimMailbox": "goals.get", "MarkMailboxDelivered": "goals.get", "ConsumeMailbox": "goals.get",
	"GetMailbox": "goals.get", "ListMailbox": "goals.get", "AcknowledgeMailbox": "goals.get", "BlockMailbox": "goals.get",
	"OpenCouncilRound": "goals.direct", "SkipCouncil": "council.skip",
	"CreateIntake": "goals.create", "GetIntake": "goals.get", "ApplyIntake": "goals.create",
	"ApplyWizardGaps":             "goals.create",
	"AcceptIntakeRecommendations": "goals.create", "GetIntakeContext": "goals.get",
	"PrepareIntakeDossier": "goals.create", "PrepareWizardDossier": "goals.create",
	"GetIntakeDossier": "goals.get", "ConfirmIntakeDossier": "goals.create",
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func canonicalRegistry(registry document) ([]byte, error) {
	for index := range registry.Commands {
		for _, target := range []*json.RawMessage{&registry.Commands[index].InputSchema, &registry.Commands[index].OutputSchema} {
			var value any
			decoder := json.NewDecoder(bytes.NewReader(*target))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				return nil, errors.New("commandgen.schema_invalid")
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			*target = encoded
		}
	}
	return json.Marshal(registry)
}

func header(buffer *bytes.Buffer, packageName, digest string) {
	fmt.Fprintln(buffer, "// Code generated by commandgen; DO NOT EDIT.")
	fmt.Fprintf(buffer, "// RegistrySourceSHA256 %s\n", digest)
	fmt.Fprintf(buffer, "package %s\n\n", packageName)
}

func generateDefinitions(registry document, digest string) []byte {
	var out bytes.Buffer
	header(&out, "commands", digest)
	fmt.Fprintf(&out, "const RegistrySourceSHA256 = %q\n\n", digest)
	fmt.Fprintln(&out, "var compiledDefinitions = []Definition{")
	for _, item := range registry.Commands {
		fmt.Fprintf(&out, "{ID:%q, Version:%q, Kind:Kind(%q), Handler:%q, Permission:%q, Audience:Audience(%q), ExecutionBound:%t, ReplayMode:ReplayMode(%q), InputSchema:[]byte(%q), OutputSchema:[]byte(%q), DescriptionKey:%q, ErrorCodes:%#v, HTTP:HTTPBinding{Method:%q, Path:%q}, MCP:MCPBinding{Tool:%q, Annotations:MCPAnnotations{ReadOnly:%t, Destructive:%t, Idempotent:%t, OpenWorld:%t}}, CLI:CLIBinding{Path:%#v}},\n", item.ID, item.Version, item.Kind, item.Handler, item.Permission, item.Audience, item.ExecutionBound, item.ReplayMode, string(item.InputSchema), string(item.OutputSchema), item.DescriptionKey, item.ErrorCodes, item.HTTP.Method, item.HTTP.Path, item.MCP.Tool, *item.MCP.Annotations.ReadOnly, *item.MCP.Annotations.Destructive, *item.MCP.Annotations.Idempotent, *item.MCP.Annotations.OpenWorld, item.CLI.Path)
	}
	fmt.Fprintln(&out, "}")
	return out.Bytes()
}
