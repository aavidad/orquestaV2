package staticcapacity

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestObserverRenuevaCapacidadBrutaIncluidoCeroPresente(t *testing.T) {
	ahora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	config := Config{ReferenciaFuente: "source:static", ReferenciaPool: "pool:default", Vigencia: time.Minute, Ahora: func() time.Time { return ahora }}
	observador, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	primera, err := observador.ObserveCapacity(context.Background(), config.ReferenciaFuente, config.ReferenciaPool)
	ahora = ahora.Add(30 * time.Second)
	segunda, segundoErr := observador.ObserveCapacity(context.Background(), config.ReferenciaFuente, config.ReferenciaPool)
	plazas := segunda.Resources.Slots
	if err != nil || segundoErr != nil || primera.ObservedAt.Equal(segunda.ObservedAt) ||
		!segunda.ExpiresAt.Equal(ahora.Add(time.Minute)) || !plazas.Limit.Present || !plazas.Remaining.Present ||
		plazas.Limit.Value != 0 || plazas.Remaining.Value != plazas.Limit.Value {
		t.Fatalf("observaciones primera=%+v/%v segunda=%+v/%v", primera, err, segunda, segundoErr)
	}
	for _, dimension := range []application.AgentCapacityDimension{segunda.Resources.Seconds, segunda.Resources.Messages, segunda.Resources.Tokens, segunda.Resources.Credits} {
		if dimension.Applicability != application.AgentCapacityApplicabilityNotApplicable {
			t.Fatalf("dimensión inventada: %+v", dimension)
		}
	}
}

func TestObserverRechazaConfiguracionScopeYContextoInvalidos(t *testing.T) {
	ahora := time.Unix(100, 0).UTC()
	valida := Config{ReferenciaFuente: "source:static", ReferenciaPool: "pool:default", Plazas: 1, Vigencia: time.Minute, Ahora: func() time.Time { return ahora }}
	invalidas := []Config{valida, valida, valida, valida}
	invalidas[0].ReferenciaFuente, invalidas[1].Plazas, invalidas[2].Vigencia, invalidas[3].Ahora = "", -1, 0, nil
	for indice, config := range invalidas {
		if _, err := New(config); err != ErrInvalidConfig {
			t.Fatalf("configuración %d error=%v", indice, err)
		}
	}
	observador, _ := New(valida)
	if _, err := observador.ObserveCapacity(context.Background(), "source:other", valida.ReferenciaPool); err != ErrScopeMismatch {
		t.Fatalf("scope incorrecto: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := observador.ObserveCapacity(cancelado, valida.ReferenciaFuente, valida.ReferenciaPool); err != context.Canceled {
		t.Fatalf("contexto cancelado: %v", err)
	}
}
