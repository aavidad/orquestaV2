package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"orquesta/capacidadapp"
)

type gestorRuntimeModelosOllama struct {
	endpoint string
	client   *http.Client
}

func nuevoGestorRuntimeModelosOllama(endpoint string, client *http.Client) capacidadapp.GestorRuntimeModelos {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if client == nil {
		client = clienteHTTPOllamaLocal()
	}
	return gestorRuntimeModelosOllama{
		endpoint: endpoint,
		client:   client,
	}
}

func (g gestorRuntimeModelosOllama) ListarModelosActivos() ([]capacidadapp.ModeloRuntimeActivo, error) {
	if strings.TrimSpace(g.endpoint) == "" {
		return nil, fmt.Errorf("endpoint de ollama no configurado")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, g.endpoint+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama devolvio status %d al listar modelos activos", resp.StatusCode)
	}
	var decoded struct {
		Models []struct {
			Name      string `json:"name"`
			Model     string `json:"model"`
			SizeVRAM  int64  `json:"size_vram"`
			Size      int64  `json:"size"`
			ExpiresAt string `json:"expires_at"`
			Details   struct {
				Format            string `json:"format"`
				Family            string `json:"family"`
				ParameterSize     string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	items := make([]capacidadapp.ModeloRuntimeActivo, 0, len(decoded.Models))
	for _, item := range decoded.Models {
		modelo := strings.TrimSpace(item.Name)
		if modelo == "" {
			modelo = strings.TrimSpace(item.Model)
		}
		var activoEn *time.Time
		if raw := strings.TrimSpace(item.ExpiresAt); raw != "" {
			if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
				activoEn = &ts
			}
		}
		procesador := "runtime_local"
		if item.SizeVRAM > 0 {
			procesador = "gpu"
		}
		items = append(items, capacidadapp.ModeloRuntimeActivo{
			Modelo:     modelo,
			ID:         strings.TrimSpace(item.Model),
			Tamano:     item.Details.ParameterSize,
			Procesador: procesador,
			Contexto:   item.Details.QuantizationLevel,
			Hasta:      strings.TrimSpace(item.ExpiresAt),
			Runtime:    "ollama",
			ActivoEn:   activoEn,
		})
	}
	return items, nil
}

func (g gestorRuntimeModelosOllama) DescargarModelo(modelo string) error {
	if strings.TrimSpace(g.endpoint) == "" {
		return fmt.Errorf("endpoint de ollama no configurado")
	}
	modelo = strings.TrimSpace(modelo)
	if modelo == "" {
		return fmt.Errorf("modelo obligatorio")
	}
	payload := map[string]any{
		"model":      modelo,
		"prompt":     "",
		"stream":     false,
		"keep_alive": 0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, g.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ollama devolvio status %d al descargar modelo %q", resp.StatusCode, modelo)
	}
	return nil
}
