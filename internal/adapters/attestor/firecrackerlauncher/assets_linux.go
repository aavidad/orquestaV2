//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	maxExecutableAssetBytes = int64(128 << 20)
	maxKernelAssetBytes     = int64(2 << 30)
	maxGuestAssetBytes      = int64(8 << 30)
	maxGuestManifestBytes   = int64(64 << 10)

	guestMemoryMiBBytes               = uint64(1 << 20)
	guestScratchFixedReserveBytes     = uint64(64 << 20)
	guestScratchCacheReserveBytes     = uint64(256 << 20)
	guestScratchTmpfsPercent          = uint32(75)
	guestKernelRuntimeHeadroomPercent = uint32(25)
	guestMemoryFormula                = "ceil(ceil((unpacked_bytes+scratch_fixed_reserve_bytes+scratch_cache_reserve_bytes)/MiB)*100/tmpfs_percent)"
)

type pinnedAsset struct {
	file   *os.File
	size   int64
	digest string
}

type assetSet struct {
	firecracker           *pinnedAsset
	jailer                *pinnedAsset
	kernel                *pinnedAsset
	guest                 *pinnedAsset
	manifest              *pinnedAsset
	digest                string
	minimumGuestMemoryMiB uint32
	close                 sync.Once
	closeErr              error
}

type guestManifestDocument struct {
	SchemaVersion         string `json:"schema_version"`
	Platform              string `json:"platform"`
	SourceCommit          string `json:"source_commit"`
	RunnerSHA256          string `json:"runner_sha256"`
	BusyboxSHA256         string `json:"busybox_sha256"`
	ToolchainTreeSHA256   string `json:"toolchain_tree_sha256"`
	ImageSHA256           string `json:"image_sha256"`
	UnpackedBytes         uint64 `json:"unpacked_bytes"`
	MinimumGuestMemoryMiB uint32 `json:"minimum_guest_memory_mib"`
	MemoryContract        struct {
		ScratchFixedReserveBytes     uint64 `json:"scratch_fixed_reserve_bytes"`
		ScratchCacheReserveBytes     uint64 `json:"scratch_cache_reserve_bytes"`
		TmpfsPercent                 uint32 `json:"tmpfs_percent"`
		KernelRuntimeHeadroomPercent uint32 `json:"kernel_runtime_headroom_percent"`
		Formula                      string `json:"formula"`
	} `json:"memory_contract"`
	Build struct {
		CGOEnabled            bool   `json:"cgo_enabled"`
		Trimpath              bool   `json:"trimpath"`
		BuildVCS              bool   `json:"buildvcs"`
		RunnerDoubleBuild     bool   `json:"runner_double_build"`
		Source                string `json:"source"`
		Archive               string `json:"archive"`
		Owner                 string `json:"owner"`
		MtimeEpoch            int64  `json:"mtime_epoch"`
		GzipNameTime          bool   `json:"gzip_name_time"`
		ToolchainDirectories  string `json:"toolchain_directories"`
		ToolchainExecutables  string `json:"toolchain_executables"`
		ToolchainData         string `json:"toolchain_data"`
		ToolchainSymlinks     string `json:"toolchain_symlinks"`
		ToolchainNobodyGoTest bool   `json:"toolchain_nobody_go_test"`
	} `json:"build"`
}

func openAssetSet(config Config, owner uint32) (*assetSet, error) {
	specifications := []struct {
		path, digest string
		mode         uint32
		maxBytes     int64
	}{
		{config.FirecrackerCommand, config.FirecrackerSHA256, 0o755, maxExecutableAssetBytes},
		{config.JailerCommand, config.JailerSHA256, 0o755, maxExecutableAssetBytes},
		{config.KernelImage, config.KernelSHA256, 0o444, maxKernelAssetBytes},
		{config.GuestImage, config.GuestSHA256, 0o444, maxGuestAssetBytes},
		{config.GuestManifest, config.GuestManifestSHA256, 0o444, maxGuestManifestBytes},
	}
	opened := make([]*pinnedAsset, 0, len(specifications))
	fail := func() (*assetSet, error) {
		for _, asset := range opened {
			_ = asset.file.Close()
		}
		return nil, launcherError(CodeAssetsUnsafe)
	}
	for _, specification := range specifications {
		asset, err := openPinnedAsset(
			specification.path,
			specification.digest,
			owner,
			specification.mode,
			specification.maxBytes,
		)
		if err != nil {
			return fail()
		}
		opened = append(opened, asset)
	}
	assets := &assetSet{
		firecracker: opened[0], jailer: opened[1], kernel: opened[2],
		guest: opened[3], manifest: opened[4],
	}
	minimumGuestMemoryMiB, err := validateGuestManifest(
		assets.manifest,
		config.GuestSHA256,
	)
	if err != nil {
		return fail()
	}
	assets.minimumGuestMemoryMiB = minimumGuestMemoryMiB
	assets.digest = canonicalAssetDigest(config)
	return assets, nil
}

func openPinnedAsset(
	path, wantDigest string,
	owner, mode uint32,
	maxBytes int64,
) (*pinnedAsset, error) {
	if maxBytes <= 0 || !validDigest(wantDigest) ||
		validateTrustedAncestors(path, owner) != nil {
		return nil, launcherError(CodeAssetsUnsafe)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeAssetsUnsafe)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-asset")
	var before unix.Stat_t
	if unix.Fstat(fd, &before) != nil ||
		before.Mode&unix.S_IFMT != unix.S_IFREG ||
		before.Uid != owner || before.Gid != owner || before.Nlink != 1 ||
		before.Mode&0o777 != mode || before.Size <= 0 || before.Size > maxBytes {
		_ = file.Close()
		return nil, launcherError(CodeAssetsUnsafe)
	}
	digest, err := digestRegularFile(context.Background(), file, before.Size)
	var after unix.Stat_t
	if err != nil || digest != wantDigest || unix.Fstat(fd, &after) != nil ||
		!sameTrustedAssetMetadata(before, after) {
		_ = file.Close()
		return nil, launcherError(CodeAssetsUnsafe)
	}
	return &pinnedAsset{file: file, size: before.Size, digest: digest}, nil
}

func digestRegularFile(ctx context.Context, file *os.File, size int64) (string, error) {
	digest := sha256.New()
	reader := io.NewSectionReader(file, 0, size)
	buffer := make([]byte, 128*1024)
	for copied := int64(0); copied < size; {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		count, err := reader.Read(buffer)
		if count > 0 {
			_, _ = digest.Write(buffer[:count])
			copied += int64(count)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		if count == 0 {
			return "", io.ErrUnexpectedEOF
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func sameTrustedAssetMetadata(left, right unix.Stat_t) bool {
	return left.Dev == right.Dev && left.Ino == right.Ino &&
		left.Size == right.Size && left.Mode == right.Mode &&
		left.Uid == right.Uid && left.Gid == right.Gid &&
		left.Nlink == right.Nlink && left.Mtim == right.Mtim &&
		left.Ctim == right.Ctim
}

func validateGuestManifest(asset *pinnedAsset, guestDigest string) (uint32, error) {
	if asset == nil || asset.file == nil || asset.size <= 0 ||
		asset.size > maxGuestManifestBytes {
		return 0, launcherError(CodeAssetsUnsafe)
	}
	content, err := io.ReadAll(io.NewSectionReader(asset.file, 0, asset.size))
	if err != nil {
		return 0, launcherError(CodeAssetsUnsafe)
	}
	var document guestManifestDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return 0, launcherError(CodeAssetsUnsafe)
	}
	minimumGuestMemoryMiB, memoryOK := minimumGuestMemoryForManifest(document.UnpackedBytes)
	if document.SchemaVersion != "orquesta_test_attestor_guest.v0" ||
		document.Platform != "linux/amd64" ||
		!validSourceCommit(document.SourceCommit) ||
		!validPrefixedDigest(document.RunnerSHA256) ||
		!validPrefixedDigest(document.BusyboxSHA256) ||
		!validPrefixedDigest(document.ToolchainTreeSHA256) ||
		document.ImageSHA256 != "sha256:"+guestDigest ||
		document.Build.CGOEnabled || !document.Build.Trimpath ||
		document.Build.BuildVCS || !document.Build.RunnerDoubleBuild ||
		document.Build.Source != "exact_commit_private_export" ||
		document.Build.Archive != "newc" ||
		document.Build.Owner != "0:0" || document.Build.MtimeEpoch != 0 ||
		document.Build.GzipNameTime ||
		document.Build.ToolchainDirectories != "0555" ||
		document.Build.ToolchainExecutables != "0555" ||
		document.Build.ToolchainData != "0444" ||
		document.Build.ToolchainSymlinks != "relative_internal" ||
		!document.Build.ToolchainNobodyGoTest ||
		document.MemoryContract.ScratchFixedReserveBytes != guestScratchFixedReserveBytes ||
		document.MemoryContract.ScratchCacheReserveBytes != guestScratchCacheReserveBytes ||
		document.MemoryContract.TmpfsPercent != guestScratchTmpfsPercent ||
		document.MemoryContract.KernelRuntimeHeadroomPercent != guestKernelRuntimeHeadroomPercent ||
		document.MemoryContract.Formula != guestMemoryFormula ||
		!memoryOK || document.MinimumGuestMemoryMiB != minimumGuestMemoryMiB {
		return 0, launcherError(CodeAssetsUnsafe)
	}
	return minimumGuestMemoryMiB, nil
}

func minimumGuestMemoryForManifest(unpackedBytes uint64) (uint32, bool) {
	if unpackedBytes == 0 ||
		unpackedBytes > ^uint64(0)-guestScratchFixedReserveBytes-guestScratchCacheReserveBytes {
		return 0, false
	}
	requiredBytes := unpackedBytes +
		guestScratchFixedReserveBytes +
		guestScratchCacheReserveBytes
	requiredMiB := requiredBytes / guestMemoryMiBBytes
	if requiredBytes%guestMemoryMiBBytes != 0 {
		requiredMiB++
	}
	if requiredMiB > ^uint64(0)/100 {
		return 0, false
	}
	scaled := requiredMiB * 100
	minimumMiB := scaled / uint64(guestScratchTmpfsPercent)
	if scaled%uint64(guestScratchTmpfsPercent) != 0 {
		minimumMiB++
	}
	if minimumMiB == 0 || minimumMiB > uint64(^uint32(0)) {
		return 0, false
	}
	return uint32(minimumMiB), true
}

func validSourceCommit(value string) bool {
	return (len(value) == 40 || len(value) == 64) &&
		len(bytes.Trim([]byte(value), "0123456789abcdef")) == 0
}

func validPrefixedDigest(value string) bool {
	return len(value) == len("sha256:")+sha256.Size*2 &&
		value[:len("sha256:")] == "sha256:" && validDigest(value[len("sha256:"):])
}

func canonicalAssetDigest(config Config) string {
	payload := struct {
		Schema        string `json:"schema"`
		Firecracker   string `json:"firecracker_sha256"`
		Jailer        string `json:"jailer_sha256"`
		Kernel        string `json:"kernel_sha256"`
		Guest         string `json:"guest_sha256"`
		GuestManifest string `json:"guest_manifest_sha256"`
	}{
		Schema:      "orquesta.firecracker-launcher.assets.v1",
		Firecracker: config.FirecrackerSHA256, Jailer: config.JailerSHA256,
		Kernel: config.KernelSHA256, Guest: config.GuestSHA256,
		GuestManifest: config.GuestManifestSHA256,
	}
	canonical, _ := json.Marshal(payload)
	return digestBytes(canonical)
}

func (assets *assetSet) Close() error {
	if assets == nil {
		return nil
	}
	assets.close.Do(func() {
		assets.closeErr = errors.Join(
			closePinnedAsset(assets.firecracker),
			closePinnedAsset(assets.jailer),
			closePinnedAsset(assets.kernel),
			closePinnedAsset(assets.guest),
			closePinnedAsset(assets.manifest),
		)
	})
	return assets.closeErr
}

func closePinnedAsset(asset *pinnedAsset) error {
	if asset == nil || asset.file == nil {
		return nil
	}
	return asset.file.Close()
}
