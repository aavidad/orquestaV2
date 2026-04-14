/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package tareasapp

import (
	"orquesta/db"
	"orquesta/tareaspolicy"
)

func validateTaskDependencies(store Store, task *db.Tarea) error {
	if store == nil || task == nil {
		return nil
	}
	return tareaspolicy.ValidateDependencies(&tareaspolicy.TaskSnapshot{
		ID:            task.ID,
		Title:         task.Titulo,
		DependencyIDs: task.Dependencias,
	}, func(depID int64) (*tareaspolicy.TaskSnapshot, error) {
		dep, err := store.GetTask(depID)
		if err != nil {
			return nil, err
		}
		if dep == nil {
			return nil, nil
		}
		return &tareaspolicy.TaskSnapshot{
			ID:              dep.ID,
			Title:           dep.Titulo,
			Status:          string(dep.Estado),
			ContractDefined: dep.ContratoDefinido,
		}, nil
	})
}
