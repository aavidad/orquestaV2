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
	if ctx == nil {
		ctx = context.Background()
	}
	var sections []idleSelfImprovementBacklogSectionV0
	for _, command := range selfAuditCommandsV0() {
		if ctx.Err() != nil {
			break
		}
		result := runSelfAuditCommandV0(ctx, projectDir, command)
		findings := selfAuditFindingsFromCommandV0(projectDir, command, result)
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
	projectDir string,
	command selfAuditCommandV0,
	result selfAuditCommandResultV0,
) []selfAuditFindingV0 {
	if strings.TrimSpace(result.Output) == "" {
		return nil
	}
	var findings []selfAuditFindingV0
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(result.Output, "\r\n", "\n"), "\n") {
		if finding, ok := selfAuditFindingFromLineV0(projectDir, command, line); ok {
			key := selfAuditFindingDedupeKeyV0(finding)
			if seen[key] {
				continue
			}
			seen[key] = true
			findings = append(findings, finding)
		}
	}
	globalFindings := selfAuditGlobalFindingsFromOutputV0(command, result.Output)
	if strings.TrimSpace(command.ToolRef) == "go-test-race" && len(findings) > 0 {
		globalFindings = nil
	}
	for _, finding := range globalFindings {
		key := selfAuditFindingDedupeKeyV0(finding)
		if seen[key] {
			continue
		}
		seen[key] = true
		findings = append(findings, finding)
	}
	return findings
}

func selfAuditFindingFromLineV0(
	projectDir string,
	command selfAuditCommandV0,
	line string,
) (selfAuditFindingV0, bool) {
	rel, lineNumber, message, ok := selfAuditPathLineMessageFromLineV0(projectDir, line)
	if !ok {
		return selfAuditFindingV0{}, false
	}
	if selfAuditLineMessageIsNoiseV0(command, message) {
		message = ""
	}
	if message == "" {
		message = selfAuditDefaultLineFindingMessageV0(command, rel, lineNumber)
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

func selfAuditPathLineMessageFromLineV0(
	projectDir string,
	line string,
) (string, int, string, bool) {
	line = strings.TrimSpace(line)
	goIndex := strings.LastIndex(line, ".go:")
	if goIndex < 0 {
		return "", 0, "", false
	}
	pathPart := strings.TrimSpace(line[:goIndex+len(".go")])
	if fields := strings.Fields(pathPart); len(fields) > 0 {
		pathPart = fields[len(fields)-1]
	}
	rel, ok := cleanSelfAuditFindingPathV0(projectDir, pathPart)
	if !ok {
		return "", 0, "", false
	}
	rest := strings.TrimSpace(line[goIndex+len(".go:"):])
	lineDigits := selfAuditLeadingDigitsV0(rest)
	if lineDigits == "" {
		return "", 0, "", false
	}
	lineNumber, err := strconv.Atoi(lineDigits)
	if err != nil || lineNumber <= 0 {
		return "", 0, "", false
	}
	message := strings.TrimSpace(rest[len(lineDigits):])
	if strings.HasPrefix(message, ":") {
		message = strings.TrimSpace(strings.TrimPrefix(message, ":"))
		columnDigits := selfAuditLeadingDigitsV0(message)
		if columnDigits != "" {
			message = strings.TrimSpace(message[len(columnDigits):])
			message = strings.TrimSpace(strings.TrimPrefix(message, ":"))
		}
	}
	return rel, lineNumber, message, true
}

func selfAuditGlobalFindingsFromOutputV0(
	command selfAuditCommandV0,
	output string,
) []selfAuditFindingV0 {
	switch strings.TrimSpace(command.ToolRef) {
	case "govulncheck":
		return selfAuditGovulncheckFindingsFromOutputV0(command, output)
	case "go-test-race":
		if strings.Contains(output, "WARNING: DATA RACE") || strings.Contains(output, "DATA RACE") {
			return []selfAuditFindingV0{{
				ToolRef: "go-test-race",
				Path:    idleSelfImprovementBacklogDocRelV0,
				Line:    1,
				Code:    "data-race",
				Message: "go test -race reporta una data race sin ruta recuperable; registrar triage y acotar write-set",
				Command: strings.TrimSpace(command.TestCommand),
			}}
		}
	}
	return nil
}

func selfAuditGovulncheckFindingsFromOutputV0(
	command selfAuditCommandV0,
	output string,
) []selfAuditFindingV0 {
	seen := map[string]bool{}
	var findings []selfAuditFindingV0
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		for _, token := range strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ',' || r == ';' || r == ':' || r == '(' || r == ')' || r == '[' || r == ']'
		}) {
			code := strings.TrimSpace(token)
			if !strings.HasPrefix(code, "GO-") || seen[code] {
				continue
			}
			seen[code] = true
			findings = append(findings, selfAuditFindingV0{
				ToolRef: "govulncheck",
				Path:    "go.mod",
				Line:    1,
				Code:    code,
				Message: "govulncheck reporta vulnerabilidad " + code,
				Command: strings.TrimSpace(command.TestCommand),
			})
		}
	}
	return findings
}

func selfAuditDefaultLineFindingMessageV0(
	command selfAuditCommandV0,
	path string,
	lineNumber int,
) string {
	if strings.TrimSpace(command.ToolRef) == "go-test-race" {
		return "go test -race reporta actividad concurrente en " + path + ":" + strconv.Itoa(lineNumber)
	}
	return strings.TrimSpace(command.ToolRef) + " reporta hallazgo en " + path + ":" + strconv.Itoa(lineNumber)
}

func selfAuditLineMessageIsNoiseV0(command selfAuditCommandV0, message string) bool {
	message = strings.TrimSpace(message)
	return strings.TrimSpace(command.ToolRef) == "go-test-race" &&
		(strings.HasPrefix(message, "+0x") || strings.HasPrefix(message, "0x"))
}

func selfAuditFindingDedupeKeyV0(finding selfAuditFindingV0) string {
	return strings.Join([]string{
		strings.TrimSpace(finding.ToolRef),
		strings.TrimSpace(finding.Path),
		strconv.Itoa(finding.Line),
		strings.TrimSpace(finding.Code),
	}, "|")
}

func selfAuditLeadingDigitsV0(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		out.WriteRune(r)
	}
	return out.String()
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

func cleanSelfAuditFindingPathV0(projectDir string, value string) (string, bool) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	if value == "" || strings.Contains(value, "\\") {
		return "", false
	}
	if filepath.IsAbs(value) {
		if strings.TrimSpace(projectDir) == "" {
			return "", false
		}
		rel, err := filepath.Rel(projectDir, value)
		if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			return "", false
		}
		value = filepath.ToSlash(rel)
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
	if strings.TrimSpace(toolRef) == "go-test-race" {
		return "data-race"
	}
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
