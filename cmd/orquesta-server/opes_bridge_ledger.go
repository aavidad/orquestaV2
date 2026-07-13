package main

import (
	"context"
	"path/filepath"

	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

const opesBridgeExternalSystemV0 = "opes"

func opesBridgeInputLedgerFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	dryRun bool,
	stateDirs ...string,
) (externalBridgeInputLedgerV0, error) {
	if dryRun || opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeInputLedgerDisabledV0, false) {
		return nil, nil
	}
	path := opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeInputLedgerPathV0)
	if path == "" {
		if len(stateDirs) > 0 && filepath.Clean(stateDirs[0]) != "." {
			path = filepath.Join(stateDirs[0], "external-bridge-input-ledger.json")
			return newFileExternalBridgeInputLedgerV0(path)
		}
		serverConfig, err := serverConfigFromEnvV0()
		if err != nil {
			return nil, err
		}
		defer releaseServerProjectConfigSnapshotV0(serverConfig)
		path = filepath.Join(serverConfig.StateDir, "external-bridge-input-ledger.json")
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
	metadata ...externalBridgeInputRunMetadataV0,
) error {
	return externalBridgeRecordSubmittedInputV0(
		ctx,
		ledger,
		opesBridgeExternalSystemV0,
		job.ID,
		runRef,
		changeRef,
		metadata...,
	)
}

func opesBridgeInputLedgerKeyV0(jobRef string) string {
	return externalBridgeInputLedgerKeyV0(opesBridgeExternalSystemV0, jobRef)
}
