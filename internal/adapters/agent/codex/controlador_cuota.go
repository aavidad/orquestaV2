package codex

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"orquesta/internal/adapters/agent/codex/appserver"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type SumideroObservacionCuota func(context.Context, application.AgentQuotaObservation, []byte) error

type ConfiguracionControladorCuota struct {
	Comando, DirectorioCuenta             string
	Entorno                               map[string]string
	ReferenciaColocacion                  ports.AgentPlacementRef
	MaximoBytesTrama                      int
	VigenciaObservacion, DemoraReconexion time.Duration
	Ahora                                 func() time.Time
	Sumidero                              SumideroObservacionCuota
	argumentosParaPruebas                 []string
}

type ControladorCuota struct {
	configuracion ConfiguracionControladorCuota
	comando       string
	entorno       []string
	ciclo         context.Context
	cancelar      context.CancelFunc
	terminado     chan struct{}
	inicial       chan error
	inicialUnaVez sync.Once
	cierreUnaVez  sync.Once
	ultima        *application.AgentQuotaObservation
}

func (adaptador *Adapter) IniciarControladoresCuota(
	ctx context.Context, parametros application.ConfiguracionControladoresCuotaAgente,
) ([]application.ControladorCuotaAgente, error) {
	if adaptador == nil || adaptador.accountHomePath == "" {
		return nil, nil
	}
	colocacion, err := ports.NewAgentPlacementRef("placement:" + adaptador.accountProfileBindingRef)
	if err != nil {
		return nil, err
	}
	controlador, err := IniciarControladorCuota(ctx, ConfiguracionControladorCuota{
		Comando: adaptador.command, DirectorioCuenta: adaptador.accountHomePath,
		Entorno: adaptador.config.Environment, ReferenciaColocacion: colocacion,
		MaximoBytesTrama:    adaptador.config.AppServerMaxFrameBytes,
		VigenciaObservacion: parametros.VigenciaObservacion, DemoraReconexion: parametros.DemoraReconexion,
		Ahora: parametros.Ahora, Sumidero: parametros.Sumidero,
	})
	if err != nil {
		return nil, err
	}
	return []application.ControladorCuotaAgente{controlador}, nil
}

func (pool *Pool) IniciarControladoresCuota(
	ctx context.Context, parametros application.ConfiguracionControladoresCuotaAgente,
) ([]application.ControladorCuotaAgente, error) {
	if pool == nil {
		return nil, &Error{Code: CodeStateInvalid}
	}
	var controladores []application.ControladorCuotaAgente
	for _, perfil := range pool.profiles {
		nuevos, err := perfil.adapter.IniciarControladoresCuota(ctx, parametros)
		if err != nil {
			for _, controlador := range controladores {
				_ = controlador.Cerrar(ctx)
			}
			return nil, err
		}
		controladores = append(controladores, nuevos...)
	}
	return controladores, nil
}

func IniciarControladorCuota(padre context.Context, configuracion ConfiguracionControladorCuota) (*ControladorCuota, error) {
	if padre == nil || configuracion.ReferenciaColocacion.String() == "" || configuracion.MaximoBytesTrama < 1 ||
		configuracion.VigenciaObservacion <= 0 || configuracion.DemoraReconexion <= 0 || configuracion.Ahora == nil ||
		configuracion.Ahora().IsZero() || configuracion.Sumidero == nil || !filepath.IsAbs(configuracion.DirectorioCuenta) ||
		filepath.Clean(configuracion.DirectorioCuenta) != configuracion.DirectorioCuenta {
		return nil, &Error{Code: CodeStateInvalid}
	}
	entorno := cloneEnvironment(configuracion.Entorno)
	entorno["HOME"], entorno["CODEX_HOME"] = configuracion.DirectorioCuenta, configuracion.DirectorioCuenta
	exacto, err := exactEnvironment(entorno)
	if err != nil {
		return nil, err
	}
	comando, err := resolveCommand(configuracion.Comando, entorno)
	if err != nil {
		return nil, err
	}
	ciclo, cancelar := context.WithCancel(padre)
	controlador := &ControladorCuota{
		configuracion: configuracion, comando: comando, entorno: exacto, ciclo: ciclo, cancelar: cancelar,
		terminado: make(chan struct{}), inicial: make(chan error, 1),
	}
	go controlador.ejecutar()
	return controlador, nil
}

func (controlador *ControladorCuota) EsperarInicial(ctx context.Context) error {
	if controlador == nil || ctx == nil {
		return &Error{Code: CodeStateInvalid}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-controlador.inicial:
		return err
	}
}

func (controlador *ControladorCuota) Cerrar(ctx context.Context) error {
	if controlador == nil || ctx == nil {
		return &Error{Code: CodeStateInvalid}
	}
	controlador.cierreUnaVez.Do(controlador.cancelar)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-controlador.terminado:
		return nil
	}
}

func (controlador *ControladorCuota) ejecutar() {
	defer close(controlador.terminado)
	for {
		err := controlador.intercambiar()
		controlador.marcarInicial(err)
		if controlador.ciclo.Err() != nil {
			return
		}
		if controlador.ultima != nil {
			desconocida := *controlador.ultima
			desconocida.Status, desconocida.ResetAt, desconocida.RetryAt =
				application.AgentQuotaUnknown, time.Time{}, time.Time{}
			desconocida.ObservedAt = controlador.configuracion.Ahora().Round(0).UTC()
			desconocida.ExpiresAt = desconocida.ObservedAt.Add(controlador.configuracion.VigenciaObservacion)
			_ = controlador.configuracion.Sumidero(controlador.ciclo, desconocida, nil)
			controlador.ultima = nil
		}
		temporizador := time.NewTimer(controlador.configuracion.DemoraReconexion)
		select {
		case <-controlador.ciclo.Done():
			if !temporizador.Stop() {
				<-temporizador.C
			}
			return
		case <-temporizador.C:
		}
	}
}

func (controlador *ControladorCuota) marcarInicial(err error) {
	controlador.inicialUnaVez.Do(func() { controlador.inicial <- err })
}

func (controlador *ControladorCuota) intercambiar() error {
	ctx, cancelar := context.WithCancelCause(controlador.ciclo)
	argumentos := controlador.configuracion.argumentosParaPruebas
	if argumentos == nil {
		argumentos = []string{"app-server"}
	}
	comando := exec.CommandContext(ctx, controlador.comando, argumentos...)
	comando.Env, comando.Stderr = append([]string(nil), controlador.entorno...), io.Discard
	configureProcessGroup(comando, func() error { return context.Cause(ctx) })
	entrada, err := comando.StdinPipe()
	if err != nil {
		cancelar(err)
		return &Error{Code: CodeProcessStartFailed}
	}
	salida, err := comando.StdoutPipe()
	if err != nil {
		cancelar(err)
		return &Error{Code: CodeProcessStartFailed}
	}
	if err = comando.Start(); err != nil {
		cancelar(err)
		return &Error{Code: CodeProcessStartFailed}
	}
	defer func() {
		cancelar(errors.New("codex.quota_session_closed"))
		_ = entrada.Close()
		_ = comando.Wait()
		_ = cleanupProcessGroup(comando)
	}()
	escribir := func(trama []byte) error {
		escritos, err := entrada.Write(trama)
		if err != nil || escritos != len(trama) {
			return io.ErrShortWrite
		}
		return nil
	}
	decodificador, _ := appserver.NewDecoder(salida, controlador.configuracion.MaximoBytesTrama)
	sesion := appserver.NewQuotaSession()
	trama, err := sesion.Initialize("init", appserver.ClientInfo{Name: "orquesta", Version: "1", Title: "quota"})
	if err != nil || escribir(trama) != nil {
		return &Error{Code: CodeUnavailable}
	}
	mensaje, err := decodificador.Next()
	if err != nil {
		return &Error{Code: CodeUnavailable}
	}
	if metodo, aceptacion := sesion.Accept(mensaje); aceptacion != nil || metodo != "initialize" {
		return &Error{Code: CodeOutputInvalid}
	}
	inicializada, _ := sesion.Initialized()
	lectura, _ := sesion.ReadQuota(int64(1))
	if escribir(inicializada) != nil || escribir(lectura) != nil {
		return &Error{Code: CodeUnavailable}
	}
	for {
		mensaje, err = decodificador.Next()
		if err != nil {
			return &Error{Code: CodeUnavailable}
		}
		metodo, aceptacion := sesion.Accept(mensaje)
		if aceptacion != nil {
			return &Error{Code: CodeOutputInvalid}
		}
		if metodo != "account/rateLimits/read" && metodo != "account/rateLimits/updated" {
			continue
		}
		traducida, err := traducirLecturaCuotaCodex(mensaje.Payload, controlador.configuracion.ReferenciaColocacion,
			controlador.configuracion.VigenciaObservacion, controlador.configuracion.Ahora)
		if err != nil || controlador.configuracion.Sumidero(ctx, traducida.Observacion, traducida.Evidencia) != nil {
			return &Error{Code: CodeOutputInvalid}
		}
		controlador.ultima = &traducida.Observacion
		controlador.marcarInicial(nil)
	}
}
