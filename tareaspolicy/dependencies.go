/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package tareaspolicy

import "fmt"

const StatusCompleted = "completada"

type TaskSnapshot struct {
	ID              int64
	Title           string
	Status          string
	ContractDefined bool
	DependencyIDs   []int64
}

type LookupFunc func(id int64) (*TaskSnapshot, error)

func ValidateDependencies(task *TaskSnapshot, lookup LookupFunc) error {
	if task == nil || lookup == nil {
		return nil
	}
	for _, depID := range task.DependencyIDs {
		dep, err := lookup(depID)
		if err != nil {
			return fmt.Errorf("dependencia #%d no encontrada", depID)
		}
		if dep == nil {
			return fmt.Errorf("dependencia #%d no encontrada", depID)
		}
		if !dep.ContractDefined {
			if dep.Status != StatusCompleted {
				return fmt.Errorf(
					"la tarea #%d depende de #%d ('%s') que aún no tiene contrato/interfaz definido ni está completada (OP-069): usa 'orquesta tarea contrato %d' y completa la tarea previa antes de tomarla",
					task.ID, depID, dep.Title, depID,
				)
			}
			return fmt.Errorf(
				"la tarea #%d depende de #%d ('%s') que aún no tiene contrato/interfaz definido (OP-069): usa 'orquesta tarea contrato %d' para registrarlo primero",
				task.ID, depID, dep.Title, depID,
			)
		}
		if dep.Status != StatusCompleted {
			return fmt.Errorf(
				"la tarea #%d depende de #%d ('%s') que aún no está completada (OP-069): espera a que termine antes de tomar esta tarea",
				task.ID, depID, dep.Title,
			)
		}
	}
	return nil
}
