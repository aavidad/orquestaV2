package orquestaweb

import (
	"context"
	"errors"

	orquestadirector "orquesta/modulos/orquesta-director"
)

const WebBootstrapProyectoErrClienteV0 = "bootstrap_cliente_error"

type BootstrapProyectoDesdeAppSpecClientV0 interface {
	BootstrapProyectoDesdeAppSpec(
		ctx context.Context,
		cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0,
	) (WebBootstrapProyectoViewModelV0, error)
}

type BootstrapProyectoDesdeAppSpecFuncV0 func(
	orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0,
) (orquestadirector.BootstrapProyectoDesdeAppSpecResultV0, error)

type LocalBootstrapProyectoDesdeAppSpecClientV0 struct {
	Bootstrap BootstrapProyectoDesdeAppSpecFuncV0
}

type WebBootstrapProyectoClientErrorV0 struct {
	Code string
}

func (err WebBootstrapProyectoClientErrorV0) Error() string {
	return err.Code
}

func NewLocalBootstrapProyectoDesdeAppSpecClientV0() LocalBootstrapProyectoDesdeAppSpecClientV0 {
	return LocalBootstrapProyectoDesdeAppSpecClientV0{
		Bootstrap: orquestadirector.BootstrapProyectoDesdeAppSpecV0,
	}
}

func (client LocalBootstrapProyectoDesdeAppSpecClientV0) BootstrapProyectoDesdeAppSpec(
	ctx context.Context,
	cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0,
) (WebBootstrapProyectoViewModelV0, error) {
	if err := ctx.Err(); err != nil {
		return WebBootstrapProyectoViewModelV0{}, WebBootstrapProyectoClientErrorV0{Code: WebBootstrapProyectoErrClienteV0}
	}
	bootstrap := client.Bootstrap
	if bootstrap == nil {
		bootstrap = orquestadirector.BootstrapProyectoDesdeAppSpecV0
	}
	result, err := bootstrap(cmd)
	if err != nil {
		if vm, ok := NewWebBootstrapProyectoErrorViewModelV0(err); ok {
			return vm, nil
		}
		return WebBootstrapProyectoViewModelV0{}, WebBootstrapProyectoClientErrorV0{Code: WebBootstrapProyectoErrClienteV0}
	}
	return NewWebBootstrapProyectoViewModelV0(result), nil
}

func IsWebBootstrapProyectoClientErrorCodeV0(err error, code string) bool {
	var clientErr WebBootstrapProyectoClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
