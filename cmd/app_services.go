package cmd

import (
	"orquesta/agentesapp"
	"orquesta/configapp"
	"orquesta/lenguajeapp"
	"orquesta/progresoapp"
)

var agentesService = agentesapp.NewService(agentesapp.Repository{})
var configService = configapp.NewService(configapp.Repository{})
var lenguajeService = lenguajeapp.NewService(lenguajeapp.Repository{})
var progresoService = progresoapp.NewService(progresoapp.Repository{})
