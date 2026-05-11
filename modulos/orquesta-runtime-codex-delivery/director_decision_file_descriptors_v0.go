package orquestaruntimecodexdelivery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
)

const DefaultDirectorAgentDecisionFileNameV0 = "director_decisions.json"

type CodexReceiptDirectorDecisionFileDescriptorProviderV0 struct {
	Store    CodexReceiptDescriptorStorePortV0
	FileName string
}

var _ orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorProviderPortV0 = CodexReceiptDirectorDecisionFileDescriptorProviderV0{}

func (provider CodexReceiptDirectorDecisionFileDescriptorProviderV0) ListDirectorAgentDecisionFilesV0(
	ctx context.Context,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) ([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider.Store == nil {
		return nil, fmt.Errorf("director_decision_file_descriptors: store requerido")
	}
	fileName := directorDecisionFileNameV0(provider.FileName)
	if fileName == "" {
		return nil, fmt.Errorf("director_decision_file_descriptors: file_name invalido")
	}
	runID := strings.TrimSpace(request.RunID)
	receipts, err := provider.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		CodexReceiptDescriptorRequestV0{
			RunID:         runID,
			CorrelationID: strings.TrimSpace(request.CorrelationID),
		},
	)
	if err != nil {
		return nil, err
	}
	return directorDecisionDescriptorsFromReceiptsV0(receipts, runID, fileName)
}

func directorDecisionDescriptorsFromReceiptsV0(
	receipts []CodexReceiptDescriptorV0,
	runID string,
	fileName string,
) ([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0, error) {
	descriptors := make(
		[]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0,
		0,
		len(receipts),
	)
	for _, receipt := range receipts {
		receiptRunID := strings.TrimSpace(receipt.RunID)
		if runID != "" && receiptRunID != runID {
			continue
		}
		ackPath := strings.TrimSpace(receipt.AckPath)
		if ackPath == "" {
			return nil, fmt.Errorf("director_decision_file_descriptors: ack_path requerido")
		}
		decisionPath := filepath.Join(filepath.Dir(ackPath), fileName)
		exists, err := directorDecisionFileExistsV0(decisionPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		descriptors = append(descriptors, orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0{
			DescriptorRef: directorDecisionDescriptorRefV0(receipt),
			RunID:         receiptRunID,
			Path:          decisionPath,
		})
	}
	return descriptors, nil
}

func directorDecisionFileExistsV0(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("director_decision_file_descriptors: decision_file no regular")
		}
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("director_decision_file_descriptors: stat decision_file: %s", directorDecisionStatErrorKindV0(err))
}

func directorDecisionStatErrorKindV0(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "permiso"
	}
	return "fallo"
}

func directorDecisionFileNameV0(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = DefaultDirectorAgentDecisionFileNameV0
	}
	if fileName == "." || fileName == ".." {
		return ""
	}
	if filepath.Base(fileName) != fileName || strings.Contains(fileName, `\`) {
		return ""
	}
	return fileName
}

func directorDecisionDescriptorRefV0(receipt CodexReceiptDescriptorV0) string {
	ref := strings.TrimSpace(receipt.DescriptorRef)
	if ref == "" {
		ref = codexReceiptDescriptorRefV0(receipt.RunID, receipt.AgentRef)
	}
	return ref + "-director-decisions"
}
