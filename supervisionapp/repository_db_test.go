package supervisionapp

import (
	"errors"
	"testing"

	"orquesta/db"
)

func TestRepositoryWithDefaultsCompletaOverridesParciales(t *testing.T) {
	customErr := errors.New("override get project")
	repo := Repository{
		getProject: func(ref string) (*db.Proyecto, error) {
			return nil, customErr
		},
	}

	filled := repo.withDefaults()
	if filled.getProject == nil {
		t.Fatal("getProject no debería quedar nil")
	}
	if filled.getProjectAutonomy == nil {
		t.Fatal("getProjectAutonomy no debería quedar nil")
	}
	if filled.listProjectAutonomy == nil {
		t.Fatal("listProjectAutonomy no debería quedar nil")
	}
	if filled.upsertProjectAutonomy == nil {
		t.Fatal("upsertProjectAutonomy no debería quedar nil")
	}
	if filled.registerAutonomyCycle == nil {
		t.Fatal("registerAutonomyCycle no debería quedar nil")
	}
	if filled.listAutonomyCycles == nil {
		t.Fatal("listAutonomyCycles no debería quedar nil")
	}
	if filled.markProjectAutonomySeen == nil {
		t.Fatal("markProjectAutonomySeen no debería quedar nil")
	}
	if filled.markProjectAutonomyReviewed == nil {
		t.Fatal("markProjectAutonomyReviewed no debería quedar nil")
	}

	if _, err := filled.getProject("demo"); !errors.Is(err, customErr) {
		t.Fatalf("el override parcial debe preservarse, err=%v", err)
	}
}
