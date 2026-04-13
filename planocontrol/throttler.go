/*
Software libre bajo licencia GNU GPL v3
*/

package planocontrol

import (
	"sync"
	"time"
)

// Throttler permite limitar la frecuencia de ejecución de acciones basadas en una clave.
type Throttler struct {
	mu   sync.Mutex
	last map[string]time.Time
}

// NewThrottler crea una instancia de Throttler lista para usar.
func NewThrottler() *Throttler {
	return &Throttler{
		last: make(map[string]time.Time),
	}
}

// Allow verifica si ha pasado el intervalo indicado desde la última vez que se permitió la acción para esa clave.
// Usa la hora actual UTC.
func (t *Throttler) Allow(key string, interval time.Duration) bool {
	return t.AllowAt(key, interval, time.Now().UTC())
}

// AllowAt verifica si ha pasado el intervalo indicado desde la última vez que se permitió la acción para esa clave,
// usando la marca de tiempo proporcionada.
func (t *Throttler) AllowAt(key string, interval time.Duration, now time.Time) bool {
	if interval <= 0 {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil {
		t.last = make(map[string]time.Time)
	}

	if last, ok := t.last[key]; ok && now.Sub(last) < interval {
		return false
	}
	t.last[key] = now.UTC()
	return true
}

// Reset borra todo el historial de throttling.
func (t *Throttler) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.last = nil
}

// Set fija manualmente la marca temporal de una clave. Se usa para pruebas y
// para forzar ventanas de cooldown predecibles.
func (t *Throttler) Set(key string, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil {
		t.last = make(map[string]time.Time)
	}
	t.last[key] = at.UTC()
}

// Gate permite limitar la frecuencia de una acción global basada en una expiración.
type Gate struct {
	mu      sync.Mutex
	expires time.Time
}

// NewGate crea un Gate vacío.
func NewGate() *Gate {
	return &Gate{}
}

// Allow verifica si ha pasado el tiempo de expiración y, si es así, lo actualiza con el intervalo indicado.
// Usa la hora actual UTC.
func (g *Gate) Allow(interval time.Duration) bool {
	return g.AllowAt(interval, time.Now().UTC())
}

// AllowAt verifica si ha pasado el tiempo de expiración usando la marca de tiempo proporcionada.
func (g *Gate) AllowAt(interval time.Duration, now time.Time) bool {
	if interval <= 0 {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.expires.IsZero() && now.Before(g.expires) {
		return false
	}
	g.expires = now.Add(interval).UTC()
	return true
}

// Reset limpia la expiración del gate.
func (g *Gate) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expires = time.Time{}
}

// Set fija manualmente la expiración del gate. Se usa para pruebas.
func (g *Gate) Set(at time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expires = at.UTC()
}
