//go:build linux

package firecrackerattestor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	maxObservedEntries = 4096
	maxControlBytes    = 64 << 10
)

type LinuxMonitor struct {
	config Config
	unit   UnitPort
}

func NewLinuxMonitor(config Config, unit UnitPort) (*LinuxMonitor, error) {
	if validateSupervisorConfig(config) != nil || unit == nil {
		return nil, ErrInvalid
	}
	return &LinuxMonitor{config: config, unit: unit}, nil
}

func (monitor *LinuxMonitor) CapturePhase(
	ctx context.Context,
	baseline UnitIdentity,
	run WorkloadRun,
	expected uint32,
) (PhaseEvidence, error) {
	if monitor == nil || ctx == nil || run == nil ||
		expected == 0 || expected > ExpectedConcurrentRuns {
		return PhaseEvidence{}, ErrInvalid
	}
	evidence := PhaseEvidence{
		RequestedRuns: expected, LimitsExact: true,
		MemorySwapMaxZero: true, NetworkAbsent: true, APIAbsent: true,
		VsockAbsent: true, SerialAbsent: true, UnitIdentityStable: true,
	}
	ticker := time.NewTicker(monitor.config.PollInterval)
	defer ticker.Stop()
	for {
		sample, complete, err := monitor.sample(ctx, baseline)
		if err != nil {
			return PhaseEvidence{}, err
		}
		evidence.Samples++
		if sample.count > expected {
			return PhaseEvidence{}, errors.New("firecracker_attestor_e2e.high_water_exceeded")
		}
		if complete && sample.count >= evidence.HighWaterRuns {
			evidence.HighWaterRuns = sample.count
			evidence.RunIDs = append([]string(nil), sample.ids...)
			evidence.FirecrackerPIDs = append([]int(nil), sample.pids...)
			evidence.LimitsExact = evidence.LimitsExact && sample.limitsExact
			evidence.MemorySwapMaxZero = evidence.MemorySwapMaxZero && sample.swapZero
			evidence.NetworkAbsent = evidence.NetworkAbsent && sample.networkAbsent
			evidence.APIAbsent = evidence.APIAbsent && sample.apiAbsent
			evidence.VsockAbsent = evidence.VsockAbsent && sample.vsockAbsent
			evidence.SerialAbsent = evidence.SerialAbsent && sample.serialAbsent
			evidence.UnitIdentityStable = evidence.UnitIdentityStable && sample.unitStable
		}
		if evidence.HighWaterRuns == expected {
			select {
			case <-run.Done():
				return evidence, nil
			default:
			}
		}
		select {
		case <-ctx.Done():
			return PhaseEvidence{}, errors.New("firecracker_attestor_e2e.phase_timeout")
		case <-run.Done():
			if evidence.HighWaterRuns != expected {
				return PhaseEvidence{}, errors.New("firecracker_attestor_e2e.high_water_not_observed")
			}
			return evidence, nil
		case <-ticker.C:
		}
	}
}

func (monitor *LinuxMonitor) WaitQuiescent(
	ctx context.Context,
	baseline UnitIdentity,
) (CleanupEvidence, error) {
	if monitor == nil || ctx == nil {
		return CleanupEvidence{}, ErrInvalid
	}
	ticker := time.NewTicker(monitor.config.PollInterval)
	defer ticker.Stop()
	var cleanSince time.Time
	var stable uint64
	for {
		sample, _, err := monitor.sample(ctx, baseline)
		if err != nil {
			return CleanupEvidence{}, err
		}
		residualProcesses, err := monitor.residualProcesses()
		if err != nil {
			return CleanupEvidence{}, err
		}
		cleanup := CleanupEvidence{
			StableSamples:     stable,
			ResidualRuns:      sample.count,
			ResidualCgroups:   sample.cgroups,
			ResidualProcesses: residualProcesses,
			UnitStopped:       !baseline.Active,
		}
		if !baseline.Active {
			cleanup.SocketAbsent, err = monitor.launcherSocketAbsent()
			if err != nil {
				return CleanupEvidence{}, err
			}
		}
		if sample.count == 0 && sample.cgroups == 0 && residualProcesses == 0 &&
			sample.unitStable && (baseline.Active || cleanup.SocketAbsent) {
			stable++
			cleanup.StableSamples = stable
			if cleanSince.IsZero() {
				cleanSince = time.Now()
			}
			if stable >= 2 && time.Since(cleanSince) >= monitor.config.StableFor {
				return cleanup, nil
			}
		} else {
			stable = 0
			cleanSince = time.Time{}
		}
		select {
		case <-ctx.Done():
			return CleanupEvidence{}, errors.New("firecracker_attestor_e2e.cleanup_timeout")
		case <-ticker.C:
		}
	}
}

type linuxSample struct {
	count         uint32
	cgroups       uint32
	ids           []string
	pids          []int
	limitsExact   bool
	swapZero      bool
	networkAbsent bool
	apiAbsent     bool
	vsockAbsent   bool
	serialAbsent  bool
	unitStable    bool
}

func (monitor *LinuxMonitor) sample(
	ctx context.Context,
	baseline UnitIdentity,
) (linuxSample, bool, error) {
	current, err := monitor.unit.Observe(ctx, baseline.UnitName)
	if err != nil {
		return linuxSample{}, false, err
	}
	stable := unitIdentityMatches(baseline, current)
	runs, err := openDirectoryNoLinks(monitor.config.Candidate.RuntimeRoot, "runs")
	if err != nil {
		return linuxSample{}, false, err
	}
	defer runs.Close()
	entries, err := readBoundedDirectory(runs, maxObservedEntries)
	if err != nil {
		return linuxSample{}, false, err
	}
	sample := linuxSample{
		count: uint32(len(entries)), limitsExact: true, swapZero: true,
		networkAbsent: true, apiAbsent: true, vsockAbsent: true,
		serialAbsent: true, unitStable: stable,
	}
	cgroupParent, err := openDirectoryNoLinks(
		monitor.config.Candidate.CgroupRoot,
		monitor.config.Candidate.ParentCgroup,
	)
	if err != nil {
		return linuxSample{}, false, err
	}
	defer cgroupParent.Close()
	cgroupEntries, err := readBoundedDirectory(cgroupParent, maxObservedEntries)
	if err != nil {
		return linuxSample{}, false, err
	}
	cgroupDirectories := make([]os.DirEntry, 0, len(cgroupEntries))
	for _, entry := range cgroupEntries {
		if entry.IsDir() {
			cgroupDirectories = append(cgroupDirectories, entry)
		}
	}
	sample.cgroups = uint32(len(cgroupDirectories))
	if len(entries) == 0 {
		return sample, len(cgroupDirectories) == 0, nil
	}
	if len(cgroupDirectories) != len(entries) {
		return sample, false, nil
	}
	cgroupSet := make(map[string]struct{}, len(cgroupDirectories))
	for _, entry := range cgroupDirectories {
		if !validRunID(entry.Name()) {
			return linuxSample{}, false, errors.New("firecracker_attestor_e2e.cgroup_entry_unsafe")
		}
		cgroupSet[entry.Name()] = struct{}{}
	}
	for _, entry := range entries {
		id := entry.Name()
		if !entry.IsDir() || !validRunID(id) {
			return linuxSample{}, false, errors.New("firecracker_attestor_e2e.run_entry_unsafe")
		}
		if _, ok := cgroupSet[id]; !ok {
			return sample, false, nil
		}
		run, err := openDirectoryAt(runs, id)
		if err != nil {
			return sample, false, nil
		}
		observation, complete, observeErr := monitor.observeRun(run, cgroupParent, id)
		run.Close()
		if observeErr != nil {
			return linuxSample{}, false, observeErr
		}
		if !complete {
			return sample, false, nil
		}
		sample.ids = append(sample.ids, id)
		sample.pids = append(sample.pids, observation.pid)
		sample.limitsExact = sample.limitsExact && observation.limitsExact
		sample.swapZero = sample.swapZero && observation.swapZero
		sample.networkAbsent = sample.networkAbsent && observation.networkAbsent
		sample.apiAbsent = sample.apiAbsent && observation.apiAbsent
		sample.vsockAbsent = sample.vsockAbsent && observation.vsockAbsent
		sample.serialAbsent = sample.serialAbsent && observation.serialAbsent
	}
	sort.Strings(sample.ids)
	sort.Ints(sample.pids)
	return sample, true, nil
}

func unitIdentityMatches(baseline, current UnitIdentity) bool {
	if baseline.UnitName == "" || current.UnitName != baseline.UnitName {
		return false
	}
	if baseline.Active {
		return current.Active && baseline.MainPID > 1 &&
			current.MainPID == baseline.MainPID &&
			validInvocationID(baseline.InvocationID) &&
			current.InvocationID == baseline.InvocationID &&
			current.Loaded && !current.NeedDaemonReload &&
			current.FragmentPath == baseline.FragmentPath
	}
	return !current.Active && current.MainPID == 0 &&
		current.Loaded && !current.NeedDaemonReload &&
		current.FragmentPath == baseline.FragmentPath
}

type runObservation struct {
	pid           int
	limitsExact   bool
	swapZero      bool
	networkAbsent bool
	apiAbsent     bool
	vsockAbsent   bool
	serialAbsent  bool
}

func (monitor *LinuxMonitor) observeRun(
	run *os.File,
	cgroups *os.File,
	id string,
) (runObservation, bool, error) {
	marker, err := readFileAt(run, ".orquesta-firecracker-run", 4096)
	if err != nil {
		return runObservation{}, false, nil
	}
	expectedMarker := "schema=orquesta_firecracker_launcher_run.v1\nid=" + id +
		"\nnonce="
	if !bytes.HasPrefix(marker, []byte(expectedMarker)) ||
		!bytes.Contains(marker, []byte(
			"\ncgroup="+monitor.config.Candidate.ParentCgroup+"/"+id+"\n",
		)) {
		return runObservation{}, false, errors.New("firecracker_attestor_e2e.run_marker_invalid")
	}
	leaf, err := openDirectoryAt(cgroups, id)
	if err != nil {
		return runObservation{}, false, nil
	}
	defer leaf.Close()
	limits, swapZero, err := validateObservedLimits(leaf)
	if err != nil {
		return runObservation{}, false, nil
	}
	procs, err := readVirtualFileAt(leaf, "cgroup.procs", maxControlBytes)
	if err != nil {
		return runObservation{}, false, nil
	}
	fields := strings.Fields(string(procs))
	if len(fields) != 1 {
		return runObservation{}, false, errors.New("firecracker_attestor_e2e.cgroup_process_count_invalid")
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil || pid <= 1 {
		return runObservation{}, false, errors.New("firecracker_attestor_e2e.cgroup_pid_invalid")
	}
	configRelative := filepath.Join(
		"jail", "firecracker", id, "root", "firecracker.json",
	)
	config, err := readFileAt(run, configRelative, maxControlBytes)
	if err != nil {
		return runObservation{}, false, nil
	}
	isolation, err := validateFirecrackerDocument(config)
	if err != nil {
		return runObservation{}, false, err
	}
	cmdline, netnsEqual, err := monitor.observeProcess(pid)
	if err != nil {
		return runObservation{}, false, nil
	}
	apiAbsent := bytes.Contains(cmdline, []byte("\x00--no-api\x00")) &&
		!bytes.Contains(cmdline, []byte("--api-sock"))
	noSockets, err := treeHasNoSockets(run, 0, maxObservedEntries)
	if err != nil {
		return runObservation{}, false, err
	}
	return runObservation{
		pid: pid, limitsExact: limits, swapZero: swapZero,
		networkAbsent: isolation.networkAbsent && netnsEqual,
		apiAbsent:     apiAbsent && noSockets,
		vsockAbsent:   isolation.vsockAbsent,
		serialAbsent:  isolation.serialAbsent,
	}, true, nil
}

type isolationDocument struct {
	networkAbsent bool
	vsockAbsent   bool
	serialAbsent  bool
}

func validateFirecrackerDocument(content []byte) (isolationDocument, error) {
	var document struct {
		BootSource struct {
			BootArgs string `json:"boot_args"`
		} `json:"boot-source"`
		Machine struct {
			MemoryMiB uint32 `json:"mem_size_mib"`
		} `json:"machine-config"`
		Network []json.RawMessage `json:"network-interfaces"`
		Vsock   json.RawMessage   `json:"vsock"`
	}
	if json.Unmarshal(content, &document) != nil ||
		document.Machine.MemoryMiB != ExpectedGuestMemoryMiB {
		return isolationDocument{}, errors.New("firecracker_attestor_e2e.firecracker_config_invalid")
	}
	vsockAbsent := len(document.Vsock) == 0 || bytes.Equal(document.Vsock, []byte("null"))
	serialAbsent := strings.Contains(document.BootSource.BootArgs, "8250.nr_uarts=0") &&
		!strings.Contains(document.BootSource.BootArgs, "console=")
	return isolationDocument{
		networkAbsent: len(document.Network) == 0,
		vsockAbsent:   vsockAbsent, serialAbsent: serialAbsent,
	}, nil
}

func validateObservedLimits(leaf *os.File) (bool, bool, error) {
	expected := map[string]string{
		"cpu.max":          "200000 100000",
		"memory.max":       "5368709120",
		"memory.swap.max":  "0",
		"memory.oom.group": "1",
		"pids.max":         "512",
	}
	exact := true
	swapZero := false
	for name, want := range expected {
		content, err := readVirtualFileAt(leaf, name, maxControlBytes)
		if err != nil {
			return false, false, err
		}
		got := strings.TrimSpace(string(content))
		exact = exact && got == want
		if name == "memory.swap.max" {
			swapZero = got == "0"
		}
	}
	return exact, swapZero, nil
}

func (monitor *LinuxMonitor) observeProcess(pid int) ([]byte, bool, error) {
	proc, err := openDirectoryNoLinks("/proc", strconv.Itoa(pid))
	if err != nil {
		return nil, false, err
	}
	defer proc.Close()
	cmdline, err := readVirtualFileAt(proc, "cmdline", maxControlBytes)
	if err != nil {
		return nil, false, err
	}
	netFD, err := unix.Openat(
		int(proc.Fd()), "ns/net",
		unix.O_RDONLY|unix.O_CLOEXEC, 0,
	)
	if err != nil {
		return nil, false, err
	}
	defer unix.Close(netFD)
	expectedFD, err := unix.Open(
		monitor.config.Candidate.NetNSPath,
		unix.O_PATH|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return nil, false, err
	}
	defer unix.Close(expectedFD)
	var got, want unix.Stat_t
	if unix.Fstat(netFD, &got) != nil || unix.Fstat(expectedFD, &want) != nil {
		return nil, false, ErrInvalid
	}
	return cmdline, got.Dev == want.Dev && got.Ino == want.Ino, nil
}

func (monitor *LinuxMonitor) launcherSocketAbsent() (bool, error) {
	runtime, err := openDirectoryNoLinks(
		"/", strings.TrimPrefix(monitor.config.Candidate.RuntimeRoot, "/"),
	)
	if err != nil {
		return false, err
	}
	defer runtime.Close()
	var stat unix.Stat_t
	err = unix.Fstatat(
		int(runtime.Fd()), "firecracker-launcher.sock",
		&stat, unix.AT_SYMLINK_NOFOLLOW,
	)
	if errors.Is(err, unix.ENOENT) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (monitor *LinuxMonitor) residualProcesses() (uint32, error) {
	proc, err := openDirectoryNoLinks("/", "proc")
	if err != nil {
		return 0, err
	}
	defer proc.Close()
	entries, err := readBoundedDirectory(proc, 1<<20)
	if err != nil {
		return 0, err
	}
	parent := "/" + strings.Trim(monitor.config.Candidate.ParentCgroup, "/")
	var count uint32
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil || !entry.IsDir() {
			continue
		}
		process, err := openDirectoryAt(proc, entry.Name())
		if err != nil {
			continue
		}
		content, readErr := readVirtualFileAt(process, "cgroup", maxControlBytes)
		process.Close()
		if readErr == nil && processInParentCgroup(content, parent) {
			count++
		}
	}
	return count, nil
}

func processInParentCgroup(content []byte, parent string) bool {
	if parent == "" || parent == "/" || filepath.Clean(parent) != parent {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		_, remainder, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		controllers, path, ok := strings.Cut(remainder, ":")
		if !ok || controllers != "" {
			continue
		}
		if path == parent || strings.HasPrefix(path, parent+"/") {
			return true
		}
	}
	return false
}

func openDirectoryNoLinks(root, relative string) (*os.File, error) {
	rootFD, err := unix.Openat2(unix.AT_FDCWD, root, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.directory_unsafe")
	}
	rootFile := os.NewFile(uintptr(rootFD), "firecracker-attestor-e2e-root")
	if relative == "." || relative == "" {
		return rootFile, nil
	}
	child, err := openDirectoryAt(rootFile, relative)
	rootFile.Close()
	return child, err
}

func openDirectoryAt(parent *os.File, relative string) (*os.File, error) {
	if parent == nil || relative == "" || filepath.IsAbs(relative) ||
		filepath.Clean(relative) != relative {
		return nil, ErrInvalid
	}
	fd, err := unix.Openat2(int(parent.Fd()), relative, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "firecracker-attestor-e2e-directory"), nil
}

func readFileAt(parent *os.File, relative string, maxBytes int64) ([]byte, error) {
	return readFileAtPolicy(parent, relative, maxBytes, true)
}

func readVirtualFileAt(parent *os.File, relative string, maxBytes int64) ([]byte, error) {
	return readFileAtPolicy(parent, relative, maxBytes, false)
}

func readFileAtPolicy(
	parent *os.File,
	relative string,
	maxBytes int64,
	requireStableSize bool,
) ([]byte, error) {
	if parent == nil || relative == "" || filepath.IsAbs(relative) ||
		filepath.Clean(relative) != relative || maxBytes <= 0 {
		return nil, ErrInvalid
	}
	fd, err := unix.Openat2(int(parent.Fd()), relative, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "firecracker-attestor-e2e-file")
	defer file.Close()
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Size < 0 || stat.Size > maxBytes {
		return nil, ErrInvalid
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || int64(len(content)) > maxBytes ||
		requireStableSize && int64(len(content)) != stat.Size {
		return nil, ErrInvalid
	}
	return content, nil
}

func readBoundedDirectory(directory *os.File, limit int) ([]os.DirEntry, error) {
	if directory == nil || limit <= 0 {
		return nil, ErrInvalid
	}
	if _, err := directory.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	entries, err := directory.ReadDir(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > limit {
		return nil, errors.New("firecracker_attestor_e2e.directory_oversize")
	}
	return entries, nil
}

func treeHasNoSockets(directory *os.File, visited, limit int) (bool, error) {
	if visited < 0 || limit <= 0 || visited >= limit {
		return false, errors.New("firecracker_attestor_e2e.tree_oversize")
	}
	return treeHasNoSocketsBounded(directory, &visited, limit)
}

func treeHasNoSocketsBounded(
	directory *os.File,
	visited *int,
	limit int,
) (bool, error) {
	if directory == nil || visited == nil || *visited < 0 || *visited >= limit {
		return false, errors.New("firecracker_attestor_e2e.tree_oversize")
	}
	entries, err := readBoundedDirectory(directory, limit-*visited)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if *visited >= limit {
			return false, errors.New("firecracker_attestor_e2e.tree_oversize")
		}
		(*visited)++
		var stat unix.Stat_t
		if unix.Fstatat(int(directory.Fd()), entry.Name(), &stat, unix.AT_SYMLINK_NOFOLLOW) != nil {
			return false, ErrInvalid
		}
		if stat.Mode&unix.S_IFMT == unix.S_IFSOCK {
			return false, nil
		}
		if stat.Mode&unix.S_IFMT == unix.S_IFDIR {
			child, err := openDirectoryAt(directory, entry.Name())
			if err != nil {
				return false, err
			}
			clean, walkErr := treeHasNoSocketsBounded(child, visited, limit)
			child.Close()
			if walkErr != nil || !clean {
				return clean, walkErr
			}
		}
	}
	return true, nil
}

func validRunID(value string) bool {
	const prefix = "orq-"
	if len(value) != len(prefix)+52 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range strings.TrimPrefix(value, prefix) {
		if character < 'a' || character > 'z' {
			if character < '2' || character > '7' {
				return false
			}
		}
	}
	return true
}
