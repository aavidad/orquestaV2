package oidc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"

	"orquesta/internal/identity"
)

const (
	AuthenticationMethod = "oidc"

	maxUpstreamTimeout        = 30 * time.Second
	maxIdleConnections        = 8
	maxIdleConnectionsPerHost = 4
	maxConnectionsPerHost     = 8
)

type Options struct {
	Issuer          string
	Audience        string
	ClockSkew       time.Duration
	RequiredGroups  []string
	UpstreamTimeout time.Duration
	HTTPClient      *http.Client
	Now             func() time.Time
}

// Provider is a stateless resource-server adapter. go-oidc owns discovery,
// JWKS signature and algorithm validation, cached key lookup and unknownKid
// refresh. Provider adds exact audience, subject and configurable time policy.
type Provider struct {
	verifier       *coreoidc.IDTokenVerifier
	issuer         string
	audience       string
	clockSkew      time.Duration
	requiredGroups []string
	now            func() time.Time
}

var _ identity.IdentityProvider = (*Provider)(nil)

func New(ctx context.Context, options Options) (*Provider, error) {
	if ctx == nil {
		return nil, &Error{Code: CodeContextInvalid}
	}
	if err := ctx.Err(); err != nil {
		return nil, &Error{Code: CodeContextInvalid, Cause: err}
	}
	issuer, err := identity.CanonicalIssuer(options.Issuer)
	if err != nil || !validExactValue(options.Audience) || options.ClockSkew < 0 || options.ClockSkew > 5*time.Minute {
		return nil, &Error{Code: CodeConfigInvalid, Cause: err}
	}
	requiredGroups, err := canonicalGroups(options.RequiredGroups)
	if err != nil {
		return nil, &Error{Code: CodeConfigInvalid, Cause: err}
	}
	if options.UpstreamTimeout <= 0 || options.UpstreamTimeout > maxUpstreamTimeout {
		return nil, &Error{Code: CodeConfigInvalid}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	httpClient, err := boundedUpstreamHTTPClient(options.HTTPClient, options.UpstreamTimeout)
	if err != nil {
		return nil, &Error{Code: CodeConfigInvalid, Cause: err}
	}
	discoveryContext := coreoidc.ClientContext(ctx, httpClient)
	discovered, err := coreoidc.NewProvider(discoveryContext, issuer)
	if err != nil {
		return nil, &Error{Code: CodeDiscoveryFailed, Cause: err}
	}
	provider := &Provider{
		issuer: issuer, audience: options.Audience, clockSkew: options.ClockSkew,
		requiredGroups: requiredGroups, now: now,
	}
	provider.verifier = discovered.VerifierContext(discoveryContext, &coreoidc.Config{
		ClientID: provider.audience,
		Now: func() time.Time {
			return provider.now().UTC().Add(-provider.clockSkew)
		},
	})
	return provider, nil
}

// newUpstreamHTTPClient deliberately owns every network policy it uses. It
// does not inherit process proxies or mutable package globals, and bounds both
// whole requests and the network phases that can otherwise wait indefinitely.
func newUpstreamHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: timeout,
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConnections,
		MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
		MaxConnsPerHost:       maxConnectionsPerHost,
		IdleConnTimeout:       timeout,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: timeout,
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

// boundedUpstreamHTTPClient treats an injected client only as transport trust
// material (for example, a test CA). Timeout, proxy and pool policy remain
// owned by this adapter, so injection cannot bypass the canonical limits or
// make discovery depend on mutable package-level HTTP defaults.
func boundedUpstreamHTTPClient(injected *http.Client, timeout time.Duration) (*http.Client, error) {
	if injected == nil {
		return newUpstreamHTTPClient(timeout), nil
	}
	transport, ok := injected.Transport.(*http.Transport)
	if !ok || transport == nil {
		return nil, errors.New("oidc.injected_http_transport_invalid")
	}
	bounded := transport.Clone()
	bounded.Proxy = nil
	if bounded.DialContext == nil {
		dialer := &net.Dialer{Timeout: timeout, KeepAlive: timeout}
		bounded.DialContext = dialer.DialContext
	}
	bounded.MaxIdleConns = maxIdleConnections
	bounded.MaxIdleConnsPerHost = maxIdleConnectionsPerHost
	bounded.MaxConnsPerHost = maxConnectionsPerHost
	bounded.IdleConnTimeout = timeout
	bounded.TLSHandshakeTimeout = timeout
	bounded.ResponseHeaderTimeout = timeout
	bounded.ExpectContinueTimeout = timeout
	return &http.Client{Transport: bounded, Timeout: timeout}, nil
}

func (*Provider) AuthenticationMethod() identity.AuthenticationMethod {
	return identity.AuthenticationMethod(AuthenticationMethod)
}

func (provider *Provider) Authenticate(
	ctx context.Context,
	credential identity.Credential,
) (identity.Principal, error) {
	if ctx == nil || provider == nil || provider.verifier == nil || provider.now == nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid}
	}
	if err := ctx.Err(); err != nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid, Cause: err}
	}
	var principal identity.Principal
	err := credential.Use(func(material []byte) error {
		rawToken := string(material)
		if rawToken == "" || strings.TrimSpace(rawToken) != rawToken || strings.ContainsAny(rawToken, "\r\n\x00") {
			return &Error{Code: CodeCredentialInvalid}
		}
		token, err := provider.verifier.Verify(ctx, rawToken)
		if err != nil {
			var expired *coreoidc.TokenExpiredError
			if errors.As(err, &expired) {
				return &Error{Code: CodeTimeInvalid, Cause: err}
			}
			return &Error{Code: CodeTokenInvalid, Cause: err}
		}
		if token.Issuer != provider.issuer {
			return &Error{Code: CodeTokenInvalid}
		}
		if len(token.Audience) != 1 || token.Audience[0] != provider.audience {
			return &Error{Code: CodeAudienceInvalid}
		}
		if token.Subject == "" {
			return &Error{Code: CodeSubjectInvalid}
		}
		var claims verifiedClaims
		if err := token.Claims(&claims); err != nil {
			return &Error{Code: CodeTokenInvalid, Cause: err}
		}
		if err := provider.validateTimes(token, claims); err != nil {
			return err
		}
		if !containsEveryGroup(claims.Groups, provider.requiredGroups) {
			return &Error{Code: CodeGroupDenied}
		}
		principal, err = identity.StablePrincipal(
			provider.AuthenticationMethod(), provider.issuer, token.Subject, identity.PrincipalKindHuman,
		)
		if err != nil {
			return &Error{Code: CodeIdentityInvalid, Cause: err}
		}
		return nil
	})
	if err != nil {
		var coded *Error
		if !errors.As(err, &coded) {
			return identity.Principal{}, &Error{Code: CodeCredentialInvalid, Cause: err}
		}
		return identity.Principal{}, err
	}
	return principal, nil
}

func (provider *Provider) validateTimes(token *coreoidc.IDToken, claims verifiedClaims) error {
	now := provider.now().UTC()
	if token.Expiry.IsZero() || now.After(token.Expiry.Add(provider.clockSkew)) {
		return &Error{Code: CodeTimeInvalid}
	}
	if token.IssuedAt.IsZero() || token.IssuedAt.After(now.Add(provider.clockSkew)) ||
		token.IssuedAt.After(token.Expiry) {
		return &Error{Code: CodeTimeInvalid}
	}
	if claims.NotBefore != nil {
		notBefore := time.Unix(*claims.NotBefore, 0).UTC()
		if notBefore.After(now.Add(provider.clockSkew)) || notBefore.After(token.Expiry) {
			return &Error{Code: CodeTimeInvalid}
		}
	}
	return nil
}

type verifiedClaims struct {
	NotBefore *int64   `json:"nbf"`
	Groups    []string `json:"groups"`
}

func canonicalGroups(groups []string) ([]string, error) {
	result := append([]string(nil), groups...)
	sort.Strings(result)
	for index, group := range result {
		if !validExactValue(group) || (index > 0 && result[index-1] == group) {
			return nil, &Error{Code: CodeConfigInvalid}
		}
	}
	return result, nil
}

func containsEveryGroup(actual, required []string) bool {
	if len(required) == 0 {
		return true
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, group := range actual {
		if validExactValue(group) {
			actualSet[group] = struct{}{}
		}
	}
	for _, group := range required {
		if _, found := actualSet[group]; !found {
			return false
		}
	}
	return true
}

func validExactValue(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\r\n\x00")
}
