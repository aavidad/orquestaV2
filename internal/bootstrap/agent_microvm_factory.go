package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

// maximoDescriptorPerfilMicroVMBytes es un límite intrínseco del frame local
// descriptor-perfil-lanzamiento.v1, no una opción de producto. El contrato
// público contiene solo hechos acotados (dos servicios y hashes/refs breves),
// por lo que 1 MiB deja margen amplio y evita leer entrada ilimitada antes de
// poder aplicar el decodificador estricto del protocolo.
const maximoDescriptorPerfilMicroVMBytes int64 = 1 << 20

var (
	errFactoriaAgentMicroVMSeleccionInvalida  = errors.New("bootstrap.agent_microvm_selection_invalid")
	errFactoriaAgentMicroVMDescriptorInvalido = errors.New("bootstrap.agent_microvm_profile_descriptor_invalid")
	errFactoriaAgentMicroVMClienteInvalido    = errors.New("bootstrap.agent_microvm_client_invalid")
	errFactoriaAgentMicroVMAutoridadInvalida  = errors.New("bootstrap.agent_microvm_authority_invalid")
)

type dependenciasAutoridadFisicaAgentMicroVM struct {
	lectorCredencial     credentials.UseAuthorityReader
	almacenOneShot       credentials.OneShotStore
	registroLanzamientos ports.MicroVMHostLaunchAuthorityRegistry
}

type recursoClienteAgentMicroVM struct {
	cliente           agentmicrovm.Client
	liberarConexiones func() error
}

type constructorClienteAgentMicroVM func(string) (recursoClienteAgentMicroVM, error)

// productionAgentMicroVM compone exclusivamente el backend Codex sobre una
// microVM ya administrada por Agente MicroVM. No abre KVM, no arranca ni cierra
// máquinas y no adquiere ownership del CredentialStore compartido del Runtime.
func productionAgentMicroVM(
	snapshot config.Snapshot,
	promptRenderer codex.PromptRenderer,
	credentialStore credentials.Store,
	autoridad dependenciasAutoridadFisicaAgentMicroVM,
) (*agenteMicroVM, error) {
	return productionAgentMicroVMConConstructor(
		snapshot,
		promptRenderer,
		credentialStore,
		autoridad,
		nuevoRecursoClienteAgentMicroVM,
	)
}

func productionAgentMicroVMConConstructor(
	snapshot config.Snapshot,
	promptRenderer codex.PromptRenderer,
	credentialStore credentials.Store,
	autoridad dependenciasAutoridadFisicaAgentMicroVM,
	construirCliente constructorClienteAgentMicroVM,
) (*agenteMicroVM, error) {
	if snapshot.RuntimeProvider() != "codex" || snapshot.RuntimeIsolation() != "microvm" {
		return nil, errFactoriaAgentMicroVMSeleccionInvalida
	}
	if interfazNulaAgentMicroVM(promptRenderer) || interfazNulaAgentMicroVM(credentialStore) ||
		interfazNulaAgentMicroVM(autoridad.lectorCredencial) ||
		interfazNulaAgentMicroVM(autoridad.almacenOneShot) ||
		interfazNulaAgentMicroVM(autoridad.registroLanzamientos) {
		return nil, errFactoriaAgentMicroVMAutoridadInvalida
	}
	descriptor, err := cargarDescriptorPerfilAgentMicroVM(
		snapshot.RuntimeMicroVMProfileDescriptorPath(),
		snapshot.RuntimeMicroVMExpectedProfileDescriptorSHA256(),
		os.Geteuid(),
	)
	if err != nil {
		return nil, err
	}
	colocacion, err := ports.NewAgentPlacementRef(snapshot.RuntimeMicroVMPlacementRef())
	if err != nil {
		return nil, errors.Join(errFactoriaAgentMicroVMSeleccionInvalida, err)
	}
	if construirCliente == nil {
		return nil, errFactoriaAgentMicroVMClienteInvalido
	}
	firmante, err := agentmicrovm.NewCredentialSigner(agentmicrovm.CredentialSignerConfig{
		Store:         credentialStore,
		CredentialRef: credentials.CredentialRef(snapshot.RuntimeMicroVMLaunchGrantSigningCredentialRef()),
		KeyID:         snapshot.RuntimeMicroVMLaunchGrantKeyID(),
	})
	if err != nil {
		return nil, err
	}
	resolutorCredencial, err := agentmicrovm.NewCredentialClaimResolver(
		autoridad.lectorCredencial,
		credentials.PurposeRef(codex.ProviderRef),
		[]agentmicrovm.CredentialClaimBinding{{
			PlacementRef:  colocacion,
			CredentialRef: credentials.CredentialRef(snapshot.RuntimeCodexCredentialRef()),
		}},
	)
	if err != nil {
		return nil, errors.Join(errFactoriaAgentMicroVMAutoridadInvalida, err)
	}
	broker, err := agentmicrovm.NewCredentialBroker(
		autoridad.registroLanzamientos,
		autoridad.almacenOneShot,
		snapshot.RuntimeMicroVMCredentialBrokerExchangeTimeout(),
	)
	if err != nil {
		return nil, errors.Join(errFactoriaAgentMicroVMAutoridadInvalida, err)
	}
	peerUID := snapshot.RuntimeMicroVMCredentialBrokerPeerUID()
	maximoConexiones := snapshot.RuntimeMicroVMCredentialBrokerMaxConnections()
	if peerUID < 0 || uint64(peerUID) > uint64(^uint32(0)) ||
		maximoConexiones <= 0 || uint64(maximoConexiones) > uint64(^uint32(0)) {
		return nil, errFactoriaAgentMicroVMAutoridadInvalida
	}
	servidorBroker, err := agentmicrovm.NewCredentialBrokerServer(
		agentmicrovm.CredentialBrokerServerConfig{
			SocketPath:        snapshot.RuntimeMicroVMCredentialBrokerSocketPath(),
			OwnerUID:          uint32(os.Geteuid()),
			PeerUID:           uint32(peerUID),
			ConnectionTimeout: snapshot.RuntimeMicroVMCredentialBrokerExchangeTimeout(),
			MaxConcurrent:     uint32(maximoConexiones),
		},
		broker,
	)
	if err != nil {
		return nil, errors.Join(errFactoriaAgentMicroVMAutoridadInvalida, err)
	}
	recurso, err := construirCliente(snapshot.RuntimeMicroVMSocketPath())
	if err != nil {
		return nil, errors.Join(errFactoriaAgentMicroVMClienteInvalido, err)
	}
	if recurso.cliente == nil || recurso.liberarConexiones == nil {
		if recurso.liberarConexiones != nil {
			return nil, errors.Join(errFactoriaAgentMicroVMClienteInvalido, recurso.liberarConexiones())
		}
		return nil, errFactoriaAgentMicroVMClienteInvalido
	}
	fallarCliente := func(causa error) (*agenteMicroVM, error) {
		return nil, errors.Join(causa, recurso.liberarConexiones())
	}

	capacidades := ports.AgentCapabilities{
		ProviderRef:                 codex.ProviderRef,
		ModelRef:                    codex.DefaultModelRef,
		AgentRef:                    codex.AgentRef,
		Unrestricted:                true,
		RequierePreservacionEntorno: true,
	}
	adaptador, err := agentmicrovm.New(agentmicrovm.Config{
		Client:                  recurso.cliente,
		Signer:                  firmante,
		ClaimResolver:           resolutorCredencial,
		LaunchAuthorityRegistry: autoridad.registroLanzamientos,
		Capabilities:            capacidades,
		ModelBinding: agentmicrovm.ProviderModelBinding{
			ModelRef:      codex.DefaultModelRef,
			ProviderModel: snapshot.RuntimeCodexModel(),
			Profile: agentmicrovm.ProfileBinding{
				PlacementRef: colocacion,
				Descriptor:   descriptor,
			},
		},
		PromptRenderer: promptRenderer,
	})
	if err != nil {
		return fallarCliente(err)
	}
	// El listener pertenece a la composición residente, no al contexto efímero
	// de Build. Start acredita la publicación antes de devolver el agente.
	if err := servidorBroker.Start(context.Background()); err != nil {
		return fallarCliente(err)
	}
	cerrarBrokerYCliente := func() error {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			snapshot.ServerShutdownTimeout(),
		)
		brokerErr := servidorBroker.Shutdown(shutdownCtx)
		cancel()
		if brokerErr != nil {
			// Shutdown ya ha ordenado cerrar listener y conexiones. Esperar su
			// terminación exacta evita liberar el cliente mientras el broker aún
			// pudiera consumir el store compartido.
			brokerErr = errors.Join(brokerErr, servidorBroker.Shutdown(context.Background()))
		}
		return errors.Join(brokerErr, recurso.liberarConexiones())
	}

	// El store pertenece al Runtime. El callback explícito sin efecto evita que
	// el wrapper lo descubra o cierre por aproximación estructural.
	agente, err := newAgentMicroVM(adaptador, cerrarBrokerYCliente, func() error { return nil })
	if err != nil {
		return nil, errors.Join(err, cerrarBrokerYCliente())
	}
	return agente, nil
}

func nuevoRecursoClienteAgentMicroVM(rutaSocket string) (recursoClienteAgentMicroVM, error) {
	cliente, err := microvm.Nuevo(rutaSocket)
	if err != nil {
		return recursoClienteAgentMicroVM{}, err
	}
	return recursoClienteAgentMicroVM{
		cliente: cliente,
		liberarConexiones: func() error {
			cliente.LiberarConexiones()
			return nil
		},
	}, nil
}

func cargarDescriptorPerfilAgentMicroVM(
	ruta string,
	digestEsperado string,
	ownerUID int,
) (microvm.DescriptorPerfilLanzamientoV1, error) {
	if strings.TrimSpace(ruta) != ruta || strings.ContainsRune(ruta, '\x00') ||
		!filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta || ownerUID < 0 {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	digestBytes, err := hex.DecodeString(digestEsperado)
	if err != nil || len(digestBytes) != sha256.Size || hex.EncodeToString(digestBytes) != digestEsperado {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	resuelta, err := filepath.EvalSymlinks(ruta)
	if err != nil || resuelta != ruta {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	directorio, nombre := filepath.Split(ruta)
	if nombre == "" {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	defer raiz.Close()
	antes, err := raiz.Lstat(nombre)
	if err != nil || !descriptorPerfilAgentMicroVMSeguro(antes, ownerUID) {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	fichero, err := raiz.Open(nombre)
	if err != nil {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	defer fichero.Close()
	abierto, err := fichero.Stat()
	if err != nil || !descriptorPerfilAgentMicroVMSeguro(abierto, ownerUID) || !os.SameFile(antes, abierto) {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, maximoDescriptorPerfilMicroVMBytes+1))
	if err != nil || len(contenido) == 0 || int64(len(contenido)) > maximoDescriptorPerfilMicroVMBytes {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	despues, err := raiz.Lstat(nombre)
	if err != nil || !descriptorPerfilAgentMicroVMSeguro(despues, ownerUID) || !os.SameFile(abierto, despues) {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	digest := sha256.Sum256(contenido)
	if hex.EncodeToString(digest[:]) != digestEsperado {
		return microvm.DescriptorPerfilLanzamientoV1{}, errFactoriaAgentMicroVMDescriptorInvalido
	}
	descriptor, err := microvm.DecodificarDescriptorPerfilLanzamientoV1(contenido)
	if err != nil {
		return microvm.DescriptorPerfilLanzamientoV1{}, errors.Join(errFactoriaAgentMicroVMDescriptorInvalido, err)
	}
	return descriptor, nil
}

func descriptorPerfilAgentMicroVMSeguro(info os.FileInfo, ownerUID int) bool {
	if info == nil || !info.Mode().IsRegular() || info.Size() <= 0 ||
		info.Size() > maximoDescriptorPerfilMicroVMBytes || info.Mode().Perm()&0o400 == 0 ||
		info.Mode().Perm()&0o022 != 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat != nil && int(stat.Uid) == ownerUID && stat.Nlink == 1
}
