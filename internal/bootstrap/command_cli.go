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

	"orquesta/internal/i18n"
	"orquesta/internal/identity"
	cliinterface "orquesta/internal/interfaces/cli"
	sdkcommands "orquesta/sdk/commands"
)

const (
	commandUsageKey = "cli.command.usage"
)

var (
	errCommandURLInvalid        = errors.New("cli.url_invalid")
	errCommandURLInsecure       = errors.New("cli.url_insecure")
	errCommandCredentialInvalid = errors.New("cli.credential_invalid")
	errCommandRequestInvalid    = errors.New("cli.request_invalid")
	errCommandOriginInvalid     = errors.New("cli.request_origin_invalid")
)

type cliDiagnostic struct {
	code       string
	messageKey string
}

func RunCommand(arguments []string, catalog *i18n.Catalog, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("orquesta command", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	locale := flags.String("locale", i18n.DefaultLocale, "")
	baseURL := flags.String("url", "", "")
	credentialPath := flags.String("credential-file", "", "")
	maxCredentialBytes := flags.Int64("max-credential-bytes", 0, "")
	maxResponseBytes := flags.Int64("max-response-bytes", 0, "")
	timeout := flags.Duration("timeout", 0, "")
	requestRef := flags.String("request-ref", "", "")
	projectRef := flags.String("project-ref", "", "")
	executionRef := flags.String("execution-ref", "", "")
	payloadValue := flags.String("payload", "", "")
	parseErr := flags.Parse(arguments)
	if parseErr != nil && !errors.Is(parseErr, flag.ErrHelp) {
		writeCLIDiagnostic(stderr, catalog, *locale, cliDiagnostic{
			code: "cli.command_arguments_invalid", messageKey: "error.cli.command_arguments_invalid",
		})
		return 2
	}
	if catalog == nil {
		_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
		return 1
	}
	if _, err := catalog.Resolve(*locale); err != nil {
		writeCLIDiagnostic(stderr, catalog, i18n.DefaultLocale, cliDiagnostic{
			code: "cli.locale_invalid", messageKey: "error.cli.locale_invalid",
		})
		return 2
	}
	if errors.Is(parseErr, flag.ErrHelp) {
		if usage, err := catalog.Text(*locale, commandUsageKey); err == nil {
			_, _ = fmt.Fprintln(stdout, usage)
			return 0
		}
		_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
		return 1
	}
	path := flags.Args()
	if *baseURL == "" || strings.TrimSpace(*credentialPath) != *credentialPath || *credentialPath == "" ||
		*maxCredentialBytes <= 0 || *maxResponseBytes <= 0 || *timeout <= 0 ||
		strings.TrimSpace(*requestRef) != *requestRef || *requestRef == "" ||
		strings.TrimSpace(*projectRef) != *projectRef || *projectRef == "" ||
		*payloadValue == "" || !json.Valid([]byte(*payloadValue)) || len(path) == 0 {
		writeCLIDiagnostic(stderr, catalog, *locale, cliDiagnostic{
			code: "cli.command_arguments_invalid", messageKey: "error.cli.command_arguments_invalid",
		})
		return 2
	}
	origin, err := commandOrigin(*baseURL)
	if err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, diagnosticForCommandError(err))
		return 2
	}
	credential, err := readCommandCredential(*credentialPath, *maxCredentialBytes)
	if err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, diagnosticForCommandError(err))
		return 2
	}
	httpClient := &http.Client{
		Transport: commandCredentialTransport{credential: credential, origin: origin},
		Timeout:   *timeout,
	}
	client, err := sdkcommands.New(sdkcommands.Config{
		BaseURL:          strings.TrimRight(*baseURL, "/"),
		HTTPClient:       httpClient,
		MaxResponseBytes: *maxResponseBytes,
	})
	if err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, cliDiagnostic{
			code: sdkcommands.CodeInternal, messageKey: "error.internal",
		})
		return 2
	}
	runner, err := cliinterface.New(client)
	if err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, cliDiagnostic{
			code: sdkcommands.CodeInternal, messageKey: "error.internal",
		})
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := runner.Run(
		ctx, path, *requestRef, *projectRef, *executionRef, json.RawMessage(*payloadValue),
	)
	if err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, diagnosticForCommandError(err))
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		writeCLIDiagnostic(stderr, catalog, *locale, cliDiagnostic{
			code: "cli.output_invalid", messageKey: "error.cli.output_invalid",
		})
		return 1
	}
	if result.Failure != nil {
		return 1
	}
	return 0
}

func commandOrigin(raw string) (string, error) {
	if strings.TrimSpace(raw) != raw || raw == "" {
		return "", errCommandURLInvalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", errCommandURLInvalid
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" {
		host := net.ParseIP(parsed.Hostname())
		if scheme != "http" || host == nil || !host.IsLoopback() {
			return "", errCommandURLInsecure
		}
	}
	return scheme + "://" + strings.ToLower(parsed.Host), nil
}

func readCommandCredential(path string, maximum int64) (identity.Credential, error) {
	if maximum <= 0 || strings.TrimSpace(path) != path || path == "" {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	cleaned := filepath.Clean(path)
	directory, name := filepath.Split(cleaned)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	if directory == "" {
		directory = "."
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	defer root.Close()
	before, err := root.Lstat(name)
	if err != nil || !privateCredentialFile(before) {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	file, err := root.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !privateCredentialFile(opened) || !os.SameFile(before, opened) {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	material, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return identity.Credential{}, errCommandCredentialInvalid
	}
	defer clear(material)
	after, err := root.Lstat(name)
	if err != nil || !privateCredentialFile(after) || !os.SameFile(opened, after) ||
		int64(len(material)) > maximum || len(material) == 0 || strings.ContainsAny(string(material), " \t\r\n") {
		return identity.Credential{}, errCommandCredentialInvalid
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
		return nil, errCommandRequestInvalid
	}
	requestOrigin := strings.ToLower(request.URL.Scheme) + "://" + strings.ToLower(request.URL.Host)
	if requestOrigin != transport.origin {
		return nil, errCommandOriginInvalid
	}
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	if err := transport.credential.Use(func(material []byte) error {
		if strings.ContainsAny(string(material), " \t\r\n") {
			return errCommandCredentialInvalid
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

func diagnosticForCommandError(err error) cliDiagnostic {
	switch {
	case errors.Is(err, errCommandURLInvalid):
		return cliDiagnostic{code: "cli.url_invalid", messageKey: "error.cli.url_invalid"}
	case errors.Is(err, errCommandURLInsecure):
		return cliDiagnostic{code: "cli.url_insecure", messageKey: "error.cli.url_insecure"}
	case errors.Is(err, errCommandCredentialInvalid):
		return cliDiagnostic{code: "cli.credential_invalid", messageKey: "error.cli.credential_invalid"}
	case errors.Is(err, sdkcommands.ErrRedirectRejected):
		return cliDiagnostic{code: "cli.redirect_rejected", messageKey: "error.cli.redirect_rejected"}
	case errors.Is(err, errCommandRequestInvalid):
		return cliDiagnostic{code: "cli.request_invalid", messageKey: "error.cli.request_invalid"}
	case errors.Is(err, errCommandOriginInvalid):
		return cliDiagnostic{code: "cli.request_origin_invalid", messageKey: "error.cli.request_origin_invalid"}
	case errors.Is(err, cliinterface.ErrCommandRunnerUnavailable):
		return cliDiagnostic{code: "cli.command_runner_unavailable", messageKey: "error.cli.command_runner_unavailable"}
	case errors.Is(err, cliinterface.ErrCommandUnknown):
		return cliDiagnostic{code: "cli.command_unknown", messageKey: "error.cli.command_unknown"}
	default:
		return cliDiagnostic{code: sdkcommands.CodeUnavailable, messageKey: "error.unavailable"}
	}
}

func writeCLIDiagnostic(writer io.Writer, catalog *i18n.Catalog, locale string, diagnostic cliDiagnostic) {
	if catalog != nil {
		if text, err := catalog.Text(locale, diagnostic.messageKey); err == nil {
			_, _ = fmt.Fprintf(writer, "%s code=%s\n", text, diagnostic.code)
			return
		}
		if text, err := catalog.Text(i18n.DefaultLocale, diagnostic.messageKey); err == nil {
			_, _ = fmt.Fprintf(writer, "%s code=%s\n", text, diagnostic.code)
			return
		}
	}
	_, _ = fmt.Fprintf(writer, "code=%s\n", diagnostic.code)
}
