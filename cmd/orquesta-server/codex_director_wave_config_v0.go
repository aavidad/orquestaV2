package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func codexDirectorWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexDirectorWaveConfigV0, error) {
	projectConfig := codexWaveProjectConfigFromArgsEnvV0(args)
	flags := flag.NewFlagSet("codex-launch-director-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var domainContextFiles codexDirectorStringListFlagV0
	directorWaveLimits := codexDirectorWaveLimitsEnvConfigFromEnvV0()

	agents := flags.Int("agents", directorWaveLimits.Agents, "numero de agentes que puede lanzar el Director")
	waveRef := flags.String("wave-ref", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRefV0, ""), "ref opaca de la ola")
	objective := flags.String("objective", "", "objetivo para el Director Operativo")
	objectiveFile := flags.String("objective-file", "", "archivo con objetivo para el Director Operativo")
	writeSet := flags.String("write-set", "", "write-set separado por comas")
	requiredTests := flags.String("required-tests", "", "tests requeridos separados por comas")
	branchRef := flags.String("branch-ref", "", "ref de rama aislada")
	worktreeRef := flags.String("worktree-ref", "", "ref de worktree aislada")
	requestRef := flags.String("request-ref", "", "request_ref opcional")
	runRef := flags.String("run-ref", "", "run_ref opcional")
	projectRef := flags.String("project-ref", envOrDefaultV0(envCodexDirectorProjectRefV0, "orquesta"), "project_ref opaco")
	domainRefs := flags.String("domain-ref", strings.TrimSpace(os.Getenv(envCodexDirectorDomainRefsV0)), "domain_refs separados por comas")
	projectDir := flags.String("project-dir", strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0)), "directorio de trabajo del proyecto")
	runtimeDir := flags.String("runtime-dir", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRuntimeWorkDirV0, ""), "directorio runtime de esta ola")
	commandPath := flags.String("command", strings.TrimSpace(os.Getenv(envCodexCommandV0)), "ruta al binario codex")
	sourceCodeHome := flags.String("source-code-home", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSourceCodeHomeV0, ""), "CODEX_HOME fuente para copiar credenciales/config")
	model := flags.String("model", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveModelV0, ""), "modelo Codex opcional")
	reasoningEffort := flags.String("reasoning-effort", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveReasoningEffortV0, envOrDefaultV0(envCodexReasoningEffortV0, "medium")), "esfuerzo de razonamiento")
	profile := flags.String("profile", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveProfileV0, ""), "perfil Codex opcional")
	sandbox := flags.String("sandbox", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSandboxV0, "workspace-write"), "sandbox Codex")
	approval := flags.String("approval-policy", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveApprovalPolicyV0, "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveExtraArgsV0, ""), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveIsolateHomeV0, false), "copiar CODEX_HOME por agente bajo el runtime")
	strictCredentialProjection := flags.Bool("strict-credential-projection", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveStrictCredentialProjectionV0, false), "exigir proyeccion aislada con auth/config presentes")
	projectMemories := flags.Bool("project-memories", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectMemoriesV0, false), "permitir proyeccion explicita de memories de CODEX_HOME")
	projectionMaxFiles := flags.Int("projection-max-files", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFilesV0, codexWaveProjectionDefaultMaxFilesV0), "maximo de ficheros a copiar desde CODEX_HOME")
	projectionMaxFileBytes := flags.Int("projection-max-file-bytes", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFileBytesV0, int(codexWaveProjectionDefaultMaxFileBytesV0)), "maximo de bytes por fichero proyectado")
	projectionMaxTotalBytes := flags.Int("projection-max-total-bytes", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxTotalBytesV0, int(codexWaveProjectionDefaultMaxTotalBytesV0)), "maximo total de bytes proyectados desde CODEX_HOME")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeV0, false), "purgar runtime de esta ola antes de materializar")
	purgeConfirm := flags.String("confirm-purge-runtime", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeConfirmV0, ""), "confirmacion explicita: debe coincidir con wave-ref")
	purgeReportOnly := flags.Bool("purge-runtime-report", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeReportV0, false), "reportar purga sin borrar")
	allowUnmanagedLaunch := flags.Bool("allow-unmanaged-launch", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveAllowUnmanagedLaunchV0, false), "breakglass auditado para lanzar agentes fuera del servidor/cola de Orquesta")
	unmanagedLaunchReason := flags.String("unmanaged-launch-reason", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchReasonV0, ""), "motivo auditado para el breakglass unmanaged")
	unmanagedLaunchConfirm := flags.String("confirm-unmanaged-launch", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchConfirmV0, ""), "confirmacion explicita unmanaged: debe coincidir con wave-ref")
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
	objectiveText, objectiveInputReceipt, err := codexDirectorObjectiveTextV0(*objective, *objectiveFile, flags.Args())
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
		projectionMaxFiles:      *projectionMaxFiles,
		projectionMaxFileBytes:  *projectionMaxFileBytes,
		projectionMaxTotalBytes: *projectionMaxTotalBytes,
		allowUnmanagedLaunch:    *allowUnmanagedLaunch, unmanagedLaunchReason: *unmanagedLaunchReason,
		unmanagedLaunchConfirm: *unmanagedLaunchConfirm,
		projectMemories:        *projectMemories, requestRef: *requestRef, runRef: *runRef,
		projectRef: *projectRef, domainRefs: *domainRefs, worktreeRef: *worktreeRef,
		branchRef: *branchRef, writeSet: *writeSet, requiredTests: *requiredTests,
		allowRecursive: *allowRecursive, maxDepth: *maxDepth, maxChildren: *maxChildren,
		recursiveAgentBudget: *recursiveAgentBudget, strictDirectorGuards: *strictDirectorGuards,
		allowGlobalWriteSet: *allowGlobalWriteSet, allowPlaceholderTests: *allowPlaceholderTests,
		guardOverrideReason: *guardOverrideReason, guardOverrideEvidenceRefs: *guardOverrideEvidenceRefs,
		domainContextFiles: append(codexDirectorCSVV0(os.Getenv(envCodexDirectorDomainContextFilesV0)), []string(domainContextFiles)...),
		operatorInputs:     []codexWaveOperatorInputReceiptV0{objectiveInputReceipt},
	})
}

type parsedCodexDirectorWaveFlagsV0 struct {
	agents, maxDepth, maxChildren, recursiveAgentBudget                 int
	projectionMaxFiles, projectionMaxFileBytes, projectionMaxTotalBytes int
	waveRef, objectiveText, projectWorkDir, rootRuntimeDir              string
	resolvedCommand, sourceHome, model, reasoningEffort, profile        string
	sandbox, approval, extraArgs, requestRef, runRef, projectRef        string
	domainRefs, worktreeRef, branchRef, writeSet, requiredTests         string
	guardOverrideReason, guardOverrideEvidenceRefs                      string
	purgeConfirm                                                        string
	unmanagedLaunchReason, unmanagedLaunchConfirm                       string
	isolateHome, dryRun, purgeRuntime, allowRecursive                   bool
	purgeReportOnly                                                     bool
	allowUnmanagedLaunch                                                bool
	strictCredentialProjection, projectMemories                         bool
	strictDirectorGuards, allowGlobalWriteSet, allowPlaceholderTests    bool
	domainContextFiles                                                  []string
	operatorInputs                                                      []codexWaveOperatorInputReceiptV0
}

func codexDirectorWaveConfigFromParsedFlagsV0(values parsedCodexDirectorWaveFlagsV0) (codexDirectorWaveConfigV0, error) {
	branchValue := strings.TrimSpace(values.branchRef)
	writeSetValues := codexDirectorCSVV0(values.writeSet)
	requiredTestValues := codexDirectorCSVV0(values.requiredTests)
	effectiveStrictDirectorGuards := values.strictDirectorGuards || !values.dryRun
	domainContextBlocks, domainContextReceipts, err := codexDirectorDomainContextBlocksFromFilesV0(values.domainContextFiles)
	if err != nil {
		return codexDirectorWaveConfigV0{}, err
	}
	operatorInputs := append(codexWaveOperatorInputReceiptsCopyV0(values.operatorInputs), domainContextReceipts...)
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
			PathEnv: codexWaveStringValueFromProjectConfigFileV0(codexWaveProjectConfigFromEnvBestEffortV0(), envCodexWavePathV0, envOrDefaultV0(envCodexPathV0, os.Getenv("PATH"))),
			Model:   strings.TrimSpace(values.model), ReasoningEffort: codexReasoningEffortPolicyV0(values.reasoningEffort),
			Profile: strings.TrimSpace(values.profile), Sandbox: strings.TrimSpace(values.sandbox),
			ApprovalPolicy: strings.TrimSpace(values.approval), ExtraArgs: strings.Fields(values.extraArgs),
			IsolateHome: values.isolateHome, DryRun: values.dryRun, PurgeRuntime: values.purgeRuntime,
			PurgeConfirm: values.purgeConfirm, PurgeReportOnly: values.purgeReportOnly,
			AllowUnmanagedLaunch: values.allowUnmanagedLaunch, UnmanagedLaunchReason: strings.TrimSpace(values.unmanagedLaunchReason),
			UnmanagedLaunchConfirm: strings.TrimSpace(values.unmanagedLaunchConfirm),
			CredentialProjectionPolicy: codexWaveCredentialProjectionPolicyWithBoundsV0(
				values.strictCredentialProjection,
				values.projectMemories,
				values.projectionMaxFiles,
				int64(values.projectionMaxFileBytes),
				int64(values.projectionMaxTotalBytes),
			),
			OperatorInputs: operatorInputs,
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

func codexDirectorObjectiveTextV0(objective string, objectiveFile string, trailing []string) (string, codexWaveOperatorInputReceiptV0, error) {
	if strings.TrimSpace(objectiveFile) != "" {
		text, receipt, err := codexWaveOperatorTextFromFileV0("objective", objectiveFile)
		if err != nil {
			return "", codexWaveOperatorInputReceiptV0{}, err
		}
		return text, receipt, nil
	}
	if strings.TrimSpace(objective) != "" {
		text := strings.TrimSpace(objective)
		return text, codexWaveOperatorInputReceiptFromTextV0("objective", "operator_inline", text), nil
	}
	if len(trailing) > 0 {
		text := strings.TrimSpace(strings.Join(trailing, " "))
		if text != "" {
			return text, codexWaveOperatorInputReceiptFromTextV0("objective", "operator_trailing", text), nil
		}
	}
	return "", codexWaveOperatorInputReceiptV0{}, errors.New("objective_required")
}

func codexDirectorDomainContextBlocksFromFilesV0(paths []string) ([]codexDirectorDomainContextBlockV0, []codexWaveOperatorInputReceiptV0, error) {
	blocks := make([]codexDirectorDomainContextBlockV0, 0, len(paths))
	receipts := make([]codexWaveOperatorInputReceiptV0, 0, len(paths))
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		text, receipt, err := codexWaveOperatorTextFromFileV0("domain_context", path)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, codexDirectorDomainContextBlockV0{SourceRef: filepath.Base(path), Text: text})
		receipts = append(receipts, receipt)
	}
	return blocks, receipts, nil
}

func codexDirectorDefaultRefV0(value string, prefix string, waveRef string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	waveRef = strings.TrimSpace(waveRef)
	if waveRef == "" {
		waveRef, _ = codexWaveRefGeneratorV0.NextRefV0("codex-wave-v0", "director-wave")
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
