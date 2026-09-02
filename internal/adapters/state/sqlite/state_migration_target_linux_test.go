//go:build linux

package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateMigrationTargetRejectsHardlinkOwnerUnsafeParentAndAncestorSymlink(t *testing.T) {
	t.Run("hardlink", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
		if err := os.Link(path, path+".hardlink"); err != nil {
			t.Fatal(err)
		}
		if target, err := openStateMigrationTarget(path, uint32(os.Geteuid())); err == nil || target != nil ||
			!strings.Contains(sqliteTestErrorChain(err), "database_invalid") {
			t.Fatalf("hardlink target=%v error=%s", target, sqliteTestErrorChain(err))
		}
	})

	t.Run("owner", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
		if target, err := openStateMigrationTarget(path, uint32(os.Geteuid()+1)); err == nil || target != nil {
			t.Fatalf("owner target=%v error=%s", target, sqliteTestErrorChain(err))
		}
	})

	t.Run("unsafe-parent", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
		if err := os.Chmod(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if target, err := openStateMigrationTarget(path, uint32(os.Geteuid())); err == nil || target != nil ||
			!strings.Contains(sqliteTestErrorChain(err), "directory_invalid") {
			t.Fatalf("unsafe parent target=%v error=%s", target, sqliteTestErrorChain(err))
		}
	})

	t.Run("ancestor-symlink", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
		aliasRoot := t.TempDir()
		alias := filepath.Join(aliasRoot, "state-alias")
		if err := os.Symlink(filepath.Dir(path), alias); err != nil {
			t.Fatal(err)
		}
		aliasedPath := filepath.Join(alias, filepath.Base(path))
		if target, err := openStateMigrationTarget(aliasedPath, uint32(os.Geteuid())); err == nil || target != nil ||
			!strings.Contains(sqliteTestErrorChain(err), "directory_invalid") {
			t.Fatalf("ancestor symlink target=%v error=%s", target, sqliteTestErrorChain(err))
		}
	})
}

func TestStateMigrationTargetDetectsPathSubstitutionAfterOpen(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
	target, err := openStateMigrationTarget(path, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := os.Rename(path, path+".replaced"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := target.verify(); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "target_changed") {
		t.Fatalf("substitution error=%s", sqliteTestErrorChain(err))
	}
}
