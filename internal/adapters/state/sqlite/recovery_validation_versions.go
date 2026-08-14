package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

const (
	recoverySchemaV09                     = 5
	recoverySchemaV10                     = 6
	recoverySchemaV12                     = 7
	recoverySchemaV13                     = 8
	recoverySchemaV14                     = 9
	recoverySchemaV15                     = 10
	recoverySchemaV16                     = 11
	recoverySchemaV17                     = 12
	recoverySchemaV18                     = 13
	recoverySchemaV19                     = 14
	recoverySchemaV20                     = 15
	recoverySchemaV21                     = 16
	recoverySchemaV23Intake               = 17
	recoverySchemaV23Dossier              = 18
	recoverySchemaV23Confirmation         = 19
	recoverySchemaV23WizardGaps           = 20
	recoverySchemaV23                     = 21
	recoverySchemaV38Physical             = 22
	recoverySchemaV38Capacity             = 23
	recoverySchemaV38Claim                = 24
	recoverySchemaV38Environment          = 25
	recoverySchemaV38EnvironmentGate      = 26
	recoverySchemaV38AttemptLease         = 27
	recoverySchemaV38RecoveryClaim        = 28
	recoverySchemaV38PreservationRatchet  = 29
	recoverySchemaV38RecoveryRequeue      = 30
	recoverySchemaV38EgressAuthority      = 31
	recoverySchemaV38MicroVMHostLaunch    = 32
	recoverySchemaV38MicroVMHostSession   = 33
	recoverySchemaV38PhysicalManifest     = 34
	recoverySchemaV38EnvironmentLifecycle = 35
	recoverySchemaV23WizardGapsSnapshot   = 36
	recoverySchemaV38AgentProviderRequest = 37
	recoverySchemaV38AgentProviderStop    = 38
	recoverySchemaV38AgentProviderStopKey = 39
	recoverySchemaV38StopRecoveryClaim    = 40
	recoverySchemaV38StopRecoveryTerminal = 41
	recoverySchemaLatest                  = recoverySchemaV38StopRecoveryTerminal
)

func migrationSchemaRef(migrations []migration) string {
	hash := sha256.New()
	for _, migration := range migrations {
		fmt.Fprintf(hash, "%d\x00%s\x00%s\n", migration.version, migration.name, migration.checksum)
	}
	return schemaRefPrefix + hex.EncodeToString(hash.Sum(nil))
}

func recoveryMigrationPrefix(migrations []migration, version int) ([]migration, error) {
	if version != recoverySchemaV09 && version != recoverySchemaV10 &&
		version != recoverySchemaV12 && version != recoverySchemaV13 && version != recoverySchemaV14 &&
		version != recoverySchemaV15 && version != recoverySchemaV16 && version != recoverySchemaV17 &&
		version != recoverySchemaV18 && version != recoverySchemaV19 && version != recoverySchemaV20 &&
		version != recoverySchemaV21 && version != recoverySchemaV23Intake &&
		version != recoverySchemaV23Dossier &&
		version != recoverySchemaV23Confirmation &&
		version != recoverySchemaV23WizardGaps &&
		version != recoverySchemaV23 && version != recoverySchemaV38Physical &&
		version != recoverySchemaV38Capacity && version != recoverySchemaV38Claim &&
		version != recoverySchemaV38Environment && version != recoverySchemaV38EnvironmentGate &&
		version != recoverySchemaV38AttemptLease && version != recoverySchemaV38RecoveryClaim &&
		version != recoverySchemaV38PreservationRatchet && version != recoverySchemaV38RecoveryRequeue &&
		version != recoverySchemaV38EgressAuthority && version != recoverySchemaV38MicroVMHostLaunch &&
		version != recoverySchemaV38MicroVMHostSession && version != recoverySchemaV38PhysicalManifest &&
		version != recoverySchemaV38EnvironmentLifecycle && version != recoverySchemaV23WizardGapsSnapshot &&
		version != recoverySchemaV38AgentProviderRequest && version != recoverySchemaV38AgentProviderStop &&
		version != recoverySchemaV38AgentProviderStopKey && version != recoverySchemaV38StopRecoveryClaim &&
		version != recoverySchemaV38StopRecoveryTerminal {
		return nil, errors.New("sqlite.recovery_schema_version_invalid")
	}
	if version > len(migrations) || migrations[version-1].version != version {
		return nil, errors.New("sqlite.recovery_schema_version_unsupported")
	}
	return migrations[:version], nil
}

type canonicalRecoverySchemaResult struct {
	digest string
	err    error
}

var canonicalRecoverySchemas = struct {
	sync.Mutex
	byVersion map[int]canonicalRecoverySchemaResult
}{byVersion: make(map[int]canonicalRecoverySchemaResult)}

func canonicalSchemaInventoryDigest(version int) (string, error) {
	canonicalRecoverySchemas.Lock()
	defer canonicalRecoverySchemas.Unlock()
	if cached, ok := canonicalRecoverySchemas.byVersion[version]; ok {
		return cached.digest, cached.err
	}
	result := canonicalRecoverySchemaResult{}
	migrations, err := loadMigrations()
	if err != nil {
		result.err = err
	} else if prefix, prefixErr := recoveryMigrationPrefix(migrations, version); prefixErr != nil {
		result.err = prefixErr
	} else {
		database, openErr := sql.Open(driverName, ":memory:")
		if openErr != nil {
			result.err = openErr
		} else {
			database.SetMaxOpenConns(1)
			if applyErr := applyRecoveryMigrationPrefix(context.Background(), database, prefix); applyErr != nil {
				result.err = applyErr
			} else {
				result.digest, result.err = schemaInventoryDigest(context.Background(), database)
			}
			if closeErr := database.Close(); result.err == nil {
				result.err = closeErr
			}
		}
	}
	canonicalRecoverySchemas.byVersion[version] = result
	return result.digest, result.err
}

func applyRecoveryMigrationPrefix(ctx context.Context, database *sql.DB, migrations []migration) error {
	connection, err := database.Conn(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	transaction, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if _, _, err := applyMigrationSteps(ctx, transaction, migrations, 0); err != nil {
		return err
	}
	return transaction.Commit()
}
