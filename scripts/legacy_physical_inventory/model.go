// Este fichero define el contrato JSON estable y los resúmenes acotados.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/sys/unix"
	"unicode/utf8"
)

const (
	schemaVersion        = "legacy_physical_inventory/v1"
	manifestDigestDomain = "sha256_canonical_json_manifest_sha256_empty_v1"
	modeMetadata         = "metadata_only"
	modeContent          = "source_content"
)

type pathSegment struct {
	Encoding string `json:"encoding"`
	Value    string `json:"value"`
}
type binaryValue struct {
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}
type record struct {
	Schema              string        `json:"schema"`
	Kind                string        `json:"kind"`
	Root                string        `json:"root"`
	Mode                string        `json:"mode,omitempty"`
	Path                []pathSegment `json:"path,omitempty"`
	FileMode            *uint32       `json:"file_mode,omitempty"`
	ModeText            string        `json:"mode_text,omitempty"`
	Size                *int64        `json:"size,omitempty"`
	UID                 *uint32       `json:"uid,omitempty"`
	GID                 *uint32       `json:"gid,omitempty"`
	LocalIdentitySHA256 string        `json:"local_identity_sha256,omitempty"`
	ContentState        string        `json:"content_state,omitempty"`
	ContentSHA256       string        `json:"content_sha256,omitempty"`
	ContentBytes        int64         `json:"content_bytes,omitempty"`
	ChangedDuringScan   bool          `json:"changed_during_scan,omitempty"`
	SpecialType         string        `json:"special_type,omitempty"`
	RejectedSegment     *binaryValue  `json:"rejected_segment,omitempty"`
	Reason              string        `json:"reason,omitempty"`
	Operation           string        `json:"operation,omitempty"`
	ErrorCode           string        `json:"error_code,omitempty"`
}
type rootSummary struct {
	Alias    string           `json:"alias"`
	Mode     string           `json:"mode"`
	Complete bool             `json:"complete"`
	Counts   map[string]int64 `json:"counts"`
}
type manifest struct {
	Schema               string           `json:"schema"`
	Algorithm            string           `json:"algorithm"`
	Complete             bool             `json:"complete"`
	JSONLSHA256          string           `json:"jsonl_sha256"`
	JSONLBytes           int64            `json:"jsonl_bytes"`
	Entries              int64            `json:"entries"`
	HashedBytes          int64            `json:"hashed_bytes"`
	MaxEntries           int64            `json:"max_entries"`
	MaxDirectoryEntries  int64            `json:"max_directory_entries"`
	MaxDepth             int              `json:"max_depth"`
	MaxPathBytes         int64            `json:"max_path_bytes"`
	MaxOutputBytes       int64            `json:"max_output_bytes"`
	MaxHashBytes         int64            `json:"max_hash_bytes"`
	MaxFileBytes         int64            `json:"max_file_bytes"`
	TimeoutNanoseconds   int64            `json:"timeout_nanoseconds"`
	Roots                []rootSummary    `json:"roots"`
	Denied               []deniedManifest `json:"denied"`
	LocalIdentityScope   string           `json:"local_identity_scope"`
	ManifestSHA256       string           `json:"manifest_sha256"`
	ManifestDigestDomain string           `json:"manifest_digest_domain"`
	JSONLName            string           `json:"jsonl_name"`
}
type deniedManifest struct {
	Root string        `json:"root"`
	Path []pathSegment `json:"path"`
}

func encodeSegment(value string) pathSegment {
	if utf8.ValidString(value) {
		return pathSegment{Encoding: "utf8", Value: value}
	}
	return pathSegment{Encoding: "base64", Value: base64.StdEncoding.EncodeToString([]byte(value))}
}
func encodeBinary(value string) *binaryValue {
	sum := sha256.Sum256([]byte(value))
	return &binaryValue{SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(value))}
}
func encodePath(parts []string) []pathSegment {
	if len(parts) == 0 {
		return nil
	}
	result := make([]pathSegment, len(parts))
	for index, part := range parts {
		result[index] = encodeSegment(part)
	}
	return result
}
func metadataRecord(kind, alias, mode string, parts []string, stat *unix.Stat_t) record {
	fileMode, size, uid, gid := stat.Mode, stat.Size, stat.Uid, stat.Gid
	return record{
		Schema:              schemaVersion,
		Kind:                kind,
		Root:                alias,
		Mode:                mode,
		Path:                encodePath(parts),
		FileMode:            &fileMode,
		ModeText:            modeText(stat.Mode),
		Size:                &size,
		UID:                 &uid,
		GID:                 &gid,
		LocalIdentitySHA256: localIdentity(stat),
	}
}
func modeText(mode uint32) string {
	permissions := fmt.Sprintf("%04o", mode&0o7777)
	switch mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return "directory:" + permissions
	case unix.S_IFREG:
		return "regular:" + permissions
	case unix.S_IFLNK:
		return "symlink:" + permissions
	case unix.S_IFIFO:
		return "fifo:" + permissions
	case unix.S_IFSOCK:
		return "socket:" + permissions
	case unix.S_IFCHR:
		return "character_device:" + permissions
	case unix.S_IFBLK:
		return "block_device:" + permissions
	default:
		return "unknown:" + permissions
	}
}
func localIdentity(stat *unix.Stat_t) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf(
		"legacy_physical_inventory/local/v1\x00%d\x00%d", stat.Dev, stat.Ino,
	)))
	return hex.EncodeToString(sum[:])
}
func stableErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errorsIs(err, errBudget):
		return "budget_exhausted"
	case errorsIs(err, errDirectoryBudget):
		return "directory_budget_exhausted"
	case errorsIs(err, errChangedDuringScan):
		return "changed_during_scan"
	case errorsIs(err, errOutputBudget):
		return "output_budget_exhausted"
	case errorsIs(err, errNoAtime):
		return "noatime_required"
	case errorsIs(err, errMountIDUnavailable):
		return "mount_id_unavailable"
	case errorsIs(err, unix.EACCES):
		return "permission_denied"
	case errorsIs(err, unix.ENOENT):
		return "not_found"
	case errorsIs(err, unix.ELOOP):
		return "symbolic_link_rejected"
	case errorsIs(err, unix.EXDEV):
		return "mount_boundary"
	case errorsIs(err, unix.ENOSYS):
		return "openat2_unavailable"
	case errorsIs(err, unix.ENOTDIR):
		return "not_directory"
	default:
		return "io_error"
	}
}
func publicErrorCode(err error) string {
	switch {
	case isRecoveryError(err):
		return "recovery_failed"
	case errorsIs(err, errUnexpectedArguments):
		return "unexpected_positional_arguments"
	case errorsIs(err, errInvalidInput):
		return "invalid_input"
	case errorsIs(err, errOutputPath):
		return "invalid_output_path"
	case errorsIs(err, errPhysicalOverlap):
		return "physical_path_overlap"
	case errorsIs(err, errNoAtime):
		return "noatime_required"
	case errorsIs(err, errMountIDUnavailable):
		return "mount_id_unavailable"
	case errorsIs(err, errRootAnchor):
		return "root_anchor_failed"
	case errorsIs(err, errOutputDirectory):
		return "private_output_directory_required"
	case errorsIs(err, errOutputLocked):
		return "output_locked"
	case errorsIs(err, errOutputNameExists):
		return "output_name_exists"
	case errorsIs(err, errRecoveryConflict):
		return "recovery_conflict"
	case errorsIs(err, errStageReplaced):
		return "stage_replaced"
	case errorsIs(err, errOutputBudget):
		return "output_budget_exhausted"
	case errorsIs(err, errInvalidManifestSeal):
		return "invalid_manifest_seal"
	case errorsIs(err, errIncomplete):
		return "inventory_incomplete"
	case errorsIs(err, errBudget):
		return "budget_exhausted"
	case errorsIs(err, errPublication):
		return "publication_failed"
	default:
		return "operation_failed"
	}
}
