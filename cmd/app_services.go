package cmd

import (
	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/conectoresapp"
	"orquesta/configuracionapp"
	"orquesta/db"
	"orquesta/gobernanzaapp"
	"orquesta/lenguajeapp"
	"orquesta/microprogramacionapp"
	"orquesta/progresoapp"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
)

var capacidadService = capacidadapp.NewService(capacidadapp.Repository{})
var agentesService = agentesapp.NewService(agentesapp.Repository{}, capacidadService)
var configService = configuracionapp.NewService(configuracionapp.Repository{})
var conectoresService = conectoresapp.NewService(conectoresapp.Repository{})
var gobernanzaService = gobernanzaapp.NewService(gobernanzaapp.Repository{})
var lenguajeService = lenguajeapp.NewService(lenguajeapp.Repository{})
var microprogramacionService = microprogramacionapp.NewService(db.SqliteMicroprogramacionRepo{})
var progresoService = progresoapp.NewService(progresoapp.Repository{})
var reviewService = reviewapp.NewService(reviewapp.NewRepository())
var supervisionService = supervisionapp.NewService(supervisionapp.NewRepository())

func init() {
	capacidadService.SetPhaseProvider(progresoService)
}
