package progresoapp

import "orquesta/db"

type Repository struct{}

func (Repository) CalculateProjectSummary(proyecto string) (*ResumenProgresoProyecto, error) {
	item, err := db.CalcularResumenProgresoProyecto(proyecto)
	if err != nil || item == nil {
		return nil, err
	}
	return mapResumenProgreso(item), nil
}

func (Repository) ListProjectPhases(proyecto string) ([]*FaseProyecto, error) {
	items, err := db.ListarFasesProyecto(proyecto)
	if err != nil {
		return nil, err
	}
	out := make([]*FaseProyecto, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapFaseProyecto(item))
	}
	return out, nil
}

func (Repository) RegisterProjectPhase(fase *FaseProyecto) (int64, error) {
	return db.RegistrarFaseProyecto(mapDBFaseProyecto(fase))
}

func (Repository) GetProjectPhase(id int64) (*FaseProyecto, error) {
	item, err := db.GetFaseProyecto(id)
	if err != nil || item == nil {
		return nil, err
	}
	return mapFaseProyecto(item), nil
}

func (Repository) UpdateProjectPhase(fase *FaseProyecto) error {
	return db.ActualizarFaseProyecto(mapDBFaseProyecto(fase))
}

func (Repository) RegisterTaskProgress(avance *AvanceTarea) error {
	return db.RegistrarAvanceTarea(mapDBAvanceTarea(avance))
}

func mapFaseProyecto(in *db.FaseProyecto) *FaseProyecto {
	if in == nil {
		return nil
	}
	return &FaseProyecto{
		ID:          in.ID,
		Proyecto:    in.Proyecto,
		Nombre:      in.Nombre,
		Descripcion: in.Descripcion,
		Orden:       in.Orden,
		Peso:        in.Peso,
		Estado:      in.Estado,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
	}
}

func mapDBFaseProyecto(in *FaseProyecto) *db.FaseProyecto {
	if in == nil {
		return nil
	}
	return &db.FaseProyecto{
		ID:          in.ID,
		Proyecto:    in.Proyecto,
		Nombre:      in.Nombre,
		Descripcion: in.Descripcion,
		Orden:       in.Orden,
		Peso:        in.Peso,
		Estado:      in.Estado,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
	}
}

func mapDBAvanceTarea(in *AvanceTarea) *db.AvanceTarea {
	if in == nil {
		return nil
	}
	return &db.AvanceTarea{
		TareaID:        in.TareaID,
		Proyecto:       in.Proyecto,
		FaseID:         in.FaseID,
		ProgresoPct:    in.ProgresoPct,
		ActualizadoPor: in.ActualizadoPor,
		UpdatedAt:      in.UpdatedAt,
	}
}

func mapResumenProgreso(in *db.ResumenProgresoProyecto) *ResumenProgresoProyecto {
	if in == nil {
		return nil
	}
	out := &ResumenProgresoProyecto{
		Proyecto:          in.Proyecto,
		ProgresoPct:       in.ProgresoPct,
		TareasTotales:     in.TareasTotales,
		TareasCompletadas: in.TareasCompletadas,
		Fases:             make([]*FaseProgresoDetalle, 0, len(in.Fases)),
		TareasSinFase:     make([]*TareaProgresoDetalle, 0, len(in.TareasSinFase)),
	}
	for _, item := range in.Fases {
		if item == nil {
			continue
		}
		out.Fases = append(out.Fases, mapFaseProgresoDetalle(item))
	}
	for _, item := range in.TareasSinFase {
		if item == nil {
			continue
		}
		out.TareasSinFase = append(out.TareasSinFase, mapTareaProgresoDetalle(item))
	}
	return out
}

func mapFaseProgresoDetalle(in *db.FaseProgresoDetalle) *FaseProgresoDetalle {
	if in == nil {
		return nil
	}
	out := &FaseProgresoDetalle{
		Fase:              mapFaseProyecto(in.Fase),
		ProgresoPct:       in.ProgresoPct,
		Tareas:            make([]*TareaProgresoDetalle, 0, len(in.Tareas)),
		TareasTotales:     in.TareasTotales,
		TareasCompletadas: in.TareasCompletadas,
	}
	for _, item := range in.Tareas {
		if item == nil {
			continue
		}
		out.Tareas = append(out.Tareas, mapTareaProgresoDetalle(item))
	}
	return out
}

func mapTareaProgresoDetalle(in *db.TareaProgresoDetalle) *TareaProgresoDetalle {
	if in == nil {
		return nil
	}
	out := &TareaProgresoDetalle{
		FaseID:      in.FaseID,
		FaseNombre:  in.FaseNombre,
		Proyecto:    in.Proyecto,
		ProgresoPct: in.ProgresoPct,
		Actualizado: in.Actualizado,
		Manual:      in.Manual,
	}
	if in.Tarea != nil {
		out.TareaID = in.Tarea.ID
		out.TareaTitulo = in.Tarea.Titulo
		out.EstadoTarea = string(in.Tarea.Estado)
		out.ModuloTarea = in.Tarea.Modulo
		out.PrioridadRaw = string(in.Tarea.Prioridad)
	}
	return out
}
