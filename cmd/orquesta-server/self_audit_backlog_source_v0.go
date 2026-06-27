package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const selfAuditCommandTimeoutV0 = 20 * time.Second

type selfAuditCommandV0 struct {
	ToolRef     string
	CommandPath string
	Args        []string
	TestCommand string
}

type selfAuditCommandResultV0 struct {
	Output string
}

type selfAuditFindingV0 struct {
	ToolRef string
	Path    string
	Line    int
	Code    string
	Message string
	Command string
}

var runSelfAuditCommandV0 = runSelfAuditCommandDefaultV0

func selfAuditBacklogSectionsV0(
	ctx context.Context,
	projectDir string,
) []idleSelfImprovementBacklogSectionV0 {
	projectDir = strings.TrimSpace(projectDir)
	if projectDir == "" {
		return nil
	}
	var sections []idleSelfImprovementBacklogSectionV0
	for _, command := range selfAuditCommandsV0() {
		result := runSelfAuditCommandV0(ctx, projectDir, command)
		findings := selfAuditFindingsFromCommandV0(command, result)
		for _, finding := range findings {
			sections = append(sections, selfAuditBacklogSectionV0(finding))
		}
	}
	return sections
}

func selfAuditCommandsV0() []selfAuditCommandV0 {
	return []selfAuditCommandV0{
		{ToolRef: "go-vet", CommandPath: "go", Args: []string{"vet", "./..."}, TestCommand: "go vet ./..."},
		{ToolRef: "staticcheck", CommandPath: "staticcheck", Args: []string{"./..."}, TestCommand: "staticcheck ./..."},
		{ToolRef: "govulncheck", CommandPath: "govulncheck", Args: []string{"./..."}, TestCommand: "govulncheck ./..."},
		{ToolRef: "go-test-race", CommandPath: "go", Args: []string{"test", "-race", "./..."}, TestCommand: "go test -race ./..."},
	}
}

func runSelfAuditCommandDefaultV0(
	ctx context.Context,
	projectDir string,
	command selfAuditCommandV0,
) selfAuditCommandResultV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, selfAuditCommandTimeoutV0)
	defer cancel()
	cmd := exec.CommandContext(runCtx, command.CommandPath, command.Args...)
	cmd.Dir = projectDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if runCtx.Err() != nil || err != nil {
		return selfAuditCommandResultV0{Output: output.String()}
	}
	return selfAuditCommandResultV0{Output: output.String()}
}

func selfAuditFindingsFromCommandV0(
	command selfAuditCommandV0,
	result selfAuditCommandResultV0,
) []selfAuditFindingV0 {
	if strings.TrimSpace(result.Output) == "" {
		return nil
	}
	var findings []selfAuditFindingV0
	for _, line := range strings.Split(strings.ReplaceAll(result.Output, "\r\n", "\n"), "\n") {
		if finding, ok := selfAuditFindingFromLineV0(command, line); ok {
			findings = append(findings, finding)
		}
	}
	return findings
}

func selfAuditFindingFromLineV0(
	command selfAuditCommandV0,
	line string,
) (selfAuditFindingV0, bool) {
	parts := strings.SplitN(strings.TrimSpace(line), ":", 4)
	if len(parts) < 4 {
		return selfAuditFindingV0{}, false
	}
	rel, ok := cleanSelfAuditFindingPathV0(parts[0])
	if !ok {
		return selfAuditFindingV0{}, false
	}
	lineNumber, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || lineNumber <= 0 {
		return selfAuditFindingV0{}, false
	}
	message := strings.TrimSpace(parts[3])
	if message == "" {
		return selfAuditFindingV0{}, false
	}
	return selfAuditFindingV0{
		ToolRef: strings.TrimSpace(command.ToolRef),
		Path:    rel,
		Line:    lineNumber,
		Code:    selfAuditFindingCodeV0(command.ToolRef, message),
		Message: selfAuditCompactMessageV0(message),
		Command: strings.TrimSpace(command.TestCommand),
	}, true
}

func selfAuditBacklogSectionV0(finding selfAuditFindingV0) idleSelfImprovementBacklogSectionV0 {
	ref := "self-audit-" + idleSelfImprovementBacklogHashV0(strings.Join([]string{
		finding.ToolRef,
		finding.Path,
		strconv.Itoa(finding.Line),
		finding.Code,
		strings.ToLower(finding.Message),
	}, "|"))
	section := idleSelfImprovementBacklogSectionV0{
		Ref:        ref,
		Heading:    "SELF-AUDIT " + finding.ToolRef + " " + finding.Path + ":" + strconv.Itoa(finding.Line),
		Objective:  "corregir hallazgo " + finding.ToolRef + " " + finding.Code + " en " + finding.Path + ": " + finding.Message,
		SourcePath: "self_audit://" + finding.ToolRef,
		SourceKind: "self_audit",
		Owner:      selfAuditOwnerForPathV0(finding.Path),
		Scope:      []string{finding.Path},
		Criteria: []string{
			finding.ToolRef + " no reporta el hallazgo " + finding.Code + " para " + finding.Path,
		},
		Tests: []string{finding.Command},
		Inputs: []string{
			"self_audit_tool:" + finding.ToolRef,
			"self_audit_finding_ref:" + ref,
			"self_audit_path:" + finding.Path,
		},
		Outputs:    []string{finding.Path},
		SourceLine: finding.Line,
		StateEvidenceRefs: []string{
			"evidence-ref-self-audit-backlog-source",
			"evidence-ref-self-audit-" + finding.ToolRef,
		},
	}
	section.TaskInstanceRef = idleSelfImprovementBacklogTaskInstanceRefV0(section)
	return section
}

func cleanSelfAuditFindingPathV0(value string) (string, bool) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return "", false
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return "", false
		}
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean != value || !strings.Contains(clean, ".") {
		return "", false
	}
	return clean, true
}

func selfAuditFindingCodeV0(toolRef string, message string) string {
	message = strings.TrimSpace(message)
	if start := strings.LastIndex(message, "("); start >= 0 && strings.HasSuffix(message, ")") {
		code := strings.TrimSpace(strings.TrimSuffix(message[start+1:], ")"))
		if code != "" && !strings.Contains(code, " ") {
			return code
		}
	}
	if code := selfAuditFirstTokenWithPrefixV0(message, "GO-"); code != "" {
		return code
	}
	return strings.TrimSpace(toolRef) + "-" + idleSelfImprovementBacklogHashV0(strings.ToLower(message))
}

func selfAuditFirstTokenWithPrefixV0(value string, prefix string) string {
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ',' || r == ';' || r == ':' || r == '(' || r == ')'
	}) {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(token, prefix) {
			return token
		}
	}
	return ""
}

func selfAuditCompactMessageV0(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) > 220 {
		value = strings.TrimSpace(value[:220]) + "..."
	}
	return value
}

func selfAuditOwnerForPathV0(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "modulos/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
	}
	if strings.HasPrefix(path, "cmd/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
	}
	return fmt.Sprintf("self-audit:%s", path)
}
