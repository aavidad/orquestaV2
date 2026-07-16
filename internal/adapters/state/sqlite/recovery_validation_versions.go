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
	recoverySchemaV09 = 5
	recoverySchemaV10 = 6
	recoverySchemaV12 = 7
	recoverySchemaV13 = 8
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
		version != recoverySchemaV12 && version != recoverySchemaV13 {
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
