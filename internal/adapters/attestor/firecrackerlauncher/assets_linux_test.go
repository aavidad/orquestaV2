//go:build linux

package firecrackerlauncher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetPreflightPinsExpectedRootEquivalentFilesAndManifest(t *testing.T) {
	config := assetConfigForTest(t)
	for _, specification := range []struct {
		name, path, digest string
		mode               uint32
		max                int64
	}{
		{"firecracker", config.FirecrackerCommand, config.FirecrackerSHA256, 0o755, maxExecutableAssetBytes},
		{"jailer", config.JailerCommand, config.JailerSHA256, 0o755, maxExecutableAssetBytes},
		{"kernel", config.KernelImage, config.KernelSHA256, 0o444, maxKernelAssetBytes},
		{"guest", config.GuestImage, config.GuestSHA256, 0o444, maxGuestAssetBytes},
		{"manifest", config.GuestManifest, config.GuestManifestSHA256, 0o444, maxGuestManifestBytes},
	} {
		asset, err := openPinnedAsset(
			specification.path,
			specification.digest,
			uint32(os.Geteuid()),
			specification.mode,
			specification.max,
		)
		if err != nil {
			t.Fatalf("%s: %v", specification.name, err)
		}
		_ = asset.file.Close()
	}
	assets, err := openAssetSet(config, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer assets.Close()
	if !validDigest(assets.digest) ||
		assets.digest != canonicalAssetDigest(config) ||
		assets.minimumGuestMemoryMiB != 470 {
		t.Fatalf("asset digest=%q minimum=%d", assets.digest, assets.minimumGuestMemoryMiB)
	}
}

func TestGuestManifestAcceptsCurrentBuilderSchema(t *testing.T) {
	imageDigest := strings.Repeat("a", 64)
	raw := []byte(fmt.Sprintf(
		`{"schema_version":"orquesta_test_attestor_guest.v0","platform":"linux/amd64","source_commit":"%s","runner_sha256":"sha256:%s","busybox_sha256":"sha256:%s","busybox_version":"v1.36.1","toolchain_tree_sha256":"sha256:%s","toolchain_version":"go1.25.0","image_sha256":"sha256:%s","unpacked_bytes":%d,"minimum_guest_memory_mib":%d,"memory_contract":{"scratch_fixed_reserve_bytes":%d,"scratch_cache_reserve_bytes":%d,"tmpfs_percent":%d,"kernel_runtime_headroom_percent":%d,"formula":"%s"},"build":{"cgo_enabled":false,"trimpath":true,"buildvcs":false,"runner_double_build":true,"source":"exact_commit_private_export","archive":"newc","owner":"0:0","mtime_epoch":0,"gzip_name_time":false,"toolchain_directories":"0555","toolchain_executables":"0555","toolchain_data":"0444","toolchain_symlinks":"relative_internal","toolchain_nobody_go_test":true}}`+"\n",
		strings.Repeat("b", 40),
		strings.Repeat("c", 64),
		strings.Repeat("d", 64),
		strings.Repeat("e", 64),
		imageDigest,
		uint64(32<<20),
		uint32(470),
		guestScratchFixedReserveBytes,
		guestScratchCacheReserveBytes,
		guestScratchTmpfsPercent,
		guestKernelRuntimeHeadroomPercent,
		guestMemoryFormula,
	))
	path := filepath.Join(t.TempDir(), "manifest.json")
	writeStrictAssetForTest(t, path, raw, 0o444)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if minimum, err := validateGuestManifest(
		&pinnedAsset{file: file, size: int64(len(raw)), digest: digestBytes(raw)},
		imageDigest,
	); err != nil || minimum != 470 {
		t.Fatal(err)
	}
}

func TestCanonicalAssetDigestMatchesInstallerGolden(t *testing.T) {
	config := Config{
		FirecrackerSHA256:   strings.Repeat("1", 64),
		JailerSHA256:        strings.Repeat("2", 64),
		KernelSHA256:        strings.Repeat("3", 64),
		GuestSHA256:         strings.Repeat("4", 64),
		GuestManifestSHA256: strings.Repeat("5", 64),
	}
	const want = "1b248da4d891d9e08e1febb62e93ff1c317917035c1bb1382da2712c064a549d"
	if got := canonicalAssetDigest(config); got != want {
		t.Fatalf("canonical asset digest=%q want=%q", got, want)
	}
}

func TestAssetPreflightRejectsTamperModesLinksDigestsAndManifest(t *testing.T) {
	tests := map[string]func(*testing.T, *Config){
		"digest": func(_ *testing.T, config *Config) {
			config.KernelSHA256 = strings.Repeat("f", 64)
		},
		"mode": func(t *testing.T, config *Config) {
			if err := os.Chmod(config.GuestImage, 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"hardlink": func(t *testing.T, config *Config) {
			if err := os.Link(
				config.FirecrackerCommand,
				filepath.Join(t.TempDir(), "second-link"),
			); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, config *Config) {
			link := filepath.Join(t.TempDir(), "guest-link")
			if err := os.Symlink(config.GuestImage, link); err != nil {
				t.Fatal(err)
			}
			config.GuestImage = link
		},
		"manifest_image_mismatch": func(t *testing.T, config *Config) {
			document := validGuestManifestForTest(strings.Repeat("e", 64))
			if err := os.Chmod(config.GuestManifest, 0o600); err != nil {
				t.Fatal(err)
			}
			writeStrictAssetForTest(t, config.GuestManifest, document, 0o444)
			config.GuestManifestSHA256 = digestBytes(document)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := assetConfigForTest(t)
			mutate(t, &config)
			if _, err := openAssetSet(config, uint32(os.Geteuid())); ErrorCode(err) != CodeAssetsUnsafe {
				t.Fatalf("unsafe asset accepted: %v", err)
			}
		})
	}
}

func TestGuestManifestRequiresExactReproducibleSourceFields(t *testing.T) {
	imageDigest := strings.Repeat("e", 64)
	tests := map[string]func(*guestManifestDocument){
		"runner_not_double_built": func(document *guestManifestDocument) {
			document.Build.RunnerDoubleBuild = false
		},
		"source_not_exact_private_export": func(document *guestManifestDocument) {
			document.Build.Source = "working_tree"
		},
		"unpacked_size_missing": func(document *guestManifestDocument) {
			document.UnpackedBytes = 0
		},
		"unpacked_size_overflow": func(document *guestManifestDocument) {
			document.UnpackedBytes = ^uint64(0)
		},
		"minimum_memory_false": func(document *guestManifestDocument) {
			document.MinimumGuestMemoryMiB++
		},
		"memory_formula_changed": func(document *guestManifestDocument) {
			document.MemoryContract.Formula = "other"
		},
		"toolchain_nobody_not_tested": func(document *guestManifestDocument) {
			document.Build.ToolchainNobodyGoTest = false
		},
		"busybox_version_missing": func(document *guestManifestDocument) {
			document.BusyboxVersion = ""
		},
		"busybox_version_unsafe": func(document *guestManifestDocument) {
			document.BusyboxVersion = "v1.36.1\nforged"
		},
		"toolchain_version_missing": func(document *guestManifestDocument) {
			document.ToolchainVersion = ""
		},
		"toolchain_version_unsafe": func(document *guestManifestDocument) {
			document.ToolchainVersion = "go1.25.0 linux/amd64"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			var document guestManifestDocument
			if err := json.Unmarshal(validGuestManifestForTest(imageDigest), &document); err != nil {
				t.Fatal(err)
			}
			mutate(&document)
			raw, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "manifest.json")
			writeStrictAssetForTest(t, path, append(raw, '\n'), 0o444)
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if _, err := validateGuestManifest(
				&pinnedAsset{file: file, size: int64(len(raw) + 1)},
				imageDigest,
			); ErrorCode(err) != CodeAssetsUnsafe {
				t.Fatalf("unsafe manifest accepted: %v", err)
			}
		})
	}
}

func TestLauncherConfigRequiresEveryExpectedHashManifestAndEmptyNetNSPath(t *testing.T) {
	tests := map[string]func(*Config){
		"firecracker_hash": func(config *Config) { config.FirecrackerSHA256 = "" },
		"jailer_hash":      func(config *Config) { config.JailerSHA256 = "" },
		"kernel_hash":      func(config *Config) { config.KernelSHA256 = "" },
		"guest_hash":       func(config *Config) { config.GuestSHA256 = "" },
		"manifest_path":    func(config *Config) { config.GuestManifest = "" },
		"manifest_hash":    func(config *Config) { config.GuestManifestSHA256 = "" },
		"netns_path":       func(config *Config) { config.NetNSPath = "" },
		"cleanup_entries":  func(config *Config) { config.MaxCleanupEntries = 0 },
		"cleanup_depth":    func(config *Config) { config.MaxCleanupDepth = 0 },
		"cleanup_entries_hard": func(config *Config) {
			config.MaxCleanupEntries = maxCleanupEntriesHard + 1
		},
		"cleanup_depth_hard": func(config *Config) {
			config.MaxCleanupDepth = maxCleanupDepthHard + 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := validConfigForTest(t.TempDir())
			mutate(&config)
			if ErrorCode(ValidateConfig(config)) != CodeConfigInvalid {
				t.Fatal("incomplete physical config accepted")
			}
		})
	}
}

func assetConfigForTest(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := validConfigForTest(filepath.Join(root, "runtime"))
	config.FirecrackerCommand = filepath.Join(root, "firecracker")
	config.JailerCommand = filepath.Join(root, "jailer")
	config.KernelImage = filepath.Join(root, "vmlinux")
	config.GuestImage = filepath.Join(root, "guest.cpio.gz")
	config.GuestManifest = filepath.Join(root, "guest.manifest.json")
	firecracker := []byte("firecracker-static-v1.16.1")
	jailer := []byte("jailer-static-v1.16.1")
	kernel := []byte("kernel")
	guest := []byte("guest")
	manifest := validGuestManifestForTest(digestBytes(guest))
	writeStrictAssetForTest(t, config.FirecrackerCommand, firecracker, 0o755)
	writeStrictAssetForTest(t, config.JailerCommand, jailer, 0o755)
	writeStrictAssetForTest(t, config.KernelImage, kernel, 0o444)
	writeStrictAssetForTest(t, config.GuestImage, guest, 0o444)
	writeStrictAssetForTest(t, config.GuestManifest, manifest, 0o444)
	config.FirecrackerSHA256 = digestBytes(firecracker)
	config.JailerSHA256 = digestBytes(jailer)
	config.KernelSHA256 = digestBytes(kernel)
	config.GuestSHA256 = digestBytes(guest)
	config.GuestManifestSHA256 = digestBytes(manifest)
	return config
}

func validGuestManifestForTest(imageDigest string) []byte {
	var document guestManifestDocument
	document.SchemaVersion = "orquesta_test_attestor_guest.v0"
	document.Platform = "linux/amd64"
	document.SourceCommit = strings.Repeat("a", 40)
	document.RunnerSHA256 = "sha256:" + strings.Repeat("1", 64)
	document.BusyboxSHA256 = "sha256:" + strings.Repeat("2", 64)
	document.BusyboxVersion = "v1.36.1"
	document.ToolchainTreeSHA256 = "sha256:" + strings.Repeat("3", 64)
	document.ToolchainVersion = "go1.25.0"
	document.ImageSHA256 = "sha256:" + imageDigest
	document.UnpackedBytes = 32 << 20
	document.MinimumGuestMemoryMiB = 470
	document.MemoryContract.ScratchFixedReserveBytes = guestScratchFixedReserveBytes
	document.MemoryContract.ScratchCacheReserveBytes = guestScratchCacheReserveBytes
	document.MemoryContract.TmpfsPercent = guestScratchTmpfsPercent
	document.MemoryContract.KernelRuntimeHeadroomPercent = guestKernelRuntimeHeadroomPercent
	document.MemoryContract.Formula = guestMemoryFormula
	document.Build.Trimpath = true
	document.Build.RunnerDoubleBuild = true
	document.Build.Source = "exact_commit_private_export"
	document.Build.Archive = "newc"
	document.Build.Owner = "0:0"
	document.Build.ToolchainDirectories = "0555"
	document.Build.ToolchainExecutables = "0555"
	document.Build.ToolchainData = "0444"
	document.Build.ToolchainSymlinks = "relative_internal"
	document.Build.ToolchainNobodyGoTest = true
	content, _ := json.Marshal(document)
	return append(content, '\n')
}

func writeStrictAssetForTest(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
