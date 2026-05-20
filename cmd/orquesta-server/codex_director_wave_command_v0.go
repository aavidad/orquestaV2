package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
)

const codexDirectorWaveSummarySchemaVersionV0 = "orquesta_codex_director_wave_launch.v0"

type codexDirectorWaveSummaryV0 struct {
	SchemaVersion string                                                  `json:"schema_version"`
	Request       orquestadirectoroperativo.OperationalDirectorRequestV0  `json:"request"`
	Plan          orquestadirectoroperativo.OperationalDirectorPlanV0     `json:"plan"`
	WaveWork      orquestadirectoroperativo.OperationalDirectorWaveWorkV0 `json:"wave_work"`
	Launch        codexWaveLaunchSummaryV0                                `json:"launch"`
	ChildLaunches []codexDirectorChildWaveSummaryV0                       `json:"child_launches,omitempty"`
	Issues        []orquestadirectoroperativo.OperationalDirectorIssueV0  `json:"issues,omitempty"`
}

type codexDirectorChildWaveSummaryV0 struct {
	ParentAgentRef string                   `json:"parent_agent_ref"`
	ParentIndex    int                      `json:"parent_index"`
	Launch         codexWaveLaunchSummaryV0 `json:"launch"`
}

type codexDirectorWaveConfigV0 struct {
	Wave codexWaveConfigV0

	RequestRef               string
	RunRef                   string
	ProjectRef               string
	DomainRefs               []string
	Objective                string
	WorktreeRef              string
	BranchRef                string
	WriteSet                 []string
	RequiredTests            []string
	AllowRecursiveDelegation bool
	MaxDelegationDepth       int
	MaxSubagentsPerAgent     int
	StrictDirectorGuards     bool
}

func codexLaunchDirectorWaveCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexDirectorWaveConfigFromArgsV0(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-director-wave: %v\n", err)
		return 2
	}
	summary, err := runCodexLaunchDirectorWaveV0(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-director-wave: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	if len(summary.Issues) > 0 || len(summary.Launch.Errors) > 0 || codexDirectorChildLaunchHasErrorsV0(summary.ChildLaunches) {
		return 1
	}
	return 0
}

func codexDirectorChildLaunchHasErrorsV0(childLaunches []codexDirectorChildWaveSummaryV0) bool {
	for _, child := range childLaunches {
		if len(child.Launch.Errors) > 0 {
			return true
		}
	}
	return false
}

func codexDirectorWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexDirectorWaveConfigV0, error) {
	flags := flag.NewFlagSet("codex-launch-director-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)

	agents := flags.Int("agents", intEnvOrDefaultV0("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", 6), "numero de agentes que puede lanzar el Director")
	waveRef := flags.String("wave-ref", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_REF")), "ref opaca de la ola")
	objective := flags.String("objective", "", "objetivo para el Director Operativo")
	objectiveFile := flags.String("objective-file", "", "archivo con objetivo para el Director Operativo")
	writeSet := flags.String("write-set", "", "write-set separado por comas")
	requiredTests := flags.String("required-tests", "", "tests requeridos separados por comas")
	branchRef := flags.String("branch-ref", "", "ref de rama aislada")
	worktreeRef := flags.String("worktree-ref", "", "ref de worktree aislada")
	requestRef := flags.String("request-ref", "", "request_ref opcional")
	runRef := flags.String("run-ref", "", "run_ref opcional")
	projectRef := flags.String("project-ref", envOrDefaultV0("ORQUESTA_CODEX_DIRECTOR_PROJECT_REF", "orquesta"), "project_ref opaco")
	domainRefs := flags.String("domain-ref", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_DIRECTOR_DOMAIN_REFS")), "domain_refs separados por comas")
	projectDir := flags.String("project-dir", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROJECT_WORKDIR")), "directorio de trabajo del proyecto")
	runtimeDir := flags.String("runtime-dir", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_RUNTIME_WORKDIR")), "directorio runtime de esta ola")
	commandPath := flags.String("command", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_COMMAND")), "ruta al binario codex")
	sourceCodeHome := flags.String("source-code-home", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_SOURCE_CODEX_HOME")), "CODEX_HOME fuente para copiar credenciales/config")
	model := flags.String("model", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_MODEL", "ORQUESTA_CODEX_MODEL"), "modelo Codex opcional")
	reasoningEffort := flags.String("reasoning-effort", envOrDefaultV0("ORQUESTA_CODEX_WAVE_REASONING_EFFORT", envOrDefaultV0("ORQUESTA_CODEX_REASONING_EFFORT", "xhigh")), "esfuerzo de razonamiento")
	profile := flags.String("profile", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_PROFILE", "ORQUESTA_CODEX_PROFILE"), "perfil Codex opcional")
	sandbox := flags.String("sandbox", envOrDefaultV0("ORQUESTA_CODEX_WAVE_SANDBOX", "danger-full-access"), "sandbox Codex")
	approval := flags.String("approval-policy", envOrDefaultV0("ORQUESTA_CODEX_WAVE_APPROVAL_POLICY", "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_EXTRA_ARGS")), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ISOLATE_HOME", false), "copiar CODEX_HOME por agente bajo el runtime")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME", false), "purgar runtime de esta ola antes de materializar")
	allowRecursive := flags.Bool("allow-recursive-delegation", false, "permitir que el Director gobierne delegacion recursiva")
	maxDepth := flags.Int("max-delegation-depth", 0, "profundidad maxima de delegacion")
	maxChildren := flags.Int("max-subagents-per-agent", 0, "fanout maximo por agente")
	strictDirectorGuards := flags.Bool("strict-director-guards", false, "exigir branch/write-set/tests explicitos del Director")

	if err := flags.Parse(args); err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	objectiveText, err := codexDirectorObjectiveTextV0(*objective, *objectiveFile, flags.Args())
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	projectWorkDir, err := codexWaveProjectDirV0(*projectDir)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	ref, err := codexWaveRefV0(*waveRef)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	rootRuntimeDir, err := codexWaveRuntimeDirV0(*runtimeDir, projectWorkDir, ref)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	resolvedCommand, err := codexWaveCommandPathV0(*commandPath)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	sourceHome := codexWaveSourceCodeHomeV0(*sourceCodeHome)
	if *isolateHome && sourceHome == "" {
		return codexDirectorWaveConfigV0{}, errors.New("source_code_home_unavailable")
	}
	if *agents <= 0 {
		return codexDirectorWaveConfigV0{}, errors.New("agents_out_of_range")
	}
	branchValue := strings.TrimSpace(*branchRef)
	writeSetValues := codexDirectorCSVV0(*writeSet)
	requiredTestValues := codexDirectorCSVV0(*requiredTests)
	if !*strictDirectorGuards {
		if branchValue == "" {
			branchValue = codexDirectorCurrentBranchRefV0(projectWorkDir)
		}
		if len(writeSetValues) == 0 {
			writeSetValues = []string{"workspace"}
		}
		if len(requiredTestValues) == 0 {
			requiredTestValues = []string{"operator-validation-required"}
		}
	}

	return codexDirectorWaveConfigV0{
		Wave: codexWaveConfigV0{
			Agents:          *agents,
			WaveRef:         ref,
			Prompt:          objectiveText,
			ProjectWorkDir:  projectWorkDir,
			RuntimeWorkDir:  rootRuntimeDir,
			CommandPath:     resolvedCommand,
			SourceCodeHome:  sourceHome,
			PathEnv:         envOrDefaultV0("ORQUESTA_CODEX_WAVE_PATH", envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH"))),
			Model:           strings.TrimSpace(*model),
			ReasoningEffort: strings.TrimSpace(*reasoningEffort),
			Profile:         strings.TrimSpace(*profile),
			Sandbox:         strings.TrimSpace(*sandbox),
			ApprovalPolicy:  strings.TrimSpace(*approval),
			ExtraArgs:       strings.Fields(*extraArgs),
			IsolateHome:     *isolateHome,
			DryRun:          *dryRun,
			PurgeRuntime:    *purgeRuntime,
		},
		RequestRef:               codexDirectorDefaultRefV0(*requestRef, "req", ref),
		RunRef:                   codexDirectorDefaultRefV0(*runRef, "run", ref),
		ProjectRef:               strings.TrimSpace(*projectRef),
		DomainRefs:               codexDirectorCSVV0(*domainRefs),
		Objective:                objectiveText,
		WorktreeRef:              codexDirectorDefaultRefV0(*worktreeRef, "worktree", ref),
		BranchRef:                branchValue,
		WriteSet:                 writeSetValues,
		RequiredTests:            requiredTestValues,
		AllowRecursiveDelegation: *allowRecursive,
		MaxDelegationDepth:       *maxDepth,
		MaxSubagentsPerAgent:     *maxChildren,
		StrictDirectorGuards:     *strictDirectorGuards,
	}, nil
}

func runCodexLaunchDirectorWaveV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
) (codexDirectorWaveSummaryV0, error) {
	request := orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:               config.RequestRef,
		RunRef:                   config.RunRef,
		ProjectRef:               config.ProjectRef,
		Objective:                config.Objective,
		Mode:                     orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		DomainRefs:               append([]string(nil), config.DomainRefs...),
		WorktreeRef:              config.WorktreeRef,
		WorktreeIsolated:         true,
		BranchRef:                config.BranchRef,
		WriteSet:                 append([]string(nil), config.WriteSet...),
		RequiredTests:            append([]string(nil), config.RequiredTests...),
		MaxParallelAgents:        config.Wave.Agents,
		AllowRecursiveDelegation: config.AllowRecursiveDelegation,
		MaxDelegationDepth:       config.MaxDelegationDepth,
		MaxSubagentsPerAgent:     config.MaxSubagentsPerAgent,
	}
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(request)
	summary := codexDirectorWaveSummaryV0{
		SchemaVersion: codexDirectorWaveSummarySchemaVersionV0,
		Request:       request,
		Plan:          result.Plan,
		Issues:        append([]orquestadirectoroperativo.OperationalDirectorIssueV0(nil), result.Issues...),
	}
	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		return summary, nil
	}
	work := orquestadirectoroperativo.BuildOperationalDirectorWaveWorkV0(result.Plan)
	summary.WaveWork = work
	if len(work.Issues) > 0 {
		summary.Issues = append(summary.Issues, work.Issues...)
		return summary, nil
	}

	launchConfig := config.Wave
	launchConfig.Agents = result.Plan.MaxParallelAgents
	launchConfig.Prompt = result.Plan.Objective
	launchConfig.AgentPrompts = codexDirectorAgentPromptsV0(result.Plan, work)
	launch, err := runCodexLaunchWaveV0(ctx, launchConfig)
	if err != nil {
		return summary, err
	}
	summary.Launch = launch
	childLaunches, err := runCodexLaunchDirectorChildWavesV0(ctx, config, result.Plan, work, launch)
	if err != nil {
		return summary, err
	}
	summary.ChildLaunches = childLaunches
	return summary, nil
}

func runCodexLaunchDirectorChildWavesV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentLaunch codexWaveLaunchSummaryV0,
) ([]codexDirectorChildWaveSummaryV0, error) {
	if !plan.RecursiveDelegation || plan.MaxSubagentsPerAgent <= 0 {
		return nil, nil
	}
	out := make([]codexDirectorChildWaveSummaryV0, 0, len(parentLaunch.Agents))
	for index, parent := range parentLaunch.Agents {
		parentIndex := index + 1
		childWaveRef := fmt.Sprintf("%s-parent-%02d-children", config.Wave.WaveRef, parentIndex)
		childConfig := config.Wave
		childConfig.WaveRef = childWaveRef
		childConfig.Agents = plan.MaxSubagentsPerAgent
		childConfig.RuntimeWorkDir = filepath.Join(config.Wave.RuntimeWorkDir, "children", childWaveRef)
		childConfig.Prompt = plan.Objective
		childConfig.AgentPrompts = codexDirectorChildAgentPromptsV0(
			plan,
			work,
			parent.AgentRef,
			parentIndex,
			len(parentLaunch.Agents),
			childConfig.Agents,
		)
		launch, err := runCodexLaunchWaveV0(ctx, childConfig)
		if err != nil {
			return out, err
		}
		out = append(out, codexDirectorChildWaveSummaryV0{
			ParentAgentRef: parent.AgentRef,
			ParentIndex:    parentIndex,
			Launch:         launch,
		})
	}
	return out, nil
}

func codexDirectorAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
) []string {
	total := plan.MaxParallelAgents
	if total <= 0 {
		total = 1
	}
	prompts := make([]string, 0, total)
	launchItem := codexDirectorLaunchItemV0(work)
	for i := 1; i <= total; i++ {
		prompts = append(prompts, codexDirectorAgentPromptV0(plan, work, launchItem, i, total))
	}
	return prompts
}

func codexDirectorAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	index int,
	total int,
) string {
	var b strings.Builder
	assigned := codexDirectorShardV0(plan.WriteSet, index, total)
	b.WriteString("Director Operativo Orquesta aprobo esta ola de trabajo.\n\n")
	b.WriteString("Plan:\n")
	b.WriteString("- plan_ref: " + plan.PlanRef + "\n")
	b.WriteString("- request_ref: " + plan.RequestRef + "\n")
	b.WriteString("- run_ref: " + plan.RunRef + "\n")
	b.WriteString("- mode: " + string(plan.Mode) + "\n")
	b.WriteString("- wave_ref: " + codexDirectorWaveRefForItemV0(work, launchItem.ItemID) + "\n")
	b.WriteString("- launch_item_ref: " + launchItem.ItemID + "\n")
	b.WriteString("- agent_index: " + strconv.Itoa(index) + " de " + strconv.Itoa(total) + "\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(plan.Objective + "\n\n")
	codexDirectorWriteRecursiveConfigV0(&b, plan)
	codexDirectorWriteDomainContextV0(&b, plan)
	b.WriteString("Write-set global autorizado:\n")
	codexDirectorWriteListV0(&b, plan.WriteSet)
	b.WriteString("\nWrite-set asignado a este agente:\n")
	if len(assigned) == 0 {
		b.WriteString("- Sin shard exclusivo: trabaja solo en apoyo/review dentro del write-set global y coordina por resumen final.\n")
	} else {
		codexDirectorWriteListV0(&b, assigned)
	}
	b.WriteString("\nTests requeridos por el Director:\n")
	codexDirectorWriteListV0(&b, plan.RequiredTests)
	b.WriteString("\nCriterios de aceptacion:\n")
	codexDirectorWriteListV0(&b, launchItem.AcceptanceCriteria)
	b.WriteString("\nReglas:\n")
	b.WriteString("- No cambies ficheros fuera del write-set autorizado.\n")
	b.WriteString("- No borres nada sin revisar uso actual y dejar evidencia en tu resumen.\n")
	b.WriteString("- Si tu shard no basta, explica el bloqueo; no invadas el shard de otro agente sin justificarlo.\n")
	b.WriteString("- Al terminar, resume rutas tocadas, pruebas ejecutadas, resultado y bloqueos.\n")
	return b.String()
}

func codexDirectorChildAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childTotal int,
) []string {
	prompts := make([]string, 0, childTotal)
	launchItem := codexDirectorLaunchItemV0(work)
	for childIndex := 1; childIndex <= childTotal; childIndex++ {
		prompts = append(prompts, codexDirectorChildAgentPromptV0(
			plan,
			launchItem,
			parentAgentRef,
			parentIndex,
			parentTotal,
			childIndex,
			childTotal,
		))
	}
	return prompts
}

func codexDirectorChildAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childIndex int,
	childTotal int,
) string {
	var b strings.Builder
	assigned := codexDirectorShardV0(plan.WriteSet, parentIndex, parentTotal)
	b.WriteString("Subagente Codex gobernado por Director Operativo Orquesta.\n\n")
	b.WriteString("Linaje:\n")
	b.WriteString("- plan_ref: " + plan.PlanRef + "\n")
	b.WriteString("- run_ref: " + plan.RunRef + "\n")
	b.WriteString("- parent_agent_ref: " + parentAgentRef + "\n")
	b.WriteString("- parent_agent_index: " + strconv.Itoa(parentIndex) + " de " + strconv.Itoa(parentTotal) + "\n")
	b.WriteString("- child_agent_index: " + strconv.Itoa(childIndex) + " de " + strconv.Itoa(childTotal) + "\n")
	b.WriteString("- launch_item_ref: " + launchItem.ItemID + "\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(plan.Objective + "\n\n")
	codexDirectorWriteRecursiveConfigV0(&b, plan)
	codexDirectorWriteDomainContextV0(&b, plan)
	b.WriteString("Write-set asignado al agente padre y compartido por este subarbol:\n")
	if len(assigned) == 0 {
		b.WriteString("- Sin shard exclusivo: trabaja solo en apoyo/review dentro del write-set global.\n")
	} else {
		codexDirectorWriteListV0(&b, assigned)
	}
	b.WriteString("\nRol sugerido del subagente:\n")
	b.WriteString("- " + codexDirectorChildRoleV0(childIndex) + "\n")
	b.WriteString("\nTests requeridos por el Director:\n")
	codexDirectorWriteListV0(&b, plan.RequiredTests)
	b.WriteString("\nCriterios de aceptacion:\n")
	codexDirectorWriteListV0(&b, launchItem.AcceptanceCriteria)
	b.WriteString("\nReglas:\n")
	b.WriteString("- Trabaja solo dentro del write-set del subarbol o entrega informe de bloqueo.\n")
	b.WriteString("- No borres nada sin revisar uso actual y dejar evidencia en tu resumen.\n")
	b.WriteString("- No lances mas subagentes desde este hijo: la ola hija ya fue materializada por Orquesta.\n")
	b.WriteString("- Al terminar, resume artefactos producidos, pruebas ejecutadas y huecos pendientes.\n")
	return b.String()
}

func codexDirectorChildRoleV0(index int) string {
	switch index {
	case 1:
		return "estructura, indice, mapa conceptual y plan de secciones"
	case 2:
		return "fuentes oficiales, normativa, doctrina y trazabilidad"
	case 3:
		return "desarrollo teorico principal con tono A1"
	case 4:
		return "ejemplos, supuestos practicos, errores frecuentes y notas de test"
	case 5:
		return "visuales utiles, tablas comparativas y esquemas responsivos"
	case 6:
		return "revision pedagogica, ensamblado, validacion de palabras y checklist A1"
	default:
		return "apoyo editorial acotado y revision"
	}
}

func codexDirectorWriteRecursiveConfigV0(
	b *strings.Builder,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
) {
	b.WriteString("Delegacion recursiva:\n")
	if !plan.RecursiveDelegation {
		b.WriteString("- recursive_delegation: false\n\n")
		return
	}
	b.WriteString("- recursive_delegation: true\n")
	b.WriteString("- max_delegation_depth: " + strconv.Itoa(plan.MaxDelegationDepth) + "\n")
	b.WriteString("- max_subagents_per_agent: " + strconv.Itoa(plan.MaxSubagentsPerAgent) + "\n")
	b.WriteString("- Orquesta materializa los subagentes autorizados con parent_agent_ref; no improvises hijos fuera de esos limites.\n\n")
}

func codexDirectorWriteDomainContextV0(
	b *strings.Builder,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
) {
	if !codexDirectorPlanUsesOPESV0(plan) {
		return
	}
	b.WriteString("Contexto de dominio OPES inyectado por el conector:\n")
	b.WriteString("- field: opes_global_editorial_policy_2026_05_18\n")
	for _, rule := range orquestaopesbridge.OPESGlobalEditorialPolicyV0() {
		b.WriteString("- " + rule + "\n")
	}
	b.WriteString("- field: opes_html_publication_policy_2026_05_19\n")
	for _, rule := range orquestaopesbridge.OPESHTMLPublicationPolicyV0() {
		b.WriteString("- " + rule + "\n")
	}
	b.WriteString("\n")
}

func codexDirectorPlanUsesOPESV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
) bool {
	if strings.EqualFold(strings.TrimSpace(plan.ProjectRef), "opes") {
		return true
	}
	for _, ref := range plan.DomainRefs {
		if strings.EqualFold(strings.TrimSpace(ref), "opes") {
			return true
		}
	}
	return false
}

func codexDirectorLaunchItemV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
) orquestadirectoroperativo.OperationalDirectorWorkItemV0 {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.Kind == orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0 {
				return item
			}
		}
	}
	return orquestadirectoroperativo.OperationalDirectorWorkItemV0{}
}

func codexDirectorWaveRefForItemV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	itemRef string,
) string {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.ItemID == itemRef {
				return wave.WaveID
			}
		}
	}
	return ""
}

func codexDirectorShardV0(values []string, index int, total int) []string {
	if total <= 0 || index <= 0 {
		return nil
	}
	out := make([]string, 0)
	for i, value := range values {
		if i%total == index-1 {
			out = append(out, value)
		}
	}
	return out
}

func codexDirectorObjectiveTextV0(objective string, objectiveFile string, trailing []string) (string, error) {
	if strings.TrimSpace(objectiveFile) != "" {
		data, err := os.ReadFile(objectiveFile)
		if err != nil {
			return "", err
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", errors.New("objective_empty")
		}
		return text, nil
	}
	if strings.TrimSpace(objective) != "" {
		return strings.TrimSpace(objective), nil
	}
	if len(trailing) > 0 {
		text := strings.TrimSpace(strings.Join(trailing, " "))
		if text != "" {
			return text, nil
		}
	}
	return "", errors.New("objective_required")
}

func codexDirectorDefaultRefV0(value string, prefix string, waveRef string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	waveRef = strings.TrimSpace(waveRef)
	if waveRef == "" {
		waveRef = time.Now().UTC().Format("20060102T150405Z")
	}
	return prefix + "-" + waveRef
}

func codexDirectorCurrentBranchRefV0(projectWorkDir string) string {
	out, err := exec.Command("git", "-C", projectWorkDir, "branch", "--show-current").Output()
	if err == nil {
		if branch := strings.TrimSpace(string(out)); branch != "" {
			return branch
		}
	}
	return "local-branch"
}

func codexDirectorCSVV0(value string) []string {
	out := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func codexDirectorWriteListV0(b *strings.Builder, values []string) {
	if len(values) == 0 {
		b.WriteString("- ninguno\n")
		return
	}
	for _, value := range values {
		b.WriteString("- " + value + "\n")
	}
}
