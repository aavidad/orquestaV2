// Package mcpinterface exposes the application through the official MCP Go
// SDK. It owns transport concerns only; lifecycle transitions remain in the
// application orchestrator.
package mcpinterface

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
)

type Config struct {
	Orchestrator    *application.Orchestrator
	Identity        identity.Provider
	Catalog         *i18n.Catalog
	Locale          string
	MaxListLimit    int
	MaxRequestBytes int64
	Version         string
}

type Interface struct {
	orchestrator *application.Orchestrator
	identity     identity.Provider
	catalog      *i18n.Catalog
	locale       string
	maxListLimit int
	version      string
	server       *sdkmcp.Server
	handler      http.Handler
}

func New(config Config) (*Interface, error) {
	switch {
	case config.Orchestrator == nil:
		return nil, errors.New("mcp.orchestrator_required")
	case config.Identity == nil:
		return nil, errors.New("mcp.identity_required")
	case config.Catalog == nil:
		return nil, errors.New("mcp.catalog_required")
	case strings.TrimSpace(config.Locale) == "" || strings.TrimSpace(config.Locale) != config.Locale:
		return nil, errors.New("mcp.locale_invalid")
	case config.MaxListLimit <= 0:
		return nil, errors.New("mcp.max_list_limit_invalid")
	case config.MaxRequestBytes <= 0 || config.MaxRequestBytes == math.MaxInt64:
		return nil, errors.New("mcp.max_request_bytes_invalid")
	case strings.TrimSpace(config.Version) == "" || strings.TrimSpace(config.Version) != config.Version:
		return nil, errors.New("mcp.version_invalid")
	}

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "orquesta", Version: config.Version},
		&sdkmcp.ServerOptions{
			Instructions: config.Catalog.Text(config.Locale, "server.instructions"),
			PageSize:     config.MaxListLimit,
			Capabilities: &sdkmcp.ServerCapabilities{},
		},
	)
	result := &Interface{
		orchestrator: config.Orchestrator,
		identity:     config.Identity,
		catalog:      config.Catalog,
		locale:       config.Locale,
		maxListLimit: config.MaxListLimit,
		version:      config.Version,
		server:       server,
	}
	result.registerTools()

	streamable := sdkmcp.NewStreamableHTTPHandler(
		func(*http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			DisableLocalhostProtection: false,
		},
	)
	limited := requestBodyLimit(config.MaxRequestBytes, streamable)
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
			http.Error(writer, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
			return
		}
		payload, err := io.ReadAll(io.LimitReader(request.Body, maximum+1))
		_ = request.Body.Close()
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if int64(len(payload)) > maximum {
			http.Error(writer, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(payload))
		request.ContentLength = int64(len(payload))
		next.ServeHTTP(writer, request)
	})
}
