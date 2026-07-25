//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type fakeRunnerNamespace struct {
	mu          sync.Mutex
	revalidated int
	closed      bool
	err         error
}

func (namespace *fakeRunnerNamespace) revalidate() error {
	namespace.mu.Lock()
	defer namespace.mu.Unlock()
	namespace.revalidated++
	return namespace.err
}

func (namespace *fakeRunnerNamespace) Close() error {
	namespace.mu.Lock()
	defer namespace.mu.Unlock()
	namespace.closed = true
	return nil
}

type fakeRunnerCgroup struct {
	mu         sync.Mutex
	prepared   []string
	killed     []string
	cleaned    []string
	cleanupErr error
}

func (cgroup *fakeRunnerCgroup) prepare(id string) error {
	cgroup.mu.Lock()
	defer cgroup.mu.Unlock()
	cgroup.prepared = append(cgroup.prepared, id)
	return nil
}

func (cgroup *fakeRunnerCgroup) kill(id string) error {
	cgroup.mu.Lock()
	defer cgroup.mu.Unlock()
	cgroup.killed = append(cgroup.killed, id)
	return nil
}

func (cgroup *fakeRunnerCgroup) cleanup(
	_ context.Context,
	id string,
	_ LaunchRequest,
	_ bool,
) error {
	cgroup.mu.Lock()
	defer cgroup.mu.Unlock()
	cgroup.cleaned = append(cgroup.cleaned, id)
	return cgroup.cleanupErr
}

func (*fakeRunnerCgroup) Close() error { return nil }

type recordingCommandFactory struct {
	mu       sync.Mutex
	mode     string
	args     [][]string
	configs  [][]byte
	layouts  []recordedJailLayout
	commands []*exec.Cmd
}

func (factory *recordingCommandFactory) build(_ string, arguments ...string) *exec.Cmd {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	copied := append([]string(nil), arguments...)
	factory.args = append(factory.args, copied)
	root := helperJailRoot(arguments)
	if raw, err := os.ReadFile(filepath.Join(root, firecrackerConfig)); err == nil {
		factory.configs = append(factory.configs, raw)
	}
	factory.layouts = append(factory.layouts, recordJailLayout(root))
	helperArguments := []string{
		"-test.run=^TestPhysicalRunnerHelperProcess$",
		"--",
		"--helper-mode=" + factory.mode,
	}
	helperArguments = append(helperArguments, arguments...)
	command := exec.Command(os.Args[0], helperArguments...)
	factory.commands = append(factory.commands, command)
	return command
}

func TestPhysicalRunnerHelperProcess(t *testing.T) {
	mode := argumentValue(os.Args, "--helper-mode")
	if mode == "" {
		t.Skip("helper only")
	}
	if mode == "hang" {
		signal.Ignore(syscall.SIGTERM)
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("D"), 128))
		for {
			time.Sleep(time.Hour)
		}
	}
	root := helperJailRoot(os.Args)
	output := filepath.Join(root, outputDriveJailName)
	file, err := os.OpenFile(output, os.O_WRONLY, 0)
	if err != nil {
		os.Exit(91)
	}
	_, writeErr := file.WriteAt([]byte("ORQ-PHYSICAL-OUTPUT"), 0)
	syncErr := file.Sync()
	closeErr := file.Close()
	_, _ = os.Stdout.Write(bytes.Repeat([]byte("D"), 128))
	if errors.Join(writeErr, syncErr, closeErr) != nil {
		os.Exit(92)
	}
}

func TestPhysicalRunnerBuildsIsolatedJailerCommandAndCopiesOutput(t *testing.T) {
	runner, factory, cgroup, namespace, request, input, output := physicalRunnerFixture(t, "success")
	result, err := runner.Run(context.Background(), request, input, output)
	if err != nil {
		t.Fatal(err)
	}
	if result.AssetDigest != runner.assets.digest ||
		result.CapturedOutputBytes != uint64(runner.config.MaxDiagnosticBytes) {
		t.Fatalf("result=%+v", result)
	}
	content := make([]byte, len("ORQ-PHYSICAL-OUTPUT"))
	if _, err := output.ReadAt(content, 0); err != nil ||
		string(content) != "ORQ-PHYSICAL-OUTPUT" {
		t.Fatalf("output=%q err=%v", content, err)
	}
	assertJailerIsolationArguments(t, factory.args[0], runner.config, request)
	assertPrecreatedJailerLayout(t, factory.layouts[0], runner)
	assertFirecrackerConfiguration(t, factory.configs[0], request)
	id, _ := runID(request.Nonce)
	cgroup.mu.Lock()
	if len(cgroup.prepared) != 1 || cgroup.prepared[0] != id ||
		len(cgroup.cleaned) != 1 || cgroup.cleaned[0] != id {
		t.Fatalf("cgroup prepared=%v cleaned=%v", cgroup.prepared, cgroup.cleaned)
	}
	cgroup.mu.Unlock()
	namespace.mu.Lock()
	if namespace.revalidated != 1 {
		t.Fatalf("netns revalidations=%d", namespace.revalidated)
	}
	namespace.mu.Unlock()
	assertRunsEmpty(t, runner.runs.path)
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPhysicalRunnerRejectsGuestMemoryBelowPinnedManifest(t *testing.T) {
	runner, factory, cgroup, namespace, request, input, output := physicalRunnerFixture(t, "success")
	runner.assets.minimumGuestMemoryMiB = request.GuestMemoryMiB + 1
	if _, err := runner.Run(context.Background(), request, input, output); ErrorCode(err) != CodeResourceUnsafe {
		t.Fatalf("memory below manifest minimum accepted: %v", err)
	}
	if len(factory.args) != 0 || len(cgroup.prepared) != 0 || namespace.revalidated != 0 {
		t.Fatalf(
			"effects before manifest memory rejection: commands=%d cgroups=%d netns=%d",
			len(factory.args), len(cgroup.prepared), namespace.revalidated,
		)
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestJailerArgumentsNeverDetachObservedFirecrackerLifecycle(t *testing.T) {
	config := validConfigForTest("/run/orquesta-firecracker")
	request := validLaunchRequestForTest()
	workspace := &runWorkspace{
		id:              "orq-" + strings.Repeat("a", 52),
		path:            "/run/orquesta-firecracker/runs/test",
		firecrackerPath: "/run/orquesta-firecracker/runs/test/firecracker",
	}
	arguments := buildJailerArguments(config, request, workspace)
	for _, forbidden := range []string{"--new-pid-ns", "--daemonize"} {
		for _, argument := range arguments {
			if argument == forbidden {
				t.Fatalf("%s detaches command.Wait from Firecracker", forbidden)
			}
		}
	}
}

func TestPhysicalRunnerTimeoutTermsKillsWaitsAndCleans(t *testing.T) {
	runner, factory, cgroup, _, request, input, output := physicalRunnerFixture(t, "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err := runner.Run(ctx, request, input, output)
	if ErrorCode(err) != CodeExecutionTimeout {
		t.Fatalf("timeout err=%v", err)
	}
	factory.mu.Lock()
	command := factory.commands[0]
	factory.mu.Unlock()
	if command.ProcessState == nil ||
		!errors.Is(unix.Kill(command.Process.Pid, 0), unix.ESRCH) {
		t.Fatalf("helper not reaped: pid=%d state=%v", command.Process.Pid, command.ProcessState)
	}
	cgroup.mu.Lock()
	if len(cgroup.killed) == 0 || len(cgroup.cleaned) != 1 {
		t.Fatalf("cgroup kill=%v cleanup=%v", cgroup.killed, cgroup.cleaned)
	}
	cgroup.mu.Unlock()
	assertRunsEmpty(t, runner.runs.path)
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPhysicalRunnerCleanupFailureDominatesSuccessfulExecution(t *testing.T) {
	runner, _, cgroup, _, request, input, output := physicalRunnerFixture(t, "success")
	cgroup.cleanupErr = errors.New("injected cgroup cleanup failure")
	result, err := runner.Run(context.Background(), request, input, output)
	if ErrorCode(err) != CodeCleanupFailed || result != (RunResult{}) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if ErrorCode(runner.Close()) != CodeCleanupFailed {
		t.Fatal("runner close hid prior cleanup failure")
	}
}

func TestPhysicalRunnerCloseCancelsKillsWaitsAndReapsActiveRun(t *testing.T) {
	runner, factory, _, _, request, input, output := physicalRunnerFixture(t, "hang")
	runDone := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), request, input, output)
		runDone <- err
	}()
	deadline := time.Now().Add(time.Second)
	for !runnerHasStartedCommand(runner) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !runnerHasStartedCommand(runner) {
		t.Fatal("helper did not start")
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-runDone; ErrorCode(err) != CodeUnavailable {
		t.Fatalf("active run error=%v", err)
	}
	factory.mu.Lock()
	command := factory.commands[0]
	factory.mu.Unlock()
	if command.ProcessState == nil ||
		!errors.Is(unix.Kill(command.Process.Pid, 0), unix.ESRCH) {
		t.Fatalf("active helper not reaped: pid=%d state=%v", command.Process.Pid, command.ProcessState)
	}
}

func TestPhysicalRunnerConcurrentCloseReapsOneActiveRun(t *testing.T) {
	runner, _, _, _, request, input, output := physicalRunnerFixture(t, "hang")
	runner.config.CleanupTimeout = 250 * time.Millisecond
	runDone := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), request, input, output)
		runDone <- err
	}()
	deadline := time.Now().Add(time.Second)
	for !runnerHasStartedCommand(runner) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !runnerHasStartedCommand(runner) {
		t.Fatal("helper did not start")
	}
	closeErrors := make(chan error, 4)
	for range 4 {
		go func() { closeErrors <- runner.Close() }()
	}
	for range 4 {
		if err := <-closeErrors; err != nil {
			t.Fatal(err)
		}
	}
	if err := <-runDone; ErrorCode(err) != CodeUnavailable {
		t.Fatalf("active run error=%v", err)
	}
	assertRunsEmpty(t, runner.runs.path)
}

func TestPhysicalRunnerSupportsConcurrentDistinctNonces(t *testing.T) {
	runner, _, _, _, request, inputOne, outputOne := physicalRunnerFixture(t, "success")
	inputTwo, digestTwo, err := NewSealedInput(
		inputDrivePayloadForTest("physical-two"),
		runner.config.MaxInputBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer inputTwo.Close()
	outputTwo, err := NewOutputDescriptor(request.OutputDriveBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer outputTwo.Close()
	requestTwo := request
	requestTwo.Nonce = strings.Repeat("b", 64)
	requestTwo.InputDigest = digestTwo
	errs := make(chan error, 2)
	go func() {
		_, runErr := runner.Run(context.Background(), request, inputOne, outputOne)
		errs <- runErr
	}()
	go func() {
		_, runErr := runner.Run(context.Background(), requestTwo, inputTwo, outputTwo)
		errs <- runErr
	}()
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	assertRunsEmpty(t, runner.runs.path)
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunIDIsBoundedInjectiveAndJailerSafe(t *testing.T) {
	seen := make(map[string]struct{})
	for _, nonce := range []string{
		strings.Repeat("0", 64),
		strings.Repeat("0", 63) + "1",
		strings.Repeat("f", 64),
	} {
		id, err := runID(nonce)
		if err != nil || len(id) > 64 || !validJailID(id) {
			t.Fatalf("nonce=%q id=%q err=%v", nonce, id, err)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("non-injective id=%q", id)
		}
		seen[id] = struct{}{}
	}
	if _, err := runID("../../host"); ErrorCode(err) != CodeProtocolInvalid {
		t.Fatalf("host path accepted: %v", err)
	}
}

func TestPhysicalProcessWaitUsesOneDeadlineWhenWaitNeverReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cgroup := &fakeRunnerCgroup{}
	started := time.Now()
	result := waitPhysicalProcessDone(
		ctx,
		nil,
		cgroup,
		"orq-"+strings.Repeat("a", 52),
		24*time.Millisecond,
		make(chan error),
	)
	elapsed := time.Since(started)
	if result.reaped || ErrorCode(result.err) != CodeCleanupFailed ||
		elapsed < 20*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Fatalf("result=%+v elapsed=%s", result, elapsed)
	}
	cgroup.mu.Lock()
	defer cgroup.mu.Unlock()
	if len(cgroup.killed) != 1 {
		t.Fatalf("cgroup kills=%v", cgroup.killed)
	}
}

func TestPhysicalRunnerCloseUsesOneDeadlineWhenActiveNeverFinishes(t *testing.T) {
	runner, _, cgroup, _, _, _, _ := physicalRunnerFixture(t, "success")
	active := &activePhysicalRun{
		cancel:  func() {},
		cgroups: cgroup,
		id:      "orq-" + strings.Repeat("a", 52),
	}
	if !runner.register(active) {
		t.Fatal("could not register synthetic active run")
	}
	started := time.Now()
	err := runner.Close()
	elapsed := time.Since(started)
	if ErrorCode(err) != CodeCleanupFailed ||
		elapsed < 15*time.Millisecond ||
		elapsed > 100*time.Millisecond {
		t.Fatalf("close err=%v elapsed=%s", err, elapsed)
	}
	runner.unregister(active)
}

func TestActivePhysicalRunTermsBeforeCgroupKill(t *testing.T) {
	cgroup := &fakeRunnerCgroup{}
	active := &activePhysicalRun{
		cancel:  func() {},
		cgroups: cgroup,
		id:      "orq-" + strings.Repeat("a", 52),
	}
	if err := active.signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	cgroup.mu.Lock()
	if len(cgroup.killed) != 0 {
		t.Fatalf("TERM killed cgroup: %v", cgroup.killed)
	}
	cgroup.mu.Unlock()
	if err := active.kill(); err != nil {
		t.Fatal(err)
	}
	cgroup.mu.Lock()
	defer cgroup.mu.Unlock()
	if len(cgroup.killed) != 1 || cgroup.killed[0] != active.id {
		t.Fatalf("KILL did not kill exact cgroup: %v", cgroup.killed)
	}
}

func physicalRunnerFixture(
	t *testing.T,
	mode string,
) (
	*physicalRunner,
	*recordingCommandFactory,
	*fakeRunnerCgroup,
	*fakeRunnerNamespace,
	LaunchRequest,
	*os.File,
	*os.File,
) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "runtime")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	owner := uint32(os.Geteuid())
	runs, err := openRunsRoot(runtimeRoot, root, owner)
	if err != nil {
		t.Fatal(err)
	}
	config := validConfigForTest(root)
	config.AllowedUID, config.AllowedGID = owner+1, owner+1
	config.JailUID, config.JailGID = owner, uint32(os.Getegid())
	config.CleanupTimeout = 20 * time.Millisecond
	config.MaxDiagnosticBytes = 32
	assets := fakeAssetSetForRunnerTest(t)
	namespace := &fakeRunnerNamespace{}
	cgroup := &fakeRunnerCgroup{}
	factory := &recordingCommandFactory{mode: mode}
	runner := newPhysicalRunnerWithDependencies(
		config,
		assets,
		namespace,
		cgroup,
		runtimeRoot,
		runs,
		factory.build,
	)
	request := validLaunchRequestForTest()
	request.Nonce = strings.Repeat("a", 64)
	input, digest, err := NewSealedInput(
		inputDrivePayloadForTest("physical-one"),
		config.MaxInputBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = input.Close() })
	request.InputDigest = digest
	output, err := NewOutputDescriptor(request.OutputDriveBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = output.Close() })
	return runner, factory, cgroup, namespace, request, input, output
}

func fakeAssetSetForRunnerTest(t *testing.T) *assetSet {
	t.Helper()
	newAsset := func(content string) *pinnedAsset {
		file, err := os.CreateTemp(t.TempDir(), "asset-")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteString(content); err != nil {
			t.Fatal(err)
		}
		return &pinnedAsset{file: file, size: int64(len(content)), digest: digestBytes([]byte(content))}
	}
	return &assetSet{
		firecracker:           newAsset("firecracker"),
		jailer:                newAsset("jailer"),
		kernel:                newAsset("kernel"),
		guest:                 newAsset("guest"),
		manifest:              newAsset("manifest"),
		digest:                digestBytes([]byte("physical-assets")),
		minimumGuestMemoryMiB: minGuestMemoryMiB,
	}
}

func helperJailRoot(arguments []string) string {
	base, id := argumentValue(arguments, "--chroot-base-dir"), argumentValue(arguments, "--id")
	if base == "" || id == "" {
		return ""
	}
	return filepath.Join(base, "firecracker", id, "root")
}

func argumentValue(arguments []string, name string) string {
	prefix := name + "="
	for index, argument := range arguments {
		if strings.HasPrefix(argument, prefix) {
			return strings.TrimPrefix(argument, prefix)
		}
		if argument == name && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

func assertJailerIsolationArguments(
	t *testing.T,
	arguments []string,
	config Config,
	request LaunchRequest,
) {
	t.Helper()
	joined := "\x00" + strings.Join(arguments, "\x00") + "\x00"
	for _, required := range []string{
		"\x00--netns\x00" + config.NetNSPath + "\x00",
		"\x00--cgroup-version\x002\x00",
		"\x00--cgroup\x00cpu.max=",
		"\x00--cgroup\x00memory.max=",
		"\x00--cgroup\x00memory.swap.max=0\x00",
		"\x00--cgroup\x00memory.oom.group=1\x00",
		"\x00--cgroup\x00pids.max=",
		"\x00--resource-limit\x00fsize=",
		"\x00--resource-limit\x00no-file=256\x00",
		"\x00--\x00--no-api\x00--config-file\x00/firecracker.json\x00",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %q", required, arguments)
		}
	}
	for _, forbidden := range []string{"--new-pid-ns", "--daemonize", "--api-sock"} {
		if strings.Contains(joined, "\x00"+forbidden+"\x00") {
			t.Fatalf("forbidden %s in %q", forbidden, arguments)
		}
	}
	cgroups := optionValues(arguments, "--cgroup")
	wantCgroups := []string{
		"cpu.max=" + strconv.FormatUint(request.CPUQuotaMicros, 10) +
			" " + strconv.FormatUint(request.CPUPeriodMicros, 10),
		"memory.max=" + strconv.FormatUint(request.MemoryMaxBytes, 10),
		"memory.swap.max=0",
		"memory.oom.group=1",
		"pids.max=" + strconv.FormatUint(uint64(request.PIDsMax), 10),
	}
	if !slices.Equal(cgroups, wantCgroups) {
		t.Fatalf("cgroup arguments=%q want=%q", cgroups, wantCgroups)
	}
}

func assertFirecrackerConfiguration(t *testing.T, raw []byte, request LaunchRequest) {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if networks, ok := document["network-interfaces"].([]any); !ok || len(networks) != 0 {
		t.Fatalf("network interfaces=%v: %s", document["network-interfaces"], raw)
	}
	for _, forbidden := range []string{"vsock", "logger", "metrics", "balloon"} {
		if _, ok := document[forbidden]; ok {
			t.Fatalf("%s present: %s", forbidden, raw)
		}
	}
	drives, ok := document["drives"].([]any)
	if !ok || len(drives) != 2 {
		t.Fatalf("drives=%v", document["drives"])
	}
	input := drives[0].(map[string]any)
	output := drives[1].(map[string]any)
	if input["path_on_host"] != "/input.drive" || input["is_read_only"] != true ||
		output["path_on_host"] != "/output.drive" || output["is_read_only"] != false {
		t.Fatalf("drive policy input=%v output=%v", input, output)
	}
	boot := document["boot-source"].(map[string]any)
	if !strings.Contains(boot["boot_args"].(string), "8250.nr_uarts=0") ||
		!strings.Contains(boot["boot_args"].(string), "init=/init") {
		t.Fatalf("boot=%v", boot)
	}
	machine := document["machine-config"].(map[string]any)
	wantVCPUs := float64(
		(request.CPUQuotaMicros + request.CPUPeriodMicros - 1) /
			request.CPUPeriodMicros,
	)
	if machine["vcpu_count"] != wantVCPUs ||
		machine["mem_size_mib"] != float64(request.GuestMemoryMiB) {
		t.Fatalf("machine=%v", machine)
	}
}

func assertPrecreatedJailerLayout(
	t *testing.T,
	layout recordedJailLayout,
	runner *physicalRunner,
) {
	t.Helper()
	if layout.err != nil {
		t.Fatal(layout.err)
	}
	rootStat := layout.root
	if rootStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		rootStat.Mode&0o777 != 0o700 ||
		rootStat.Uid != runner.runs.owner ||
		rootStat.Gid != runner.runs.owner {
		t.Fatalf("jail root stat=%+v", rootStat)
	}
	for name, mode := range map[string]uint32{
		kernelJailName:      unix.S_IFREG | 0o400,
		guestJailName:       unix.S_IFREG | 0o400,
		inputDriveJailName:  unix.S_IFREG | 0o400,
		outputDriveJailName: unix.S_IFREG | 0o600,
		firecrackerConfig:   unix.S_IFREG | 0o400,
	} {
		stat := layout.files[name]
		if stat.Mode != mode ||
			stat.Uid != runner.config.JailUID ||
			stat.Gid != runner.config.JailGID ||
			stat.Nlink != 1 {
			t.Fatalf("%s stat=%+v", name, stat)
		}
	}
}

type recordedJailLayout struct {
	root  unix.Stat_t
	files map[string]unix.Stat_t
	err   error
}

func recordJailLayout(root string) recordedJailLayout {
	layout := recordedJailLayout{files: make(map[string]unix.Stat_t)}
	if err := unix.Lstat(root, &layout.root); err != nil {
		layout.err = err
		return layout
	}
	for _, name := range []string{
		kernelJailName,
		guestJailName,
		inputDriveJailName,
		outputDriveJailName,
		firecrackerConfig,
	} {
		var stat unix.Stat_t
		if err := unix.Lstat(filepath.Join(root, name), &stat); err != nil {
			layout.err = err
			return layout
		}
		layout.files[name] = stat
	}
	return layout
}

func optionValues(arguments []string, name string) []string {
	var values []string
	for index := 0; index < len(arguments); index++ {
		if arguments[index] == name && index+1 < len(arguments) {
			values = append(values, arguments[index+1])
			index++
		}
	}
	return values
}

func assertRunsEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("runs residual entries=%v err=%v", entries, err)
	}
}

func runnerHasStartedCommand(runner *physicalRunner) bool {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	for active := range runner.active {
		active.mu.Lock()
		started := active.command != nil && active.command.Process != nil
		active.mu.Unlock()
		if started {
			return true
		}
	}
	return false
}
