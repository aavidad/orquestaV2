//go:build linux

package firecrackerattestor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const systemctlPath = "/usr/bin/systemctl"

type LinuxUnit struct{}

func (LinuxUnit) Preflight(ctx context.Context, config Config) (CandidateIdentity, error) {
	if os.Geteuid() != 0 || validateSupervisorConfig(config) != nil {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.preflight_denied")
	}
	candidate := config.Candidate
	if !validCandidateUnitPaths(candidate) {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.candidate_invalid")
	}
	checks := []trustedFileCheck{
		{candidate.UnitPath, candidate.UnitSHA256, false, 64 << 10},
		{candidate.PrimitivesUnitPath, candidate.PrimitivesUnitSHA256, false, 64 << 10},
		{candidate.LauncherPath, candidate.LauncherSHA256, true, 128 << 20},
		{candidate.ConfigPath, candidate.ConfigSHA256, false, 64 << 10},
		{candidate.SupervisorPath, candidate.SupervisorSHA256, true, 128 << 20},
	}
	for _, check := range checks {
		if err := verifyRootFile(check); err != nil {
			return CandidateIdentity{}, err
		}
	}
	if err := verifyRootExecutable(systemctlPath); err != nil {
		return CandidateIdentity{}, err
	}
	initial, err := observeSystemdUnit(ctx, candidate.UnitName)
	if err != nil || initial.Active || initial.MainPID != 0 ||
		!initial.Loaded || initial.NeedDaemonReload ||
		initial.FragmentPath != candidate.UnitPath {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.candidate_already_active")
	}
	document, err := readCandidateConfig(candidate.ConfigPath)
	if err != nil || document.SocketPath != candidate.LauncherSocketPath ||
		document.RuntimeRoot != candidate.RuntimeRoot ||
		document.CgroupRoot != candidate.CgroupRoot ||
		document.ParentCgroup != candidate.ParentCgroup ||
		document.NetNSPath != candidate.NetNSPath ||
		document.MaxMemoryBytes != ExpectedMemoryMaxBytes ||
		document.MaxPIDs != ExpectedPIDsMax ||
		document.MaxCPUQuotaMicros != ExpectedCPUQuotaMicros ||
		document.MaxConcurrentRuns != ExpectedConcurrentRuns ||
		document.AllowedUID != config.ChildUID ||
		document.AllowedGID != config.ChildGID ||
		document.FirecrackerCommand == "" || document.JailerCommand == "" ||
		document.KernelImage == "" || document.GuestImage == "" ||
		document.GuestManifest == "" {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.config_mismatch")
	}
	assetDigest := candidateAssetDigest(document)
	if assetDigest != candidate.AssetDigest {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.asset_mismatch")
	}
	unitContent, err := readTrustedFile(candidate.UnitPath, 64<<10)
	if err != nil ||
		!bytes.Contains(unitContent, []byte("User=0\n")) ||
		!bytes.Contains(unitContent, []byte("NoNewPrivileges=yes\n")) ||
		!bytes.Contains(unitContent, []byte("IPAddressDeny=any\n")) ||
		!bytes.Contains(unitContent, []byte(
			"Requires="+filepath.Base(candidate.PrimitivesUnitPath)+"\n",
		)) ||
		!bytes.Contains(unitContent, []byte(
			"ExecStart="+candidate.LauncherPath+" --config "+candidate.ConfigPath+"\n",
		)) {
		return CandidateIdentity{}, errors.New("firecracker_attestor_e2e.unit_mismatch")
	}
	if err := verifyIsolationRoots(candidate); err != nil {
		return CandidateIdentity{}, err
	}
	return CandidateIdentity{
		UnitSHA256:           candidate.UnitSHA256,
		PrimitivesUnitSHA256: candidate.PrimitivesUnitSHA256,
		LauncherSHA256:       candidate.LauncherSHA256,
		ConfigSHA256:         candidate.ConfigSHA256,
		SupervisorSHA256:     candidate.SupervisorSHA256,
		AssetDigest:          candidate.AssetDigest,
	}, nil
}

func (LinuxUnit) Start(ctx context.Context, unit string) (UnitIdentity, error) {
	if !validUnitNameShape(unit) {
		return UnitIdentity{}, ErrInvalid
	}
	if _, err := runSystemctl(ctx, "start", unit); err != nil {
		return UnitIdentity{}, err
	}
	identity, err := observeSystemdUnit(ctx, unit)
	if err != nil || !identity.Active {
		return UnitIdentity{}, errors.Join(
			errors.New("firecracker_attestor_e2e.unit_start_failed"), err,
		)
	}
	return identity, nil
}

func (LinuxUnit) Observe(ctx context.Context, unit string) (UnitIdentity, error) {
	if !validUnitNameShape(unit) {
		return UnitIdentity{}, ErrInvalid
	}
	return observeSystemdUnit(ctx, unit)
}

func (LinuxUnit) Stop(ctx context.Context, unit string) error {
	if !validUnitNameShape(unit) {
		return ErrInvalid
	}
	if _, err := runSystemctl(ctx, "stop", unit); err != nil {
		return err
	}
	identity, err := observeSystemdUnit(ctx, unit)
	if err != nil {
		return err
	}
	if identity.Active || identity.MainPID != 0 {
		return errors.New("firecracker_attestor_e2e.unit_stop_failed")
	}
	return nil
}

func runSystemctl(ctx context.Context, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, systemctlPath, arguments...)
	command.Env = []string{
		"LANG=C", "LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin",
		"SYSTEMD_COLORS=0", "SYSTEMD_PAGER=cat",
	}
	command.Stdin = nil
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return nil, errors.New("firecracker_attestor_e2e.systemctl_failed")
	}
	if output.Len() > 64<<10 {
		return nil, errors.New("firecracker_attestor_e2e.systemctl_output_oversize")
	}
	return output.Bytes(), nil
}

func observeSystemdUnit(ctx context.Context, unit string) (UnitIdentity, error) {
	output, err := runSystemctl(
		ctx, "show", unit,
		"--property=MainPID", "--property=InvocationID", "--property=ActiveState",
		"--property=FragmentPath", "--property=LoadState",
		"--property=NeedDaemonReload",
	)
	if err != nil {
		return UnitIdentity{}, err
	}
	properties := make(map[string]string, 6)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
		}
		if _, duplicate := properties[key]; duplicate {
			return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
		}
		properties[key] = value
	}
	if len(properties) != 6 {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
	}
	pid, err := strconv.Atoi(properties["MainPID"])
	if err != nil || pid < 0 {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
	}
	active := properties["ActiveState"] == "active"
	if !active && properties["ActiveState"] != "inactive" &&
		properties["ActiveState"] != "failed" && properties["ActiveState"] != "deactivating" {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
	}
	loaded := properties["LoadState"] == "loaded"
	if !loaded {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_not_loaded")
	}
	needReload := properties["NeedDaemonReload"] == "yes"
	if !needReload && properties["NeedDaemonReload"] != "no" {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
	}
	fragment := properties["FragmentPath"]
	if fragment == "" || !filepath.IsAbs(fragment) || filepath.Clean(fragment) != fragment {
		return UnitIdentity{}, errors.New("firecracker_attestor_e2e.unit_show_invalid")
	}
	return UnitIdentity{
		UnitName: unit, MainPID: pid,
		InvocationID: strings.ToLower(properties["InvocationID"]), Active: active,
		FragmentPath: fragment, Loaded: loaded, NeedDaemonReload: needReload,
	}, nil
}

type trustedFileCheck struct {
	path       string
	digest     string
	executable bool
	maxBytes   int64
}

func verifyRootFile(check trustedFileCheck) error {
	file, err := openVerifiedRootFile(check)
	if err != nil {
		return err
	}
	return file.Close()
}

func openVerifiedRootFile(check trustedFileCheck) (*os.File, error) {
	if check.path == "" || !filepath.IsAbs(check.path) ||
		filepath.Clean(check.path) != check.path || check.maxBytes <= 0 {
		return nil, errors.New("firecracker_attestor_e2e.path_invalid")
	}
	if err := VerifyTrustedAncestors(check.path); err != nil {
		return nil, err
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, check.path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.file_unsafe")
	}
	file := os.NewFile(uintptr(fd), "firecracker-attestor-e2e-verified")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Uid != 0 || stat.Gid != 0 || stat.Nlink != 1 ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Size <= 0 || stat.Size > check.maxBytes ||
		stat.Mode&0o022 != 0 ||
		(check.executable && stat.Mode&0o111 == 0) {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.binary_metadata_unsafe")
	}
	digest := sha256.New()
	if copied, err := io.Copy(digest, io.NewSectionReader(file, 0, stat.Size)); err != nil ||
		copied != stat.Size || hex.EncodeToString(digest.Sum(nil)) != check.digest {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.binary_identity_mismatch")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.file_unsafe")
	}
	return file, nil
}

// VerifyTrustedAncestors pins every directory above path to root:root without
// group/world write. The leaf is validated independently by its consumer.
func VerifyTrustedAncestors(path string) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("firecracker_attestor_e2e.path_invalid")
	}
	parent := filepath.Dir(path)
	fd, err := unix.Open(
		"/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return errors.New("firecracker_attestor_e2e.ancestor_unsafe")
	}
	defer func() { _ = unix.Close(fd) }()
	var rootStat unix.Stat_t
	if unix.Fstat(fd, &rootStat) != nil || !trustedAncestorStat(&rootStat) {
		return errors.New("firecracker_attestor_e2e.ancestor_unsafe")
	}
	if parent == "/" {
		return nil
	}
	for _, component := range strings.Split(strings.TrimPrefix(parent, "/"), "/") {
		next, err := unix.Openat2(fd, component, &unix.OpenHow{
			Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
			Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
				unix.RESOLVE_NO_MAGICLINKS,
		})
		if err != nil {
			return errors.New("firecracker_attestor_e2e.ancestor_unsafe")
		}
		var stat unix.Stat_t
		safe := unix.Fstat(next, &stat) == nil && trustedAncestorStat(&stat)
		_ = unix.Close(fd)
		fd = next
		if !safe {
			return errors.New("firecracker_attestor_e2e.ancestor_unsafe")
		}
	}
	return nil
}

func trustedAncestorStat(stat *unix.Stat_t) bool {
	return stat != nil && stat.Mode&unix.S_IFMT == unix.S_IFDIR &&
		stat.Uid == 0 && stat.Gid == 0 && stat.Mode&0o022 == 0
}

func readTrustedFile(path string, maxBytes int64) ([]byte, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		maxBytes <= 0 {
		return nil, errors.New("firecracker_attestor_e2e.path_invalid")
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.file_unsafe")
	}
	file := os.NewFile(uintptr(fd), "firecracker-attestor-e2e-trusted")
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 ||
		info.Size() > maxBytes {
		return nil, errors.New("firecracker_attestor_e2e.file_unsafe")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || int64(len(content)) != info.Size() {
		return nil, errors.New("firecracker_attestor_e2e.file_unsafe")
	}
	return content, nil
}

func verifyRootExecutable(path string) error {
	var stat unix.Stat_t
	if unix.Stat(path, &stat) != nil || stat.Uid != 0 || stat.Gid != 0 ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Mode&0o022 != 0 || stat.Mode&0o111 == 0 {
		return errors.New("firecracker_attestor_e2e.systemctl_unsafe")
	}
	return nil
}

type candidateConfigDocument struct {
	SocketPath          string `json:"socket_path"`
	RuntimeRoot         string `json:"runtime_root"`
	FirecrackerCommand  string `json:"firecracker_command"`
	FirecrackerSHA256   string `json:"firecracker_sha256"`
	JailerCommand       string `json:"jailer_command"`
	JailerSHA256        string `json:"jailer_sha256"`
	KernelImage         string `json:"kernel_image"`
	KernelSHA256        string `json:"kernel_sha256"`
	GuestImage          string `json:"guest_image"`
	GuestSHA256         string `json:"guest_sha256"`
	GuestManifest       string `json:"guest_manifest"`
	GuestManifestSHA256 string `json:"guest_manifest_sha256"`
	NetNSPath           string `json:"netns_path"`
	CgroupRoot          string `json:"cgroup_root"`
	ParentCgroup        string `json:"parent_cgroup"`
	AllowedUID          uint32 `json:"allowed_uid"`
	AllowedGID          uint32 `json:"allowed_gid"`
	MaxMemoryBytes      uint64 `json:"max_memory_bytes"`
	MaxPIDs             uint32 `json:"max_pids"`
	MaxCPUQuotaMicros   uint64 `json:"max_cpu_quota_micros"`
	MaxConcurrentRuns   uint32 `json:"max_concurrent_runs"`
}

func readCandidateConfig(path string) (candidateConfigDocument, error) {
	content, err := readTrustedFile(path, 64<<10)
	if err != nil {
		return candidateConfigDocument{}, err
	}
	var all map[string]json.RawMessage
	if json.Unmarshal(content, &all) != nil {
		return candidateConfigDocument{}, ErrInvalid
	}
	var document candidateConfigDocument
	if json.Unmarshal(content, &document) != nil {
		return candidateConfigDocument{}, ErrInvalid
	}
	required := []string{
		"socket_path", "runtime_root", "firecracker_command", "firecracker_sha256",
		"jailer_command", "jailer_sha256", "kernel_image", "kernel_sha256",
		"guest_image", "guest_sha256", "guest_manifest",
		"guest_manifest_sha256", "netns_path", "cgroup_root", "parent_cgroup",
		"allowed_uid", "allowed_gid", "max_memory_bytes", "max_pids",
		"max_cpu_quota_micros", "max_concurrent_runs",
	}
	for _, name := range required {
		if _, ok := all[name]; !ok {
			return candidateConfigDocument{}, ErrInvalid
		}
	}
	return document, nil
}

func candidateAssetDigest(document candidateConfigDocument) string {
	payload := struct {
		Schema              string `json:"schema"`
		FirecrackerSHA256   string `json:"firecracker_sha256"`
		JailerSHA256        string `json:"jailer_sha256"`
		KernelSHA256        string `json:"kernel_sha256"`
		GuestSHA256         string `json:"guest_sha256"`
		GuestManifestSHA256 string `json:"guest_manifest_sha256"`
	}{
		"orquesta.firecracker-launcher.assets.v1",
		document.FirecrackerSHA256, document.JailerSHA256,
		document.KernelSHA256, document.GuestSHA256,
		document.GuestManifestSHA256,
	}
	content, _ := json.Marshal(payload)
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func validUnitName(unit, digest string) bool {
	return unit == "orquesta-firecracker-attestor-"+digest+".service"
}

func validCandidateUnitPaths(candidate Candidate) bool {
	return validUnitName(candidate.UnitName, candidate.UnitSHA256) &&
		candidate.UnitPath == filepath.Join("/etc/systemd/system", candidate.UnitName) &&
		filepath.Dir(candidate.PrimitivesUnitPath) == "/etc/systemd/system" &&
		filepath.Base(candidate.PrimitivesUnitPath) ==
			"orquesta-firecracker-primitives-"+
				candidate.PrimitivesUnitSHA256+".service" &&
		filepath.Clean(candidate.UnitPath) == candidate.UnitPath
}

func validUnitNameShape(unit string) bool {
	const prefix = "orquesta-firecracker-attestor-"
	const suffix = ".service"
	if !strings.HasPrefix(unit, prefix) || !strings.HasSuffix(unit, suffix) {
		return false
	}
	return validDigest(strings.TrimSuffix(strings.TrimPrefix(unit, prefix), suffix))
}

func verifyIsolationRoots(candidate Candidate) error {
	for _, path := range []string{
		candidate.RuntimeRoot,
		filepath.Join(candidate.CgroupRoot, candidate.ParentCgroup),
	} {
		var stat unix.Stat_t
		if unix.Lstat(path, &stat) != nil ||
			stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
			stat.Uid != 0 || stat.Mode&0o022 != 0 {
			return errors.New("firecracker_attestor_e2e.isolation_root_unsafe")
		}
	}
	var netnsStat, selfNetnsStat unix.Stat_t
	if unix.Stat(candidate.NetNSPath, &netnsStat) != nil ||
		unix.Stat("/proc/self/ns/net", &selfNetnsStat) != nil ||
		netnsStat.Ino == selfNetnsStat.Ino && netnsStat.Dev == selfNetnsStat.Dev {
		return errors.New("firecracker_attestor_e2e.netns_unsafe")
	}
	return nil
}
