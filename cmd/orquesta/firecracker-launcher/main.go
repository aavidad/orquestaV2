//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"orquesta/internal/adapters/attestor/firecrackerlauncher"
)

const maxConfigFileBytes = int64(64 << 10)

type configDocument struct {
	SocketPath             string `json:"socket_path"`
	RuntimeRoot            string `json:"runtime_root"`
	FirecrackerCommand     string `json:"firecracker_command"`
	FirecrackerSHA256      string `json:"firecracker_sha256"`
	JailerCommand          string `json:"jailer_command"`
	JailerSHA256           string `json:"jailer_sha256"`
	KernelImage            string `json:"kernel_image"`
	KernelSHA256           string `json:"kernel_sha256"`
	GuestImage             string `json:"guest_image"`
	GuestSHA256            string `json:"guest_sha256"`
	GuestManifest          string `json:"guest_manifest"`
	GuestManifestSHA256    string `json:"guest_manifest_sha256"`
	NetNSPath              string `json:"netns_path"`
	CgroupRoot             string `json:"cgroup_root"`
	ParentCgroup           string `json:"parent_cgroup"`
	AllowedUID             uint32 `json:"allowed_uid"`
	AllowedGID             uint32 `json:"allowed_gid"`
	JailUID                uint32 `json:"jail_uid"`
	JailGID                uint32 `json:"jail_gid"`
	MaxInputBytes          int64  `json:"max_input_bytes"`
	MaxOutputDriveBytes    uint64 `json:"max_output_drive_bytes"`
	MaxCapturedOutputBytes uint64 `json:"max_captured_output_bytes"`
	MaxMemoryBytes         uint64 `json:"max_memory_bytes"`
	MaxPIDs                uint32 `json:"max_pids"`
	MaxCPUQuotaMicros      uint64 `json:"max_cpu_quota_micros"`
	MaxConcurrentRuns      uint32 `json:"max_concurrent_runs"`
	MaxTimeout             string `json:"max_timeout"`
	CleanupTimeout         string `json:"cleanup_timeout"`
	MaxCleanupEntries      uint32 `json:"max_cleanup_entries"`
	MaxCleanupDepth        uint32 `json:"max_cleanup_depth"`
	MaxDiagnosticBytes     int64  `json:"max_diagnostic_bytes"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) != 2 || arguments[0] != "--config" {
		writeCode(stderr, firecrackerlauncher.CodeConfigInvalid)
		return 2
	}
	config, err := loadConfig(arguments[1], 0)
	if err != nil {
		writeCode(stderr, firecrackerlauncher.CodeConfigInvalid)
		return 2
	}
	server, err := firecrackerlauncher.NewServer(config)
	if err != nil {
		writeCode(stderr, safeErrorCode(err))
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.Serve(ctx); err != nil {
		writeCode(stderr, safeErrorCode(err))
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "code=ok")
	return 0
}

func loadConfig(path string, trustedOwner uint32) (firecrackerlauncher.Config, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	file, err := firecrackerlauncher.OpenTrustedConfigFile(path, trustedOwner, maxConfigFileBytes)
	if err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	content, err := io.ReadAll(io.LimitReader(file, maxConfigFileBytes+1))
	if err != nil || int64(len(content)) != info.Size() {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	if !hasCanonicalConfigKeys(content) {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	var document configDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	maxTimeout, err := time.ParseDuration(document.MaxTimeout)
	if err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	cleanupTimeout, err := time.ParseDuration(document.CleanupTimeout)
	if err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	config := firecrackerlauncher.Config{
		SocketPath: document.SocketPath, RuntimeRoot: document.RuntimeRoot,
		FirecrackerCommand: document.FirecrackerCommand, FirecrackerSHA256: document.FirecrackerSHA256,
		JailerCommand: document.JailerCommand, JailerSHA256: document.JailerSHA256,
		KernelImage: document.KernelImage, KernelSHA256: document.KernelSHA256,
		GuestImage: document.GuestImage, GuestSHA256: document.GuestSHA256,
		GuestManifest: document.GuestManifest, GuestManifestSHA256: document.GuestManifestSHA256,
		NetNSPath:  document.NetNSPath,
		CgroupRoot: document.CgroupRoot, ParentCgroup: document.ParentCgroup,
		AllowedUID: document.AllowedUID, AllowedGID: document.AllowedGID,
		JailUID: document.JailUID, JailGID: document.JailGID,
		MaxInputBytes:          document.MaxInputBytes,
		MaxOutputDriveBytes:    document.MaxOutputDriveBytes,
		MaxCapturedOutputBytes: document.MaxCapturedOutputBytes,
		MaxMemoryBytes:         document.MaxMemoryBytes, MaxPIDs: document.MaxPIDs,
		MaxCPUQuotaMicros: document.MaxCPUQuotaMicros,
		MaxConcurrentRuns: document.MaxConcurrentRuns,
		MaxTimeout:        maxTimeout, CleanupTimeout: cleanupTimeout,
		MaxCleanupEntries:  document.MaxCleanupEntries,
		MaxCleanupDepth:    document.MaxCleanupDepth,
		MaxDiagnosticBytes: document.MaxDiagnosticBytes,
	}
	if err := firecrackerlauncher.ValidateConfig(config); err != nil {
		return firecrackerlauncher.Config{}, firecrackerlauncherError()
	}
	return config, nil
}

func hasCanonicalConfigKeys(content []byte) bool {
	allowed := make(map[string]struct{})
	documentType := reflect.TypeOf(configDocument{})
	for index := 0; index < documentType.NumField(); index++ {
		allowed[documentType.Field(index).Tag.Get("json")] = struct{}{}
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	token, err := decoder.Token()
	open, ok := token.(json.Delim)
	if err != nil || !ok || open != '{' {
		return false
	}
	seen := make(map[string]struct{}, len(allowed))
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		if _, ok := allowed[key]; !ok {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return false
		}
	}
	token, err = decoder.Token()
	close, ok := token.(json.Delim)
	if err != nil || !ok || close != '}' || len(seen) != len(allowed) {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}

func firecrackerlauncherError() error {
	return &firecrackerlauncher.Error{Code: firecrackerlauncher.CodeConfigInvalid}
}

func safeErrorCode(err error) string {
	code := firecrackerlauncher.ErrorCode(err)
	if code == "" {
		return firecrackerlauncher.CodeUnavailable
	}
	return code
}

func writeCode(writer io.Writer, code string) {
	_, _ = fmt.Fprintf(writer, "code=%s\n", code)
}
