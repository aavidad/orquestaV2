package capacidadapp

import (
	"strings"

	"orquesta/db"
)

type Repository struct{}

func (Repository) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return db.ListarPoolsResumen(activo)
}

func (Repository) GetPool(slug string) (*db.PoolCapacidad, error) {
	return db.GetPool(slug)
}

func (Repository) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return db.ListarModelosPool(slug)
}

func (Repository) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return db.GuardarPool(pool)
}

func (Repository) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return db.GuardarPoolModelo(poolSlug, modelo)
}

func (Repository) SeedInitialModels() error {
	return db.SeedModelosIniciales()
}

func (Repository) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return db.ListarPoliticasModelo(scopeTipo, scopeRef, activa)
}

func (Repository) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return db.GuardarPoliticaModelo(policy)
}

func (Repository) SeedInitialModelPolicies() error {
	return db.SeedPoliticasModeloIniciales()
}

func (Repository) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return db.ResolverPoliticaModelo(input)
}

func (Repository) ListProjectTasks(proyecto string) ([]*db.Tarea, error) {
	proyecto = strings.TrimSpace(proyecto)
	if proyecto == "" {
		return nil, nil
	}
	item, err := db.GetProyecto(proyecto)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return db.ListarTareas(db.FiltroTareas{ProyectoID: &item.ID})
}

func (Repository) GetTask(id int64) (*db.Tarea, error) {
	return db.GetTarea(id)
}

func (Repository) TakeTask(id int64, agente string) error {
	return db.TomarTarea(id, agente)
}

func (Repository) ReassignTask(id int64, agente string) error {
	return db.ReasignarTarea(id, agente)
}

func (Repository) StartTask(id int64, agente string) error {
	return db.IniciarTarea(id, agente)
}

func (Repository) ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error) {
	return db.ListarFasesProyecto(strings.TrimSpace(proyecto))
}

func (Repository) RegisterProjectPhase(fase *db.FaseProyecto) (int64, error) {
	return db.RegistrarFaseProyecto(fase)
}

func (Repository) GetProjectPhase(id int64) (*db.FaseProyecto, error) {
	return db.GetFaseProyecto(id)
}

func (Repository) UpdateProjectPhase(fase *db.FaseProyecto) error {
	return db.ActualizarFaseProyecto(fase)
}
