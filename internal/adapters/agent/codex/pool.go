package codex

import (
	"context"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	CodePoolConfigInvalid     = "codex.pool_config_invalid"
	CodePoolProfileWithdrawal = "codex.pool_profile_withdrawal"
	CodePoolRoutingInvalid    = "codex.pool_routing_invalid"

	poolProfilesDirectory = "profiles"
	poolProfilePrefix     = "account-"
)

var errPoolShutdown = errors.New("codex.pool_shutdown")

// AccountHome identifies one persistent Codex account without exposing it to
// provider-neutral state. Root contains Profile; auth.json remains in place.
type AccountHome struct {
	Root    string
	Profile string
}

// PoolConfig applies one homogeneous adapter configuration to every account.
// Adapter.WorkRoot is the private pool root. Per-account WorkRoots are derived
// from opaque account-profile refs and Adapter.MaxConcurrentExecutions is
// deliberately replaced with one.
type PoolConfig struct {
	Adapter      Config
	AccountHomes []AccountHome
}

// Pool is one Codex adapter surface backed by one capacity-one Adapter per
// persistent account. It owns no execution lifecycle or routing store:
// request.json journals remain the durable routing authority.
type Pool struct {
	profiles          []*poolProfile
	aggregateCapacity int

	mu         sync.Mutex
	closed     bool
	routes     map[string]*poolRoute
	operations sync.WaitGroup

	lifecycle       context.Context
	cancelLifecycle context.CancelCauseFunc
	shutdownOnce    sync.Once
	shutdownDone    chan struct{}
	shutdownErr     error
}

type poolProfile struct {
	ref      string
	workRoot string
	adapter  *Adapter
}

type poolRoute struct {
	profile   *poolProfile
	selecting chan struct{}
}

type poolProfileSpec struct {
	home          AccountHome
	ref           string
	directoryName string
}

func NewPool(config PoolConfig) (*Pool, error) {
	specs, poolRoot, err := preparePoolConfig(config)
	if err != nil {
		return nil, err
	}
	profiles := make([]*poolProfile, 0, len(specs))
	fail := func(cause error) (*Pool, error) {
		closeErrs := make([]error, 0, len(profiles)+1)
		closeErrs = append(closeErrs, cause)
		for _, profile := range profiles {
			closeErrs = append(closeErrs, profile.adapter.Close())
		}
		return nil, errors.Join(closeErrs...)
	}
	for _, spec := range specs {
		adapterConfig := config.Adapter
		adapterConfig.WorkRoot = filepath.Join(poolRoot, poolProfilesDirectory, spec.directoryName)
		adapterConfig.AccountHomeRoot = spec.home.Root
		adapterConfig.AccountProfile = spec.home.Profile
		adapterConfig.MaxConcurrentExecutions = 1
		adapterConfig.Environment = cloneEnvironment(config.Adapter.Environment)
		adapter, adapterErr := New(adapterConfig)
		if adapterErr != nil {
			return fail(adapterErr)
		}
		profiles = append(profiles, &poolProfile{
			ref: spec.ref, workRoot: adapterConfig.WorkRoot, adapter: adapter,
		})
	}
	lifecycle, cancelLifecycle := context.WithCancelCause(context.Background())
	pool := &Pool{
		profiles: profiles, aggregateCapacity: config.Adapter.MaxConcurrentExecutions,
		routes:    make(map[string]*poolRoute),
		lifecycle: lifecycle, cancelLifecycle: cancelLifecycle,
		shutdownDone: make(chan struct{}),
	}
	if err := pool.rebuildRoutingIndex(); err != nil {
		return fail(err)
	}
	return pool, nil
}

func preparePoolConfig(config PoolConfig) ([]poolProfileSpec, string, error) {
	base := config.Adapter
	if len(config.AccountHomes) == 0 ||
		base.AccountHomeRoot != "" || base.AccountProfile != "" ||
		base.CredentialStore != nil || base.CredentialRef != "" {
		return nil, "", &Error{Code: CodePoolConfigInvalid}
	}
	if base.MaxConcurrentExecutions <= 0 {
		return nil, "", &Error{Code: CodeMaxConcurrentInvalid}
	}
	poolRoot, root, err := openPrivateRoot(base.WorkRoot)
	if err != nil {
		return nil, "", err
	}
	defer root.Close()
	canonicalPoolRoot, err := filepath.EvalSymlinks(poolRoot)
	if err != nil {
		return nil, "", &Error{Code: CodeWorkRootInvalid, Cause: err}
	}

	specs := make([]poolProfileSpec, 0, len(config.AccountHomes))
	refs := make(map[string]struct{}, len(config.AccountHomes))
	directories := make(map[string]struct{}, len(config.AccountHomes))
	for _, home := range config.AccountHomes {
		profileConfig := base
		profileConfig.AccountHomeRoot = home.Root
		profileConfig.AccountProfile = home.Profile
		profileConfig.MaxConcurrentExecutions = 1
		if err := validateAccountProfileConfig(profileConfig); err != nil {
			return nil, "", &Error{Code: CodePoolConfigInvalid, Cause: err}
		}
		ref, err := deriveAccountProfileRef(home.Root, home.Profile)
		if err != nil {
			return nil, "", err
		}
		directoryName, err := poolProfileDirectoryName(ref)
		if err != nil {
			return nil, "", err
		}
		if _, duplicate := refs[ref]; duplicate {
			return nil, "", &Error{Code: CodePoolConfigInvalid}
		}
		if _, duplicate := directories[directoryName]; duplicate {
			return nil, "", &Error{Code: CodePoolConfigInvalid}
		}
		canonicalHome, err := filepath.EvalSymlinks(filepath.Join(home.Root, home.Profile))
		if err != nil {
			return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
		}
		if filesystemPathsOverlap(canonicalPoolRoot, canonicalHome) {
			return nil, "", &Error{Code: CodeWorkRootInvalid}
		}
		refs[ref] = struct{}{}
		directories[directoryName] = struct{}{}
		specs = append(specs, poolProfileSpec{
			home: home, ref: ref, directoryName: directoryName,
		})
	}
	sort.Slice(specs, func(left, right int) bool { return specs[left].ref < specs[right].ref })
	if err := preparePoolProfileDirectories(root, directories); err != nil {
		return nil, "", err
	}
	return specs, poolRoot, nil
}

func poolProfileDirectoryName(ref string) (string, error) {
	if !validAccountProfileRef(ref) {
		return "", &Error{Code: CodePoolConfigInvalid}
	}
	encoded := strings.TrimPrefix(ref, accountProfileRefPrefix)
	return poolProfilePrefix + encoded, nil
}

func filesystemPathsOverlap(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	return left == right || pathWithinFilesystem(left, right) || pathWithinFilesystem(right, left)
}

func pathWithinFilesystem(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func preparePoolProfileDirectories(root *os.Root, configured map[string]struct{}) error {
	entries, err := readRootDirectory(root, ".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != poolProfilesDirectory {
			return &Error{Code: CodePoolConfigInvalid}
		}
	}
	if _, err := root.Lstat(poolProfilesDirectory); errors.Is(err, fs.ErrNotExist) {
		if err := root.Mkdir(poolProfilesDirectory, 0o700); err != nil {
			return &Error{Code: CodeWorkRootCreateFailed, Cause: err}
		}
		if err := syncDirectoryAndParent(poolProfilesDirectory, func(directory string) error {
			return syncCodexDirectory(root, directory)
		}); err != nil {
			return &Error{Code: CodeWorkRootCreateFailed, Cause: err}
		}
	} else if err != nil {
		return &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	info, err := root.Lstat(poolProfilesDirectory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	entries, err = readRootDirectory(root, poolProfilesDirectory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := path.Join(poolProfilesDirectory, entry.Name())
		info, statErr := root.Lstat(entryPath)
		if statErr != nil || entry.Type()&os.ModeSymlink != 0 ||
			info.Mode()&os.ModeSymlink != 0 || !entry.IsDir() || !info.IsDir() ||
			info.Mode().Perm()&0o077 != 0 {
			return &Error{Code: CodeWorkRootInvalid}
		}
		if _, present := configured[entry.Name()]; present {
			continue
		}
		if !validPoolProfileDirectoryName(entry.Name()) {
			return &Error{Code: CodeWorkRootInvalid}
		}
		retiredEntries, err := readRootDirectory(root, entryPath)
		if err != nil {
			return err
		}
		if len(retiredEntries) != 0 {
			return &Error{Code: CodePoolProfileWithdrawal}
		}
	}
	return nil
}

func validPoolProfileDirectoryName(value string) bool {
	encoded := strings.TrimPrefix(value, poolProfilePrefix)
	if poolProfilePrefix+encoded != value || len(encoded) != 64 ||
		encoded != strings.ToLower(encoded) {
		return false
	}
	decoded, err := hex.DecodeString(encoded)
	return err == nil && len(decoded) == 32
}

func readRootDirectory(root *os.Root, directory string) ([]os.DirEntry, error) {
	handle, err := root.Open(directory)
	if err != nil {
		return nil, &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	defer handle.Close()
	entries, err := handle.ReadDir(-1)
	if err != nil {
		return nil, &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	return entries, nil
}

func (pool *Pool) rebuildRoutingIndex() error {
	digestOwners := make(map[string]*poolProfile)
	for _, profile := range pool.profiles {
		if err := pool.indexProfileJournals(profile, digestOwners); err != nil {
			return err
		}
	}
	return nil
}

func (pool *Pool) indexProfileJournals(
	profile *poolProfile,
	digestOwners map[string]*poolProfile,
) error {
	if err := profile.adapter.validatePrivateDirectory("executions"); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	entries, err := readAdapterDirectory(profile.adapter, "executions")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		digest := entry.Name()
		if !validExecutionDirectoryDigest(digest) ||
			entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			return &Error{Code: CodePoolRoutingInvalid}
		}
		if owner := digestOwners[digest]; owner != nil && owner != profile {
			return &Error{Code: CodePoolRoutingInvalid}
		}
		digestOwners[digest] = profile
		runPath := path.Join("executions", digest)
		record, found, err := profile.adapter.readLaunchRecord(runPath)
		if err != nil {
			return err
		}
		if !found {
			// The profile has materialized an execution boundary but lacks the
			// durable routing authority that must precede process creation.
			// Never guess another account after this crash frontier.
			return &Error{Code: CodePoolRoutingInvalid}
		}
		executionRef, err := goal.NewExecutionRef(record.ExecutionRef)
		if err != nil || path.Base(executionPath(executionRef)) != digest ||
			profile.adapter.validateAccountProfileBinding(record) != nil {
			return &Error{Code: CodePoolRoutingInvalid, Cause: err}
		}
		key := executionRef.String()
		if existing := pool.routes[key]; existing != nil && existing.profile != profile {
			return &Error{Code: CodePoolRoutingInvalid}
		}
		pool.routes[key] = &poolRoute{profile: profile}
	}
	return nil
}

func readAdapterDirectory(adapter *Adapter, directory string) ([]os.DirEntry, error) {
	handle, err := adapter.root.Open(directory)
	if err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	defer handle.Close()
	entries, err := handle.ReadDir(-1)
	if err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	return entries, nil
}

func validExecutionDirectoryDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func (pool *Pool) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return ports.AgentCapabilities{}, err
	}
	defer end()
	var result ports.AgentCapabilities
	for index, profile := range pool.profiles {
		capabilities, err := profile.adapter.Capabilities(operationContext)
		if err != nil {
			return ports.AgentCapabilities{}, err
		}
		if index == 0 {
			result = capabilities
			continue
		}
		if !sameAgentCapabilities(result, capabilities) {
			return ports.AgentCapabilities{}, &Error{Code: CodePoolConfigInvalid}
		}
	}
	return result, nil
}

func sameAgentCapabilities(left, right ports.AgentCapabilities) bool {
	return left.ProviderRef == right.ProviderRef &&
		left.ModelRef == right.ModelRef &&
		left.AgentRef == right.AgentRef &&
		left.Unrestricted == right.Unrestricted &&
		slices.Equal(left.RoleKeys, right.RoleKeys) &&
		slices.Equal(left.SkillRefs, right.SkillRefs) &&
		slices.Equal(left.ToolRefs, right.ToolRefs) &&
		slices.Equal(left.CapabilityRefs, right.CapabilityRefs)
}

func (pool *Pool) Launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	defer end()
	seleccionado := pool.perfilColocacion(request.ReferenciaColocacion)
	if seleccionado == nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodePoolRoutingInvalid}
	}
	profile, selectProfile, err := pool.route(operationContext, request.ExecutionRef, true)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if !selectProfile {
		if profile != seleccionado {
			return ports.AgentLaunchReceipt{}, &Error{Code: CodePoolRoutingInvalid}
		}
		return profile.adapter.Launch(ctx, request)
	}
	receipt, launchErr := seleccionado.adapter.Launch(ctx, request)
	if err := pool.finishSelection(request.ExecutionRef, seleccionado, launchErr == nil); err != nil {
		return ports.AgentLaunchReceipt{}, errors.Join(err, launchErr)
	}
	return receipt, launchErr
}

func (pool *Pool) perfilColocacion(referencia ports.AgentPlacementRef) *poolProfile {
	for _, perfil := range pool.profiles {
		if referencia.String() == "placement:"+perfil.ref {
			return perfil
		}
	}
	return nil
}

func (pool *Pool) Observe(
	ctx context.Context,
	executionRef goal.ExecutionRef,
) (ports.AgentObservation, error) {
	if executionRef.String() == "" {
		return ports.AgentObservation{}, &ports.AgentContractError{Code: "agent.execution_ref_required"}
	}
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	defer end()
	profile, _, err := pool.route(operationContext, executionRef, false)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	return profile.adapter.Observe(operationContext, executionRef)
}

func (pool *Pool) ControlCapabilities(
	ctx context.Context,
) (ports.AgentControlCapabilities, error) {
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return ports.AgentControlCapabilities{}, err
	}
	defer end()
	var result ports.AgentControlCapabilities
	for index, profile := range pool.profiles {
		capabilities, err := profile.adapter.ControlCapabilities(operationContext)
		if err != nil {
			return ports.AgentControlCapabilities{}, err
		}
		if index == 0 {
			result = capabilities
		} else if result != capabilities {
			return ports.AgentControlCapabilities{}, &Error{Code: CodePoolConfigInvalid}
		}
	}
	return result, nil
}

func (pool *Pool) Stop(
	ctx context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	if err := ports.ValidateAgentStopRequest(request); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	defer end()
	profile, _, err := pool.route(operationContext, request.ExecutionRef, false)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	return profile.adapter.Stop(operationContext, request)
}

func (pool *Pool) route(
	ctx context.Context,
	executionRef goal.ExecutionRef,
	allowSelection bool,
) (*poolProfile, bool, error) {
	key := executionRef.String()
	for {
		pool.mu.Lock()
		if pool.closed {
			pool.mu.Unlock()
			return nil, false, &Error{Code: CodeUnavailable}
		}
		if existing := pool.routes[key]; existing != nil && existing.selecting != nil {
			selecting := existing.selecting
			pool.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err()
			case <-selecting:
			}
			continue
		}
		scanned, found, err := pool.scanExecutionRouteLocked(executionRef)
		if err != nil {
			pool.mu.Unlock()
			return nil, false, err
		}
		if found {
			pool.routes[key] = &poolRoute{profile: scanned}
			pool.mu.Unlock()
			return scanned, false, nil
		}
		delete(pool.routes, key)
		if !allowSelection {
			pool.mu.Unlock()
			return nil, false, &Error{Code: CodeExecutionNotFound}
		}
		active, err := pool.aggregateActiveCountLocked()
		if err != nil {
			pool.mu.Unlock()
			return nil, false, err
		}
		if active >= pool.aggregateCapacity {
			pool.mu.Unlock()
			return nil, false, &Error{
				Code: CodeCapacityUnavailable, TemporaryFailure: true,
			}
		}
		pool.routes[key] = &poolRoute{selecting: make(chan struct{})}
		pool.mu.Unlock()
		return nil, true, nil
	}
}

// aggregateActiveCountLocked derives aggregate occupancy from the same
// request/terminal journals used for routing. A selecting route is the only
// in-memory reservation and exists solely across the pre-journal launch
// frontier. Ambiguous or corrupt durable state never frees capacity.
func (pool *Pool) aggregateActiveCountLocked() (int, error) {
	active := 0
	for key, route := range pool.routes {
		if route == nil {
			return 0, &Error{Code: CodePoolRoutingInvalid}
		}
		if route.selecting != nil {
			active++
			continue
		}
		if route.profile == nil {
			return 0, &Error{Code: CodePoolRoutingInvalid}
		}
		executionRef, err := goal.NewExecutionRef(key)
		if err != nil || executionRef.String() != key {
			return 0, &Error{Code: CodePoolRoutingInvalid, Cause: err}
		}
		record, runPath, found, err := route.profile.adapter.loadLaunchRecord(executionRef)
		if err != nil {
			return 0, err
		}
		if !found {
			return 0, &Error{Code: CodePoolRoutingInvalid}
		}
		_, terminal, err := route.profile.adapter.loadCausalTerminal(
			runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes,
		)
		if err != nil {
			return 0, err
		}
		if !terminal {
			active++
		}
	}
	return active, nil
}

func (pool *Pool) scanExecutionRouteLocked(
	executionRef goal.ExecutionRef,
) (*poolProfile, bool, error) {
	runPath := executionPath(executionRef)
	var owner *poolProfile
	for _, profile := range pool.profiles {
		if err := profile.adapter.validatePrivateDirectory(runPath); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, false, err
		}
		if owner != nil && owner != profile {
			return nil, false, &Error{Code: CodePoolRoutingInvalid}
		}
		owner = profile
		record, found, err := profile.adapter.readLaunchRecord(runPath)
		if err != nil {
			return nil, false, err
		}
		if !found {
			return nil, false, &Error{Code: CodePoolRoutingInvalid}
		}
		if record.ExecutionRef != executionRef.String() ||
			profile.adapter.validateAccountProfileBinding(record) != nil {
			return nil, false, &Error{Code: CodePoolRoutingInvalid}
		}
	}
	return owner, owner != nil, nil
}

func (pool *Pool) finishSelection(
	executionRef goal.ExecutionRef,
	expected *poolProfile,
	requireJournal bool,
) error {
	key := executionRef.String()
	pool.mu.Lock()
	defer pool.mu.Unlock()
	selection := pool.routes[key]
	if pool.closed {
		if selection != nil && selection.selecting != nil {
			close(selection.selecting)
		}
		delete(pool.routes, key)
		return nil
	}
	owner, found, err := pool.scanExecutionRouteLocked(executionRef)
	if err == nil && requireJournal && !found {
		err = &Error{Code: CodePoolRoutingInvalid}
	}
	if err == nil && found && owner != expected {
		err = &Error{Code: CodePoolRoutingInvalid}
	}
	if selection != nil && selection.selecting != nil {
		close(selection.selecting)
	}
	if found {
		pool.routes[key] = &poolRoute{profile: owner}
	} else {
		delete(pool.routes, key)
	}
	return err
}

func (pool *Pool) BindSessionResolver(resolver SessionResolver) error {
	if resolver == nil {
		return &Error{Code: CodeSessionInvalid}
	}
	_, end, err := pool.beginOperation(context.Background())
	if err != nil {
		return err
	}
	defer end()
	errs := make([]error, 0, len(pool.profiles))
	for _, profile := range pool.profiles {
		errs = append(errs, profile.adapter.BindSessionResolver(resolver))
	}
	return errors.Join(errs...)
}

func (pool *Pool) BindRuntimeScope(ctx context.Context, scope string) error {
	operationContext, end, err := pool.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer end()
	errs := make([]error, 0, len(pool.profiles))
	for _, profile := range pool.profiles {
		errs = append(errs, profile.adapter.BindRuntimeScope(operationContext, scope))
	}
	return errors.Join(errs...)
}

func (pool *Pool) BindWorkspacePathResolver(resolver WorkspacePathResolver) error {
	if resolver == nil {
		return &Error{Code: CodeWorkspaceResolverInvalid}
	}
	_, end, err := pool.beginOperation(context.Background())
	if err != nil {
		return err
	}
	defer end()
	errs := make([]error, 0, len(pool.profiles))
	for _, profile := range pool.profiles {
		errs = append(errs, profile.adapter.BindWorkspacePathResolver(resolver))
	}
	return errors.Join(errs...)
}

func (pool *Pool) beginOperation(
	ctx context.Context,
) (context.Context, func(), error) {
	if pool == nil {
		return nil, nil, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return nil, nil, &Error{Code: CodeUnavailable}
	}
	pool.operations.Add(1)
	pool.mu.Unlock()

	operationContext, cancel := context.WithCancelCause(ctx)
	stopLifecycle := context.AfterFunc(pool.lifecycle, func() {
		cancel(errPoolShutdown)
	})
	var once sync.Once
	end := func() {
		once.Do(func() {
			stopLifecycle()
			cancel(nil)
			pool.operations.Done()
		})
	}
	return operationContext, end, nil
}

func (pool *Pool) Shutdown(ctx context.Context) error {
	if pool == nil {
		return nil
	}
	pool.cancelLifecycle(errPoolShutdown)
	pool.mu.Lock()
	pool.closed = true
	pool.shutdownOnce.Do(func() {
		go pool.finalizeShutdown()
	})
	done := pool.shutdownDone
	pool.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return pool.shutdownErr
	}
}

func (pool *Pool) finalizeShutdown() {
	errs := make([]error, len(pool.profiles))
	var shutdowns sync.WaitGroup
	shutdowns.Add(len(pool.profiles))
	for index, profile := range pool.profiles {
		go func(index int, adapter *Adapter) {
			defer shutdowns.Done()
			// Adapter.Shutdown may return its caller timeout while its unique
			// finalizer continues waiting for operation owners. Pool must not
			// publish completion until that durable/process cleanup is real.
			_ = adapter.Close()
			<-adapter.shutdownDone
			adapter.mu.Lock()
			errs[index] = adapter.shutdownErr
			adapter.mu.Unlock()
		}(index, profile.adapter)
	}
	// Child lifecycles start first so an Adapter operation blocked in external
	// authority receives its own shutdown cancellation. Pool operations then
	// drain without replacing the callerContext retained by accepted runs.
	pool.operations.Wait()
	shutdowns.Wait()
	pool.mu.Lock()
	pool.shutdownErr = errors.Join(errs...)
	close(pool.shutdownDone)
	pool.mu.Unlock()
}

func (pool *Pool) Close() error {
	return pool.Shutdown(context.Background())
}

var _ application.AgentLauncher = (*Pool)(nil)
var _ application.AgentObserver = (*Pool)(nil)
var _ application.AgentController = (*Pool)(nil)
