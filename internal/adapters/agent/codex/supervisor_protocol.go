package codex

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	localSupervisorArgument       = "__orquesta_internal_codex_supervisor_v1"
	supervisorEnvelopeSchema      = 1
	completionProofSchema         = 1
	completionProofFileName       = "completion-proof.json"
	maxSupervisorEnvelopeBytes    = 64 << 20
	maxCompletionProofExtraBytes  = 16 << 10
	supervisorCauseNatural        = "natural"
	supervisorCauseTimeout        = "timeout"
	supervisorInternalFailureExit = 125
)

type supervisorEnvelope struct {
	SchemaVersion        int      `json:"schema_version"`
	SupervisorInstance   string   `json:"supervisor_instance"`
	ExecutionRef         string   `json:"execution_ref"`
	RequestHash          string   `json:"request_hash"`
	SpecHash             string   `json:"spec_hash"`
	ArtifactMediaType    string   `json:"artifact_media_type"`
	RuntimeScope         string   `json:"runtime_scope"`
	CompletionPublicKey  string   `json:"completion_public_key"`
	CompletionPrivateKey []byte   `json:"completion_private_key"`
	Command              string   `json:"command"`
	Arguments            []string `json:"arguments"`
	Environment          []string `json:"environment"`
	WorkingDirectory     string   `json:"working_directory"`
	Prompt               string   `json:"prompt"`
	RunDirectory         string   `json:"run_directory"`
	ResultPath           string   `json:"result_path"`
	TimeoutNanos         int64    `json:"timeout_nanos"`
	PipeDrainDelayNanos  int64    `json:"pipe_drain_delay_nanos"`
	MaxDiagnosticBytes   int64    `json:"max_diagnostic_bytes"`
	MaxOutputBytes       int64    `json:"max_output_bytes"`
}

type supervisorCommand struct {
	command                            *exec.Cmd
	gateReader, gateWriter, sealed     *os.File
	diagnosticReader, diagnosticWriter *os.File
}

type completionProof struct {
	SchemaVersion      int    `json:"schema_version"`
	SupervisorInstance string `json:"supervisor_instance"`
	ExecutionRef       string `json:"execution_ref"`
	RequestHash        string `json:"request_hash"`
	SpecHash           string `json:"spec_hash"`
	ArtifactMediaType  string `json:"artifact_media_type"`
	RuntimeScope       string `json:"runtime_scope"`

	SupervisorPID         int    `json:"supervisor_pid"`
	SupervisorPGID        int    `json:"supervisor_pgid"`
	SupervisorBootID      string `json:"supervisor_boot_id"`
	SupervisorBirthMarker string `json:"supervisor_birth_marker"`
	WorkerPID             int    `json:"worker_pid"`
	WorkerPGID            int    `json:"worker_pgid"`
	WorkerBootID          string `json:"worker_boot_id"`
	WorkerBirthMarker     string `json:"worker_birth_marker"`

	Cause          string `json:"cause"`
	Exited         bool   `json:"exited"`
	ExitCode       int    `json:"exit_code"`
	Signal         int    `json:"signal"`
	TreeGone       bool   `json:"tree_gone"`
	ResultFound    bool   `json:"result_found"`
	ResultSize     int64  `json:"result_size"`
	ResultHash     string `json:"result_sha256"`
	ResultTooLarge bool   `json:"result_too_large"`

	DiagnosticSize      int64  `json:"diagnostic_size"`
	DiagnosticHash      string `json:"diagnostic_sha256"`
	DiagnosticTruncated bool   `json:"diagnostic_truncated"`
	Signature           string `json:"signature"`
}

func IsLocalSupervisorInvocation(arguments []string) bool {
	return len(arguments) == 1 && arguments[0] == localSupervisorArgument
}

// RunLocalSupervisor is the private, descriptor-authenticated entry point used
// by the Orquesta binary. It deliberately accepts no paths, environment or
// launch material from argv.
func RunLocalSupervisor() int {
	if err := platformRunLocalSupervisor(); err != nil {
		return supervisorInternalFailureExit
	}
	return 0
}

func newSupervisorInstance() (string, error) {
	var material [32]byte
	if _, err := io.ReadFull(rand.Reader, material[:]); err != nil {
		return "", err
	}
	return "supervisor:" + hex.EncodeToString(material[:]), nil
}

func validateSupervisorEnvelope(envelope supervisorEnvelope) error {
	if envelope.SchemaVersion != supervisorEnvelopeSchema ||
		!validSupervisorToken(envelope.SupervisorInstance, "supervisor:") ||
		envelope.ExecutionRef == "" || envelope.RequestHash == "" || envelope.SpecHash == "" ||
		envelope.ArtifactMediaType == "" ||
		envelope.RuntimeScope == "" || envelope.Command == "" || !filepath.IsAbs(envelope.Command) ||
		!validCompletionPublicKey(envelope.CompletionPublicKey) ||
		len(envelope.CompletionPrivateKey) != ed25519.PrivateKeySize ||
		!bytes.Equal(
			ed25519.PrivateKey(envelope.CompletionPrivateKey).Public().(ed25519.PublicKey),
			decodeCompletionPublicKey(envelope.CompletionPublicKey),
		) ||
		envelope.WorkingDirectory == "" || !filepath.IsAbs(envelope.WorkingDirectory) ||
		envelope.RunDirectory == "" || !filepath.IsAbs(envelope.RunDirectory) ||
		envelope.ResultPath == "" || !filepath.IsAbs(envelope.ResultPath) ||
		filepath.Dir(envelope.ResultPath) != envelope.RunDirectory ||
		envelope.TimeoutNanos <= 0 || envelope.PipeDrainDelayNanos < 0 ||
		envelope.MaxDiagnosticBytes <= 0 || envelope.MaxOutputBytes <= 0 {
		return errors.New("invalid supervisor envelope")
	}
	for _, value := range append(append([]string(nil), envelope.Arguments...), envelope.Environment...) {
		if strings.ContainsRune(value, 0) {
			return errors.New("invalid supervisor envelope value")
		}
	}
	return nil
}

func newCompletionSigningKey() (string, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, err
	}
	encoded := "ed25519:" + base64.RawStdEncoding.EncodeToString(publicKey)
	return encoded, privateKey, nil
}

func validCompletionPublicKey(value string) bool {
	return len(decodeCompletionPublicKey(value)) == ed25519.PublicKeySize
}

func decodeCompletionPublicKey(value string) []byte {
	if !strings.HasPrefix(value, "ed25519:") {
		return nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "ed25519:"))
	if err != nil || base64.RawStdEncoding.EncodeToString(payload) != strings.TrimPrefix(value, "ed25519:") {
		return nil
	}
	return payload
}

func completionProofSigningPayload(proof completionProof) ([]byte, error) {
	proof.Signature = ""
	return json.Marshal(proof)
}

func signCompletionProof(proof *completionProof, privateKey []byte) error {
	if proof == nil || len(privateKey) != ed25519.PrivateKeySize {
		return errors.New("invalid completion signing key")
	}
	payload, err := completionProofSigningPayload(*proof)
	if err != nil {
		return err
	}
	defer clearBytes(payload)
	signature := ed25519.Sign(ed25519.PrivateKey(privateKey), payload)
	proof.Signature = base64.RawStdEncoding.EncodeToString(signature)
	clearBytes(signature)
	return nil
}

func verifyCompletionProofSignature(record processRecord, proof completionProof) bool {
	publicKey := decodeCompletionPublicKey(record.CompletionPublicKey)
	signature, err := base64.RawStdEncoding.DecodeString(proof.Signature)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}
	if base64.RawStdEncoding.EncodeToString(signature) != proof.Signature {
		clearBytes(signature)
		return false
	}
	defer clearBytes(signature)
	payload, err := completionProofSigningPayload(proof)
	if err != nil {
		return false
	}
	defer clearBytes(payload)
	return ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature)
}

func validSupervisorToken(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
}

func decodeSupervisorEnvelope(reader io.Reader) (supervisorEnvelope, error) {
	limited := &io.LimitedReader{R: reader, N: maxSupervisorEnvelopeBytes + 1}
	payload, err := io.ReadAll(limited)
	if err != nil || int64(len(payload)) > maxSupervisorEnvelopeBytes {
		clearBytes(payload)
		return supervisorEnvelope{}, errors.New("invalid supervisor envelope")
	}
	defer clearBytes(payload)
	var envelope supervisorEnvelope
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return supervisorEnvelope{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return supervisorEnvelope{}, err
	}
	if err := validateSupervisorEnvelope(envelope); err != nil {
		return supervisorEnvelope{}, err
	}
	return envelope, nil
}

func digestBytes(payload []byte) string {
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func digestFile(filePath string, maximum int64) (int64, string, bool, bool, error) {
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return 0, "", false, false, syncSupervisorDirectory(filepath.Dir(filePath))
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Mode().Perm()&0o077 != 0 {
		return 0, "", false, false, errors.New("invalid supervisor result")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", false, false, err
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return 0, "", false, false, err
	}
	if info.Size() > maximum {
		return info.Size(), "", true, true, syncSupervisorDirectory(filepath.Dir(filePath))
	}
	digest := sha256.New()
	size, err := io.Copy(digest, file)
	if err != nil || size != info.Size() {
		return 0, "", false, false, errors.New("unstable supervisor result")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(info, after) || after.Size() != size {
		return 0, "", false, false, errors.New("unstable supervisor result")
	}
	if err := syncSupervisorDirectory(filepath.Dir(filePath)); err != nil {
		return 0, "", false, false, err
	}
	return size, "sha256:" + hex.EncodeToString(digest.Sum(nil)), true, false, nil
}

func syncSupervisorDirectory(directoryPath string) error {
	directory, err := os.Open(directoryPath)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func completionProofPath(runDirectory string) string {
	return filepath.Join(runDirectory, completionProofFileName)
}

func publishCompletionProof(runDirectory string, proof completionProof) error {
	payload, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	var suffix [12]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return err
	}
	temporary := filepath.Join(runDirectory, fmt.Sprintf(".%s-%s.tmp", completionProofFileName, hex.EncodeToString(suffix[:])))
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	removeTemporary := true
	defer func() {
		_ = file.Close()
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	if count, err := file.Write(payload); err != nil || count != len(payload) {
		return errors.New("write completion proof")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	finalPath := completionProofPath(runDirectory)
	if err := os.Link(temporary, finalPath); err != nil {
		return err
	}
	if err := os.Remove(temporary); err != nil {
		return err
	}
	removeTemporary = false
	directory, err := os.Open(runDirectory)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func supervisorDurations(envelope supervisorEnvelope) (time.Duration, time.Duration) {
	return time.Duration(envelope.TimeoutNanos), time.Duration(envelope.PipeDrainDelayNanos)
}

func clearSupervisorEnvelope(envelope *supervisorEnvelope) {
	if envelope == nil {
		return
	}
	clearEnvironment(envelope.Environment)
	clearBytes(envelope.CompletionPrivateKey)
	envelope.CompletionPrivateKey = nil
	for index := range envelope.Arguments {
		envelope.Arguments[index] = ""
	}
	envelope.Prompt, envelope.Command, envelope.WorkingDirectory = "", "", ""
}
