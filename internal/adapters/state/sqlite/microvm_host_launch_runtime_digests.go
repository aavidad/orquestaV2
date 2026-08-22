package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (repository *Repository) ResolveRuntime(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchRuntimeDigestsV1, error) {
	if ctx == nil || key.ActionFence > math.MaxInt64 || ports.ValidateMicroVMHostLaunchAuthorityKey(key) != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, invalid(errors.New("sqlite.microvm_host_launch_runtime_digests_key_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, err
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, microVMHostLaunchContextError(ctx, err)
	}
	defer transaction.Rollback()
	value, found, err := readMicroVMHostLaunchRuntimeDigests(ctx, transaction, key)
	if err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, microVMHostLaunchDatabaseError(ctx, err)
	}
	if !found {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, stateError(application.StateNotFound, errors.New("sqlite.microvm_host_launch_runtime_digests_not_found"))
	}
	if err := commit(transaction); err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, microVMHostLaunchContextError(ctx, err)
	}
	return value, nil
}

func insertMicroVMHostLaunchRuntimeDigests(
	ctx context.Context,
	transaction *sql.Tx,
	value ports.MicroVMHostLaunchRuntimeDigestsV1,
) error {
	_, err := transaction.ExecContext(ctx, `INSERT INTO microvm_host_launch_runtime_digests(
execution_ref,action_fence,plan_sha256,concession_sha256,kernel_sha256,initramfs_sha256,profile_sha256) VALUES(?,?,?,?,?,?,?)`,
		value.Key.RunRef.String(), value.Key.ActionFence, value.PlanSHA256, value.ConcessionSHA256,
		value.KernelSHA256, value.InitramfsSHA256, value.ProfileSHA256)
	if err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func readMicroVMHostLaunchRuntimeDigests(
	ctx context.Context,
	source queryer,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchRuntimeDigestsV1, bool, error) {
	var executionRef, plan, concession, kernel, initramfs, profile string
	var fence int64
	err := source.QueryRowContext(ctx, `SELECT execution_ref,action_fence,plan_sha256,concession_sha256,kernel_sha256,initramfs_sha256,profile_sha256
FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?`,
		key.RunRef.String(), key.ActionFence).Scan(&executionRef, &fence, &plan, &concession, &kernel, &initramfs, &profile)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, false, nil
	}
	if err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, false, mapDatabaseError(err)
	}
	runRef, refErr := goal.NewExecutionRef(executionRef)
	value := ports.MicroVMHostLaunchRuntimeDigestsV1{Key: ports.MicroVMHostLaunchAuthorityKey{RunRef: runRef, ActionFence: uint64(fence)}, PlanSHA256: plan, ConcessionSHA256: concession, KernelSHA256: kernel, InitramfsSHA256: initramfs, ProfileSHA256: profile}
	if refErr != nil || fence <= 0 || value.Key != key || ports.ValidateMicroVMHostLaunchRuntimeDigestsV1(value) != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, false, invalid(errors.New("sqlite.microvm_host_launch_runtime_digests_corrupt"))
	}
	return value, true, nil
}
