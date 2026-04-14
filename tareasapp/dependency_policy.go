/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package tareasapp

import (
	"fmt"

	"orquesta/db"
)

func validateTaskDependencies(store Store, task *db.Tarea) error {
	if store == nil || task == nil {
		return nil
	}
	for _, depID := range task.Dependencias {
		dep, err := store.GetTask(depID)
		if err != nil {
			return fmt.Errorf("dependencia #%d no encontrada", depID)
		}
		if dep == nil {
			return fmt.Errorf("dependencia #%d no encontrada", depID)
		}
		if !dep.ContratoDefinido {
			if dep.Estado != db.TareaCompletada {
				return fmt.Errorf(
					"la tarea #%d depende de #%d ('%s') que aún no tiene contrato/interfaz definido ni está completada (OP-069): usa 'orquesta tarea contrato %d' y completa la tarea previa antes de tomarla",
					task.ID, depID, dep.Titulo, depID,
				)
			}
			return fmt.Errorf(
				"la tarea #%d depende de #%d ('%s') que aún no tiene contrato/interfaz definido (OP-069): usa 'orquesta tarea contrato %d' para registrarlo primero",
				task.ID, depID, dep.Titulo, depID,
			)
		}
		if dep.Estado != db.TareaCompletada {
			return fmt.Errorf(
				"la tarea #%d depende de #%d ('%s') que aún no está completada (OP-069): espera a que termine antes de tomar esta tarea",
				task.ID, depID, dep.Titulo,
			)
		}
	}
	return nil
}
