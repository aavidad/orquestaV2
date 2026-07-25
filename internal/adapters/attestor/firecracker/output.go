package firecracker

import (
	"crypto/sha256"
	"io"
)

type Verdict byte

const (
	VerdictPassed Verdict = 1
	VerdictFailed Verdict = 2
)

type RequiredTestOutcome struct {
	RequiredTestRef string
	ExitCode        uint8
	OutputDigest    string
}

// Output carries either typed required-test outcomes or one stable machine
// error. SubjectDigest is mandatory in both variants.
type Output struct {
	SubjectDigest string
	RunNonce      string
	Verdict       Verdict
	Outcomes      []RequiredTestOutcome
	ErrorCode     string
}

// OutputDriveSize returns the smallest sector-aligned drive for output.
// Callers may preallocate a larger sector-aligned drive up to
// MaxOutputDriveBytes; WriteOutputDrive zero-fills all remaining bytes.
func OutputDriveSize(output Output) (uint64, error) {
	payloadBytes, _, err := validateOutput(output)
	if err != nil {
		return 0, err
	}
	frameBytes, ok := checkedAdd(headerSize+trailerSize, payloadBytes)
	if !ok {
		return 0, protocolError(CodeLimit, nil)
	}
	driveBytes, ok := sectorBytes(frameBytes)
	if !ok || driveBytes > MaxOutputDriveBytes {
		return 0, protocolError(CodeLimit, nil)
	}
	return driveBytes, nil
}

// WriteOutputDrive first zeroes the preallocated drive, writes payload and
// trailer, and syncs them. Only then does it publish the valid header at offset
// zero and sync again. A zero/partial/stale drive cannot decode as complete.
func WriteOutputDrive(destination OutputDrive, driveBytes uint64, output Output) error {
	minimum, err := OutputDriveSize(output)
	if err != nil {
		return err
	}
	if err := validateDriveBytes(driveBytes, MaxOutputDriveBytes); err != nil {
		return err
	}
	if driveBytes < minimum {
		return protocolError(CodeTruncated, nil)
	}
	payloadBytes, kind, _ := validateOutput(output)
	recordCount := uint32(len(output.Outcomes))
	rawHeader, frameBytes, err := buildHeader(outputMagic, kind, payloadBytes, 0, 0, recordCount)
	if err != nil {
		return err
	}
	if err := writeZeroesAt(destination, driveBytes); err != nil {
		return err
	}
	frameDigest := sha256.New()
	if _, err := frameDigest.Write(rawHeader[:]); err != nil {
		return protocolError(CodeIO, err)
	}
	payloadOffset := &offsetWriter{destination: destination, offset: headerSize}
	payloadWriter := io.MultiWriter(payloadOffset, frameDigest)
	if err := writeOutputPayload(payloadWriter, kind, output); err != nil {
		return protocolError(CodeIO, err)
	}
	trailer := buildTrailer(kind, frameBytes, payloadBytes, frameDigest.Sum(nil))
	if err := writeAtAll(destination, trailer[:], headerSize+int64(payloadBytes)); err != nil {
		return protocolError(CodeIO, err)
	}
	if err := destination.Sync(); err != nil {
		return protocolError(CodeIO, err)
	}
	if err := writeAtAll(destination, rawHeader[:], 0); err != nil {
		return protocolError(CodeIO, err)
	}
	if err := destination.Sync(); err != nil {
		return protocolError(CodeIO, err)
	}
	return nil
}

func ReadOutputDrive(source io.Reader, driveBytes uint64) (Output, error) {
	var output Output
	if err := validateDriveBytes(driveBytes, MaxOutputDriveBytes); err != nil {
		return output, err
	}
	header, err := readHeader(source, driveBytes, outputMagic, kindOutputResults, kindOutputError)
	if err != nil {
		return output, err
	}
	if header.sectionBytes != 0 || header.snapshotBytes != 0 ||
		header.payloadBytes > MaxOutputDriveBytes-headerSize-trailerSize {
		return output, protocolError(CodeOutputInvalid, nil)
	}
	if header.kind == kindOutputResults && (header.recordCount == 0 || header.recordCount > MaxRequiredTests) ||
		header.kind == kindOutputError && header.recordCount != 0 {
		return output, protocolError(CodeOutputInvalid, nil)
	}
	frameDigest := sha256.New()
	if _, err := frameDigest.Write(header.raw[:]); err != nil {
		return output, protocolError(CodeIO, err)
	}
	payload := &io.LimitedReader{R: io.TeeReader(source, frameDigest), N: int64(header.payloadBytes)}
	output, err = readOutputPayload(payload, header)
	if err != nil {
		return Output{}, err
	}
	if payload.N != 0 {
		return Output{}, protocolError(CodeOutputInvalid, nil)
	}
	if err := verifyFrameSuffix(source, driveBytes, header, frameDigest); err != nil {
		return Output{}, err
	}
	return output, nil
}

func validateOutput(output Output) (uint64, byte, error) {
	subjectDigest, subjectOK := digestBytes(output.SubjectDigest)
	runNonce, nonceOK := digestBytes(output.RunNonce)
	if !subjectOK || !nonceOK {
		return 0, 0, protocolError(CodeOutputInvalid, nil)
	}
	payloadBytes := uint64(len(subjectDigest) + len(runNonce))
	if output.ErrorCode != "" {
		if output.Verdict != 0 || len(output.Outcomes) != 0 || !validErrorCode(output.ErrorCode) {
			return 0, 0, protocolError(CodeOutputInvalid, nil)
		}
		payloadBytes += uint64(4 + len(output.ErrorCode))
		return payloadBytes, kindOutputError, nil
	}
	if output.Verdict != VerdictPassed && output.Verdict != VerdictFailed ||
		len(output.Outcomes) == 0 || len(output.Outcomes) > MaxRequiredTests {
		return 0, 0, protocolError(CodeOutputInvalid, nil)
	}
	payloadBytes += 8
	seen, hasFailure := make(map[string]struct{}, len(output.Outcomes)), false
	for _, outcome := range output.Outcomes {
		if !validReference(outcome.RequiredTestRef, MaxReferenceBytes) {
			return 0, 0, protocolError(CodeOutputInvalid, nil)
		}
		if _, duplicate := seen[outcome.RequiredTestRef]; duplicate {
			return 0, 0, protocolError(CodeOutputInvalid, nil)
		}
		seen[outcome.RequiredTestRef] = struct{}{}
		if _, valid := digestBytes(outcome.OutputDigest); !valid {
			return 0, 0, protocolError(CodeOutputInvalid, nil)
		}
		payloadBytes += uint64(4 + len(outcome.RequiredTestRef) + 4 + sha256.Size)
		hasFailure = hasFailure || outcome.ExitCode != 0
	}
	if output.Verdict == VerdictPassed && hasFailure || output.Verdict == VerdictFailed && !hasFailure {
		return 0, 0, protocolError(CodeOutputInvalid, nil)
	}
	return payloadBytes, kindOutputResults, nil
}

func writeOutputPayload(destination io.Writer, kind byte, output Output) error {
	subjectDigest, _ := digestBytes(output.SubjectDigest)
	runNonce, _ := digestBytes(output.RunNonce)
	if err := writeAll(destination, subjectDigest[:]); err != nil {
		return err
	}
	if err := writeAll(destination, runNonce[:]); err != nil {
		return err
	}
	if kind == kindOutputError {
		return writeString(destination, output.ErrorCode)
	}
	if err := writeAll(destination, []byte{byte(output.Verdict), 0, 0, 0}); err != nil {
		return err
	}
	if err := writeUint32(destination, uint32(len(output.Outcomes))); err != nil {
		return err
	}
	for _, outcome := range output.Outcomes {
		if err := writeString(destination, outcome.RequiredTestRef); err != nil {
			return err
		}
		if err := writeAll(destination, []byte{outcome.ExitCode, 0, 0, 0}); err != nil {
			return err
		}
		outputDigest, _ := digestBytes(outcome.OutputDigest)
		if err := writeAll(destination, outputDigest[:]); err != nil {
			return err
		}
	}
	return nil
}

func readOutputPayload(source io.Reader, header frameHeader) (Output, error) {
	var output Output
	var subjectDigest, runNonce [sha256.Size]byte
	if _, err := io.ReadFull(source, subjectDigest[:]); err != nil {
		return output, protocolError(CodeTruncated, err)
	}
	if _, err := io.ReadFull(source, runNonce[:]); err != nil {
		return output, protocolError(CodeTruncated, err)
	}
	output.SubjectDigest = digestString(subjectDigest[:])
	output.RunNonce = digestString(runNonce[:])
	if header.kind == kindOutputError {
		code, err := readString(source, MaxErrorCodeBytes, CodeOutputInvalid)
		if err != nil {
			return Output{}, err
		}
		output.ErrorCode = code
		if _, _, err := validateOutput(output); err != nil {
			return Output{}, err
		}
		return output, nil
	}
	var verdict [4]byte
	if _, err := io.ReadFull(source, verdict[:]); err != nil {
		return Output{}, protocolError(CodeTruncated, err)
	}
	if !allZero(verdict[1:]) {
		return Output{}, protocolError(CodeOutputInvalid, nil)
	}
	output.Verdict = Verdict(verdict[0])
	count, err := readUint32(source)
	if err != nil {
		return Output{}, err
	}
	if count != header.recordCount || count == 0 || count > MaxRequiredTests {
		return Output{}, protocolError(CodeOutputInvalid, nil)
	}
	output.Outcomes = make([]RequiredTestOutcome, 0, count)
	for index := uint32(0); index < count; index++ {
		ref, readErr := readString(source, MaxReferenceBytes, CodeOutputInvalid)
		if readErr != nil {
			return Output{}, readErr
		}
		var exitCode [4]byte
		if _, readErr = io.ReadFull(source, exitCode[:]); readErr != nil {
			return Output{}, protocolError(CodeTruncated, readErr)
		}
		if !allZero(exitCode[1:]) {
			return Output{}, protocolError(CodeOutputInvalid, nil)
		}
		var outputDigest [sha256.Size]byte
		if _, readErr = io.ReadFull(source, outputDigest[:]); readErr != nil {
			return Output{}, protocolError(CodeTruncated, readErr)
		}
		output.Outcomes = append(output.Outcomes, RequiredTestOutcome{
			RequiredTestRef: ref,
			ExitCode:        exitCode[0],
			OutputDigest:    digestString(outputDigest[:]),
		})
	}
	if _, _, err := validateOutput(output); err != nil {
		return Output{}, err
	}
	return output, nil
}
