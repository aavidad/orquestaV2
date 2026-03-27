package cmd

import (
	"orquesta/agentesapp"
	"orquesta/conectoresapp"
	"orquesta/configapp"
	"orquesta/gobernanzaapp"
	"orquesta/lenguajeapp"
	"orquesta/progresoapp"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
)

var agentesService = agentesapp.NewService(agentesapp.Repository{})
var configService = configapp.NewService(configapp.Repository{})
var conectoresService = conectoresapp.NewService(conectoresapp.Repository{})
var gobernanzaService = gobernanzaapp.NewService(gobernanzaapp.Repository{})
var lenguajeService = lenguajeapp.NewService(lenguajeapp.Repository{})
var progresoService = progresoapp.NewService(progresoapp.Repository{})
var reviewService = reviewapp.NewService(reviewapp.NewRepository())
var supervisionService = supervisionapp.NewService(supervisionapp.NewRepository())
