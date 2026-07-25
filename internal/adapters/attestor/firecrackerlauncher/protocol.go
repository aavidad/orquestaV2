package firecrackerlauncher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"

	firecrackerdrive "orquesta/internal/testattestorprotocol/rawdrive"
)

const (
	protocolVersion        = uint16(1)
	maxProtocolPacket      = 8 * 1024
	maxProtocolField       = 256
	requestMagic           = "ORQ-FCLAUNCH-1\x00"
	responseMagic          = "ORQ-FCRESPONSE-1\x00"
	defaultCPUPeriod       = uint64(100_000)
	minGuestMemoryMiB      = uint32(128)
	maxFirecrackerVCPUs    = uint32(32)
	minimumMemoryMaxBytes  = uint64(minGuestMemoryMiB+128) << 20
	firecrackerSectorBytes = uint64(firecrackerdrive.SectorSize)
	maxInputDriveBytesHard = int64(firecrackerdrive.MaxInputDriveBytes)
	maxOutputDriveBytes    = firecrackerdrive.MaxOutputDriveBytes
	maxCapturedOutputBytes = firecrackerdrive.MaxCapturedOutputBytes
)

type LaunchRequest struct {
	Nonce                  string
	InputDigest            string
	SubjectDigest          string
	PolicyDigest           string
	Timeout                time.Duration
	GuestMemoryMiB         uint32
	MemoryMaxBytes         uint64
	PIDsMax                uint32
	CPUQuotaMicros         uint64
	CPUPeriodMicros        uint64
	OutputDriveBytes       uint64
	MaxCapturedOutputBytes uint64
}

type LaunchResponse struct {
	Nonce        string
	Code         string
	OutputDigest string
	AssetDigest  string
}

func marshalRequest(request LaunchRequest) ([]byte, error) {
	if !validLaunchRequestShape(request) {
		return nil, launcherError(CodeProtocolInvalid)
	}
	var payload bytes.Buffer
	writeProtocolPrefix(&payload, requestMagic)
	for _, value := range []string{
		request.Nonce, request.InputDigest, request.SubjectDigest, request.PolicyDigest,
	} {
		writeProtocolString(&payload, value)
	}
	for _, value := range []uint64{
		uint64(request.Timeout), uint64(request.GuestMemoryMiB), request.MemoryMaxBytes,
		uint64(request.PIDsMax), request.CPUQuotaMicros, request.CPUPeriodMicros,
		request.OutputDriveBytes, request.MaxCapturedOutputBytes,
	} {
		_ = binary.Write(&payload, binary.BigEndian, value)
	}
	if payload.Len() > maxProtocolPacket {
		return nil, launcherError(CodeProtocolInvalid)
	}
	return payload.Bytes(), nil
}

func unmarshalRequest(payload []byte) (LaunchRequest, error) {
	reader := bytes.NewReader(payload)
	if !readProtocolPrefix(reader, requestMagic) {
		return LaunchRequest{}, launcherError(CodeProtocolInvalid)
	}
	values := make([]string, 4)
	for index := range values {
		value, err := readProtocolString(reader)
		if err != nil {
			return LaunchRequest{}, launcherError(CodeProtocolInvalid)
		}
		values[index] = value
	}
	numbers := make([]uint64, 8)
	for index := range numbers {
		if binary.Read(reader, binary.BigEndian, &numbers[index]) != nil {
			return LaunchRequest{}, launcherError(CodeProtocolInvalid)
		}
	}
	if reader.Len() != 0 || numbers[1] > uint64(^uint32(0)) || numbers[3] > uint64(^uint32(0)) {
		return LaunchRequest{}, launcherError(CodeProtocolInvalid)
	}
	request := LaunchRequest{
		Nonce: values[0], InputDigest: values[1], SubjectDigest: values[2], PolicyDigest: values[3],
		Timeout: time.Duration(numbers[0]), GuestMemoryMiB: uint32(numbers[1]),
		MemoryMaxBytes: numbers[2], PIDsMax: uint32(numbers[3]),
		CPUQuotaMicros: numbers[4], CPUPeriodMicros: numbers[5],
		OutputDriveBytes: numbers[6], MaxCapturedOutputBytes: numbers[7],
	}
	if !validLaunchRequestShape(request) {
		return LaunchRequest{}, launcherError(CodeProtocolInvalid)
	}
	return request, nil
}

func marshalResponse(response LaunchResponse) ([]byte, error) {
	if !validNonce(response.Nonce) || !validResponseCode(response.Code) ||
		response.Code == responseCodeOK && (!validDigest(response.OutputDigest) || !validDigest(response.AssetDigest)) ||
		response.Code != responseCodeOK && (response.OutputDigest != "" || response.AssetDigest != "") {
		return nil, launcherError(CodeResponseInvalid)
	}
	var payload bytes.Buffer
	writeProtocolPrefix(&payload, responseMagic)
	for _, value := range []string{response.Nonce, response.Code, response.OutputDigest, response.AssetDigest} {
		writeProtocolString(&payload, value)
	}
	return payload.Bytes(), nil
}

func unmarshalResponse(payload []byte) (LaunchResponse, error) {
	reader := bytes.NewReader(payload)
	if !readProtocolPrefix(reader, responseMagic) {
		return LaunchResponse{}, launcherError(CodeResponseInvalid)
	}
	values := make([]string, 4)
	for index := range values {
		value, err := readProtocolString(reader)
		if err != nil {
			return LaunchResponse{}, launcherError(CodeResponseInvalid)
		}
		values[index] = value
	}
	if reader.Len() != 0 {
		return LaunchResponse{}, launcherError(CodeResponseInvalid)
	}
	response := LaunchResponse{Nonce: values[0], Code: values[1], OutputDigest: values[2], AssetDigest: values[3]}
	canonical, err := marshalResponse(response)
	if err != nil || !bytes.Equal(canonical, payload) {
		return LaunchResponse{}, launcherError(CodeResponseInvalid)
	}
	return response, nil
}

func writeProtocolPrefix(writer io.Writer, magic string) {
	_, _ = io.WriteString(writer, magic)
	_ = binary.Write(writer, binary.BigEndian, protocolVersion)
}

func readProtocolPrefix(reader *bytes.Reader, magic string) bool {
	prefix := make([]byte, len(magic))
	var version uint16
	return func() bool {
		_, err := io.ReadFull(reader, prefix)
		return err == nil && string(prefix) == magic &&
			binary.Read(reader, binary.BigEndian, &version) == nil && version == protocolVersion
	}()
}

func writeProtocolString(writer io.Writer, value string) {
	_ = binary.Write(writer, binary.BigEndian, uint32(len(value)))
	_, _ = io.WriteString(writer, value)
}

func readProtocolString(reader *bytes.Reader) (string, error) {
	var size uint32
	if binary.Read(reader, binary.BigEndian, &size) != nil || size > maxProtocolField || uint64(size) > uint64(reader.Len()) {
		return "", errors.New("invalid field")
	}
	value := make([]byte, size)
	if _, err := io.ReadFull(reader, value); err != nil || strings.ContainsRune(string(value), 0) {
		return "", errors.New("invalid field")
	}
	return string(value), nil
}

func validLaunchRequestShape(request LaunchRequest) bool {
	return validNonce(request.Nonce) && validDigest(request.InputDigest) &&
		validDigest(request.SubjectDigest) && validDigest(request.PolicyDigest) &&
		request.Timeout > 0 && request.GuestMemoryMiB >= minGuestMemoryMiB &&
		request.MemoryMaxBytes > uint64(request.GuestMemoryMiB)<<20 &&
		request.PIDsMax > 0 && request.CPUQuotaMicros > 0 &&
		request.CPUPeriodMicros == defaultCPUPeriod &&
		validOutputDriveBytes(request.OutputDriveBytes) &&
		request.MaxCapturedOutputBytes > 0 &&
		request.MaxCapturedOutputBytes <= maxCapturedOutputBytes
}

func validNonce(value string) bool {
	return len(value) == sha256.Size*2 && strings.Trim(value, "0123456789abcdef") == ""
}

func validDigest(value string) bool {
	return len(value) == sha256.Size*2 && strings.Trim(value, "0123456789abcdef") == ""
}

func validResponseCode(value string) bool {
	if value == responseCodeOK {
		return true
	}
	if len(value) == 0 || len(value) > 160 || !strings.HasPrefix(value, "test_attestor.") {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func digestBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func validOutputDriveBytes(value uint64) bool {
	return value > 0 && value <= maxOutputDriveBytes && value%firecrackerSectorBytes == 0
}
