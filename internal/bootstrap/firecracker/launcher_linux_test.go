//go:build linux

package firecracker

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/adapters/attestor/firecrackerlauncher"
)

func TestLauncherCLIRequiresOnlyCanonicalConfigArgument(t *testing.T) {
	for _, arguments := range [][]string{
		nil,
		{"--socket", "/run/orquesta/launcher.sock"},
		{"--config"},
		{"-config", "/etc/orquesta/launcher.json"},
		{"--config", "/etc/orquesta/launcher.json", "--extra"},
	} {
		var stdout, stderr bytes.Buffer
		if status := RunFirecrackerLauncher(arguments, &stdout, &stderr); status != 2 ||
			stderr.String() != "code=test_attestor.firecracker_launcher_config_invalid\n" ||
			stdout.Len() != 0 {
			t.Fatalf("arguments=%q status/output=%d %q %q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestLauncherCLIPreservesSafeKVMPreflightSubstageCodes(t *testing.T) {
	codes := []string{
		firecrackerlauncher.CodeKVMOpenUnavailable,
		firecrackerlauncher.CodeKVMMetadataUnsafe,
		firecrackerlauncher.CodeKVMAPIUnavailable,
		firecrackerlauncher.CodeKVMVersionUnsupported,
	}
	for _, code := range codes {
		var output bytes.Buffer
		err := &firecrackerlauncher.Error{Code: code}
		if got := safeErrorCode(err); got != code {
			t.Fatalf("code=%q got=%q", code, got)
		}
		writeCode(&output, safeErrorCode(err))
		if output.String() != "code="+code+"\n" ||
			strings.Contains(output.String(), "/dev/kvm") ||
			strings.Contains(output.String(), "operation not permitted") {
			t.Fatalf("unsafe diagnostic output: %q", output.String())
		}
	}
}

func TestLoadConfigIsStrictAndValidatesOwnershipAndPermissions(t *testing.T) {
	owner := uint32(os.Geteuid())
	t.Run("valid", func(t *testing.T) {
		path := writeConfigForTest(t, validConfigDocumentForTest(), 0o600)
		if _, err := loadConfig(path, owner); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		document := validConfigDocumentForTest()
		content, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		content = []byte(strings.TrimSuffix(string(content), "}") + `,"unknown":true}`)
		path := writeRawConfigForTest(t, content, 0o600)
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("unknown config field accepted")
		}
	})
	t.Run("partial", func(t *testing.T) {
		path := writeRawConfigForTest(t, []byte(`{"socket_path":"/run/orquesta/launcher.sock"}`), 0o600)
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("partial config accepted")
		}
	})
	t.Run("case_alias", func(t *testing.T) {
		content, err := json.Marshal(validConfigDocumentForTest())
		if err != nil {
			t.Fatal(err)
		}
		content = bytes.Replace(content, []byte(`"socket_path"`), []byte(`"SOCKET_PATH"`), 1)
		path := writeRawConfigForTest(t, content, 0o600)
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("case-insensitive alias accepted")
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		content, err := json.Marshal(validConfigDocumentForTest())
		if err != nil {
			t.Fatal(err)
		}
		content = []byte(strings.TrimSuffix(string(content), "}") + `,"socket_path":"/run/other.sock"}`)
		path := writeRawConfigForTest(t, content, 0o600)
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("duplicate config field accepted")
		}
	})
	t.Run("writable_by_group", func(t *testing.T) {
		path := writeConfigForTest(t, validConfigDocumentForTest(), 0o600)
		if err := os.Chmod(path, 0o620); err != nil {
			t.Fatal(err)
		}
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("group-writable config accepted")
		}
	})
	t.Run("wrong_owner", func(t *testing.T) {
		path := writeConfigForTest(t, validConfigDocumentForTest(), 0o600)
		if _, err := loadConfig(path, owner+1); err == nil {
			t.Fatal("config with untrusted owner accepted")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		target := writeConfigForTest(t, validConfigDocumentForTest(), 0o600)
		link := filepath.Join(t.TempDir(), "launcher.json")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := loadConfig(link, owner); err == nil {
			t.Fatal("symlink config accepted")
		}
	})
	t.Run("intermediate_symlink", func(t *testing.T) {
		base := t.TempDir()
		if err := os.Chmod(base, 0o700); err != nil {
			t.Fatal(err)
		}
		realParent := filepath.Join(base, "real")
		if err := os.Mkdir(realParent, 0o700); err != nil {
			t.Fatal(err)
		}
		content, err := json.Marshal(validConfigDocumentForTest())
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(realParent, "launcher.json")
		if err := os.WriteFile(target, content, 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "link")
		if err := os.Symlink(realParent, link); err != nil {
			t.Fatal(err)
		}
		if _, err := loadConfig(filepath.Join(link, "launcher.json"), owner); err == nil {
			t.Fatal("intermediate config symlink accepted")
		}
	})
	t.Run("writable_ancestor", func(t *testing.T) {
		base := t.TempDir()
		if err := os.Chmod(base, 0o700); err != nil {
			t.Fatal(err)
		}
		unsafeParent := filepath.Join(base, "unsafe")
		if err := os.Mkdir(unsafeParent, 0o770); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(unsafeParent, 0o770); err != nil {
			t.Fatal(err)
		}
		content, err := json.Marshal(validConfigDocumentForTest())
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(unsafeParent, "launcher.json")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadConfig(path, owner); err == nil {
			t.Fatal("config below replaceable ancestor accepted")
		}
	})
}

func validConfigDocumentForTest() configDocument {
	return configDocument{
		SocketPath:             "/run/orquesta-firecracker/launcher.sock",
		RuntimeRoot:            "/run/orquesta-firecracker",
		FirecrackerCommand:     "/usr/local/bin/firecracker",
		FirecrackerSHA256:      strings.Repeat("1", 64),
		JailerCommand:          "/usr/local/bin/jailer",
		JailerSHA256:           strings.Repeat("2", 64),
		KernelImage:            "/usr/local/lib/orquesta/vmlinux",
		KernelSHA256:           strings.Repeat("3", 64),
		GuestImage:             "/usr/local/lib/orquesta/guest.cpio",
		GuestSHA256:            strings.Repeat("4", 64),
		GuestManifest:          "/usr/local/lib/orquesta/guest.manifest.json",
		GuestManifestSHA256:    strings.Repeat("5", 64),
		NetNSPath:              "/run/netns/orquesta-firecracker-empty",
		CgroupRoot:             "/sys/fs/cgroup",
		ParentCgroup:           "orquesta-firecracker",
		AllowedUID:             1000,
		AllowedGID:             1000,
		JailUID:                65534,
		JailGID:                65534,
		MaxInputBytes:          1 << 20,
		MaxOutputDriveBytes:    256 << 10,
		MaxCapturedOutputBytes: 16 << 20,
		MaxMemoryBytes:         1 << 30,
		MaxPIDs:                256,
		MaxCPUQuotaMicros:      200_000,
		MaxConcurrentRuns:      2,
		MaxTimeout:             "3s",
		CleanupTimeout:         "1s",
		MaxCleanupEntries:      1024,
		MaxCleanupDepth:        16,
		MaxDiagnosticBytes:     64 << 10,
	}
}

func writeConfigForTest(t *testing.T, document configDocument, mode os.FileMode) string {
	t.Helper()
	content, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return writeRawConfigForTest(t, content, mode)
}

func writeRawConfigForTest(t *testing.T, content []byte, mode os.FileMode) string {
	t.Helper()
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "launcher.json")
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}
