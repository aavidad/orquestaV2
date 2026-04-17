package supervisionapp

import (
	"strings"
	"time"

	"orquesta/db"
)

// Repository es el adaptador de la capa de aplicación hacia db.
type Repository struct {
	getProject                  func(string) (*db.Proyecto, error)
	getProjectAutonomy          func(int64) (*db.ProyectoAutonomia, error)
	listProjectAutonomy         func(*bool) ([]*db.ProyectoAutonomia, error)
	upsertProjectAutonomy       func(*db.ProyectoAutonomia) (int64, error)
	registerAutonomyCycle       func(*db.AutonomiaCiclo) (int64, error)
	listAutonomyCycles          func(db.FiltroAutonomiaCiclos) ([]*db.AutonomiaCiclo, error)
	markProjectAutonomyReviewed func(int64, time.Time) error
	markProjectAutonomySeen     func(int64, time.Time) error
}

// NewRepository crea un repositorio por defecto contra la implementación de db.
func NewRepository() Repository {
	return Repository{
		getProject:                  db.GetProyecto,
		getProjectAutonomy:          db.GetProyectoAutonomia,
		listProjectAutonomy:         db.ListarProyectosAutonomia,
		upsertProjectAutonomy:       db.UpsertProyectoAutonomia,
		registerAutonomyCycle:       db.RegistrarAutonomiaCiclo,
		listAutonomyCycles:          db.ListarAutonomiaCiclos,
		markProjectAutonomySeen:     db.MarcarProyectoAutonomiaSupervisado,
		markProjectAutonomyReviewed: db.MarcarProyectoAutonomiaRevisado,
	}
}

func (r Repository) withDefaults() Repository {
	defaults := NewRepository()
	if r.getProject == nil {
		r.getProject = defaults.getProject
	}
	if r.getProjectAutonomy == nil {
		r.getProjectAutonomy = defaults.getProjectAutonomy
	}
	if r.listProjectAutonomy == nil {
		r.listProjectAutonomy = defaults.listProjectAutonomy
	}
	if r.upsertProjectAutonomy == nil {
		r.upsertProjectAutonomy = defaults.upsertProjectAutonomy
	}
	if r.registerAutonomyCycle == nil {
		r.registerAutonomyCycle = defaults.registerAutonomyCycle
	}
	if r.listAutonomyCycles == nil {
		r.listAutonomyCycles = defaults.listAutonomyCycles
	}
	if r.markProjectAutonomySeen == nil {
		r.markProjectAutonomySeen = defaults.markProjectAutonomySeen
	}
	if r.markProjectAutonomyReviewed == nil {
		r.markProjectAutonomyReviewed = defaults.markProjectAutonomyReviewed
	}
	return r
}

func (r Repository) GetProject(ref string) (*db.Proyecto, error) {
	repo := r.withDefaults()
	return repo.getProject(ref)
}

func (r Repository) GetProjectAutonomy(proyectoID int64) (*Policy, error) {
	repo := r.withDefaults()
	item, err := repo.getProjectAutonomy(proyectoID)
	if err != nil {
		return nil, err
	}
	return dbPolicyToPolicy(item), nil
}

func (r Repository) ListProjectAutonomy(enabled *bool) ([]*Policy, error) {
	repo := r.withDefaults()
	raw, err := repo.listProjectAutonomy(enabled)
	if err != nil {
		return nil, err
	}
	out := make([]*Policy, 0, len(raw))
	for _, item := range raw {
		c := dbPolicyToPolicy(item)
		if c != nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r Repository) UpsertProjectAutonomy(item *Policy) (int64, error) {
	repo := r.withDefaults()
	return repo.upsertProjectAutonomy(policyToDBPolicy(item))
}

func (r Repository) RegisterAutonomyCycle(item *Cycle) (int64, error) {
	repo := r.withDefaults()
	return repo.registerAutonomyCycle(cycleToDBCycle(item))
}

func (r Repository) ListAutonomyCycles(filter CycleFilter) ([]*Cycle, error) {
	repo := r.withDefaults()
	raw, err := repo.listAutonomyCycles(db.FiltroAutonomiaCiclos{
		ProyectoID: filter.ProyectoID,
		Kind:       filter.Kind,
		Agente:     filter.Agente,
		Limit:      filter.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*Cycle, 0, len(raw))
	for _, item := range raw {
		c := dbCycleToCycle(item)
		if c != nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r Repository) MarkProjectAutonomySupervised(proyectoID int64, when time.Time) error {
	repo := r.withDefaults()
	return repo.markProjectAutonomySeen(proyectoID, when)
}

func (r Repository) MarkProjectAutonomyReviewed(proyectoID int64, when time.Time) error {
	repo := r.withDefaults()
	return repo.markProjectAutonomyReviewed(proyectoID, when)
}

func dbPolicyToPolicy(item *db.ProyectoAutonomia) *Policy {
	if item == nil {
		return nil
	}
	return &Policy{
		ProyectoID:           item.ProyectoID,
		ProyectoSlug:         item.ProyectoSlug,
		Enabled:              item.Enabled,
		ObjetivoGeneral:      item.ObjetivoGeneral,
		DefinitionOfDoneJSON: item.DefinitionOfDoneJSON,
		MaxWorkers:           item.MaxWorkers,
		SupervisorAgente:     item.SupervisorAgente,
		ReviewerAgente:       item.ReviewerAgente,
		ReserveReviewer:      item.ReserveReviewer,
		ReserveSupervisor:    item.ReserveSupervisor,
		ReviewRequired:       item.ReviewRequired,
		AutoCreateTasks:      item.AutoCreateTasks,
		AutoCloseProject:     item.AutoCloseProject,
		EstadoAutonomia:      EstadoAutonomiaProyecto(item.EstadoAutonomia),
		LastSupervisionAt:    item.LastSupervisionAt,
		LastReviewAt:         item.LastReviewAt,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func policyToDBPolicy(item *Policy) *db.ProyectoAutonomia {
	if item == nil {
		return nil
	}
	out := &db.ProyectoAutonomia{
		ProyectoID:           item.ProyectoID,
		Enabled:              item.Enabled,
		ObjetivoGeneral:      item.ObjetivoGeneral,
		DefinitionOfDoneJSON: item.DefinitionOfDoneJSON,
		MaxWorkers:           item.MaxWorkers,
		SupervisorAgente:     item.SupervisorAgente,
		ReviewerAgente:       item.ReviewerAgente,
		ReserveReviewer:      item.ReserveReviewer,
		ReserveSupervisor:    item.ReserveSupervisor,
		ReviewRequired:       item.ReviewRequired,
		AutoCreateTasks:      item.AutoCreateTasks,
		AutoCloseProject:     item.AutoCloseProject,
		EstadoAutonomia:      db.EstadoAutonomiaProyecto(item.EstadoAutonomia),
		LastSupervisionAt:    item.LastSupervisionAt,
		LastReviewAt:         item.LastReviewAt,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
	trimIfNeeded(out)
	return out
}

func trimIfNeeded(item *db.ProyectoAutonomia) {
	if item == nil {
		return
	}
	item.ObjetivoGeneral = strings.TrimSpace(item.ObjetivoGeneral)
	item.DefinitionOfDoneJSON = strings.TrimSpace(item.DefinitionOfDoneJSON)
	item.SupervisorAgente = strings.TrimSpace(item.SupervisorAgente)
	item.ReviewerAgente = strings.TrimSpace(item.ReviewerAgente)
}

func dbCycleToCycle(item *db.AutonomiaCiclo) *Cycle {
	if item == nil {
		return nil
	}
	return &Cycle{
		ID:           item.ID,
		ProyectoID:   item.ProyectoID,
		ProyectoSlug: item.ProyectoSlug,
		Kind:         item.Kind,
		Agente:       item.Agente,
		SesionID:     item.SesionID,
		RuntimeID:    item.RuntimeID,
		InputJSON:    item.InputJSON,
		DecisionJSON: item.DecisionJSON,
		Resultado:    item.Resultado,
		CreatedAt:    item.CreatedAt,
	}
}

func cycleToDBCycle(item *Cycle) *db.AutonomiaCiclo {
	if item == nil {
		return nil
	}
	return &db.AutonomiaCiclo{
		ID:           item.ID,
		ProyectoID:   item.ProyectoID,
		Kind:         item.Kind,
		Agente:       item.Agente,
		SesionID:     item.SesionID,
		RuntimeID:    item.RuntimeID,
		InputJSON:    item.InputJSON,
		DecisionJSON: item.DecisionJSON,
		Resultado:    item.Resultado,
	}
}
