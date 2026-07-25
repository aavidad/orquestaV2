package firecracker

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestInputDriveRoundTripCanonicalAndPreallocated(t *testing.T) {
	input := validInput()
	snapshot := validSnapshot("tree payload")
	minimum, err := InputDriveSize(input, uint64(len(snapshot)))
	if err != nil {
		t.Fatal(err)
	}
	driveBytes := minimum + 2*SectorSize
	first := encodeInput(t, input, snapshot, driveBytes)
	second := encodeInput(t, input, snapshot, driveBytes)
	if !bytes.Equal(first, second) {
		t.Fatal("same input produced different drive bytes")
	}
	var restoredSnapshot bytes.Buffer
	restored, err := ReadInputDrive(bytes.NewReader(first), uint64(len(first)), &restoredSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, input) || !bytes.Equal(restoredSnapshot.Bytes(), snapshot) {
		t.Fatalf("roundtrip mismatch input=%+v snapshot=%x", restored, restoredSnapshot.Bytes())
	}
	frameBytes := binary.BigEndian.Uint64(first[48:56])
	if !allZero(first[frameBytes:]) {
		t.Fatal("input drive padding is not zero")
	}
}

func TestInputDriveRejectsInvalidMetadataAndSnapshotLength(t *testing.T) {
	valid := validInput()
	tests := []struct {
		name  string
		input Input
		code  string
	}{
		{"subject digest", withInput(valid, func(value *Input) { value.SubjectDigest = strings.Repeat("A", 64) }), CodeMetadataInvalid},
		{"run nonce missing", withInput(valid, func(value *Input) { value.RunNonce = "" }), CodeMetadataInvalid},
		{"run nonce uppercase", withInput(valid, func(value *Input) { value.RunNonce = strings.Repeat("A", 64) }), CodeMetadataInvalid},
		{"policy ref", withInput(valid, func(value *Input) { value.PolicyRef = " policy:x" }), CodeMetadataInvalid},
		{"output zero", withInput(valid, func(value *Input) { value.MaxOutputBytes = 0 }), CodeMetadataInvalid},
		{"output limit", withInput(valid, func(value *Input) { value.MaxOutputBytes = MaxCapturedOutputBytes + 1 }), CodeMetadataInvalid},
		{"no tests", withInput(valid, func(value *Input) { value.RequiredTests = nil }), CodeMetadataInvalid},
		{"duplicate", withInput(valid, func(value *Input) { value.RequiredTests = append(value.RequiredTests, value.RequiredTests[0]) }), CodeMetadataInvalid},
		{"unsafe cwd", withInput(valid, func(value *Input) { value.RequiredTests[0].WorkingDirectory = "../host" }), CodeMetadataInvalid},
		{"nul argument", withInput(valid, func(value *Input) { value.RequiredTests[0].Arguments = []string{"x\x00y"} }), CodeMetadataInvalid},
		{"argument limit", withInput(valid, func(value *Input) {
			value.RequiredTests[0].Arguments = []string{strings.Repeat("x", MaxArgumentBytes+1)}
		}), CodeMetadataInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := InputDriveSize(test.input, uint64(len(validSnapshot("")))); ErrorCode(err) != test.code {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}
	if _, err := InputDriveSize(valid, uint64(len(snapshotMagic)-1)); ErrorCode(err) != CodeLimit {
		t.Fatalf("short snapshot code=%q", ErrorCode(err))
	}
	if _, err := InputDriveSize(valid, MaxSnapshotBytes+1); ErrorCode(err) != CodeLimit {
		t.Fatalf("large snapshot code=%q", ErrorCode(err))
	}
}

func TestInputDriveRejectsSourceMismatch(t *testing.T) {
	input := validInput()
	snapshot := validSnapshot("payload")
	driveBytes, err := InputDriveSize(input, uint64(len(snapshot)))
	if err != nil {
		t.Fatal(err)
	}
	var drive bytes.Buffer
	if err := WriteInputDrive(&drive, driveBytes, input, bytes.NewReader(snapshot[:len(snapshot)-1]), uint64(len(snapshot))); ErrorCode(err) != CodeTruncated {
		t.Fatalf("short source code=%q err=%v", ErrorCode(err), err)
	}
	drive.Reset()
	extra := append(append([]byte(nil), snapshot...), 1)
	if err := WriteInputDrive(&drive, driveBytes, input, bytes.NewReader(extra), uint64(len(snapshot))); ErrorCode(err) != CodeSnapshotInvalid {
		t.Fatalf("long source code=%q err=%v", ErrorCode(err), err)
	}
	badMagic := append([]byte(nil), snapshot...)
	badMagic[0] ^= 0xff
	drive.Reset()
	if err := WriteInputDrive(&drive, driveBytes, input, bytes.NewReader(badMagic), uint64(len(badMagic))); ErrorCode(err) != CodeSnapshotInvalid {
		t.Fatalf("magic code=%q err=%v", ErrorCode(err), err)
	}
}

func TestInputDriveRejectsTamperingTruncationOverflowAndTrailingData(t *testing.T) {
	drive := encodeInput(t, validInput(), validSnapshot(strings.Repeat("s", 700)), 0)
	cases := []struct {
		name   string
		mutate func([]byte) ([]byte, uint64)
		code   string
	}{
		{"magic", func(value []byte) ([]byte, uint64) { value[0] ^= 1; return value, uint64(len(value)) }, CodeInvalid},
		{"header checksum", func(value []byte) ([]byte, uint64) { value[64] ^= 1; return value, uint64(len(value)) }, CodeTampered},
		{"payload", func(value []byte) ([]byte, uint64) { value[headerSize+10] ^= 1; return value, uint64(len(value)) }, CodeTampered},
		{"trailer", func(value []byte) ([]byte, uint64) {
			frame := binary.BigEndian.Uint64(value[48:56])
			value[frame-1] ^= 1
			return value, uint64(len(value))
		}, CodeTampered},
		{"trailing nonzero", func(value []byte) ([]byte, uint64) { value[len(value)-1] = 1; return value, uint64(len(value)) }, CodeTrailingData},
		{"reader truncation", func(value []byte) ([]byte, uint64) { return value[:len(value)-1], uint64(len(value)) }, CodeTruncated},
		{"overflow", func(value []byte) ([]byte, uint64) {
			binary.BigEndian.PutUint64(value[16:24], ^uint64(0))
			resealHeader(value)
			return value, uint64(len(value))
		}, CodeInvalid},
		{"metadata limit", func(value []byte) ([]byte, uint64) {
			binary.BigEndian.PutUint64(value[24:32], MaxInputMetadataBytes+1)
			resealHeader(value)
			return value, uint64(len(value))
		}, CodeLimit},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			value, declared := test.mutate(append([]byte(nil), drive...))
			if _, err := ReadInputDrive(bytes.NewReader(value), declared, io.Discard); ErrorCode(err) != test.code {
				t.Fatalf("code=%q want=%q err=%v", ErrorCode(err), test.code, err)
			}
		})
	}
}

func TestInputDriveStreamsSnapshotInBoundedChunks(t *testing.T) {
	input := validInput()
	snapshotBytes := uint64(2 << 20)
	reader := &snapshotPatternReader{remaining: snapshotBytes}
	driveBytes, err := InputDriveSize(input, snapshotBytes)
	if err != nil {
		t.Fatal(err)
	}
	destination := &boundedWriter{maximum: 32 * 1024}
	if err := WriteInputDrive(destination, driveBytes, input, reader, snapshotBytes); err != nil {
		t.Fatal(err)
	}
	if destination.written != driveBytes {
		t.Fatalf("written=%d want=%d", destination.written, driveBytes)
	}
}

func TestOutputDriveRoundTripsResultsFailureAndStableError(t *testing.T) {
	outputs := []Output{
		{
			SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictPassed,
			Outcomes: []RequiredTestOutcome{
				{RequiredTestRef: "required:test-b", ExitCode: 0, OutputDigest: testDigest("b")},
				{RequiredTestRef: "required:test-a", ExitCode: 0, OutputDigest: testDigest("a")},
			},
		},
		{
			SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictFailed,
			Outcomes: []RequiredTestOutcome{{RequiredTestRef: "required:test-a", ExitCode: 2, OutputDigest: testDigest("failed")}},
		},
		{SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), ErrorCode: "test_attestor.firecracker.guest_timeout"},
	}
	for _, output := range outputs {
		driveBytes, err := OutputDriveSize(output)
		if err != nil {
			t.Fatal(err)
		}
		driveBytes += 3 * SectorSize
		drive := encodeOutput(t, output, driveBytes)
		restored, err := ReadOutputDrive(bytes.NewReader(drive), uint64(len(drive)))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(restored, output) {
			t.Fatalf("roundtrip=%+v want=%+v", restored, output)
		}
	}
}

func TestOutputDriveRejectsInvalidVariantsDuplicatesAndVerdictMismatch(t *testing.T) {
	valid := validOutput()
	tests := []Output{
		{},
		{SubjectDigest: testDigest("subject"), Verdict: VerdictPassed, Outcomes: valid.Outcomes},
		withOutput(valid, func(value *Output) { value.RunNonce = strings.Repeat("A", 64) }),
		{SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictPassed, ErrorCode: "guest.failed"},
		{SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), ErrorCode: "Guest Failed"},
		{
			SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictPassed,
			Outcomes: []RequiredTestOutcome{{RequiredTestRef: "test:a", ExitCode: 1, OutputDigest: testDigest("a")}},
		},
		{
			SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictFailed,
			Outcomes: []RequiredTestOutcome{{RequiredTestRef: "test:a", ExitCode: 0, OutputDigest: testDigest("a")}},
		},
		withOutput(valid, func(value *Output) { value.Outcomes = append(value.Outcomes, value.Outcomes[0]) }),
	}
	for index, output := range tests {
		if _, err := OutputDriveSize(output); ErrorCode(err) != CodeOutputInvalid {
			t.Fatalf("case=%d code=%q err=%v", index, ErrorCode(err), err)
		}
	}
}

func TestOutputDriveRejectsUnwrittenTamperingTruncationAndTrailingData(t *testing.T) {
	drive := encodeOutput(t, validOutput(), 3*SectorSize)
	if _, err := ReadOutputDrive(bytes.NewReader(make([]byte, SectorSize)), SectorSize); ErrorCode(err) != CodeOutputIncomplete {
		t.Fatalf("zero drive code=%q err=%v", ErrorCode(err), err)
	}
	cases := []struct {
		name   string
		mutate func([]byte) ([]byte, uint64)
		code   string
	}{
		{"header checksum", func(value []byte) ([]byte, uint64) { value[64] ^= 1; return value, uint64(len(value)) }, CodeTampered},
		{"payload", func(value []byte) ([]byte, uint64) { value[headerSize+1] ^= 1; return value, uint64(len(value)) }, CodeTampered},
		{"trailer", func(value []byte) ([]byte, uint64) {
			frame := binary.BigEndian.Uint64(value[48:56])
			value[frame-1] ^= 1
			return value, uint64(len(value))
		}, CodeTampered},
		{"trailing nonzero", func(value []byte) ([]byte, uint64) { value[len(value)-1] = 1; return value, uint64(len(value)) }, CodeTrailingData},
		{"reader truncation", func(value []byte) ([]byte, uint64) { return value[:len(value)-1], uint64(len(value)) }, CodeTruncated},
		{"count limit before allocation", func(value []byte) ([]byte, uint64) {
			binary.BigEndian.PutUint32(value[40:44], MaxRequiredTests+1)
			resealHeader(value)
			return value, uint64(len(value))
		}, CodeOutputInvalid},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			value, declared := test.mutate(append([]byte(nil), drive...))
			if _, err := ReadOutputDrive(bytes.NewReader(value), declared); ErrorCode(err) != test.code {
				t.Fatalf("code=%q want=%q err=%v", ErrorCode(err), test.code, err)
			}
		})
	}
}

func TestDriveRejectsAlignmentLimitsAndWriterFailure(t *testing.T) {
	input, snapshot := validInput(), validSnapshot("")
	if err := WriteInputDrive(io.Discard, SectorSize+1, input, bytes.NewReader(snapshot), uint64(len(snapshot))); ErrorCode(err) != CodeAlignment {
		t.Fatalf("input alignment code=%q", ErrorCode(err))
	}
	if _, err := ReadOutputDrive(bytes.NewReader(nil), MaxOutputDriveBytes+SectorSize); ErrorCode(err) != CodeLimit {
		t.Fatalf("output limit code=%q", ErrorCode(err))
	}
	driveBytes, err := OutputDriveSize(validOutput())
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("write failed")
	if err := WriteOutputDrive(&memoryOutputDrive{data: make([]byte, driveBytes), writeErr: sentinel}, driveBytes, validOutput()); ErrorCode(err) != CodeIO || !errors.Is(err, sentinel) {
		t.Fatalf("writer code=%q err=%v", ErrorCode(err), err)
	}
}

func TestOutputCommitWritesHeaderOnlyAfterPayloadSync(t *testing.T) {
	output := validOutput()
	driveBytes, err := OutputDriveSize(output)
	if err != nil {
		t.Fatal(err)
	}
	success := &memoryOutputDrive{data: make([]byte, driveBytes)}
	if err := WriteOutputDrive(success, driveBytes, output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(success.headerZeroAtSync, []bool{true, false}) {
		t.Fatalf("header state at sync=%v", success.headerZeroAtSync)
	}
	beforeSync := &memoryOutputDrive{data: bytes.Repeat([]byte{0xff}, int(driveBytes)), failSync: 1}
	if err := WriteOutputDrive(beforeSync, driveBytes, output); ErrorCode(err) != CodeIO {
		t.Fatalf("before sync code=%q err=%v", ErrorCode(err), err)
	}
	assertIncompleteOutput(t, beforeSync.data)
	afterSync := &memoryOutputDrive{
		data: bytes.Repeat([]byte{0xff}, int(driveBytes)), failHeaderAfterSync: true,
	}
	if err := WriteOutputDrive(afterSync, driveBytes, output); ErrorCode(err) != CodeIO {
		t.Fatalf("after sync code=%q err=%v", ErrorCode(err), err)
	}
	if afterSync.syncs != 1 {
		t.Fatalf("syncs=%d want=1", afterSync.syncs)
	}
	assertIncompleteOutput(t, afterSync.data)
}

func FuzzReadInputDrive(f *testing.F) {
	seed := encodeInputForFuzz(f, validInput(), validSnapshot("seed"))
	f.Add(seed)
	f.Add(make([]byte, SectorSize))
	f.Fuzz(func(t *testing.T, value []byte) {
		if len(value) > 1<<20 {
			t.Skip()
		}
		_, _ = ReadInputDrive(bytes.NewReader(value), uint64(len(value)), io.Discard)
	})
}

func FuzzReadOutputDrive(f *testing.F) {
	seed := encodeOutputForFuzz(f, validOutput())
	f.Add(seed)
	f.Add(make([]byte, SectorSize))
	f.Fuzz(func(t *testing.T, value []byte) {
		if len(value) > 1<<20 {
			t.Skip()
		}
		_, _ = ReadOutputDrive(bytes.NewReader(value), uint64(len(value)))
	})
}

func validInput() Input {
	return Input{
		SubjectDigest:  testDigest("subject"),
		RunNonce:       testDigest("nonce"),
		PolicyRef:      "policy:test-attestor:firecracker:v1",
		PolicyDigest:   testDigest("policy"),
		MaxOutputBytes: 8 << 20,
		RequiredTests: []RequiredTest{
			{
				Ref: "required:test-b", ToolRef: "tool:go",
				Arguments: []string{"test", "-run", "^TestB$", "./internal/..."}, WorkingDirectory: ".",
			},
			{
				Ref: "required:test-a", ToolRef: "tool:go",
				Arguments: []string{"test", "./internal/goal"}, WorkingDirectory: "src",
			},
		},
	}
}

func validOutput() Output {
	return Output{
		SubjectDigest: testDigest("subject"), RunNonce: testDigest("nonce"), Verdict: VerdictPassed,
		Outcomes: []RequiredTestOutcome{
			{RequiredTestRef: "required:test-b", OutputDigest: testDigest("b")},
			{RequiredTestRef: "required:test-a", OutputDigest: testDigest("a")},
		},
	}
}

func validSnapshot(payload string) []byte {
	return append(append([]byte(nil), snapshotMagic...), []byte(payload)...)
}

func testDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return digestString(digest[:])
}

func encodeInput(t *testing.T, input Input, snapshot []byte, driveBytes uint64) []byte {
	t.Helper()
	if driveBytes == 0 {
		var err error
		driveBytes, err = InputDriveSize(input, uint64(len(snapshot)))
		if err != nil {
			t.Fatal(err)
		}
	}
	var drive bytes.Buffer
	if err := WriteInputDrive(&drive, driveBytes, input, bytes.NewReader(snapshot), uint64(len(snapshot))); err != nil {
		t.Fatal(err)
	}
	return drive.Bytes()
}

func encodeOutput(t *testing.T, output Output, driveBytes uint64) []byte {
	t.Helper()
	if driveBytes == 0 {
		var err error
		driveBytes, err = OutputDriveSize(output)
		if err != nil {
			t.Fatal(err)
		}
	}
	drive := &memoryOutputDrive{data: make([]byte, driveBytes)}
	if err := WriteOutputDrive(drive, driveBytes, output); err != nil {
		t.Fatal(err)
	}
	return drive.data
}

func encodeInputForFuzz(f *testing.F, input Input, snapshot []byte) []byte {
	f.Helper()
	driveBytes, err := InputDriveSize(input, uint64(len(snapshot)))
	if err != nil {
		f.Fatal(err)
	}
	var drive bytes.Buffer
	if err := WriteInputDrive(&drive, driveBytes, input, bytes.NewReader(snapshot), uint64(len(snapshot))); err != nil {
		f.Fatal(err)
	}
	return drive.Bytes()
}

func encodeOutputForFuzz(f *testing.F, output Output) []byte {
	f.Helper()
	driveBytes, err := OutputDriveSize(output)
	if err != nil {
		f.Fatal(err)
	}
	drive := &memoryOutputDrive{data: make([]byte, driveBytes)}
	if err := WriteOutputDrive(drive, driveBytes, output); err != nil {
		f.Fatal(err)
	}
	return drive.data
}

func withInput(input Input, mutate func(*Input)) Input {
	input.RequiredTests = append([]RequiredTest(nil), input.RequiredTests...)
	for index := range input.RequiredTests {
		input.RequiredTests[index].Arguments = append([]string(nil), input.RequiredTests[index].Arguments...)
	}
	mutate(&input)
	return input
}

func withOutput(output Output, mutate func(*Output)) Output {
	output.Outcomes = append([]RequiredTestOutcome(nil), output.Outcomes...)
	mutate(&output)
	return output
}

func resealHeader(value []byte) {
	clear(value[headerChecksumOffset : headerChecksumOffset+sha256.Size])
	checksum := sha256.Sum256(value[:headerSize])
	copy(value[headerChecksumOffset:headerChecksumOffset+sha256.Size], checksum[:])
}

type snapshotPatternReader struct {
	remaining uint64
	offset    uint64
}

func (reader *snapshotPatternReader) Read(value []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	if uint64(len(value)) > reader.remaining {
		value = value[:reader.remaining]
	}
	for index := range value {
		position := reader.offset + uint64(index)
		if position < uint64(len(snapshotMagic)) {
			value[index] = snapshotMagic[position]
		} else {
			value[index] = byte(position)
		}
	}
	reader.offset += uint64(len(value))
	reader.remaining -= uint64(len(value))
	return len(value), nil
}

type boundedWriter struct {
	maximum int
	written uint64
}

func (writer *boundedWriter) Write(value []byte) (int, error) {
	if len(value) > writer.maximum {
		return 0, errors.New("write exceeded streaming chunk")
	}
	writer.written += uint64(len(value))
	return len(value), nil
}

type memoryOutputDrive struct {
	data                []byte
	syncs               int
	failSync            int
	writeErr            error
	failHeaderAfterSync bool
	headerZeroAtSync    []bool
}

func (drive *memoryOutputDrive) WriteAt(value []byte, offset int64) (int, error) {
	if drive.writeErr != nil {
		return 0, drive.writeErr
	}
	if drive.failHeaderAfterSync && drive.syncs > 0 && offset == 0 {
		return 0, errors.New("header commit failed")
	}
	if offset < 0 || offset > int64(len(drive.data)) || int64(len(value)) > int64(len(drive.data))-offset {
		return 0, io.ErrShortWrite
	}
	copy(drive.data[offset:], value)
	return len(value), nil
}

func (drive *memoryOutputDrive) Sync() error {
	drive.syncs++
	drive.headerZeroAtSync = append(drive.headerZeroAtSync, allZero(drive.data[:headerSize]))
	if drive.failSync == drive.syncs {
		return errors.New("sync failed")
	}
	return nil
}

func assertIncompleteOutput(t *testing.T, drive []byte) {
	t.Helper()
	if !allZero(drive[:headerSize]) {
		t.Fatal("failed output exposed a valid/nonzero header")
	}
	if _, err := ReadOutputDrive(bytes.NewReader(drive), uint64(len(drive))); ErrorCode(err) != CodeOutputIncomplete {
		t.Fatalf("incomplete code=%q err=%v", ErrorCode(err), err)
	}
}
