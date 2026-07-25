// Package rawdrive defines the canonical raw-drive protocol shared by the host
// and isolated test-attestor guest. It contains framing only: it does not
// launch a VMM, inspect host configuration, or depend on application DTOs.
package rawdrive

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	SectorSize = 512

	MaxSnapshotBytes       uint64 = 16 << 30
	MaxInputMetadataBytes  uint64 = 1 << 20
	MaxInputDriveBytes     uint64 = MaxSnapshotBytes + MaxInputMetadataBytes + 2*SectorSize
	MaxOutputDriveBytes    uint64 = 4 << 20
	MaxCapturedOutputBytes uint64 = 1 << 30

	MaxRequiredTests       = 32
	MaxArgumentsPerTest    = 128
	MaxArgumentBytes       = 4 << 10
	MaxArgumentsBytes      = 64 << 10
	MaxReferenceBytes      = 256
	MaxWorkingDirectoryLen = 1 << 10
	MaxErrorCodeBytes      = 256

	protocolVersion uint16 = 1
	hashSHA256      byte   = 1
	headerSize             = 128
	trailerSize            = 64

	kindInput         byte = 1
	kindOutputResults byte = 2
	kindOutputError   byte = 3

	headerChecksumOffset = 64
)

var (
	inputMagic    = [8]byte{'O', 'R', 'Q', 'I', 'N', '0', '0', '1'}
	outputMagic   = [8]byte{'O', 'R', 'Q', 'O', 'U', 'T', '0', '1'}
	trailerMagic  = [8]byte{'O', 'R', 'Q', 'E', 'N', 'D', '0', '1'}
	snapshotMagic = []byte("ORQ-SNAPSHOT-2\x00")
)

const (
	CodeInvalid          = "test_attestor.raw_drive.invalid"
	CodeLimit            = "test_attestor.raw_drive.limit_exceeded"
	CodeTruncated        = "test_attestor.raw_drive.truncated"
	CodeTampered         = "test_attestor.raw_drive.tampered"
	CodeTrailingData     = "test_attestor.raw_drive.trailing_data"
	CodeAlignment        = "test_attestor.raw_drive.alignment_invalid"
	CodeIO               = "test_attestor.raw_drive.io"
	CodeMetadataInvalid  = "test_attestor.raw_drive.metadata_invalid"
	CodeSnapshotInvalid  = "test_attestor.raw_drive.snapshot_invalid"
	CodeOutputInvalid    = "test_attestor.raw_drive.output_invalid"
	CodeOutputIncomplete = "test_attestor.raw_drive.output_incomplete"
)

// Error exposes a stable machine code while retaining an optional I/O cause.
type Error struct {
	Code  string
	Cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func (err *Error) CauseCode() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var protocolErr *Error
	if errors.As(err, &protocolErr) {
		return protocolErr.Code
	}
	return ""
}

type frameHeader struct {
	kind          byte
	payloadBytes  uint64
	sectionBytes  uint64
	snapshotBytes uint64
	recordCount   uint32
	frameBytes    uint64
	expectedMagic [8]byte
	raw           [headerSize]byte
}

// OutputDrive is the durability boundary for the guest result drive. The
// implementation must make successful WriteAt calls visible to Sync.
type OutputDrive interface {
	io.WriterAt
	Sync() error
}

func buildHeader(
	magic [8]byte,
	kind byte,
	payloadBytes uint64,
	sectionBytes uint64,
	snapshotBytes uint64,
	recordCount uint32,
) ([headerSize]byte, uint64, error) {
	var raw [headerSize]byte
	frameBytes, ok := checkedAdd(headerSize+trailerSize, payloadBytes)
	if !ok {
		return raw, 0, protocolError(CodeLimit, nil)
	}
	copy(raw[0:8], magic[:])
	binary.BigEndian.PutUint16(raw[8:10], protocolVersion)
	raw[10], raw[11] = kind, hashSHA256
	binary.BigEndian.PutUint32(raw[12:16], headerSize)
	binary.BigEndian.PutUint64(raw[16:24], payloadBytes)
	binary.BigEndian.PutUint64(raw[24:32], sectionBytes)
	binary.BigEndian.PutUint64(raw[32:40], snapshotBytes)
	binary.BigEndian.PutUint32(raw[40:44], recordCount)
	binary.BigEndian.PutUint64(raw[48:56], frameBytes)
	checksum := sha256.Sum256(raw[:])
	copy(raw[headerChecksumOffset:headerChecksumOffset+sha256.Size], checksum[:])
	return raw, frameBytes, nil
}

func readHeader(source io.Reader, driveBytes uint64, magic [8]byte, allowedKinds ...byte) (frameHeader, error) {
	header := frameHeader{expectedMagic: magic}
	if driveBytes < SectorSize {
		return header, protocolError(CodeTruncated, nil)
	}
	if _, err := io.ReadFull(source, header.raw[:]); err != nil {
		return header, protocolError(CodeTruncated, err)
	}
	if bytes.Equal(header.raw[:], make([]byte, headerSize)) && magic == outputMagic {
		return header, protocolError(CodeOutputIncomplete, nil)
	}
	if !bytes.Equal(header.raw[0:8], magic[:]) ||
		binary.BigEndian.Uint16(header.raw[8:10]) != protocolVersion ||
		header.raw[11] != hashSHA256 ||
		binary.BigEndian.Uint32(header.raw[12:16]) != headerSize ||
		binary.BigEndian.Uint32(header.raw[44:48]) != 0 ||
		!allZero(header.raw[56:64]) ||
		!allZero(header.raw[96:128]) {
		return header, protocolError(CodeInvalid, nil)
	}
	header.kind = header.raw[10]
	if !containsKind(allowedKinds, header.kind) {
		return header, protocolError(CodeInvalid, nil)
	}
	wantChecksum := append([]byte(nil), header.raw[headerChecksumOffset:headerChecksumOffset+sha256.Size]...)
	clear(header.raw[headerChecksumOffset : headerChecksumOffset+sha256.Size])
	gotChecksum := sha256.Sum256(header.raw[:])
	copy(header.raw[headerChecksumOffset:headerChecksumOffset+sha256.Size], wantChecksum)
	if !bytes.Equal(wantChecksum, gotChecksum[:]) {
		return header, protocolError(CodeTampered, nil)
	}
	header.payloadBytes = binary.BigEndian.Uint64(header.raw[16:24])
	header.sectionBytes = binary.BigEndian.Uint64(header.raw[24:32])
	header.snapshotBytes = binary.BigEndian.Uint64(header.raw[32:40])
	header.recordCount = binary.BigEndian.Uint32(header.raw[40:44])
	header.frameBytes = binary.BigEndian.Uint64(header.raw[48:56])
	wantFrame, ok := checkedAdd(headerSize+trailerSize, header.payloadBytes)
	if !ok || header.frameBytes != wantFrame {
		return header, protocolError(CodeInvalid, nil)
	}
	padded, ok := sectorBytes(header.frameBytes)
	if !ok {
		return header, protocolError(CodeLimit, nil)
	}
	if padded > driveBytes {
		return header, protocolError(CodeTruncated, nil)
	}
	return header, nil
}

func buildTrailer(kind byte, frameBytes, payloadBytes uint64, digest []byte) [trailerSize]byte {
	var raw [trailerSize]byte
	copy(raw[0:8], trailerMagic[:])
	binary.BigEndian.PutUint16(raw[8:10], protocolVersion)
	raw[10], raw[11] = kind, hashSHA256
	binary.BigEndian.PutUint32(raw[12:16], trailerSize)
	binary.BigEndian.PutUint64(raw[16:24], frameBytes)
	binary.BigEndian.PutUint64(raw[24:32], payloadBytes)
	copy(raw[32:64], digest)
	return raw
}

func verifyTrailer(source io.Reader, header frameHeader, frameDigest hash.Hash) error {
	var raw [trailerSize]byte
	if _, err := io.ReadFull(source, raw[:]); err != nil {
		return protocolError(CodeTruncated, err)
	}
	if !bytes.Equal(raw[0:8], trailerMagic[:]) ||
		binary.BigEndian.Uint16(raw[8:10]) != protocolVersion ||
		raw[10] != header.kind ||
		raw[11] != hashSHA256 ||
		binary.BigEndian.Uint32(raw[12:16]) != trailerSize ||
		binary.BigEndian.Uint64(raw[16:24]) != header.frameBytes ||
		binary.BigEndian.Uint64(raw[24:32]) != header.payloadBytes {
		return protocolError(CodeTampered, nil)
	}
	if !bytes.Equal(raw[32:64], frameDigest.Sum(nil)) {
		return protocolError(CodeTampered, nil)
	}
	return nil
}

func validateDriveBytes(driveBytes, max uint64) error {
	if driveBytes == 0 || driveBytes%SectorSize != 0 {
		return protocolError(CodeAlignment, nil)
	}
	if driveBytes > max {
		return protocolError(CodeLimit, nil)
	}
	return nil
}

func sectorBytes(frameBytes uint64) (uint64, bool) {
	withPadding, ok := checkedAdd(frameBytes, SectorSize-1)
	if !ok {
		return 0, false
	}
	return withPadding / SectorSize * SectorSize, true
}

func checkedAdd(left, right uint64) (uint64, bool) {
	if right > math.MaxUint64-left {
		return 0, false
	}
	return left + right, true
}

func writeFramePrefix(destination io.Writer, rawHeader []byte) (hash.Hash, error) {
	frameDigest := sha256.New()
	if err := writeAll(io.MultiWriter(destination, frameDigest), rawHeader); err != nil {
		return nil, protocolError(CodeIO, err)
	}
	return frameDigest, nil
}

func writeFrameSuffix(
	destination io.Writer,
	driveBytes uint64,
	kind byte,
	frameBytes uint64,
	payloadBytes uint64,
	frameDigest hash.Hash,
) error {
	trailer := buildTrailer(kind, frameBytes, payloadBytes, frameDigest.Sum(nil))
	if err := writeAll(destination, trailer[:]); err != nil {
		return protocolError(CodeIO, err)
	}
	return writeZeroes(destination, driveBytes-frameBytes)
}

func verifyFrameSuffix(source io.Reader, driveBytes uint64, header frameHeader, frameDigest hash.Hash) error {
	if err := verifyTrailer(source, header, frameDigest); err != nil {
		return err
	}
	remaining := driveBytes - header.frameBytes
	var block [32 * 1024]byte
	for remaining > 0 {
		next := uint64(len(block))
		if next > remaining {
			next = remaining
		}
		if _, err := io.ReadFull(source, block[:next]); err != nil {
			return protocolError(CodeTruncated, err)
		}
		if !allZero(block[:next]) {
			return protocolError(CodeTrailingData, nil)
		}
		remaining -= next
	}
	return nil
}

func writeZeroes(destination io.Writer, count uint64) error {
	var zeroes [32 * 1024]byte
	for count > 0 {
		next := uint64(len(zeroes))
		if next > count {
			next = count
		}
		if err := writeAll(destination, zeroes[:next]); err != nil {
			return protocolError(CodeIO, err)
		}
		count -= next
	}
	return nil
}

func writeAll(destination io.Writer, value []byte) error {
	for len(value) > 0 {
		written, err := destination.Write(value)
		if written < 0 || written > len(value) {
			return io.ErrShortWrite
		}
		value = value[written:]
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func writeAtAll(destination io.WriterAt, value []byte, offset int64) error {
	for len(value) > 0 {
		written, err := destination.WriteAt(value, offset)
		if written < 0 || written > len(value) {
			return io.ErrShortWrite
		}
		value = value[written:]
		offset += int64(written)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func writeZeroesAt(destination io.WriterAt, count uint64) error {
	var zeroes [32 * 1024]byte
	var offset int64
	for count > 0 {
		next := uint64(len(zeroes))
		if next > count {
			next = count
		}
		if err := writeAtAll(destination, zeroes[:next], offset); err != nil {
			return protocolError(CodeIO, err)
		}
		offset += int64(next)
		count -= next
	}
	return nil
}

type offsetWriter struct {
	destination io.WriterAt
	offset      int64
}

func (writer *offsetWriter) Write(value []byte) (int, error) {
	if err := writeAtAll(writer.destination, value, writer.offset); err != nil {
		return 0, err
	}
	writer.offset += int64(len(value))
	return len(value), nil
}

func writeUint32(destination io.Writer, value uint32) error {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	return writeAll(destination, encoded[:])
}

func writeString(destination io.Writer, value string) error {
	if uint64(len(value)) > math.MaxUint32 {
		return protocolError(CodeLimit, nil)
	}
	if err := writeUint32(destination, uint32(len(value))); err != nil {
		return err
	}
	return writeAll(destination, []byte(value))
}

func readUint32(source io.Reader) (uint32, error) {
	var encoded [4]byte
	if _, err := io.ReadFull(source, encoded[:]); err != nil {
		return 0, protocolError(CodeTruncated, err)
	}
	return binary.BigEndian.Uint32(encoded[:]), nil
}

func readString(source io.Reader, maximum uint32, invalidCode string) (string, error) {
	length, err := readUint32(source)
	if err != nil {
		return "", err
	}
	if length > maximum {
		return "", protocolError(CodeLimit, nil)
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(source, value); err != nil {
		return "", protocolError(CodeTruncated, err)
	}
	if !utf8.Valid(value) {
		return "", protocolError(invalidCode, nil)
	}
	return string(value), nil
}

func digestBytes(value string) ([sha256.Size]byte, bool) {
	var result [sha256.Size]byte
	if len(value) != hex.EncodedLen(len(result)) || strings.Trim(value, "0123456789abcdef") != "" {
		return result, false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return result, false
	}
	copy(result[:], decoded)
	return result, true
}

func digestString(value []byte) string { return hex.EncodeToString(value) }

func validReference(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && utf8.ValidString(value) &&
		strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func validErrorCode(value string) bool {
	if value == "" || len(value) > MaxErrorCodeBytes {
		return false
	}
	separator := false
	for index, char := range []byte(value) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			separator = false
			continue
		}
		if (char == '.' || char == '_' || char == '-') && index != 0 && !separator {
			separator = true
			continue
		}
		return false
	}
	return !separator
}

func containsKind(kinds []byte, want byte) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

func allZero(value []byte) bool {
	for _, item := range value {
		if item != 0 {
			return false
		}
	}
	return true
}

func protocolError(code string, cause error) error { return &Error{Code: code, Cause: cause} }
