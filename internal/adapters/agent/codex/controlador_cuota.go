package codex

import (
	"context"
	"encoding/json"
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

type SumideroObservacionCuota func(context.Context, application.AgentQuotaObservation, json.RawMessage) error

type ConfiguracionControladorCuota struct {
	Comando, DirectorioCuenta string
	Entorno                   map[string]string
	ReferenciaColocacion      ports.AgentPlacementRef
	MaximoBytesTrama          int
	VigenciaObservacion       time.Duration
	DemoraReconexion          time.Duration
	Ahora                     func() time.Time
	Sumidero                  SumideroObservacionCuota
	argumentosParaPruebas     []string
}

type ControladorCuota struct {
	configuracion ConfiguracionControladorCuota
	comando       string
	entorno       []string
	ciclo         context.Context
	cancelar      context.CancelFunc
	terminado     chan struct{}
	inicialLista  chan struct{}
	inicialUnaVez sync.Once
	errorInicial  error
	cierreUnaVez  sync.Once
	ultima        *application.AgentQuotaObservation
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
	entornoExacto, err := exactEnvironment(entorno)
	if err != nil {
		return nil, err
	}
	comando, err := resolveCommand(configuracion.Comando, entorno)
	if err != nil {
		return nil, err
	}
	ciclo, cancelar := context.WithCancel(padre)
	controlador := &ControladorCuota{
		configuracion: configuracion, comando: comando, entorno: entornoExacto, ciclo: ciclo, cancelar: cancelar,
		terminado: make(chan struct{}), inicialLista: make(chan struct{}),
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
	case <-controlador.inicialLista:
		return controlador.errorInicial
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
		controlador.señalarInicial(err)
		if controlador.ciclo.Err() != nil {
			return
		}
		if controlador.ultima != nil {
			desconocida := *controlador.ultima
			desconocida.Status = application.AgentQuotaUnknown
			desconocida.ObservedAt = controlador.configuracion.Ahora().Round(0).UTC()
			desconocida.ExpiresAt = desconocida.ObservedAt.Add(controlador.configuracion.VigenciaObservacion)
			desconocida.ResetAt, desconocida.RetryAt = time.Time{}, time.Time{}
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

func (controlador *ControladorCuota) intercambiar() error {
	contextoEjecucion, cancelar := context.WithCancelCause(controlador.ciclo)
	argumentos := controlador.configuracion.argumentosParaPruebas
	if argumentos == nil {
		argumentos = []string{"app-server"}
	}
	comando := exec.CommandContext(contextoEjecucion, controlador.comando, argumentos...)
	comando.Env, comando.Stderr = append([]string(nil), controlador.entorno...), io.Discard
	configureProcessGroup(comando, func() error { return context.Cause(contextoEjecucion) })
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
	decodificador, _ := appserver.NewDecoder(salida, controlador.configuracion.MaximoBytesTrama)
	sesion := appserver.NewQuotaSession()
	trama, err := sesion.Initialize("init", appserver.ClientInfo{Name: "orquesta", Version: "1", Title: "quota"})
	if err != nil || escribirTramaCuota(entrada, trama) != nil {
		return &Error{Code: CodeUnavailable}
	}
	mensaje, err := decodificador.Next()
	if err != nil {
		return &Error{Code: CodeUnavailable}
	}
	if metodo, errorAceptacion := sesion.Accept(mensaje); errorAceptacion != nil || metodo != "initialize" {
		return &Error{Code: CodeOutputInvalid}
	}
	inicializada, _ := sesion.Initialized()
	lectura, _ := sesion.ReadQuota(int64(1))
	if escribirTramaCuota(entrada, inicializada) != nil || escribirTramaCuota(entrada, lectura) != nil {
		return &Error{Code: CodeUnavailable}
	}
	for {
		mensaje, err = decodificador.Next()
		if err != nil {
			return &Error{Code: CodeUnavailable}
		}
		metodo, errorAceptacion := sesion.Accept(mensaje)
		if errorAceptacion != nil {
			return &Error{Code: CodeOutputInvalid}
		}
		if metodo != "account/rateLimits/read" && metodo != "account/rateLimits/updated" {
			continue
		}
		traducida, errorTraduccion := traducirLecturaCuotaCodex(
			mensaje.Payload, controlador.configuracion.ReferenciaColocacion,
			controlador.configuracion.VigenciaObservacion, controlador.configuracion.Ahora,
		)
		if errorTraduccion != nil ||
			controlador.configuracion.Sumidero(contextoEjecucion, traducida.Observacion, traducida.Evidencia) != nil {
			return &Error{Code: CodeOutputInvalid}
		}
		controlador.ultima = &traducida.Observacion
		controlador.señalarInicial(nil)
	}
}

func (controlador *ControladorCuota) señalarInicial(err error) {
	controlador.inicialUnaVez.Do(func() {
		controlador.errorInicial = err
		close(controlador.inicialLista)
	})
}

func escribirTramaCuota(escritor io.Writer, trama []byte) error {
	escritos, err := escritor.Write(trama)
	if err != nil || escritos != len(trama) {
		return io.ErrShortWrite
	}
	return nil
}
