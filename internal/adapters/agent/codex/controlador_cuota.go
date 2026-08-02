package codex

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"time"

	"orquesta/internal/adapters/agent/codex/appserver"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type configuracionControladorCuota struct {
	application.ConfiguracionControladoresCuotaAgente
	comando               string
	entorno               []string
	referenciaColocacion  ports.AgentPlacementRef
	maximoBytesTrama      int
	argumentosParaPruebas []string
}

type ControladorCuota struct {
	configuracion               configuracionControladorCuota
	ciclo                       context.Context
	cancelar                    context.CancelFunc
	terminado                   chan error
	inicial                     chan error
	inicialUnaVez, cierreUnaVez sync.Once
	ultima                      *application.AgentQuotaObservation
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
	entorno, err := adaptador.accountExecutionEnvironment(adaptador.environment)
	if err != nil {
		return nil, err
	}
	controlador, err := iniciarControladorCuota(ctx, configuracionControladorCuota{
		ConfiguracionControladoresCuotaAgente: parametros,
		comando:                               adaptador.command, entorno: entorno, referenciaColocacion: colocacion,
		maximoBytesTrama: adaptador.config.AppServerMaxFrameBytes,
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

func iniciarControladorCuota(padre context.Context, configuracion configuracionControladorCuota) (*ControladorCuota, error) {
	if padre == nil || configuracion.comando == "" || configuracion.referenciaColocacion.String() == "" || configuracion.maximoBytesTrama < 1 ||
		configuracion.VigenciaObservacion <= 0 || configuracion.DemoraReconexion <= 0 || configuracion.Ahora == nil ||
		configuracion.Ahora().IsZero() || configuracion.Sumidero == nil {
		return nil, &Error{Code: CodeStateInvalid}
	}
	ciclo, cancelar := context.WithCancel(padre)
	controlador := &ControladorCuota{
		configuracion: configuracion, ciclo: ciclo, cancelar: cancelar,
		terminado: make(chan error), inicial: make(chan error, 1),
	}
	go controlador.ejecutar()
	return controlador, nil
}

func (controlador *ControladorCuota) EsperarInicial(ctx context.Context) error {
	if controlador == nil || ctx == nil {
		return &Error{Code: CodeStateInvalid}
	}
	return esperarControladorCuota(ctx, controlador.inicial)
}

func (controlador *ControladorCuota) Cerrar(ctx context.Context) error {
	if controlador == nil || ctx == nil {
		return &Error{Code: CodeStateInvalid}
	}
	controlador.cierreUnaVez.Do(controlador.cancelar)
	return esperarControladorCuota(ctx, controlador.terminado)
}

func esperarControladorCuota(ctx context.Context, resultado <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-resultado:
		return err
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
		espera, cancelar := context.WithTimeout(controlador.ciclo, controlador.configuracion.DemoraReconexion)
		<-espera.Done()
		cancelar()
		if controlador.ciclo.Err() != nil {
			return
		}
	}
}

func (controlador *ControladorCuota) marcarInicial(err error) {
	controlador.inicialUnaVez.Do(func() { controlador.inicial <- err })
}

func (controlador *ControladorCuota) intercambiar() error {
	ctx, cancelar := context.WithCancel(controlador.ciclo)
	argumentos := controlador.configuracion.argumentosParaPruebas
	if argumentos == nil {
		argumentos = []string{"app-server"}
	}
	comando := exec.CommandContext(ctx, controlador.configuracion.comando, argumentos...)
	comando.Env, comando.Stderr = append([]string(nil), controlador.configuracion.entorno...), io.Discard
	configureProcessGroup(comando, ctx.Err)
	entrada, err := comando.StdinPipe()
	if err != nil {
		cancelar()
		return &Error{Code: CodeProcessStartFailed}
	}
	salida, err := comando.StdoutPipe()
	if err != nil {
		cancelar()
		return &Error{Code: CodeProcessStartFailed}
	}
	if err = comando.Start(); err != nil {
		cancelar()
		return &Error{Code: CodeProcessStartFailed}
	}
	defer func() {
		cancelar()
		_ = entrada.Close()
		_ = cleanupProcessGroup(comando)
		_ = comando.Wait()
	}()
	escribir := func(trama []byte) error {
		escritos, err := entrada.Write(trama)
		if escritos != len(trama) && err == nil {
			return io.ErrShortWrite
		}
		return err
	}
	decodificador, _ := appserver.NewDecoder(salida, controlador.configuracion.maximoBytesTrama)
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
		traducida, err := traducirLecturaCuotaCodex(mensaje.Payload, controlador.configuracion.referenciaColocacion,
			controlador.configuracion.VigenciaObservacion, controlador.configuracion.Ahora)
		if err != nil || controlador.configuracion.Sumidero(ctx, traducida.Observacion, traducida.Evidencia) != nil {
			return &Error{Code: CodeOutputInvalid}
		}
		controlador.ultima = &traducida.Observacion
		controlador.marcarInicial(nil)
	}
}
