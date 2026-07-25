package codex

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
)

func (adapter *Adapter) loadCompletionProof(
	state *executionState,
) (completionProof, []byte, bool) {
	if adapter == nil || state == nil || state.process == nil ||
		state.process.SupervisorInstance == "" {
		return completionProof{}, nil, false
	}
	filePath := path.Join(state.runPath, completionProofFileName)
	info, err := adapter.root.Lstat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return completionProof{}, nil, false
	}
	maximum := int64(maxCompletionProofExtraBytes)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > maximum {
		return completionProof{}, nil, false
	}
	payload, err := adapter.root.ReadFile(filePath)
	if err != nil || int64(len(payload)) != info.Size() {
		return completionProof{}, nil, false
	}
	defer clearBytes(payload)
	var proof completionProof
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proof); err != nil {
		return completionProof{}, nil, false
	}
	if err := requireJSONEOF(decoder); err != nil {
		return completionProof{}, nil, false
	}
	result, valid := adapter.validCompletionProof(state, proof)
	if !valid {
		return completionProof{}, nil, false
	}
	return proof, result, true
}

func (adapter *Adapter) validCompletionProof(state *executionState, proof completionProof) ([]byte, bool) {
	record := state.process
	if record == nil || record.SchemaVersion != cgroupProcessSchemaVersion ||
		proof.SchemaVersion != completionProofSchema ||
		!verifyCompletionProofSignature(*record, proof) ||
		proof.SupervisorInstance != record.SupervisorInstance ||
		proof.ExecutionRef != record.ExecutionRef || proof.ExecutionRef != state.receipt.ExecutionRef.String() ||
		proof.RequestHash != record.RequestHash ||
		(proof.RequestHash != state.requestHash && proof.RequestHash != state.terminalRequestHash) ||
		proof.SpecHash != state.receipt.SpecHash || proof.ArtifactMediaType == "" ||
		proof.RuntimeScope != record.RuntimeScope ||
		proof.RuntimeScope != adapter.config.RuntimeScope ||
		proof.SupervisorPID != record.PID || proof.SupervisorPGID != record.PGID ||
		proof.SupervisorBootID != record.BootID || proof.SupervisorBirthMarker != record.BirthMarker ||
		proof.CgroupRootDevice != record.CgroupRootDevice ||
		proof.CgroupRootInode != record.CgroupRootInode ||
		proof.CgroupControlDevice != record.CgroupControlDevice ||
		proof.CgroupControlInode != record.CgroupControlInode ||
		proof.CgroupName != record.CgroupName ||
		proof.CgroupDevice != record.CgroupDevice ||
		proof.CgroupInode != record.CgroupInode ||
		!proof.TreeGone || proof.DiagnosticSize < 0 ||
		proof.DiagnosticSize > adapter.config.MaxDiagnosticBytes ||
		proof.DiagnosticHash == "" ||
		(proof.Cause != supervisorCauseNatural && proof.Cause != supervisorCauseTimeout &&
			proof.Cause != supervisorCauseStop) {
		return nil, false
	}
	if proof.WorkerPID <= 0 || proof.WorkerPGID <= 0 ||
		proof.WorkerBootID != record.BootID || proof.WorkerBirthMarker == "" {
		// A failed exec has no worker identity and therefore cannot prove a
		// completed execution. It remains fail-closed as interrupted.
		return nil, false
	}
	if proof.Cause == supervisorCauseStop {
		intent, found, err := adapter.loadWinningStopSignalIntent(state.runPath)
		if err != nil || !found ||
			proof.StopRequestHash != intent.RequestHash ||
			proof.StopIdempotency != intent.Idempotency ||
			proof.StopMode != string(intent.Mode) ||
			proof.StopSequence != intent.Sequence {
			return nil, false
		}
	} else if proof.StopRequestHash != "" || proof.StopIdempotency != "" ||
		proof.StopMode != "" || proof.StopSequence != 0 {
		return nil, false
	}
	if adapter.cgroups == nil {
		return nil, false
	}
	populated, err := adapter.cgroups.populated(*record)
	if err != nil || populated {
		return nil, false
	}
	if proof.Exited == (proof.Signal != 0) ||
		(proof.Exited && (proof.ExitCode < 0 || proof.ExitCode > 255)) ||
		(!proof.Exited && proof.ExitCode != 0) {
		return nil, false
	}
	if proof.ResultFound {
		if proof.ResultSize < 0 ||
			(!proof.ResultTooLarge && proof.ResultHash == "") ||
			(proof.ResultTooLarge && (proof.ResultSize <= state.maxOutput || proof.ResultHash != "")) {
			return nil, false
		}
	} else if proof.ResultSize != 0 || proof.ResultHash != "" || proof.ResultTooLarge {
		return nil, false
	}
	result, size, digest, found, tooLarge, err := adapter.readResultForProof(state.runPath, state.maxOutput)
	if err != nil || size != proof.ResultSize || digest != proof.ResultHash ||
		found != proof.ResultFound || tooLarge != proof.ResultTooLarge {
		clearBytes(result)
		return nil, false
	}
	return result, true
}

func (adapter *Adapter) readResultForProof(
	runPath string,
	maximum int64,
) ([]byte, int64, string, bool, bool, error) {
	filePath := path.Join(runPath, lastMessageFileName)
	info, err := adapter.root.Lstat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, "", false, false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Mode().Perm()&0o077 != 0 {
		return nil, 0, "", false, false, errors.New("invalid completion result")
	}
	if info.Size() > maximum {
		return nil, info.Size(), "", true, true, nil
	}
	payload, err := adapter.root.ReadFile(filePath)
	if err != nil {
		return nil, 0, "", false, false, err
	}
	if int64(len(payload)) != info.Size() {
		clearBytes(payload)
		return nil, 0, "", false, false, errors.New("unstable completion result")
	}
	after, err := adapter.root.Lstat(filePath)
	if err != nil || !os.SameFile(info, after) || after.Size() != int64(len(payload)) {
		clearBytes(payload)
		return nil, 0, "", false, false, errors.New("unstable completion result")
	}
	digest := sha256.Sum256(payload)
	return payload, int64(len(payload)), "sha256:" + hex.EncodeToString(digest[:]), true, false, nil
}

func (adapter *Adapter) removeCompletionArtifacts(runPath string) error {
	var removeErr error
	for _, name := range []string{completionProofFileName} {
		err := adapter.root.Remove(path.Join(runPath, name))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			removeErr = errors.Join(removeErr, err)
		}
	}
	if removeErr != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: removeErr}
	}
	return adapter.syncDirectoryCausally(runPath)
}
