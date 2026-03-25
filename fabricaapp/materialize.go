package fabricaapp

import (
	"fmt"
	"strings"

	"orquesta/db"
)

type MaterializeStore interface {
	CreateTask(t *db.Tarea) (int64, error)
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
			Prioridad:    item.Prioridad,
			Dependencias: deps,
			CreadoPor:    actor,
		})
		if err != nil {
			return MaterializeResult{}, err
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

type DBStore struct{}

func (DBStore) CreateTask(t *db.Tarea) (int64, error) {
	return db.CrearTarea(t)
}

func (DBStore) MoveTaskToBacklog(id int64) error {
	return db.EnviarTareaABacklog(id)
}

func (DBStore) DefineContract(id int64, actor string) error {
	return db.DefinirContrato(id, actor)
}
