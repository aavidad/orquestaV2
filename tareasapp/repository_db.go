package tareasapp

import "orquesta/db"

type Repository struct{}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}

func (Repository) GetTask(id int64) (*db.Tarea, error) {
	return db.GetTarea(id)
}

func (Repository) CreateTask(t *db.Tarea) (int64, error) {
	return db.CrearTarea(t)
}

func (Repository) TakeTask(id int64, agente string) error {
	return db.TomarTarea(id, agente)
}

func (Repository) StartTask(id int64, agente string) error {
	return db.IniciarTarea(id, agente)
}

func (Repository) CompleteTask(id int64, agente, commit string) error {
	return db.CompletarTarea(id, agente, commit)
}

func (Repository) BlockTask(id int64, agente, motivo string) error {
	return db.BloquearTarea(id, agente, motivo)
}

func (Repository) UnblockTask(id int64, agente, resolucion string) error {
	return db.DesbloquearTarea(id, agente, resolucion)
}

func (Repository) AnnotateTask(id int64, agente, nota string) error {
	return db.AnotarTarea(id, agente, nota)
}

func (Repository) MoveTaskToBacklog(id int64) error {
	return db.EnviarTareaABacklog(id)
}

func (Repository) ReassignTask(id int64, nuevoAgente string) error {
	return db.ReasignarTarea(id, nuevoAgente)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyectoPrepareLite(ref)
}

func (Repository) GetProposal(codigo string) (*db.Propuesta, error) {
	return db.GetPropuesta(codigo)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}
