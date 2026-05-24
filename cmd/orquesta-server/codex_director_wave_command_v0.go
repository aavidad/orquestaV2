package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

const codexDirectorWaveSummarySchemaVersionV0 = "orquesta_codex_director_wave_launch.v0"

type codexDirectorWaveSummaryV0 struct {
	SchemaVersion     string                                                     `json:"schema_version"`
	Request           orquestadirectoroperativo.OperationalDirectorRequestV0     `json:"request"`
	WorktreeIsolation codexDirectorWorktreeIsolationSummaryV0                    `json:"worktree_isolation,omitempty"`
	Plan              orquestadirectoroperativo.OperationalDirectorPlanV0        `json:"plan"`
	WaveWork          orquestadirectoroperativo.OperationalDirectorWaveWorkV0    `json:"wave_work"`
	Launch            codexWaveLaunchSummaryV0                                   `json:"launch"`
	ChildLaunches     []codexDirectorChildWaveSummaryV0                          `json:"child_launches,omitempty"`
	AgentBudget       orquestadirectoroperativo.OperationalDirectorAgentBudgetV0 `json:"agent_budget"`
	Issues            []orquestadirectoroperativo.OperationalDirectorIssueV0     `json:"issues,omitempty"`
}

type codexDirectorChildWaveSummaryV0 struct {
	ParentAgentRef            string                            `json:"parent_agent_ref"`
	ParentWaveRef             string                            `json:"parent_wave_ref,omitempty"`
	ParentIndex               int                               `json:"parent_index"`
	DelegationDepth           int                               `json:"delegation_depth"`
	MaxDelegationDepth        int                               `json:"max_delegation_depth"`
	MaxSubagentsPerAgent      int                               `json:"max_subagents_per_agent"`
	SubtreeAgentBudget        int                               `json:"subtree_agent_budget"`
	ReviewRequiredBeforeClose bool                              `json:"review_required_before_close"`
	Launch                    codexWaveLaunchSummaryV0          `json:"launch"`
	ChildLaunches             []codexDirectorChildWaveSummaryV0 `json:"child_launches,omitempty"`
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
	RecursiveAgentBudget     int
	StrictDirectorGuards     bool
	DomainContextBlocks      []codexDirectorDomainContextBlockV0
}

type codexDirectorDomainContextBlockV0 struct {
	SourceRef string
	Text      string
}

type codexDirectorStringListFlagV0 []string

func (flagValue *codexDirectorStringListFlagV0) String() string {
	if flagValue == nil {
		return ""
	}
	return strings.Join(*flagValue, ",")
}

func (flagValue *codexDirectorStringListFlagV0) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*flagValue = append(*flagValue, value)
	return nil
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
		if codexDirectorChildLaunchHasErrorsV0(child.ChildLaunches) {
			return true
		}
	}
	return false
}

func codexDirectorWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexDirectorWaveConfigV0, error) {
	flags := flag.NewFlagSet("codex-launch-director-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var domainContextFiles codexDirectorStringListFlagV0

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
	reasoningEffort := flags.String("reasoning-effort", envOrDefaultV0("ORQUESTA_CODEX_WAVE_REASONING_EFFORT", envOrDefaultV0("ORQUESTA_CODEX_REASONING_EFFORT", "medium")), "esfuerzo de razonamiento")
	profile := flags.String("profile", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_PROFILE", "ORQUESTA_CODEX_PROFILE"), "perfil Codex opcional")
	sandbox := flags.String("sandbox", envOrDefaultV0("ORQUESTA_CODEX_WAVE_SANDBOX", "workspace-write"), "sandbox Codex")
	approval := flags.String("approval-policy", envOrDefaultV0("ORQUESTA_CODEX_WAVE_APPROVAL_POLICY", "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_EXTRA_ARGS")), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ISOLATE_HOME", false), "copiar CODEX_HOME por agente bajo el runtime")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME", false), "purgar runtime de esta ola antes de materializar")
	allowRecursive := flags.Bool("allow-recursive-delegation", false, "permitir que el Director gobierne delegacion recursiva")
	maxDepth := flags.Int("max-delegation-depth", 0, "profundidad maxima de delegacion")
	maxChildren := flags.Int("max-subagents-per-agent", 0, "fanout maximo por agente")
	recursiveAgentBudget := flags.Int("recursive-agent-budget", intEnvOrDefaultV0("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", 32), "presupuesto global maximo de agentes en arbol recursivo")
	strictDirectorGuards := flags.Bool("strict-director-guards", false, "exigir branch/write-set/tests explicitos del Director")
	flags.Var(&domainContextFiles, "domain-context-file", "archivo de contexto de dominio inyectado por un adaptador externo; puede repetirse")

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
	if *recursiveAgentBudget < 0 {
		return codexDirectorWaveConfigV0{}, errors.New("recursive_agent_budget_out_of_range")
	}
	branchValue := strings.TrimSpace(*branchRef)
	writeSetValues := codexDirectorCSVV0(*writeSet)
	requiredTestValues := codexDirectorCSVV0(*requiredTests)
	domainContextFileValues := append(
		codexDirectorCSVV0(os.Getenv("ORQUESTA_CODEX_DIRECTOR_DOMAIN_CONTEXT_FILES")),
		[]string(domainContextFiles)...,
	)
	domainContextBlocks, err := codexDirectorDomainContextBlocksFromFilesV0(domainContextFileValues)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	if !*strictDirectorGuards {
		if branchValue == "" {
			branchValue = codexDirectorDefaultBranchRefV0(ref)
		}
		if len(writeSetValues) == 0 {
			writeSetValues = []string{"."}
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
			ReasoningEffort: codexReasoningEffortMinHighV0(*reasoningEffort),
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
		RecursiveAgentBudget:     *recursiveAgentBudget,
		StrictDirectorGuards:     *strictDirectorGuards,
		DomainContextBlocks:      domainContextBlocks,
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
		MaxRecursiveAgents:       config.RecursiveAgentBudget,
	}
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(request)
	summary := codexDirectorWaveSummaryV0{
		SchemaVersion: codexDirectorWaveSummarySchemaVersionV0,
		Request:       request,
		Plan:          result.Plan,
		Issues:        append([]orquestadirectoroperativo.OperationalDirectorIssueV0(nil), result.Issues...),
	}
	isolation, isolationIssues := codexDirectorPrepareWorktreeIsolationV0(ctx, config)
	summary.WorktreeIsolation = isolation
	if len(isolationIssues) > 0 {
		summary.Issues = append(summary.Issues, isolationIssues...)
		return summary, nil
	}
	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		return summary, nil
	}
	budget := orquestadirectoroperativo.BuildOperationalDirectorAgentBudgetV0(result.Plan)
	summary.AgentBudget = budget
	if budget.Exceeded {
		summary.Issues = append(summary.Issues, orquestadirectoroperativo.OperationalDirectorIssueV0{
			Code:    "recursive_agent_budget_exceeded",
			Field:   "recursive_agent_budget",
			Message: fmt.Sprintf("planned agents %d exceed budget %d", budget.PlannedAgents, budget.MaxAgents),
		})
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
	launchConfig.AgentPrompts = codexDirectorAgentPromptsV0(result.Plan, work, config.DomainContextBlocks, config.StrictDirectorGuards)
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
	return runCodexLaunchDirectorChildWavesAtDepthV0(ctx, config, plan, work, parentLaunch, 1)
}

func runCodexLaunchDirectorChildWavesAtDepthV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentLaunch codexWaveLaunchSummaryV0,
	depth int,
) ([]codexDirectorChildWaveSummaryV0, error) {
	if depth <= 0 || depth > plan.MaxDelegationDepth {
		return nil, nil
	}
	out := make([]codexDirectorChildWaveSummaryV0, 0, len(parentLaunch.Agents))
	for index, parent := range parentLaunch.Agents {
		parentIndex := index + 1
		childWaveRef := fmt.Sprintf("%s-depth-%02d-parent-%02d-children", parentLaunch.WaveRef, depth, parentIndex)
		childConfig := config.Wave
		childConfig.WaveRef = childWaveRef
		childConfig.Agents = plan.MaxSubagentsPerAgent
		childConfig.RuntimeWorkDir = filepath.Join(config.Wave.RuntimeWorkDir, "children", childWaveRef)
		childConfig.Prompt = plan.Objective
		childConfig.AgentPrompts = codexDirectorChildAgentPromptsV0(
			plan,
			work,
			config.DomainContextBlocks,
			config.StrictDirectorGuards,
			parent.AgentRef,
			parentIndex,
			len(parentLaunch.Agents),
			childConfig.Agents,
			depth,
		)
		launch, err := runCodexLaunchWaveV0(ctx, childConfig)
		if err != nil {
			return out, err
		}
		childLaunches, err := runCodexLaunchDirectorChildWavesAtDepthV0(ctx, config, plan, work, launch, depth+1)
		if err != nil {
			return out, err
		}
		out = append(out, codexDirectorChildWaveSummaryV0{
			ParentAgentRef:            parent.AgentRef,
			ParentWaveRef:             parentLaunch.WaveRef,
			ParentIndex:               parentIndex,
			DelegationDepth:           depth,
			MaxDelegationDepth:        plan.MaxDelegationDepth,
			MaxSubagentsPerAgent:      plan.MaxSubagentsPerAgent,
			SubtreeAgentBudget:        codexDirectorSubtreeAgentBudgetV0(launch, childLaunches),
			ReviewRequiredBeforeClose: true,
			Launch:                    launch,
			ChildLaunches:             childLaunches,
		})
	}
	return out, nil
}

func codexDirectorSubtreeAgentBudgetV0(
	launch codexWaveLaunchSummaryV0,
	childLaunches []codexDirectorChildWaveSummaryV0,
) int {
	total := len(launch.Agents)
	for _, child := range childLaunches {
		total += child.SubtreeAgentBudget
	}
	return total
}

func codexDirectorAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	strictDirectorGuards bool,
) []string {
	total := plan.MaxParallelAgents
	if total <= 0 {
		total = 1
	}
	prompts := make([]string, 0, total)
	launchItem := codexDirectorLaunchItemV0(work)
	for i := 1; i <= total; i++ {
		prompts = append(prompts, codexDirectorAgentPromptV0(plan, work, launchItem, domainContextBlocks, strictDirectorGuards, i, total))
	}
	return prompts
}

func codexDirectorAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	strictDirectorGuards bool,
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
	codexDirectorWriteDomainContextV0(&b, domainContextBlocks)
	b.WriteString("Write-set global autorizado:\n")
	codexDirectorWriteListV0(&b, plan.WriteSet)
	b.WriteString("\nWrite-set primario asignado a este agente:\n")
	if len(assigned) == 0 {
		b.WriteString("- Sin shard exclusivo: trabaja en apoyo/review, o en cualquier parte necesaria del write-set global si detectas un bloqueo real.\n")
	} else {
		codexDirectorWriteListV0(&b, assigned)
	}
	b.WriteString("\nTests requeridos por el Director:\n")
	codexDirectorWriteListV0(&b, plan.RequiredTests)
	b.WriteString("\nCriterios de aceptacion:\n")
	codexDirectorWriteListV0(&b, launchItem.AcceptanceCriteria)
	b.WriteString("\nReglas:\n")
	if strictDirectorGuards {
		b.WriteString("- Modo estricto: no cambies ficheros fuera del write-set autorizado.\n")
	} else {
		b.WriteString("- El write-set asignado es ownership inicial, no una venda: si necesitas tocar otro fichero del repo para cumplir el objetivo o arreglar tests, hazlo y justifica la ampliacion en tu resumen.\n")
	}
	b.WriteString("- No borres nada sin revisar uso actual y dejar evidencia en tu resumen.\n")
	b.WriteString("- Si trabajas sobre un tema o artefacto existente, estudia primero que falla y modifica solo lo necesario; rehacerlo entero es una excepcion justificada cuando sea mas sencillo o seguro que corregirlo.\n")
	b.WriteString("- No invoques spawn_agent ni lances hijos manualmente: Orquesta ya materializo los subagentes autorizados para esta ola; coordina por artefactos dentro del write-set.\n")
	b.WriteString("- No uses generadores para redactar contenido doctrinal final ni para inflar palabras con plantillas repetidas; los scripts solo pueden validar o ensamblar artefactos, y debes corregir cualquier repeticion estructural antes de cerrar.\n")
	b.WriteString("- Si tu shard no basta, explica el bloqueo; no invadas el shard de otro agente sin justificarlo.\n")
	b.WriteString("- Al terminar, resume rutas tocadas, pruebas ejecutadas, resultado y bloqueos.\n")
	return b.String()
}

func codexDirectorChildAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	strictDirectorGuards bool,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childTotal int,
	depth int,
) []string {
	prompts := make([]string, 0, childTotal)
	launchItem := codexDirectorLaunchItemV0(work)
	for childIndex := 1; childIndex <= childTotal; childIndex++ {
		prompts = append(prompts, codexDirectorChildAgentPromptV0(
			plan,
			launchItem,
			domainContextBlocks,
			strictDirectorGuards,
			parentAgentRef,
			parentIndex,
			parentTotal,
			childIndex,
			childTotal,
			depth,
		))
	}
	return prompts
}

func codexDirectorChildAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	strictDirectorGuards bool,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childIndex int,
	childTotal int,
	depth int,
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
	b.WriteString("- delegation_depth: " + strconv.Itoa(depth) + "\n")
	b.WriteString("- max_delegation_depth: " + strconv.Itoa(plan.MaxDelegationDepth) + "\n")
	b.WriteString("- max_subagents_per_agent: " + strconv.Itoa(plan.MaxSubagentsPerAgent) + "\n")
	b.WriteString("- launch_item_ref: " + launchItem.ItemID + "\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(plan.Objective + "\n\n")
	codexDirectorWriteRecursiveConfigV0(&b, plan)
	codexDirectorWriteDomainContextV0(&b, domainContextBlocks)
	b.WriteString("Write-set primario del agente padre y compartido por este subarbol:\n")
	if len(assigned) == 0 {
		b.WriteString("- Sin shard exclusivo: trabaja en apoyo/review, o en cualquier parte necesaria del write-set global si detectas un bloqueo real.\n")
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
	if strictDirectorGuards {
		b.WriteString("- Modo estricto: trabaja solo dentro del write-set del subarbol o entrega informe de bloqueo.\n")
	} else {
		b.WriteString("- El write-set del subarbol es ownership inicial; puedes tocar otro fichero del repo si es necesario para cumplir el objetivo o arreglar tests, justificandolo en tu resumen.\n")
	}
	b.WriteString("- No borres nada sin revisar uso actual y dejar evidencia en tu resumen.\n")
	b.WriteString("- Si trabajas sobre un tema o artefacto existente, estudia primero que falla y modifica solo lo necesario; rehacerlo entero es una excepcion justificada cuando sea mas sencillo o seguro que corregirlo.\n")
	b.WriteString("- No lances mas subagentes desde este hijo: la ola hija ya fue materializada por Orquesta.\n")
	b.WriteString("- No declares cierre de tu subarbol: el Director debe revisar entregas, tests y evidencias causales antes de cerrar.\n")
	b.WriteString("- No uses generadores para redactar contenido doctrinal final ni para inflar palabras con plantillas repetidas; los scripts solo pueden validar o ensamblar artefactos.\n")
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
	blocks []codexDirectorDomainContextBlockV0,
) {
	if len(blocks) == 0 {
		return
	}
	b.WriteString("Contexto de dominio inyectado por adaptador:\n")
	for _, block := range blocks {
		if strings.TrimSpace(block.Text) == "" {
			continue
		}
		sourceRef := strings.TrimSpace(block.SourceRef)
		if sourceRef == "" {
			sourceRef = "domain-context"
		}
		b.WriteString("- source_ref: " + sourceRef + "\n")
		b.WriteString(block.Text + "\n")
	}
	b.WriteString("\n")
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
	if codexDirectorWriteSetCoversWorkspaceV0(values) {
		return append([]string(nil), values...)
	}
	out := make([]string, 0)
	for i, value := range values {
		if i%total == index-1 {
			out = append(out, value)
		}
	}
	return out
}

func codexDirectorWriteSetCoversWorkspaceV0(values []string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case ".", "workspace":
			return true
		}
	}
	return false
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

func codexDirectorDomainContextBlocksFromFilesV0(
	paths []string,
) ([]codexDirectorDomainContextBlockV0, error) {
	blocks := make([]codexDirectorDomainContextBlockV0, 0, len(paths))
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("domain_context_file_read_failed %s: %w", filepath.Base(path), err)
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return nil, fmt.Errorf("domain_context_file_empty %s", filepath.Base(path))
		}
		blocks = append(blocks, codexDirectorDomainContextBlockV0{
			SourceRef: filepath.Base(path),
			Text:      text,
		})
	}
	return blocks, nil
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
