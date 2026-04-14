package tareaspolicy

import (
	"strings"
	"testing"
)

func TestValidateDependenciesRequiresCompletedDependencies(t *testing.T) {
	task := &TaskSnapshot{ID: 7, Title: "dependiente", DependencyIDs: []int64{3}}

	err := ValidateDependencies(task, func(id int64) (*TaskSnapshot, error) {
		return &TaskSnapshot{ID: id, Title: "dep", Status: "asignada", ContractDefined: true}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "aún no está completada") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestValidateDependenciesRequiresContract(t *testing.T) {
	task := &TaskSnapshot{ID: 8, Title: "dependiente", DependencyIDs: []int64{4}}

	err := ValidateDependencies(task, func(id int64) (*TaskSnapshot, error) {
		return &TaskSnapshot{ID: id, Title: "dep", Status: StatusCompleted, ContractDefined: false}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "no tiene contrato/interfaz definido") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestValidateDependenciesAllowsCompletedDependenciesWithContract(t *testing.T) {
	task := &TaskSnapshot{ID: 9, Title: "dependiente", DependencyIDs: []int64{5}}

	if err := ValidateDependencies(task, func(id int64) (*TaskSnapshot, error) {
		return &TaskSnapshot{ID: id, Title: "dep", Status: StatusCompleted, ContractDefined: true}, nil
	}); err != nil {
		t.Fatalf("ValidateDependencies: %v", err)
	}
}
