package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const serverAutoprogrammingBatchTestReceiptSchemaV0 = "orquesta.server.autoprogramming_batch_test_receipt.v0"

type serverAutoprogrammingBatchTestRunnerV0 struct {
	Executor   orquestaruntimerequiredtest.LocalCommandExecutorV0
	ReceiptDir string
}

type serverAutoprogrammingBatchTestReceiptV0 struct {
	SchemaVersion  string   `json:"schema_version"`
	ReceiptRef     string   `json:"receipt_ref"`
	BatchRef       string   `json:"batch_ref"`
	Revision       string   `json:"revision"`
	TestCommand    string   `json:"test_command"`
	TestHash       string   `json:"test_hash"`
	ClaimRef       string   `json:"claim_ref"`
	GateGeneration uint64   `json:"gate_generation"`
	Status         string   `json:"status"`
	EvidenceRefs   []string `json:"evidence_refs"`
}

var _ orquestaappcodexstack.AutoprogrammingBatchTestRunnerPortV0 = (*serverAutoprogrammingBatchTestRunnerV0)(nil)
var _ orquestaappcodexstack.AutoprogrammingBatchTestClaimReconcilerPortV0 = (*serverAutoprogrammingBatchTestRunnerV0)(nil)

func (runner *serverAutoprogrammingBatchTestRunnerV0) Close() error {
	if runner == nil {
		return nil
	}
	return runner.Executor.Close()
}

func autoprogrammingBatchTestRunnerFromConfigV0(
	serverConfig orquestaserver.ConfigV0,
	canonicalSelfProgrammingDir string,
) (*serverAutoprogrammingBatchTestRunnerV0, error) {
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	if !requiredTestRunnerEnabledFromProjectConfigV0(projectConfig) {
		return nil, nil
	}
	allowed, err := requiredTestAllowedCommandsFromProjectConfigV0(projectConfig)
	if err != nil {
		return nil, err
	}
	outputDir, err := requiredTestOutputDirFromProjectConfigV0(serverConfig, projectConfig)
	if err != nil {
		return nil, err
	}
	env, err := requiredTestEnvFromProjectConfigV0(projectConfig, allowed, outputDir)
	if err != nil {
		return nil, err
	}
	workDir, err := filepath.Abs(strings.TrimSpace(canonicalSelfProgrammingDir))
	if err != nil || strings.TrimSpace(canonicalSelfProgrammingDir) == "" {
		return nil, fmt.Errorf("autoprogramming_batch_canonical_work_dir_invalid")
	}
	executor, err := orquestaruntimerequiredtest.NewLocalCommandExecutorV0(orquestaruntimerequiredtest.LocalCommandExecutorV0{
		ProjectWorkDir:  filepath.Clean(workDir),
		OutputDir:       outputDir,
		AllowedCommands: allowed,
		Env:             env,
		MaxOutputBytes:  int64(intProjectConfigOrEnvOrDefaultV0(envRequiredTestMaxOutputBytesV0, projectConfig.RequiredTestRunner.MaxOutputBytes, 1024*1024)),
		MaxArtifacts:    intProjectConfigOrEnvOrDefaultV0(envRequiredTestOutputMaxArtifactsV0, projectConfig.RequiredTestRunner.MaxArtifacts, 200),
	})
	if err != nil {
		return nil, err
	}
	return &serverAutoprogrammingBatchTestRunnerV0{
		Executor:   executor,
		ReceiptDir: filepath.Join(outputDir, "autoprogramming-batch-receipts-v0"),
	}, nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) RunAutoprogrammingBatchTestV0(
	ctx context.Context,
	request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0,
) (orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0, error) {
	if err := runner.validateRequestV0(request); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, err
	}
	if receipt, found, err := runner.loadReceiptV0(request); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, err
	} else if found {
		return serverAutoprogrammingBatchTestRunResultFromReceiptV0(receipt), nil
	}
	receipt := runner.receiptForRequestV0(request)
	result, err := runner.Executor.RunRequiredTestCommandV0(ctx, orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
		RunRef:        request.BatchRef,
		TaskRef:       request.ClaimRef,
		TestCommand:   request.Test.Command,
		CorrelationID: receipt.ReceiptRef,
		EvidenceRefs:  []string{request.Revision, request.TestHash, fmt.Sprint(request.GateGeneration)},
	})
	if err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, err
	}
	receipt.Status = serverAutoprogrammingBatchTestReceiptStatusV0(result.Status)
	if receipt.Status == "" {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, fmt.Errorf("autoprogramming_batch_test_status_invalid")
	}
	receipt.EvidenceRefs = compactServerAutoprogrammingBatchRefsV0(result.EvidenceRefs)
	if len(receipt.EvidenceRefs) == 0 {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, fmt.Errorf("autoprogramming_batch_test_evidence_required")
	}
	if err := runner.writeReceiptV0(receipt); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, err
	}
	return serverAutoprogrammingBatchTestRunResultFromReceiptV0(receipt), nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) ReconcileAutoprogrammingBatchTestClaimV0(
	ctx context.Context,
	request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0,
) (orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0, bool, error) {
	if err := runner.validateRequestV0(request); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, false, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, false, err
	}
	receipt, found, err := runner.loadReceiptV0(request)
	if err != nil || !found {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, found, err
	}
	return serverAutoprogrammingBatchTestRunResultFromReceiptV0(receipt), true, nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) validateRequestV0(request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0) error {
	canonical, err := filepath.Abs(strings.TrimSpace(request.CanonicalWorkDir))
	if err != nil || strings.TrimSpace(request.CanonicalWorkDir) == "" || filepath.Clean(canonical) != filepath.Clean(runner.Executor.ProjectWorkDir) {
		return fmt.Errorf("autoprogramming_batch_canonical_work_dir_mismatch")
	}
	if strings.TrimSpace(request.BatchRef) == "" || strings.TrimSpace(request.Revision) == "" ||
		strings.TrimSpace(request.Test.Command) == "" || strings.TrimSpace(request.TestHash) == "" || strings.TrimSpace(request.ClaimRef) == "" || request.GateGeneration == 0 {
		return fmt.Errorf("autoprogramming_batch_test_request_invalid")
	}
	wantHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(request.Test)
	if wantHash != strings.TrimSpace(request.TestHash) {
		return fmt.Errorf("autoprogramming_batch_test_hash_mismatch")
	}
	return nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) receiptForRequestV0(request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0) serverAutoprogrammingBatchTestReceiptV0 {
	identity := strings.Join([]string{
		strings.TrimSpace(request.BatchRef),
		strings.TrimSpace(request.Revision),
		strings.TrimSpace(request.TestHash),
		strings.TrimSpace(request.ClaimRef),
		fmt.Sprint(request.GateGeneration),
	}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	return serverAutoprogrammingBatchTestReceiptV0{
		SchemaVersion:  serverAutoprogrammingBatchTestReceiptSchemaV0,
		ReceiptRef:     "autoprogramming-batch-test-receipt-v0-" + hex.EncodeToString(digest[:16]),
		BatchRef:       strings.TrimSpace(request.BatchRef),
		Revision:       strings.TrimSpace(request.Revision),
		TestCommand:    strings.TrimSpace(request.Test.Command),
		TestHash:       strings.TrimSpace(request.TestHash),
		ClaimRef:       strings.TrimSpace(request.ClaimRef),
		GateGeneration: request.GateGeneration,
	}
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) writeReceiptV0(receipt serverAutoprogrammingBatchTestReceiptV0) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("autoprogramming_batch_test_receipt_encode")
	}
	if err := os.MkdirAll(runner.ReceiptDir, 0o700); err != nil {
		return fmt.Errorf("autoprogramming_batch_test_receipt_dir")
	}
	if err := writeCommandDurableFileV0(runner.receiptPathV0(receipt.ReceiptRef), append(data, '\n'), "autoprogramming_batch_test_receipt"); err != nil {
		return err
	}
	return nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) loadReceiptV0(request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0) (serverAutoprogrammingBatchTestReceiptV0, bool, error) {
	want := runner.receiptForRequestV0(request)
	raw, err := os.ReadFile(runner.receiptPathV0(want.ReceiptRef))
	if os.IsNotExist(err) {
		return serverAutoprogrammingBatchTestReceiptV0{}, false, nil
	}
	if err != nil {
		return serverAutoprogrammingBatchTestReceiptV0{}, false, fmt.Errorf("autoprogramming_batch_test_receipt_read")
	}
	var receipt serverAutoprogrammingBatchTestReceiptV0
	if json.Unmarshal(raw, &receipt) != nil || !serverAutoprogrammingBatchTestReceiptMatchesV0(receipt, want) {
		return serverAutoprogrammingBatchTestReceiptV0{}, false, fmt.Errorf("autoprogramming_batch_test_receipt_invalid")
	}
	return receipt, true, nil
}

func (runner *serverAutoprogrammingBatchTestRunnerV0) receiptPathV0(receiptRef string) string {
	return filepath.Join(runner.ReceiptDir, filepath.Base(strings.TrimSpace(receiptRef))+".json")
}

func serverAutoprogrammingBatchTestReceiptMatchesV0(got, want serverAutoprogrammingBatchTestReceiptV0) bool {
	return got.SchemaVersion == serverAutoprogrammingBatchTestReceiptSchemaV0 &&
		got.ReceiptRef == want.ReceiptRef && got.BatchRef == want.BatchRef && got.Revision == want.Revision &&
		got.TestCommand == want.TestCommand && got.TestHash == want.TestHash && got.ClaimRef == want.ClaimRef && got.GateGeneration == want.GateGeneration &&
		(got.Status == orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0 || got.Status == orquestaautoprogramming.AutoprogrammingBatchTestReceiptFailedV0) &&
		len(compactServerAutoprogrammingBatchRefsV0(got.EvidenceRefs)) > 0
}

func serverAutoprogrammingBatchTestReceiptStatusV0(status orquestacionnucleoapp.RequiredTestEvidenceStatusV0) string {
	switch status {
	case orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0:
		return orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0
	case orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0:
		return orquestaautoprogramming.AutoprogrammingBatchTestReceiptFailedV0
	default:
		return ""
	}
}

func serverAutoprogrammingBatchTestRunResultFromReceiptV0(receipt serverAutoprogrammingBatchTestReceiptV0) orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0 {
	return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{
		ReceiptRef:   receipt.ReceiptRef,
		Status:       receipt.Status,
		EvidenceRefs: compactServerAutoprogrammingBatchRefsV0(append(append([]string(nil), receipt.EvidenceRefs...), receipt.ReceiptRef)),
	}
}

func compactServerAutoprogrammingBatchRefsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
