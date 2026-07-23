package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/identity"
	cliinterface "orquesta/internal/interfaces/cli"
	sdkcommands "orquesta/sdk/commands"
)

const CommandUsage = "orquesta command --url URL --credential-file PATH --max-credential-bytes N --max-response-bytes N --timeout DURATION --request-ref REF --project-ref REF --payload JSON [--execution-ref REF] -- COMMAND..."

func RunCommand(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("orquesta command", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	baseURL := flags.String("url", "", "")
	credentialPath := flags.String("credential-file", "", "")
	maxCredentialBytes := flags.Int64("max-credential-bytes", 0, "")
	maxResponseBytes := flags.Int64("max-response-bytes", 0, "")
	timeout := flags.Duration("timeout", 0, "")
	requestRef := flags.String("request-ref", "", "")
	projectRef := flags.String("project-ref", "", "")
	executionRef := flags.String("execution-ref", "", "")
	payloadValue := flags.String("payload", "", "")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, CommandUsage)
			return 0
		}
		_, _ = fmt.Fprintln(stderr, "cli.command_arguments_invalid")
		return 2
	}
	path := flags.Args()
	origin, err := commandOrigin(*baseURL)
	if err != nil ||
		strings.TrimSpace(*credentialPath) != *credentialPath || *credentialPath == "" ||
		*maxCredentialBytes <= 0 || *maxResponseBytes <= 0 || *timeout <= 0 ||
		strings.TrimSpace(*requestRef) != *requestRef || *requestRef == "" ||
		strings.TrimSpace(*projectRef) != *projectRef || *projectRef == "" ||
		*payloadValue == "" || !json.Valid([]byte(*payloadValue)) || len(path) == 0 {
		_, _ = fmt.Fprintln(stderr, "cli.command_arguments_invalid")
		return 2
	}
	credential, err := readCommandCredential(*credentialPath, *maxCredentialBytes)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "cli.credential_invalid")
		return 2
	}
	httpClient := &http.Client{
		Transport: commandCredentialTransport{credential: credential, origin: origin},
		Timeout:   *timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("cli.redirect_rejected")
		},
	}
	client, err := sdkcommands.New(sdkcommands.Config{
		BaseURL:          strings.TrimRight(*baseURL, "/"),
		HTTPClient:       httpClient,
		MaxResponseBytes: *maxResponseBytes,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 2
	}
	runner, err := cliinterface.New(client)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := runner.Run(
		ctx, path, *requestRef, *projectRef, *executionRef, json.RawMessage(*payloadValue),
	)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		_, _ = fmt.Fprintln(stderr, "cli.output_invalid")
		return 1
	}
	if result.Failure != nil {
		return 1
	}
	return 0
}

func commandOrigin(raw string) (string, error) {
	if strings.TrimSpace(raw) != raw || raw == "" {
		return "", errors.New("cli.url_invalid")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("cli.url_invalid")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" {
		host := net.ParseIP(parsed.Hostname())
		if scheme != "http" || host == nil || !host.IsLoopback() {
			return "", errors.New("cli.url_insecure")
		}
	}
	return scheme + "://" + strings.ToLower(parsed.Host), nil
}

func readCommandCredential(path string, maximum int64) (identity.Credential, error) {
	if maximum <= 0 || strings.TrimSpace(path) != path || path == "" {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	cleaned := filepath.Clean(path)
	directory, name := filepath.Split(cleaned)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	if directory == "" {
		directory = "."
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	defer root.Close()
	before, err := root.Lstat(name)
	if err != nil || !privateCredentialFile(before) {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	file, err := root.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !privateCredentialFile(opened) || !os.SameFile(before, opened) {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	material, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	defer clear(material)
	after, err := root.Lstat(name)
	if err != nil || !privateCredentialFile(after) || !os.SameFile(opened, after) ||
		int64(len(material)) > maximum || len(material) == 0 || strings.ContainsAny(string(material), " \t\r\n") {
		return identity.Credential{}, errors.New("cli.credential_invalid")
	}
	return identity.NewCredential(material)
}

func privateCredentialFile(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular() && info.Mode().Perm() == 0o600
}

type commandCredentialTransport struct {
	credential identity.Credential
	origin     string
	base       http.RoundTripper
}

func (transport commandCredentialTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, errors.New("cli.request_invalid")
	}
	requestOrigin := strings.ToLower(request.URL.Scheme) + "://" + strings.ToLower(request.URL.Host)
	if requestOrigin != transport.origin {
		return nil, errors.New("cli.request_origin_invalid")
	}
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	if err := transport.credential.Use(func(material []byte) error {
		if strings.ContainsAny(string(material), " \t\r\n") {
			return errors.New("cli.credential_invalid")
		}
		cloned.Header.Set("Authorization", "Bearer "+string(material))
		return nil
	}); err != nil {
		return nil, err
	}
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}
