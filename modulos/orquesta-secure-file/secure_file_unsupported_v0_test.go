//go:build !linux

package orquestasecurefile

import (
	"errors"
	"testing"
)

func TestSecureFileV0FailsClosedOnUnsupportedPlatformV0(t *testing.T) {
	if _, err := OpenDirectoryV0("/", DirectoryOptionsV0{}); !errors.Is(err, ErrUnsupportedPlatformV0) {
		t.Fatalf("unsupported platform did not fail closed: %v", err)
	}
	if _, err := ReadFileAtV0(nil, "file", FileOptionsV0{}); !errors.Is(err, ErrUnsupportedPlatformV0) {
		t.Fatalf("unsupported read did not fail closed: %v", err)
	}
	if _, _, err := CreateFileIfAbsentAtV0(nil, "file", []byte("data"), FileOptionsV0{}); !errors.Is(err, ErrUnsupportedPlatformV0) {
		t.Fatalf("unsupported create did not fail closed: %v", err)
	}
}
