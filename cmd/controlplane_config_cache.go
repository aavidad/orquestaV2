/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"sync"
	"time"

	"orquesta/db"
)

const controlPlaneConfigCacheTTL = 60 * time.Second

type configCacheEntry struct {
	value     string
	err       error
	expiresAt time.Time
}

var controlPlaneConfigCache struct {
	mu      sync.RWMutex
	entries map[string]configCacheEntry
}

func resetControlPlaneConfigCache() {
	controlPlaneConfigCache.mu.Lock()
	defer controlPlaneConfigCache.mu.Unlock()
	controlPlaneConfigCache.entries = nil
}

func controlPlaneConfigGetCached(clave string) (string, error) {
	now := time.Now()

	controlPlaneConfigCache.mu.RLock()
	if entry, ok := controlPlaneConfigCache.entries[clave]; ok && now.Before(entry.expiresAt) {
		controlPlaneConfigCache.mu.RUnlock()
		return entry.value, entry.err
	}
	controlPlaneConfigCache.mu.RUnlock()

	v, err := db.ConfigGet(clave)

	controlPlaneConfigCache.mu.Lock()
	if controlPlaneConfigCache.entries == nil {
		controlPlaneConfigCache.entries = make(map[string]configCacheEntry)
	}
	controlPlaneConfigCache.entries[clave] = configCacheEntry{
		value:     v,
		err:       err,
		expiresAt: now.Add(controlPlaneConfigCacheTTL),
	}
	controlPlaneConfigCache.mu.Unlock()

	return v, err
}
