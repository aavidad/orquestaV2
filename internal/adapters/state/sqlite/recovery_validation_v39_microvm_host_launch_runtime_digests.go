package sqlite

import (
	"context"
	"database/sql"
	"errors"
)

func validateRecoveryV39MicroVMHostLaunchRuntimeDigests(ctx context.Context, tx *sql.Tx) error {
	var epoch, legacyCount, exemptions int
	if err := tx.QueryRowContext(ctx, `SELECT schema_epoch,legacy_authority_count FROM microvm_host_launch_runtime_digest_epoch WHERE singleton=1`).Scan(&epoch, &legacyCount); err != nil || epoch != 39 {
		return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM microvm_host_launch_runtime_digest_legacy_exemptions`).Scan(&exemptions); err != nil || exemptions != legacyCount {
		return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT execution_ref,action_fence FROM microvm_host_launch_authorities ORDER BY execution_ref,action_fence`)
	if err != nil {
		return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
	}
	defer rows.Close()
	for rows.Next() {
		var executionRef string
		var fence int64
		if err := rows.Scan(&executionRef, &fence); err != nil {
			return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
		}
		if fence <= 0 {
			return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(nil)
		}
		var legacy, runtime int
		if err := tx.QueryRowContext(ctx, `SELECT
EXISTS(SELECT 1 FROM microvm_host_launch_runtime_digest_legacy_exemptions WHERE execution_ref=? AND action_fence=?),
EXISTS(SELECT 1 FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?)`,
			executionRef, fence, executionRef, fence).Scan(&legacy, &runtime); err != nil || legacy+runtime != 1 {
			return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
		}
		if runtime == 1 {
			var valid int
			if err := tx.QueryRowContext(ctx, `SELECT
length(plan_sha256)=64 AND plan_sha256 NOT GLOB '*[^0-9a-f]*' AND
length(concession_sha256)=64 AND concession_sha256 NOT GLOB '*[^0-9a-f]*' AND
length(kernel_sha256)=64 AND kernel_sha256 NOT GLOB '*[^0-9a-f]*' AND
length(initramfs_sha256)=64 AND initramfs_sha256 NOT GLOB '*[^0-9a-f]*' AND
length(profile_sha256)=64 AND profile_sha256 NOT GLOB '*[^0-9a-f]*'
FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?`, executionRef, fence).Scan(&valid); err != nil || valid != 1 {
				return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(err)
	}
	return nil
}

func invalidRecoveryV39MicroVMHostLaunchRuntimeDigests(cause error) error {
	return invalid(errors.Join(cause, errors.New("sqlite.recovery_v39_microvm_host_launch_runtime_digests_invalid")))
}
