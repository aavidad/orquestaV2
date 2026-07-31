package codex

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestControladorCuotaLeeReconectaRotaYCierraExactamente(t *testing.T) {
	if os.Getenv("ORQUESTA_QUOTA_HELPER") == "1" {
		ejecutarAyudanteCuota()
		return
	}
	contador := t.TempDir() + "/starts"
	colocacion, _ := ports.NewAgentPlacementRef("placement:codex:test")
	observaciones := make(chan application.AgentQuotaObservation, 5)
	controlador, err := IniciarControladorCuota(context.Background(), ConfiguracionControladorCuota{
		Comando: os.Args[0], DirectorioCuenta: t.TempDir(),
		Entorno:              map[string]string{"ORQUESTA_QUOTA_HELPER": "1", "ORQUESTA_QUOTA_COUNTER": contador},
		ReferenciaColocacion: colocacion, MaximoBytesTrama: 4096, VigenciaObservacion: time.Minute,
		DemoraReconexion: time.Millisecond, Ahora: func() time.Time { return time.Unix(1_000, 0).UTC() },
		Sumidero: func(_ context.Context, observacion application.AgentQuotaObservation, evidencia []byte) error {
			if strings.Contains(string(evidencia), "secret") {
				return fmt.Errorf("la evidencia conservó datos sensibles")
			}
			observaciones <- observacion
			return nil
		},
		argumentosParaPruebas: []string{"-test.run=^TestControladorCuotaLeeReconectaRotaYCierraExactamente$"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := controlador.EsperarInicial(ctx); err != nil {
		t.Fatal(err)
	}
	var recibidas []application.AgentQuotaObservation
	for len(recibidas) < 4 {
		select {
		case observacion := <-observaciones:
			recibidas = append(recibidas, observacion)
		case <-ctx.Done():
			t.Fatalf("observaciones=%d: %v", len(recibidas), ctx.Err())
		}
	}
	if recibidas[1].Status != application.AgentQuotaUnknown ||
		recibidas[0].WindowRef != recibidas[1].WindowRef ||
		recibidas[0].WindowRef == recibidas[2].WindowRef || recibidas[2].WindowRef == recibidas[3].WindowRef {
		t.Fatalf("la rotación no cambió ventanas: %+v", recibidas)
	}
	if err := controlador.Cerrar(ctx); err != nil {
		t.Fatal(err)
	}
	datos, err := os.ReadFile(contador)
	if err != nil {
		t.Fatal(err)
	}
	lineas := strings.Fields(string(datos))
	if len(lineas) != 2 {
		t.Fatalf("procesos iniciados=%d, want 2", len(lineas))
	}
	pid, _ := strconv.Atoi(lineas[1])
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("proceso de cuota huérfano: %d", pid)
	}
}

func TestControladorCuotaRechazaConfiguracionInvalida(t *testing.T) {
	if controlador, err := IniciarControladorCuota(context.Background(), ConfiguracionControladorCuota{}); controlador != nil || ErrorCode(err) != CodeStateInvalid {
		t.Fatalf("controlador=%v error=%v", controlador, err)
	}
}

func ejecutarAyudanteCuota() {
	contador := os.Getenv("ORQUESTA_QUOTA_COUNTER")
	archivo, _ := os.OpenFile(contador, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	informacion, _ := archivo.Stat()
	intento := informacion.Size()/8 + 1
	_, _ = fmt.Fprintf(archivo, "%07d\n", os.Getpid())
	_ = archivo.Close()
	lector := bufio.NewScanner(os.Stdin)
	if !lector.Scan() {
		os.Exit(2)
	}
	fmt.Printf("{\"id\":\"init\",\"result\":{\"userAgent\":\"test\",\"codexHome\":\"/test\",\"platformFamily\":\"unix\",\"platformOs\":\"linux\"}}\n")
	if !lector.Scan() || !lector.Scan() {
		os.Exit(2)
	}
	reinicio := int64(2_000 + intento*1_000)
	fmt.Printf("{\"id\":1,\"result\":%s}\n", cargaCuotaAyudante(reinicio))
	if intento == 1 {
		os.Exit(0)
	}
	fmt.Printf("{\"method\":\"account/rateLimits/updated\",\"params\":%s}\n", cargaCuotaAyudante(reinicio+1_000))
	for lector.Scan() {
	}
}

func cargaCuotaAyudante(reinicio int64) string {
	return fmt.Sprintf(`{"rateLimits":{"credits":{"secret":true},"individualLimit":null,"limitId":"codex","limitName":"secret","planType":"plus","primary":{"usedPercent":25,"windowDurationMins":300,"resetsAt":%d},"rateLimitReachedType":null,"secondary":null,"spendControlReached":false},"rateLimitResetCredits":null,"rateLimitsByLimitId":null}`, reinicio)
}
