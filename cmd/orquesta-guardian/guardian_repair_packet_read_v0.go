package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	defaultGuardianRepairPacketMaxBytesV0 = int64(256 * 1024)
	guardianRepairPacketInvalidReasonV0   = "guardian_repair_packet_invalid"
)

type guardianRepairPacketInspectionV0 struct {
	Packet    guardianRepairPacketV0
	SHA256    string
	ByteCount string
}

func inspectGuardianRepairPacketForLaunchV0(
	config guardianConfigV0,
	phase string,
	packetPath string,
) (guardianRepairPacketInspectionV0, error) {
	if err := validateGuardianRepairPacketPreOpenPathV0(config, packetPath); err != nil {
		return guardianRepairPacketInspectionV0{}, err
	}
	data, err := readGuardianRepairPacketBytesV0(packetPath)
	if err != nil {
		return guardianRepairPacketInspectionV0{}, err
	}
	var packet guardianRepairPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return guardianRepairPacketInspectionV0{}, guardianRepairPacketInvalidV0("json_invalid")
	}
	if err := validateGuardianRepairPacketEnvelopeV0(config, phase, packetPath, packet); err != nil {
		return guardianRepairPacketInspectionV0{}, err
	}
	sum := sha256.Sum256(data)
	return guardianRepairPacketInspectionV0{
		Packet:    packet,
		SHA256:    hex.EncodeToString(sum[:]),
		ByteCount: strconv.Itoa(len(data)),
	}, nil
}

func validateGuardianRepairPacketPreOpenPathV0(config guardianConfigV0, packetPath string) error {
	actual, err := filepath.Abs(strings.TrimSpace(packetPath))
	if err != nil || actual == "" {
		return guardianRepairPacketInvalidV0("path_invalid")
	}
	repairDir, err := filepath.Abs(filepath.Dir(config.RepairPacketPath))
	if err != nil || repairDir == "" {
		return guardianRepairPacketInvalidV0("path_policy_invalid")
	}
	if filepath.Clean(filepath.Dir(actual)) != filepath.Clean(repairDir) {
		return guardianRepairPacketInvalidV0("path_outside_repair_dir")
	}
	if filepath.Ext(actual) != ".json" {
		return guardianRepairPacketInvalidV0("path_ext_invalid")
	}
	return nil
}

func readGuardianRepairPacketBytesV0(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, guardianRepairPacketInvalidV0(guardianRepairPacketOpenReasonV0(err))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, guardianRepairPacketInvalidV0("symlink")
	}
	if !info.Mode().IsRegular() {
		return nil, guardianRepairPacketInvalidV0("not_regular")
	}
	if guardianRepairPacketHasMultipleLinksV0(info) {
		return nil, guardianRepairPacketInvalidV0("hardlink")
	}
	if info.Size() > defaultGuardianRepairPacketMaxBytesV0 {
		return nil, guardianRepairPacketInvalidV0("too_large")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, guardianRepairPacketInvalidV0(guardianRepairPacketOpenReasonV0(err))
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, guardianRepairPacketInvalidV0("stat_failed")
	}
	if !os.SameFile(info, openedInfo) {
		return nil, guardianRepairPacketInvalidV0("changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, defaultGuardianRepairPacketMaxBytesV0+1))
	if err != nil {
		return nil, guardianRepairPacketInvalidV0("read_failed")
	}
	if int64(len(data)) > defaultGuardianRepairPacketMaxBytesV0 {
		return nil, guardianRepairPacketInvalidV0("too_large")
	}
	return data, nil
}

func validateGuardianRepairPacketEnvelopeV0(
	config guardianConfigV0,
	phase string,
	packetPath string,
	packet guardianRepairPacketV0,
) error {
	if packet.SchemaVersion != guardianRepairPacketSchemaVersionV0 {
		return guardianRepairPacketInvalidV0("schema_invalid")
	}
	if strings.TrimSpace(packet.FailurePhase) != strings.TrimSpace(phase) {
		return guardianRepairPacketInvalidV0("failure_phase_mismatch")
	}
	if strings.TrimSpace(packet.RedactionLevel) != guardianPublicRedactionLevelV0 {
		return guardianRepairPacketInvalidV0("redaction_level_invalid")
	}
	if strings.TrimSpace(packet.Freshness) == "" || strings.TrimSpace(packet.CreatedAt) == "" {
		return guardianRepairPacketInvalidV0("freshness_missing")
	}
	if strings.TrimSpace(packet.Summary) == "" {
		return guardianRepairPacketInvalidV0("summary_missing")
	}
	return validateGuardianRepairPacketAttemptV0(config, packetPath, packet)
}

func validateGuardianRepairPacketAttemptV0(
	config guardianConfigV0,
	packetPath string,
	packet guardianRepairPacketV0,
) error {
	if packet.FailurePacketHash != guardianFailurePacketHashV0(packet) {
		return guardianRepairPacketInvalidV0("failure_packet_hash_mismatch")
	}
	if packet.RepairAttemptRef != guardianRepairAttemptRefV0(config, packet.FailurePacketHash) {
		return guardianRepairPacketInvalidV0("repair_attempt_ref_mismatch")
	}
	if packet.RepairBudget.MaxAgents <= 0 || packet.RepairBudget.MaxAttempts <= 0 {
		return guardianRepairPacketInvalidV0("repair_budget_invalid")
	}
	expected, err := filepath.Abs(guardianRepairPacketPathV0(config, packet.FailurePacketHash))
	if err != nil {
		return guardianRepairPacketInvalidV0("path_hash_invalid")
	}
	actual, err := filepath.Abs(packetPath)
	if err != nil || filepath.Clean(actual) != filepath.Clean(expected) {
		return guardianRepairPacketInvalidV0("path_hash_mismatch")
	}
	return nil
}

func guardianRepairPacketPromptSummaryV0(config guardianConfigV0, value string) string {
	value = guardianRedactOperationalTextV0(config, strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 240 {
		return value[:240]
	}
	return value
}

func guardianRepairPacketInvalidV0(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "invalid"
	}
	return fmt.Errorf("%s:%s", guardianRepairPacketInvalidReasonV0, reason)
}

func guardianRepairPacketOpenReasonV0(err error) string {
	switch {
	case os.IsNotExist(err):
		return "not_found"
	case os.IsPermission(err):
		return "permission_denied"
	default:
		return "read_failed"
	}
}

func guardianRepairPacketHasMultipleLinksV0(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
