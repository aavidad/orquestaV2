// Package mcpinterface exposes the application through the official MCP Go
// SDK. It owns transport concerns only; lifecycle transitions remain in the
// application orchestrator.
package mcpinterface

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
)

type Config struct {
	Dispatcher commandcore.Executor
	Identity   identity.Provider
	Catalog    *i18n.Catalog
	Locale     string
	Version    string
}

type Interface struct {
	server  *sdkmcp.Server
	handler http.Handler
}

func New(config Config) (*Interface, error) {
	switch {
	case config.Dispatcher == nil || !config.Dispatcher.Limits().Valid():
		return nil, errors.New("mcp.dispatcher_required")
	case config.Identity == nil:
		return nil, errors.New("mcp.identity_required")
	case config.Catalog == nil:
		return nil, errors.New("mcp.catalog_required")
	case strings.TrimSpace(config.Locale) == "" || strings.TrimSpace(config.Locale) != config.Locale:
		return nil, errors.New("mcp.locale_invalid")
	case strings.TrimSpace(config.Version) == "" || strings.TrimSpace(config.Version) != config.Version:
		return nil, errors.New("mcp.version_invalid")
	}
	if _, err := config.Catalog.Resolve(config.Locale); err != nil {
		return nil, fmt.Errorf("mcp.locale_invalid: %w", err)
	}
	instructions, err := config.Catalog.Text(config.Locale, "server.instructions")
	if err != nil {
		return nil, fmt.Errorf("mcp.instructions_unavailable: %w", err)
	}

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "orquesta", Version: config.Version},
		&sdkmcp.ServerOptions{
			Instructions: instructions,
			PageSize:     config.Dispatcher.Limits().MaxListLimit,
			Capabilities: &sdkmcp.ServerCapabilities{},
		},
	)
	if err := RegisterCommandTools(server, config.Dispatcher, config.Identity, config.Catalog, config.Locale); err != nil {
		return nil, err
	}
	result := &Interface{server: server}

	streamable := sdkmcp.NewStreamableHTTPHandler(
		func(*http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			DisableLocalhostProtection: false,
		},
	)
	limited := requestBodyLimit(config.Dispatcher.Limits().MaxRequestBytes, streamable)
	result.handler = http.NewCrossOriginProtection().Handler(limited)
	return result, nil
}

func (server *Interface) Server() *sdkmcp.Server {
	if server == nil {
		return nil
	}
	return server.server
}

func (server *Interface) Handler() http.Handler {
	if server == nil {
		return nil
	}
	return server.handler
}

func requestBodyLimit(maximum int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Body == nil || request.ContentLength == 0 {
			next.ServeHTTP(writer, request)
			return
		}
		if request.ContentLength > maximum {
			_ = request.Body.Close()
			writer.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		payload, err := io.ReadAll(io.LimitReader(request.Body, maximum+1))
		_ = request.Body.Close()
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if int64(len(payload)) > maximum {
			writer.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(payload))
		request.ContentLength = int64(len(payload))
		next.ServeHTTP(writer, request)
	})
}
