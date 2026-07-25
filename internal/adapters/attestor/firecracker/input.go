package firecracker

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math"
	"path"
	"strings"
	"unicode/utf8"
)

// RequiredTest is the adapter-local, argv-safe test declaration transported to
// the guest. Order is significant and preserved.
type RequiredTest struct {
	Ref              string
	ToolRef          string
	Arguments        []string
	WorkingDirectory string
}

// Input is the minimum logical request metadata required by the isolated guest.
// Snapshot bytes travel as a separate opaque ORQ-SNAPSHOT-2 section.
type Input struct {
	SubjectDigest  string
	RunNonce       string
	PolicyRef      string
	PolicyDigest   string
	MaxOutputBytes uint64
	RequiredTests  []RequiredTest
}

// InputDriveSize returns the smallest sector-aligned drive containing input and
// the declared opaque snapshot.
func InputDriveSize(input Input, snapshotBytes uint64) (uint64, error) {
	metadataBytes, err := validateInput(input)
	if err != nil {
		return 0, err
	}
	if snapshotBytes < uint64(len(snapshotMagic)) || snapshotBytes > MaxSnapshotBytes {
		return 0, protocolError(CodeLimit, nil)
	}
	payloadBytes, ok := checkedAdd(metadataBytes, snapshotBytes)
	if !ok {
		return 0, protocolError(CodeLimit, nil)
	}
	frameBytes, ok := checkedAdd(headerSize+trailerSize, payloadBytes)
	if !ok {
		return 0, protocolError(CodeLimit, nil)
	}
	driveBytes, ok := sectorBytes(frameBytes)
	if !ok || driveBytes > MaxInputDriveBytes {
		return 0, protocolError(CodeLimit, nil)
	}
	return driveBytes, nil
}

// WriteInputDrive writes one canonical input frame and zero-fills the complete
// declared drive. Snapshot must expose exactly snapshotBytes and starts with
// ORQ-SNAPSHOT-2. Bytes already emitted are not rolled back on source/I/O error.
func WriteInputDrive(
	destination io.Writer,
	driveBytes uint64,
	input Input,
	snapshot io.Reader,
	snapshotBytes uint64,
) error {
	minimum, err := InputDriveSize(input, snapshotBytes)
	if err != nil {
		return err
	}
	if err := validateDriveBytes(driveBytes, MaxInputDriveBytes); err != nil {
		return err
	}
	if driveBytes < minimum {
		return protocolError(CodeTruncated, nil)
	}
	metadataBytes, _ := validateInput(input)
	payloadBytes, _ := checkedAdd(metadataBytes, snapshotBytes)
	rawHeader, frameBytes, err := buildHeader(
		inputMagic, kindInput, payloadBytes, metadataBytes, snapshotBytes, uint32(len(input.RequiredTests)),
	)
	if err != nil {
		return err
	}
	frameDigest, err := writeFramePrefix(destination, rawHeader[:])
	if err != nil {
		return err
	}
	payloadWriter := io.MultiWriter(destination, frameDigest)
	if err := writeInputMetadata(payloadWriter, input); err != nil {
		return protocolError(CodeIO, err)
	}
	if err := copySnapshot(payloadWriter, snapshot, snapshotBytes); err != nil {
		return err
	}
	var extra [1]byte
	if count, readErr := snapshot.Read(extra[:]); count != 0 || readErr != io.EOF {
		return protocolError(CodeSnapshotInvalid, readErr)
	}
	return writeFrameSuffix(destination, driveBytes, kindInput, frameBytes, payloadBytes, frameDigest)
}

// ReadInputDrive validates the complete raw drive, copies the opaque snapshot
// to snapshotDestination in bounded chunks, and returns adapter-local metadata.
// On error, snapshotDestination may contain unauthenticated partial bytes; the
// caller must not use them unless this function succeeds.
func ReadInputDrive(
	source io.Reader,
	driveBytes uint64,
	snapshotDestination io.Writer,
) (Input, error) {
	var input Input
	if err := validateDriveBytes(driveBytes, MaxInputDriveBytes); err != nil {
		return input, err
	}
	header, err := readHeader(source, driveBytes, inputMagic, kindInput)
	if err != nil {
		return input, err
	}
	if header.sectionBytes > MaxInputMetadataBytes ||
		header.snapshotBytes < uint64(len(snapshotMagic)) ||
		header.snapshotBytes > MaxSnapshotBytes ||
		header.recordCount == 0 ||
		header.recordCount > MaxRequiredTests {
		return input, protocolError(CodeLimit, nil)
	}
	payloadBytes, ok := checkedAdd(header.sectionBytes, header.snapshotBytes)
	if !ok || payloadBytes != header.payloadBytes {
		return input, protocolError(CodeInvalid, nil)
	}
	frameDigest := sha256.New()
	if _, err := frameDigest.Write(header.raw[:]); err != nil {
		return input, protocolError(CodeIO, err)
	}
	payload := io.TeeReader(source, frameDigest)
	metadata := &io.LimitedReader{R: payload, N: int64(header.sectionBytes)}
	input, err = readInputMetadata(metadata, header.recordCount)
	if err != nil {
		return Input{}, err
	}
	if metadata.N != 0 {
		return Input{}, protocolError(CodeMetadataInvalid, nil)
	}
	if err := copySnapshot(snapshotDestination, payload, header.snapshotBytes); err != nil {
		return Input{}, err
	}
	if err := verifyFrameSuffix(source, driveBytes, header, frameDigest); err != nil {
		return Input{}, err
	}
	return input, nil
}

func validateInput(input Input) (uint64, error) {
	subjectDigest, subjectOK := digestBytes(input.SubjectDigest)
	runNonce, nonceOK := digestBytes(input.RunNonce)
	policyDigest, policyOK := digestBytes(input.PolicyDigest)
	if !subjectOK || !nonceOK || !policyOK || !validReference(input.PolicyRef, MaxReferenceBytes) ||
		input.MaxOutputBytes == 0 || input.MaxOutputBytes > MaxCapturedOutputBytes ||
		len(input.RequiredTests) == 0 || len(input.RequiredTests) > MaxRequiredTests {
		return 0, protocolError(CodeMetadataInvalid, nil)
	}
	size := uint64(len(subjectDigest) + len(runNonce) + len(policyDigest) + 8 + 4 + len(input.PolicyRef) + 4)
	seen := make(map[string]struct{}, len(input.RequiredTests))
	for _, test := range input.RequiredTests {
		testBytes, err := validateRequiredTest(test)
		if err != nil {
			return 0, err
		}
		if _, duplicate := seen[test.Ref]; duplicate {
			return 0, protocolError(CodeMetadataInvalid, nil)
		}
		seen[test.Ref] = struct{}{}
		var ok bool
		size, ok = checkedAdd(size, testBytes)
		if !ok || size > MaxInputMetadataBytes {
			return 0, protocolError(CodeLimit, nil)
		}
	}
	return size, nil
}

func validateRequiredTest(test RequiredTest) (uint64, error) {
	if !validReference(test.Ref, MaxReferenceBytes) ||
		!validReference(test.ToolRef, MaxReferenceBytes) ||
		!validWorkingDirectory(test.WorkingDirectory) ||
		len(test.Arguments) > MaxArgumentsPerTest {
		return 0, protocolError(CodeMetadataInvalid, nil)
	}
	size := uint64(4 + len(test.Ref) + 4 + len(test.ToolRef) + 4 + len(test.WorkingDirectory) + 4)
	totalArguments := 0
	for _, argument := range test.Arguments {
		if len(argument) > MaxArgumentBytes || !utf8.ValidString(argument) || strings.ContainsRune(argument, 0) {
			return 0, protocolError(CodeMetadataInvalid, nil)
		}
		totalArguments += len(argument)
		if totalArguments > MaxArgumentsBytes {
			return 0, protocolError(CodeLimit, nil)
		}
		var ok bool
		size, ok = checkedAdd(size, uint64(4+len(argument)))
		if !ok {
			return 0, protocolError(CodeLimit, nil)
		}
	}
	return size, nil
}

func validWorkingDirectory(value string) bool {
	if value == "" || len(value) > MaxWorkingDirectoryLen || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value || strings.ContainsAny(value, "\x00\\") || path.IsAbs(value) {
		return false
	}
	if value == "." {
		return true
	}
	cleaned := path.Clean(value)
	return cleaned == value && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func writeInputMetadata(destination io.Writer, input Input) error {
	subjectDigest, _ := digestBytes(input.SubjectDigest)
	runNonce, _ := digestBytes(input.RunNonce)
	policyDigest, _ := digestBytes(input.PolicyDigest)
	if err := writeAll(destination, subjectDigest[:]); err != nil {
		return err
	}
	if err := writeAll(destination, runNonce[:]); err != nil {
		return err
	}
	if err := writeAll(destination, policyDigest[:]); err != nil {
		return err
	}
	var maxOutput [8]byte
	binary.BigEndian.PutUint64(maxOutput[:], input.MaxOutputBytes)
	if err := writeAll(destination, maxOutput[:]); err != nil {
		return err
	}
	if err := writeString(destination, input.PolicyRef); err != nil {
		return err
	}
	if err := writeUint32(destination, uint32(len(input.RequiredTests))); err != nil {
		return err
	}
	for _, test := range input.RequiredTests {
		for _, value := range []string{test.Ref, test.ToolRef, test.WorkingDirectory} {
			if err := writeString(destination, value); err != nil {
				return err
			}
		}
		if err := writeUint32(destination, uint32(len(test.Arguments))); err != nil {
			return err
		}
		for _, argument := range test.Arguments {
			if err := writeString(destination, argument); err != nil {
				return err
			}
		}
	}
	return nil
}

func readInputMetadata(source io.Reader, recordCount uint32) (Input, error) {
	var input Input
	var subjectDigest, runNonce, policyDigest [sha256.Size]byte
	if _, err := io.ReadFull(source, subjectDigest[:]); err != nil {
		return input, protocolError(CodeTruncated, err)
	}
	if _, err := io.ReadFull(source, runNonce[:]); err != nil {
		return input, protocolError(CodeTruncated, err)
	}
	if _, err := io.ReadFull(source, policyDigest[:]); err != nil {
		return input, protocolError(CodeTruncated, err)
	}
	var maxOutput [8]byte
	if _, err := io.ReadFull(source, maxOutput[:]); err != nil {
		return input, protocolError(CodeTruncated, err)
	}
	policyRef, err := readString(source, MaxReferenceBytes, CodeMetadataInvalid)
	if err != nil {
		return input, err
	}
	count, err := readUint32(source)
	if err != nil {
		return input, err
	}
	if count != recordCount || count == 0 || count > MaxRequiredTests {
		return input, protocolError(CodeMetadataInvalid, nil)
	}
	input = Input{
		SubjectDigest:  digestString(subjectDigest[:]),
		RunNonce:       digestString(runNonce[:]),
		PolicyRef:      policyRef,
		PolicyDigest:   digestString(policyDigest[:]),
		MaxOutputBytes: binary.BigEndian.Uint64(maxOutput[:]),
		RequiredTests:  make([]RequiredTest, 0, count),
	}
	for index := uint32(0); index < count; index++ {
		ref, readErr := readString(source, MaxReferenceBytes, CodeMetadataInvalid)
		if readErr != nil {
			return Input{}, readErr
		}
		toolRef, readErr := readString(source, MaxReferenceBytes, CodeMetadataInvalid)
		if readErr != nil {
			return Input{}, readErr
		}
		workingDirectory, readErr := readString(source, MaxWorkingDirectoryLen, CodeMetadataInvalid)
		if readErr != nil {
			return Input{}, readErr
		}
		argumentCount, readErr := readUint32(source)
		if readErr != nil {
			return Input{}, readErr
		}
		if argumentCount > MaxArgumentsPerTest {
			return Input{}, protocolError(CodeLimit, nil)
		}
		test := RequiredTest{
			Ref: ref, ToolRef: toolRef, WorkingDirectory: workingDirectory,
			Arguments: make([]string, 0, argumentCount),
		}
		for argumentIndex := uint32(0); argumentIndex < argumentCount; argumentIndex++ {
			argument, argumentErr := readString(source, MaxArgumentBytes, CodeMetadataInvalid)
			if argumentErr != nil {
				return Input{}, argumentErr
			}
			test.Arguments = append(test.Arguments, argument)
		}
		input.RequiredTests = append(input.RequiredTests, test)
	}
	if _, err := validateInput(input); err != nil {
		return Input{}, err
	}
	return input, nil
}

func copySnapshot(destination io.Writer, source io.Reader, snapshotBytes uint64) error {
	if snapshotBytes < uint64(len(snapshotMagic)) || snapshotBytes > math.MaxInt64 {
		return protocolError(CodeLimit, nil)
	}
	prefix := make([]byte, len(snapshotMagic))
	if _, err := io.ReadFull(source, prefix); err != nil {
		return protocolError(CodeTruncated, err)
	}
	if !bytes.Equal(prefix, snapshotMagic) {
		return protocolError(CodeSnapshotInvalid, nil)
	}
	if err := writeAll(destination, prefix); err != nil {
		return protocolError(CodeIO, err)
	}
	remaining := snapshotBytes - uint64(len(prefix))
	var block [32 * 1024]byte
	for remaining > 0 {
		next := uint64(len(block))
		if next > remaining {
			next = remaining
		}
		if _, err := io.ReadFull(source, block[:next]); err != nil {
			return protocolError(CodeTruncated, err)
		}
		if err := writeAll(destination, block[:next]); err != nil {
			return protocolError(CodeIO, err)
		}
		remaining -= next
	}
	return nil
}
