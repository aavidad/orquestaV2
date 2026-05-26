package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

const opesBridgeExternalSystemV0 = "opes"

func opesBridgeInputLedgerFromEnvV0(
	dryRun bool,
) (externalBridgeInputLedgerV0, error) {
	if dryRun || strings.TrimSpace(os.Getenv(envOPESBridgeInputLedgerDisabledV0)) == "1" {
		return nil, nil
	}
	path := strings.TrimSpace(os.Getenv(envOPESBridgeInputLedgerPathV0))
	if path == "" {
		config, err := serverConfigFromEnvV0()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(config.StateDir, "external-bridge-input-ledger.json")
	}
	return newFileExternalBridgeInputLedgerV0(path)
}

func opesBridgeSubmittedLedgerEntryV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	job orquestaopesconnector.ExternalJobV0,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	return externalBridgeSubmittedInputLedgerEntryV0(
		ctx,
		ledger,
		opesBridgeExternalSystemV0,
		job.ID,
	)
}

func opesBridgeRecordSubmittedV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	job orquestaopesconnector.ExternalJobV0,
	runRef string,
	changeRef string,
) error {
	return externalBridgeRecordSubmittedInputV0(
		ctx,
		ledger,
		opesBridgeExternalSystemV0,
		job.ID,
		runRef,
		changeRef,
	)
}

func opesBridgeInputLedgerKeyV0(jobRef string) string {
	return externalBridgeInputLedgerKeyV0(opesBridgeExternalSystemV0, jobRef)
}
