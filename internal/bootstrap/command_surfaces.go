package bootstrap

import (
	"net/http"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
	"orquesta/internal/interfaces/httpapi"
	mcpinterface "orquesta/internal/interfaces/mcp"
)

const commandHTTPPrefix = "/api/v1/commands/"

type commandSurfaces struct {
	dispatcher  *commandcore.Dispatcher
	httpHandler http.Handler
	mcpHandler  http.Handler
}

func newCommandSurfaces(
	setup buildSetup,
	version string,
	orchestrator *application.Orchestrator,
	repository *statesqlite.Repository,
	execution commandcore.ExecutionAuthorityResolver,
) (commandSurfaces, error) {
	dispatcher, err := commandcore.NewDispatcher(orchestrator, repository, commandcore.APILimits{
		MaxRequestBytes: setup.snapshot.ServerMaxRequestBytes(),
		MaxListLimit:    int(setup.snapshot.APIMaxListLimit()),
	}, execution)
	if err != nil {
		return commandSurfaces{}, err
	}
	provider := identity.ContextProvider{}
	httpHandler, err := httpapi.New(httpapi.Config{
		Dispatcher: dispatcher,
		Identity:   provider,
	})
	if err != nil {
		return commandSurfaces{}, err
	}
	mcpServer, err := mcpinterface.New(mcpinterface.Config{
		Dispatcher: dispatcher,
		Identity:   provider,
		Catalog:    setup.catalog,
		Locale:     setup.snapshot.APILocale(),
		Version:    version,
	})
	if err != nil {
		return commandSurfaces{}, err
	}
	return commandSurfaces{
		dispatcher: dispatcher, httpHandler: httpHandler, mcpHandler: mcpServer.Handler(),
	}, nil
}
