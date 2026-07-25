package firecrackerlauncher

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestLaunchProtocolRoundTripIsCanonicalAndBounded(t *testing.T) {
	request := validLaunchRequestForTest()
	payload, err := marshalRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	got, err := unmarshalRequest(payload)
	if err != nil || got != request {
		t.Fatalf("request round-trip=%+v err=%v", got, err)
	}
	second, err := marshalRequest(got)
	if err != nil || !bytes.Equal(second, payload) {
		t.Fatalf("request not canonical: err=%v", err)
	}
	response := LaunchResponse{
		Nonce: request.Nonce, Code: responseCodeOK,
		OutputDigest: digestBytes([]byte("output")), AssetDigest: digestBytes([]byte("assets")),
	}
	encoded, err := marshalResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := unmarshalResponse(encoded)
	if err != nil || decoded != response {
		t.Fatalf("response round-trip=%+v err=%v", decoded, err)
	}
}

func TestLaunchProtocolRejectsTamperingAndUnsafeCause(t *testing.T) {
	request := validLaunchRequestForTest()
	payload, err := marshalRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	mutations := [][]byte{
		nil,
		payload[:len(payload)-1],
		append(append([]byte(nil), payload...), 0),
		append([]byte("bad"), payload...),
	}
	for index, mutation := range mutations {
		if _, err := unmarshalRequest(mutation); ErrorCode(err) != CodeProtocolInvalid {
			t.Errorf("mutation %d accepted: %v", index, err)
		}
	}
	for _, code := range []string{"", "other.failure", "test_attestor.bad\nsecret", strings.Repeat("x", 161)} {
		_, err := marshalResponse(LaunchResponse{Nonce: request.Nonce, Code: code})
		if ErrorCode(err) != CodeResponseInvalid {
			t.Errorf("unsafe response code accepted: %q err=%v", code, err)
		}
	}
	request.Nonce = strings.Repeat("a", 32)
	if _, err := marshalRequest(request); ErrorCode(err) != CodeProtocolInvalid {
		t.Fatalf("legacy 16-byte nonce accepted: %v", err)
	}
}

func TestLaunchRequestResourcesFailClosed(t *testing.T) {
	config := validConfigForTest(t.TempDir())
	base := validLaunchRequestForTest()
	mutations := map[string]func(*LaunchRequest){
		"timeout": func(value *LaunchRequest) { value.Timeout = config.MaxTimeout + time.Nanosecond },
		"memory":  func(value *LaunchRequest) { value.MemoryMaxBytes = config.MaxMemoryBytes + 1 },
		"margin":  func(value *LaunchRequest) { value.MemoryMaxBytes = uint64(value.GuestMemoryMiB) << 20 },
		"pids":    func(value *LaunchRequest) { value.PIDsMax = config.MaxPIDs + 1 },
		"cpu":     func(value *LaunchRequest) { value.CPUQuotaMicros = config.MaxCPUQuotaMicros + 1 },
		"drive_max": func(value *LaunchRequest) {
			value.OutputDriveBytes = config.MaxOutputDriveBytes + firecrackerSectorBytes
		},
		"drive_alignment": func(value *LaunchRequest) {
			value.OutputDriveBytes++
		},
		"capture_max": func(value *LaunchRequest) {
			value.MaxCapturedOutputBytes = config.MaxCapturedOutputBytes + 1
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if err := config.validateRequest(request); ErrorCode(err) != CodeResourceUnsafe {
				t.Fatalf("unsafe request accepted: %+v err=%v", request, err)
			}
		})
	}
}

func TestConfigHardLimitsFailClosed(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"input": func(config *Config) {
			config.MaxInputBytes = maxInputDriveBytesHard + int64(firecrackerSectorBytes)
		},
		"input_alignment": func(config *Config) {
			config.MaxInputBytes = int64(firecrackerSectorBytes) + 1
		},
		"capture": func(config *Config) {
			config.MaxCapturedOutputBytes = maxCapturedOutputBytes + 1
		},
		"diagnostic": func(config *Config) {
			config.MaxDiagnosticBytes = maxDiagnosticBytesHard + 1
		},
		"concurrency": func(config *Config) {
			config.MaxConcurrentRuns = maxConcurrentRunsHard + 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := validConfigForTest(t.TempDir())
			mutate(&config)
			if err := ValidateConfig(config); ErrorCode(err) != CodeConfigInvalid {
				t.Fatalf("unsafe config accepted: %+v err=%v", config, err)
			}
		})
	}
	config := validConfigForTest(t.TempDir())
	config.MaxMemoryBytes = minimumMemoryMaxBytes
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("minimum satisfiable memory rejected: %v", err)
	}
	config.MaxMemoryBytes--
	if err := ValidateConfig(config); ErrorCode(err) != CodeConfigInvalid {
		t.Fatalf("unsatisfiable memory accepted: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"peer_uid_sentinel": func(config *Config) { config.AllowedUID = ^uint32(0) },
		"peer_gid_sentinel": func(config *Config) { config.AllowedGID = ^uint32(0) },
		"jail_uid_sentinel": func(config *Config) { config.JailUID = ^uint32(0) },
		"jail_gid_sentinel": func(config *Config) { config.JailGID = ^uint32(0) },
		"shared_uid":        func(config *Config) { config.JailUID = config.AllowedUID },
		"shared_gid":        func(config *Config) { config.JailGID = config.AllowedGID },
	} {
		t.Run(name, func(t *testing.T) {
			config := validConfigForTest(t.TempDir())
			mutate(&config)
			if err := ValidateConfig(config); ErrorCode(err) != CodeConfigInvalid {
				t.Fatalf("unsafe identities accepted: %+v err=%v", config, err)
			}
		})
	}
	request := validLaunchRequestForTest()
	request.MaxCapturedOutputBytes = maxCapturedOutputBytes + 1
	if _, err := marshalRequest(request); ErrorCode(err) != CodeProtocolInvalid {
		t.Fatalf("capture above hard cap accepted: %v", err)
	}
}

func validLaunchRequestForTest() LaunchRequest {
	return LaunchRequest{
		Nonce: strings.Repeat("a", 64), InputDigest: digestBytes([]byte("input")),
		SubjectDigest: digestBytes([]byte("subject")), PolicyDigest: digestBytes([]byte("policy")),
		Timeout: 2 * time.Second, GuestMemoryMiB: 128, MemoryMaxBytes: 512 << 20,
		PIDsMax: 64, CPUQuotaMicros: 100_000, CPUPeriodMicros: defaultCPUPeriod,
		OutputDriveBytes: 256 << 10, MaxCapturedOutputBytes: 64 << 10,
	}
}
