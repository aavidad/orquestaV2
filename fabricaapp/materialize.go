package fabricaapp

import (
	"fmt"
	"strings"

	"orquesta/db"
)

type MaterializeStore interface {
	CreateTask(t *db.Tarea) (int64, error)
	GetExistingTaskID(projectID int64, blueprintKey string) int64
	MoveTaskToBacklog(id int64) error
	DefineContract(id int64, actor string) error
}

type MaterializeResult struct {
	Created int
	Backlog int
	TaskIDs map[string]int64
}

func Materialize(store MaterializeStore, projectID int64, actor string, plan GenerationResult) (MaterializeResult, error) {
	if store == nil {
		return MaterializeResult{}, fmt.Errorf("store obligatorio")
	}
	if projectID <= 0 {
		return MaterializeResult{}, fmt.Errorf("projectID obligatorio")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "alberto"
	}

	result := MaterializeResult{
		TaskIDs: make(map[string]int64, len(plan.Tasks)),
	}
	for _, item := range plan.Tasks {
		deps, err := resolveBlueprintDependencyIDs(result.TaskIDs, item.Dependencias)
		if err != nil {
			return MaterializeResult{}, err
		}
		id, err := store.CreateTask(&db.Tarea{
			Titulo:       item.Titulo,
			Descripcion:  item.Descripcion,
			ProyectoID:   &projectID,
			Modulo:       item.Modulo,
			Prioridad:    db.PrioridadTarea(item.Prioridad),
			Dependencias: deps,
			CreadoPor:    actor,
			Notas:        buildSOPNotes(item),
			BlueprintKey: item.Key,
		})
		if err != nil {
			return MaterializeResult{}, err
		}
		if id == 0 {
			// tarea ya existía con esta blueprint_key — recuperar ID para que las dependencias se resuelvan
			id = store.GetExistingTaskID(projectID, item.Key)
			if id > 0 {
				result.TaskIDs[item.Key] = id
			}
			continue
		}
		result.TaskIDs[item.Key] = id
		result.Created++
		if len(deps) > 0 {
			if err := store.MoveTaskToBacklog(id); err != nil {
				return MaterializeResult{}, err
			}
			result.Backlog++
		}
		if item.ContratoDefinido {
			if err := store.DefineContract(id, actor); err != nil {
				return MaterializeResult{}, err
			}
		}
	}
	return result, nil
}

func resolveBlueprintDependencyIDs(ids map[string]int64, keys []string) ([]int64, error) {
	out := make([]int64, 0, len(keys))
	for _, key := range keys {
		id, ok := ids[strings.TrimSpace(key)]
		if !ok || id <= 0 {
			return nil, fmt.Errorf("dependencia blueprint no resuelta: %s", key)
		}
		out = append(out, id)
	}
	return out, nil
}

func buildSOPNotes(task BlueprintTask) string {
	partes := make([]string, 0, 6)
	if task.RolSugerido != "" {
		partes = append(partes, "SOP rol sugerido: "+task.RolSugerido)
	}
	if task.Fase != "" {
		partes = append(partes, "SOP fase: "+task.Fase)
	}
	if task.Entregable != "" {
		partes = append(partes, "SOP entregable: "+task.Entregable)
	}
	if len(task.CriteriosCierre) > 0 {
		partes = append(partes, "SOP criterios de salida:")
		for idx, criterio := range task.CriteriosCierre {
			partes = append(partes, fmt.Sprintf("%d. %s", idx+1, strings.TrimSpace(criterio)))
		}
	}
	return strings.Join(partes, "\n")
}

type DBStore struct{}

func (DBStore) CreateTask(t *db.Tarea) (int64, error) {
	return db.CrearTarea(t)
}

func (DBStore) GetExistingTaskID(projectID int64, blueprintKey string) int64 {
	return db.GetTareaIDBlueprintKey(projectID, blueprintKey)
}

func (DBStore) MoveTaskToBacklog(id int64) error {
	return db.EnviarTareaABacklog(id)
}

func (DBStore) DefineContract(id int64, actor string) error {
	return db.DefinirContrato(id, actor)
}
