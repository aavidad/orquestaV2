//go:build linux

package acceptance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

var (
	v38B04Manifest = flag.String("v38-b04-manifest", "", "manifiesto B04 inmutable")
	v38B04Sibling  = flag.String("v38-b04-agentmicrovm-repo", "", "checkout exacto de agente_microvm")
)

const v38B04Protocol = "agentmicrovm.local.v1"

type v38B04ManifestV1 struct {
	Schema   string `json:"schema"`
	Protocol string `json:"protocol"`
	Orquesta struct {
		Repository     string `json:"repository"`
		SubjectCommit  string `json:"subject_commit"`
		ContractPath   string `json:"contract_path"`
		ContractSHA256 string `json:"contract_sha256"`
	} `json:"orquesta"`
	AgentMicroVM struct {
		Repository    string `json:"repository"`
		SubjectCommit string `json:"subject_commit"`
		BinarySHA256  string `json:"binary_sha256"`
		CargoPackage  string `json:"cargo_package"`
		BinaryName    string `json:"binary_name"`
	} `json:"agent_microvm"`
	Limits struct {
		BuildMilliseconds   int64 `json:"build_milliseconds"`
		StartupMilliseconds int64 `json:"startup_milliseconds"`
		ResponseBytes       int64 `json:"response_bytes"`
	} `json:"limits"`
}

func TestV38B04CrossRepositoryContract(t *testing.T) {
	if *v38B04Manifest == "" && *v38B04Sibling == "" {
		t.Skip("B04 opt-in: exige manifiesto y checkout hermano explícitos")
	}
	if *v38B04Manifest == "" || *v38B04Sibling == "" {
		t.Fatal("B04 exige juntos -v38-b04-manifest y -v38-b04-agentmicrovm-repo")
	}

	orquestaRoot := v38B04RepositoryRoot(t)
	manifest := v38B04ReadManifest(t, *v38B04Manifest)
	v38B04ValidateManifest(t, manifest)
	v38B04ValidateSubjects(t, manifest, orquestaRoot, *v38B04Sibling)

	targetRoot := filepath.Join(t.TempDir(), "cargo-target")
	binary := v38B04BuildSibling(t, manifest, *v38B04Sibling, targetRoot)
	if err := v38B04RequireDigest(binary, manifest.AgentMicroVM.BinarySHA256); err != nil {
		t.Fatal(err)
	}
	if err := v38B04RequireDigest(binary, strings.Repeat("0", 64)); err == nil {
		t.Fatal("un resumen de binario distinto fue aceptado")
	}

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	if err := os.Mkdir(runtimeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(runtimeRoot, "agentmicrovm.sock")
	config := filepath.Join(runtimeRoot, "agentmicrovm.toml")
	if err := os.WriteFile(config, []byte(fmt.Sprintf("[api]\nsocket=%q\n", socket)), 0o600); err != nil {
		t.Fatal(err)
	}

	stop := v38B04StartSibling(t, binary, config, socket, time.Duration(manifest.Limits.StartupMilliseconds)*time.Millisecond)
	t.Cleanup(stop)
	client := v38B04Client(socket, time.Duration(manifest.Limits.StartupMilliseconds)*time.Millisecond)

	status, header, body, err := v38B04Request(client, http.MethodGet, "/v1/capacidades", v38B04Protocol, nil, manifest.Limits.ResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || header != v38B04Protocol {
		t.Fatalf("capacidades status/header=%d/%q", status, header)
	}
	var capabilities struct {
		Protocol   string   `json:"protocolo"`
		Operations []string `json:"operaciones"`
		Maximum    uint32   `json:"maximo_ejecuciones"`
	}
	if err := json.Unmarshal(body, &capabilities); err != nil {
		t.Fatal(err)
	}
	if capabilities.Protocol != v38B04Protocol || capabilities.Maximum != 0 || !slices.Equal(capabilities.Operations, []string{"salud", "capacidades"}) {
		t.Fatalf("capacidades incompatibles: %+v", capabilities)
	}

	for _, protocol := range []string{"", "agentmicrovm.local.v0"} {
		status, _, _, err = v38B04Request(client, http.MethodGet, "/v1/salud", protocol, nil, manifest.Limits.ResponseBytes)
		if err != nil || status != http.StatusUpgradeRequired {
			t.Fatalf("protocolo %q status/error=%d/%v", protocol, status, err)
		}
	}
	status, _, _, err = v38B04Request(client, http.MethodPost, "/v1/salud", v38B04Protocol, nil, manifest.Limits.ResponseBytes)
	if err != nil || status != http.StatusMethodNotAllowed {
		t.Fatalf("método no admitido status/error=%d/%v", status, err)
	}
	status, _, _, err = v38B04Request(client, http.MethodPost, "/v1/ejecuciones", v38B04Protocol, bytes.NewReader(make([]byte, 16*1024+1)), manifest.Limits.ResponseBytes)
	if err != nil || status != http.StatusRequestEntityTooLarge {
		t.Fatalf("trama sobredimensionada status/error=%d/%v", status, err)
	}
	wrongClient := v38B04Client(filepath.Join(runtimeRoot, "otro.sock"), time.Second)
	if _, _, _, err = v38B04Request(wrongClient, http.MethodGet, "/v1/salud", v38B04Protocol, nil, manifest.Limits.ResponseBytes); err == nil {
		t.Fatal("un socket distinto alcanzó al candidato")
	}

	stop()
	if _, err := os.Lstat(socket); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket no retirado: %v", err)
	}
}

func v38B04ReadManifest(t *testing.T, path string) v38B04ManifestV1 {
	t.Helper()
	if !filepath.IsAbs(path) {
		t.Fatal("el manifiesto B04 debe ser absoluto")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 16*1024+1))
	decoder.DisallowUnknownFields()
	var manifest v38B04ManifestV1
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		t.Fatal("contenido posterior al manifiesto B04")
	}
	return manifest
}

func v38B04ValidateManifest(t *testing.T, manifest v38B04ManifestV1) {
	t.Helper()
	if manifest.Schema != "orquesta.v38.agentmicrovm_cross_contract.v1" || manifest.Protocol != v38B04Protocol ||
		manifest.Orquesta.Repository != "https://github.com/aavidad/orquestaV2" ||
		manifest.AgentMicroVM.Repository != "https://github.com/aavidad/agente_microvm" ||
		manifest.Orquesta.ContractPath != "acceptance/v38_agentmicrovm_cross_contract_test.go" ||
		!v38B04Hex(manifest.Orquesta.SubjectCommit, 40) || !v38B04Hex(manifest.AgentMicroVM.SubjectCommit, 40) ||
		!v38B04Hex(manifest.Orquesta.ContractSHA256, 64) || !v38B04Hex(manifest.AgentMicroVM.BinarySHA256, 64) ||
		manifest.AgentMicroVM.CargoPackage != "agente_microvm" || manifest.AgentMicroVM.BinaryName != "agente-microvm" ||
		manifest.Limits.BuildMilliseconds < 1000 || manifest.Limits.StartupMilliseconds < 100 ||
		manifest.Limits.ResponseBytes < 1024 || manifest.Limits.ResponseBytes > 1024*1024 {
		t.Fatalf("manifiesto B04 inválido: %+v", manifest)
	}
}

func v38B04ValidateSubjects(t *testing.T, manifest v38B04ManifestV1, orquestaRoot, siblingRoot string) {
	t.Helper()
	if !filepath.IsAbs(siblingRoot) {
		t.Fatal("el checkout hermano debe ser absoluto")
	}
	if got := strings.TrimSpace(v38B04Git(t, siblingRoot, "rev-parse", "HEAD")); got != manifest.AgentMicroVM.SubjectCommit {
		t.Fatalf("revisión hermana=%s, esperada=%s", got, manifest.AgentMicroVM.SubjectCommit)
	}
	if status := v38B04Git(t, siblingRoot, "status", "--porcelain", "--untracked-files=no"); status != "" {
		t.Fatalf("checkout hermano modificado: %q", status)
	}
	if got := strings.TrimSpace(v38B04Git(t, orquestaRoot, "show", manifest.Orquesta.SubjectCommit+":"+manifest.Orquesta.ContractPath)); v38B04Digest([]byte(got)) != manifest.Orquesta.ContractSHA256 {
		t.Fatal("el contrato Orquesta no coincide con su resumen sellado")
	}
	current, err := os.ReadFile(filepath.Join(orquestaRoot, manifest.Orquesta.ContractPath))
	if err != nil || v38B04Digest(bytes.TrimSpace(current)) != manifest.Orquesta.ContractSHA256 {
		t.Fatalf("el contrato activo difiere del sujeto: %v", err)
	}
	productModule := v38B04Git(t, orquestaRoot, "show", manifest.Orquesta.SubjectCommit+":go.mod")
	siblingCargo := v38B04Git(t, siblingRoot, "show", manifest.AgentMicroVM.SubjectCommit+":Cargo.toml")
	if strings.Contains(productModule, "agente_microvm") || strings.Contains(siblingCargo, "orquesta") {
		t.Fatal("los sujetos introducen importación cruzada")
	}
}

func v38B04BuildSibling(t *testing.T, manifest v38B04ManifestV1, siblingRoot, targetRoot string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(manifest.Limits.BuildMilliseconds)*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "cargo", "build", "--locked", "--release", "--package", manifest.AgentMicroVM.CargoPackage, "--bin", manifest.AgentMicroVM.BinaryName)
	command.Dir = siblingRoot
	command.Env = append(os.Environ(), "CARGO_TARGET_DIR="+targetRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build hermano: %v: %s", err, bytes.TrimSpace(output))
	}
	return filepath.Join(targetRoot, "release", manifest.AgentMicroVM.BinaryName)
}

func v38B04StartSibling(t *testing.T, binary, config, socket string, timeout time.Duration) func() {
	t.Helper()
	var output bytes.Buffer
	command := exec.Command(binary, "servir", "--config", config)
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("servicio hermano: %v: %s", err, bytes.TrimSpace(output.Bytes()))
			}
		case <-time.After(timeout):
			_ = command.Process.Kill()
			<-done
			t.Error("el servicio hermano no terminó cooperativamente")
		}
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if info, err := os.Lstat(socket); err == nil && info.Mode()&os.ModeSocket != 0 && info.Mode().Perm() == 0o600 {
			return stop
		}
		select {
		case err := <-done:
			stopped = true
			t.Fatalf("servicio terminó antes del socket: %v: %s", err, bytes.TrimSpace(output.Bytes()))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	stop()
	t.Fatal("el servicio hermano no publicó el socket privado")
	return stop
}

func v38B04Client(socket string, timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &http.Transport{
		DisableCompression: true,
		ForceAttemptHTTP2:  false,
		Proxy:              nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
}

func v38B04Request(client *http.Client, method, path, protocol string, body io.Reader, limit int64) (int, string, []byte, error) {
	request, err := http.NewRequest(method, "http://agentmicrovm.local"+path, body)
	if err != nil {
		return 0, "", nil, err
	}
	if protocol != "" {
		request.Header.Set("x-agentmicrovm-protocolo", protocol)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, "", nil, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(content)) > limit {
		return 0, "", nil, fmt.Errorf("respuesta fuera de límite: %w", err)
	}
	return response.StatusCode, response.Header.Get("x-agentmicrovm-protocolo"), content, nil
}

func v38B04RequireDigest(path, expected string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if got := v38B04Digest(content); got != expected {
		return fmt.Errorf("resumen binario=%s, esperado=%s", got, expected)
	}
	return nil
}

func v38B04Git(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", arguments, err, bytes.TrimSpace(output))
	}
	return strings.TrimSpace(string(output))
}

func v38B04RepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func v38B04Digest(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func v38B04Hex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}
