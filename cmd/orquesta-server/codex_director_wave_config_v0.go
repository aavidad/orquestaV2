package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func codexDirectorWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexDirectorWaveConfigV0, error) {
	flags := flag.NewFlagSet("codex-launch-director-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var domainContextFiles codexDirectorStringListFlagV0
	directorWaveLimits := codexDirectorWaveLimitsEnvConfigFromEnvV0()

	agents := flags.Int("agents", directorWaveLimits.Agents, "numero de agentes que puede lanzar el Director")
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
	strictCredentialProjection := flags.Bool("strict-credential-projection", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_STRICT_CREDENTIAL_PROJECTION", false), "exigir proyeccion aislada con auth/config presentes")
	projectMemories := flags.Bool("project-memories", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PROJECT_MEMORIES", false), "permitir proyeccion explicita de memories de CODEX_HOME")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME", false), "purgar runtime de esta ola antes de materializar")
	purgeConfirm := flags.String("confirm-purge-runtime", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_CONFIRM")), "confirmacion explicita: debe coincidir con wave-ref")
	purgeReportOnly := flags.Bool("purge-runtime-report", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_REPORT", false), "reportar purga sin borrar")
	allowUnmanagedLaunch := flags.Bool("allow-unmanaged-launch", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ALLOW_UNMANAGED_LAUNCH", false), "breakglass auditado para lanzar agentes fuera del servidor/cola de Orquesta")
	unmanagedLaunchReason := flags.String("unmanaged-launch-reason", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_REASON")), "motivo auditado para el breakglass unmanaged")
	unmanagedLaunchConfirm := flags.String("confirm-unmanaged-launch", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_CONFIRM")), "confirmacion explicita unmanaged: debe coincidir con wave-ref")
	allowRecursive := flags.Bool("allow-recursive-delegation", false, "permitir que el Director gobierne delegacion recursiva")
	maxDepth := flags.Int("max-delegation-depth", 0, "profundidad maxima de delegacion")
	maxChildren := flags.Int("max-subagents-per-agent", directorWaveLimits.MaxSubagentsPerAgent, "fanout maximo por agente")
	recursiveAgentBudget := flags.Int("recursive-agent-budget", directorWaveLimits.RecursiveAgentBudget, "presupuesto global maximo de agentes en arbol recursivo")
	strictDirectorGuards := flags.Bool("strict-director-guards", false, "exigir branch/write-set/tests explicitos del Director")
	allowGlobalWriteSet := flags.Bool("allow-global-write-set", false, "opt-in auditado para write_set global")
	allowPlaceholderTests := flags.Bool("allow-placeholder-tests", false, "opt-in auditado para tests placeholder")
	guardOverrideReason := flags.String("guard-override-reason", "", "razon auditada para relajar guardas estrictas")
	guardOverrideEvidenceRefs := flags.String("guard-override-evidence-ref", "", "evidence refs separadas por comas para relajar guardas estrictas")
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
	rootRuntimeDir, err := codexWaveRuntimeDirForLaunchV0(*runtimeDir, projectWorkDir, ref, *purgeRuntime)
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
	return codexDirectorWaveConfigFromParsedFlagsV0(parsedCodexDirectorWaveFlagsV0{
		agents: *agents, waveRef: ref, objectiveText: objectiveText, projectWorkDir: projectWorkDir,
		rootRuntimeDir: rootRuntimeDir, resolvedCommand: resolvedCommand, sourceHome: sourceHome,
		model: *model, reasoningEffort: *reasoningEffort, profile: *profile, sandbox: *sandbox,
		approval: *approval, extraArgs: *extraArgs, isolateHome: *isolateHome, dryRun: *dryRun,
		purgeRuntime: *purgeRuntime, strictCredentialProjection: *strictCredentialProjection,
		purgeConfirm: *purgeConfirm, purgeReportOnly: *purgeReportOnly,
		allowUnmanagedLaunch: *allowUnmanagedLaunch, unmanagedLaunchReason: *unmanagedLaunchReason,
		unmanagedLaunchConfirm: *unmanagedLaunchConfirm,
		projectMemories:        *projectMemories, requestRef: *requestRef, runRef: *runRef,
		projectRef: *projectRef, domainRefs: *domainRefs, worktreeRef: *worktreeRef,
		branchRef: *branchRef, writeSet: *writeSet, requiredTests: *requiredTests,
		allowRecursive: *allowRecursive, maxDepth: *maxDepth, maxChildren: *maxChildren,
		recursiveAgentBudget: *recursiveAgentBudget, strictDirectorGuards: *strictDirectorGuards,
		allowGlobalWriteSet: *allowGlobalWriteSet, allowPlaceholderTests: *allowPlaceholderTests,
		guardOverrideReason: *guardOverrideReason, guardOverrideEvidenceRefs: *guardOverrideEvidenceRefs,
		domainContextFiles: append(codexDirectorCSVV0(os.Getenv("ORQUESTA_CODEX_DIRECTOR_DOMAIN_CONTEXT_FILES")), []string(domainContextFiles)...),
	})
}

type parsedCodexDirectorWaveFlagsV0 struct {
	agents, maxDepth, maxChildren, recursiveAgentBudget              int
	waveRef, objectiveText, projectWorkDir, rootRuntimeDir           string
	resolvedCommand, sourceHome, model, reasoningEffort, profile     string
	sandbox, approval, extraArgs, requestRef, runRef, projectRef     string
	domainRefs, worktreeRef, branchRef, writeSet, requiredTests      string
	guardOverrideReason, guardOverrideEvidenceRefs                   string
	purgeConfirm                                                     string
	unmanagedLaunchReason, unmanagedLaunchConfirm                    string
	isolateHome, dryRun, purgeRuntime, allowRecursive                bool
	purgeReportOnly                                                  bool
	allowUnmanagedLaunch                                             bool
	strictCredentialProjection, projectMemories                      bool
	strictDirectorGuards, allowGlobalWriteSet, allowPlaceholderTests bool
	domainContextFiles                                               []string
}

func codexDirectorWaveConfigFromParsedFlagsV0(values parsedCodexDirectorWaveFlagsV0) (codexDirectorWaveConfigV0, error) {
	branchValue := strings.TrimSpace(values.branchRef)
	writeSetValues := codexDirectorCSVV0(values.writeSet)
	requiredTestValues := codexDirectorCSVV0(values.requiredTests)
	effectiveStrictDirectorGuards := values.strictDirectorGuards || !values.dryRun
	domainContextBlocks, err := codexDirectorDomainContextBlocksFromFilesV0(values.domainContextFiles)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	if !effectiveStrictDirectorGuards {
		if branchValue == "" {
			branchValue = codexDirectorDefaultBranchRefV0(values.waveRef)
		}
		if len(writeSetValues) == 0 {
			writeSetValues = []string{"."}
		}
		if len(requiredTestValues) == 0 {
			requiredTestValues = []string{"operator-validation-required"}
		}
	}
	config := codexDirectorWaveConfigV0{
		Wave: codexWaveConfigV0{
			Agents: values.agents, WaveRef: values.waveRef, Prompt: values.objectiveText,
			ProjectWorkDir: values.projectWorkDir, RuntimeWorkDir: values.rootRuntimeDir,
			CommandPath: values.resolvedCommand, SourceCodeHome: values.sourceHome,
			PathEnv: envOrDefaultV0("ORQUESTA_CODEX_WAVE_PATH", envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH"))),
			Model:   strings.TrimSpace(values.model), ReasoningEffort: codexReasoningEffortPolicyV0(values.reasoningEffort),
			Profile: strings.TrimSpace(values.profile), Sandbox: strings.TrimSpace(values.sandbox),
			ApprovalPolicy: strings.TrimSpace(values.approval), ExtraArgs: strings.Fields(values.extraArgs),
			IsolateHome: values.isolateHome, DryRun: values.dryRun, PurgeRuntime: values.purgeRuntime,
			PurgeConfirm: values.purgeConfirm, PurgeReportOnly: values.purgeReportOnly,
			AllowUnmanagedLaunch: values.allowUnmanagedLaunch, UnmanagedLaunchReason: strings.TrimSpace(values.unmanagedLaunchReason),
			UnmanagedLaunchConfirm:     strings.TrimSpace(values.unmanagedLaunchConfirm),
			CredentialProjectionPolicy: codexWaveCredentialProjectionPolicyV0FromFlags(values.strictCredentialProjection, values.projectMemories),
		},
		RequestRef: codexDirectorDefaultRefV0(values.requestRef, "req", values.waveRef),
		RunRef:     codexDirectorDefaultRefV0(values.runRef, "run", values.waveRef),
		ProjectRef: strings.TrimSpace(values.projectRef), DomainRefs: codexDirectorCSVV0(values.domainRefs),
		Objective: values.objectiveText, WorktreeRef: codexDirectorWorktreeRefV0(values.worktreeRef, values.waveRef, effectiveStrictDirectorGuards),
		BranchRef: branchValue, WriteSet: writeSetValues, RequiredTests: requiredTestValues,
		AllowRecursiveDelegation: values.allowRecursive, MaxDelegationDepth: values.maxDepth,
		MaxSubagentsPerAgent: values.maxChildren, RecursiveAgentBudget: values.recursiveAgentBudget,
		StrictDirectorGuards: effectiveStrictDirectorGuards, AllowGlobalWriteSet: values.allowGlobalWriteSet,
		AllowPlaceholderTests: values.allowPlaceholderTests, GuardOverrideReason: strings.TrimSpace(values.guardOverrideReason),
		GuardOverrideEvidenceRefs: codexDirectorCSVV0(values.guardOverrideEvidenceRefs),
		DomainContextBlocks:       domainContextBlocks,
	}
	if err := codexWaveValidateProjectionPolicyV0(config.Wave); err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	return config, nil
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

func codexDirectorDomainContextBlocksFromFilesV0(paths []string) ([]codexDirectorDomainContextBlockV0, error) {
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
		blocks = append(blocks, codexDirectorDomainContextBlockV0{SourceRef: filepath.Base(path), Text: text})
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
