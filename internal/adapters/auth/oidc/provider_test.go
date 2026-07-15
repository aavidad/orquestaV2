package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"orquesta/internal/identity"
)

type testIdentityProvider struct {
	t      *testing.T
	server *httptest.Server
	now    time.Time

	mu        sync.RWMutex
	keys      map[string]*rsa.PrivateKey
	published []string
	codes     map[string]testAuthorization
	nextCode  uint64
	jwksReads atomic.Int64
	hangJWKS  atomic.Bool

	jwksCancelled     chan struct{}
	jwksCancelledOnce sync.Once
}

type testAuthorization struct {
	clientID      string
	redirectURI   string
	codeChallenge string
	nonce         string
}

func newTestIdentityProvider(t *testing.T) *testIdentityProvider {
	t.Helper()
	provider := &testIdentityProvider{
		t: t, now: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC),
		keys: make(map[string]*rsa.PrivateKey), codes: make(map[string]testAuthorization),
		jwksCancelled: make(chan struct{}),
	}
	for _, kid := range []string{"key-before", "key-after", "key-untrusted"} {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate %s: %v", kid, err)
		}
		provider.keys[kid] = key
	}
	provider.published = []string{"key-before"}
	provider.server = httptest.NewServer(http.HandlerFunc(provider.serveHTTP))
	t.Cleanup(provider.server.Close)
	return provider
}

func (provider *testIdentityProvider) issuer() string { return provider.server.URL }

func (provider *testIdentityProvider) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	switch request.URL.Path {
	case "/.well-known/openid-configuration":
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"issuer": provider.issuer(), "jwks_uri": provider.issuer() + "/keys",
			"authorization_endpoint":                provider.issuer() + "/auth",
			"token_endpoint":                        provider.issuer() + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/keys":
		provider.jwksReads.Add(1)
		if provider.hangJWKS.Load() {
			<-request.Context().Done()
			provider.jwksCancelledOnce.Do(func() { close(provider.jwksCancelled) })
			return
		}
		provider.mu.RLock()
		keys := make([]jose.JSONWebKey, 0, len(provider.published))
		for _, kid := range provider.published {
			keys = append(keys, jose.JSONWebKey{
				Key: &provider.keys[kid].PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig",
			})
		}
		provider.mu.RUnlock()
		_ = json.NewEncoder(writer).Encode(map[string]any{"keys": keys})
	case "/auth":
		provider.serveAuthorization(writer, request)
	case "/token":
		provider.serveToken(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (provider *testIdentityProvider) serveAuthorization(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	if request.Method != http.MethodGet || query.Get("response_type") != "code" ||
		query.Get("client_id") != "orquesta-v11" || query.Get("redirect_uri") == "" ||
		query.Get("state") == "" || query.Get("nonce") == "" ||
		query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" {
		http.Error(writer, "invalid authorization request", http.StatusBadRequest)
		return
	}
	provider.mu.Lock()
	provider.nextCode++
	code := fmt.Sprintf("code-%d", provider.nextCode)
	provider.codes[code] = testAuthorization{
		clientID: query.Get("client_id"), redirectURI: query.Get("redirect_uri"),
		codeChallenge: query.Get("code_challenge"), nonce: query.Get("nonce"),
	}
	provider.mu.Unlock()
	callback, err := url.Parse(query.Get("redirect_uri"))
	if err != nil {
		http.Error(writer, "invalid redirect", http.StatusBadRequest)
		return
	}
	values := callback.Query()
	values.Set("code", code)
	values.Set("state", query.Get("state"))
	callback.RawQuery = values.Encode()
	http.Redirect(writer, request, callback.String(), http.StatusFound)
}

func (provider *testIdentityProvider) serveToken(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Header.Get("Authorization") != "" || request.ParseForm() != nil ||
		request.Form.Get("grant_type") != "authorization_code" {
		http.Error(writer, "invalid token request", http.StatusBadRequest)
		return
	}
	code := request.Form.Get("code")
	provider.mu.Lock()
	authorization, found := provider.codes[code]
	if found {
		delete(provider.codes, code)
	}
	provider.mu.Unlock()
	verifierDigest := sha256.Sum256([]byte(request.Form.Get("code_verifier")))
	challenge := base64.RawURLEncoding.EncodeToString(verifierDigest[:])
	if !found || request.Form.Get("client_id") != authorization.clientID ||
		request.Form.Get("redirect_uri") != authorization.redirectURI || challenge != authorization.codeChallenge {
		http.Error(writer, "invalid grant", http.StatusBadRequest)
		return
	}
	claims := provider.claims()
	claims["nonce"] = authorization.nonce
	idToken := provider.sign("key-before", jose.RS256, claims)
	writer.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"access_token": "ephemeral-access-token", "token_type": "Bearer", "expires_in": 300,
		"id_token": idToken,
	})
}

func (provider *testIdentityProvider) publish(kids ...string) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.published = append([]string(nil), kids...)
}

func (provider *testIdentityProvider) claims() map[string]any {
	return map[string]any{
		"iss": provider.issuer(), "aud": "orquesta-v11", "sub": "00u-v11-alice",
		"iat": provider.now.Add(-time.Minute).Unix(), "nbf": provider.now.Add(-time.Minute).Unix(),
		"exp":    provider.now.Add(5 * time.Minute).Unix(),
		"groups": []string{"orquesta-users", "project-admins"},
		"email":  "ignored@example.invalid", "name": "Ignored Name",
	}
}

func (provider *testIdentityProvider) sign(kid string, algorithm jose.SignatureAlgorithm, claims map[string]any) string {
	provider.t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		provider.t.Fatalf("marshal claims: %v", err)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: algorithm, Key: provider.keys[kid]},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), kid),
	)
	if err != nil {
		provider.t.Fatalf("new signer: %v", err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		provider.t.Fatalf("sign: %v", err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		provider.t.Fatalf("serialize: %v", err)
	}
	return compact
}

func (provider *testIdentityProvider) adapter(t *testing.T, groups ...string) *Provider {
	t.Helper()
	adapter, err := New(context.Background(), Options{
		Issuer: provider.issuer(), Audience: "orquesta-v11", ClockSkew: 30 * time.Second,
		RequiredGroups: groups, UpstreamTimeout: 2 * time.Second,
		Now: func() time.Time { return provider.now },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return adapter
}

func authenticateToken(provider *Provider, raw string) (identity.Principal, error) {
	credential, err := identity.NewCredential([]byte(raw))
	if err != nil {
		return identity.Principal{}, err
	}
	return provider.Authenticate(context.Background(), credential)
}

func TestProviderDiscoversVerifiesAndMapsStableIdentityWithoutClaimsAuthority(t *testing.T) {
	idp := newTestIdentityProvider(t)
	provider := idp.adapter(t, "orquesta-users")
	if provider.AuthenticationMethod() != "oidc" {
		t.Fatalf("AuthenticationMethod = %q", provider.AuthenticationMethod())
	}
	claims := idp.claims()
	first, err := authenticateToken(provider, idp.sign("key-before", jose.RS256, claims))
	if err != nil {
		t.Fatalf("Authenticate first: %v", err)
	}
	claims["email"] = "changed@example.invalid"
	claims["name"] = "Changed"
	claims["groups"] = []string{"project-admins", "orquesta-users", "new-external-group"}
	second, err := authenticateToken(provider, idp.sign("key-before", jose.RS256, claims))
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("mutable claims changed identity: %+v %+v %v", first, second, err)
	}
	want, err := identity.StablePrincipal("oidc", idp.issuer(), "00u-v11-alice", identity.PrincipalKindHuman)
	if err != nil || first != want {
		t.Fatalf("principal = %+v, want %+v, err=%v", first, want, err)
	}
}

func TestProviderRefreshesJWKSOnRotationAndRejectsRetiredKeyAfterRefresh(t *testing.T) {
	idp := newTestIdentityProvider(t)
	provider := idp.adapter(t)
	before := idp.sign("key-before", jose.RS256, idp.claims())
	if _, err := authenticateToken(provider, before); err != nil {
		t.Fatalf("before rotation: %v", err)
	}
	idp.publish("key-after")
	after := idp.sign("key-after", jose.RS256, idp.claims())
	if _, err := authenticateToken(provider, after); err != nil {
		t.Fatalf("after rotation: %v", err)
	}
	if _, err := authenticateToken(provider, before); !IsError(err, CodeTokenInvalid) {
		t.Fatalf("removed key error = %v", err)
	}
	untrusted := idp.sign("key-untrusted", jose.RS256, idp.claims())
	if _, err := authenticateToken(provider, untrusted); !IsError(err, CodeTokenInvalid) {
		t.Fatalf("untrusted key error = %v", err)
	}
	if idp.jwksReads.Load() < 4 {
		t.Fatalf("JWKS reads = %d, rotation/unknown kid did not refresh", idp.jwksReads.Load())
	}
}

func TestProviderRejectsIssuerAudienceSubjectTimesAlgorithmSignatureAndGroups(t *testing.T) {
	idp := newTestIdentityProvider(t)
	provider := idp.adapter(t, "orquesta-users")
	tests := []struct {
		name string
		edit func(map[string]any)
		kid  string
		alg  jose.SignatureAlgorithm
		want ErrorCode
	}{
		{"issuer", func(c map[string]any) { c["iss"] = "https://other.invalid" }, "key-before", jose.RS256, CodeTokenInvalid},
		{"audience", func(c map[string]any) { c["aud"] = "other" }, "key-before", jose.RS256, CodeTokenInvalid},
		{"multiple audiences", func(c map[string]any) { c["aud"] = []string{"orquesta-v11", "other"} }, "key-before", jose.RS256, CodeAudienceInvalid},
		{"subject", func(c map[string]any) { c["sub"] = "" }, "key-before", jose.RS256, CodeSubjectInvalid},
		{"expired", func(c map[string]any) { c["exp"] = idp.now.Add(-31 * time.Second).Unix() }, "key-before", jose.RS256, CodeTimeInvalid},
		{"not before", func(c map[string]any) { c["nbf"] = idp.now.Add(31 * time.Second).Unix() }, "key-before", jose.RS256, CodeTimeInvalid},
		{"issued at", func(c map[string]any) { c["iat"] = idp.now.Add(31 * time.Second).Unix() }, "key-before", jose.RS256, CodeTimeInvalid},
		{"issued after expiry", func(c map[string]any) { c["iat"] = idp.now.Add(6 * time.Minute).Unix() }, "key-before", jose.RS256, CodeTimeInvalid},
		{"not before after expiry", func(c map[string]any) {
			c["nbf"] = idp.now.Add(4 * time.Minute).Unix()
			c["exp"] = idp.now.Add(3 * time.Minute).Unix()
		}, "key-before", jose.RS256, CodeTimeInvalid},
		{"missing issued at", func(c map[string]any) { delete(c, "iat") }, "key-before", jose.RS256, CodeTimeInvalid},
		{"groups", func(c map[string]any) { c["groups"] = []string{"project-admins"} }, "key-before", jose.RS256, CodeGroupDenied},
		{"signature", func(map[string]any) {}, "key-untrusted", jose.RS256, CodeTokenInvalid},
		{"algorithm", func(map[string]any) {}, "key-before", jose.PS256, CodeTokenInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := idp.claims()
			test.edit(claims)
			if _, err := authenticateToken(provider, idp.sign(test.kid, test.alg, claims)); !IsError(err, test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestProviderAppliesExactConfiguredClockSkewAndInputGuards(t *testing.T) {
	idp := newTestIdentityProvider(t)
	provider := idp.adapter(t)
	for _, edit := range []func(map[string]any){
		func(c map[string]any) { c["exp"] = idp.now.Add(-29 * time.Second).Unix() },
		func(c map[string]any) { c["nbf"] = idp.now.Add(29 * time.Second).Unix() },
		func(c map[string]any) { c["iat"] = idp.now.Add(29 * time.Second).Unix() },
	} {
		claims := idp.claims()
		edit(claims)
		if _, err := authenticateToken(provider, idp.sign("key-before", jose.RS256, claims)); err != nil {
			t.Errorf("within clock skew rejected: %v", err)
		}
	}
	credential, _ := identity.NewCredential([]byte(" Bearer invalid\n"))
	if _, err := provider.Authenticate(context.Background(), credential); !IsError(err, CodeCredentialInvalid) {
		t.Fatalf("malformed credential error = %v", err)
	}
	if _, err := provider.Authenticate(nil, credential); !IsError(err, CodeContextInvalid) {
		t.Fatalf("nil context error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Authenticate(cancelled, credential); !IsError(err, CodeContextInvalid) {
		t.Fatalf("cancelled context error = %v", err)
	}
}

func TestOwnedUpstreamClientBoundsHungDiscoveryAndReleasesResources(t *testing.T) {
	baselineGoroutines := runtime.NumGoroutine()
	var openConnections atomic.Int64
	var startedOnce sync.Once
	var cancelledOnce sync.Once
	started := make(chan struct{})
	cancelled := make(chan struct{})

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		startedOnce.Do(func() { close(started) })
		<-request.Context().Done()
		cancelledOnce.Do(func() { close(cancelled) })
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			openConnections.Add(1)
		case http.StateClosed, http.StateHijacked:
			openConnections.Add(-1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	upstreamTimeout := 100 * time.Millisecond
	startedAt := time.Now()
	_, err := New(context.Background(), Options{
		Issuer: server.URL, Audience: "orquesta-v11", UpstreamTimeout: upstreamTimeout,
	})
	elapsed := time.Since(startedAt)
	if !IsError(err, CodeDiscoveryFailed) {
		t.Fatalf("hung discovery error = %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("hung discovery exceeded bounded timeout: %s", elapsed)
	}
	select {
	case <-started:
	default:
		t.Fatal("discovery request never reached the IdP")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("timed-out discovery did not cancel the IdP handler")
	}
	waitForCondition(t, 2*time.Second, func() bool { return openConnections.Load() == 0 },
		"timed-out discovery connection remained open")

	server.Close()
	runtime.GC()
	waitForCondition(t, 2*time.Second, func() bool {
		return runtime.NumGoroutine() <= baselineGoroutines+4
	}, "timed-out discovery left goroutines behind")
}

func TestOwnedUpstreamClientBoundsHungJWKS(t *testing.T) {
	idp := newTestIdentityProvider(t)
	provider, err := New(context.Background(), Options{
		Issuer: idp.issuer(), Audience: "orquesta-v11", ClockSkew: 30 * time.Second,
		UpstreamTimeout: 100 * time.Millisecond, Now: func() time.Time { return idp.now },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	idp.hangJWKS.Store(true)
	startedAt := time.Now()
	_, err = authenticateToken(provider, idp.sign("key-before", jose.RS256, idp.claims()))
	if !IsError(err, CodeTokenInvalid) {
		t.Fatalf("hung JWKS error = %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 2*time.Second {
		t.Fatalf("hung JWKS exceeded bounded timeout: %s", elapsed)
	}
	select {
	case <-idp.jwksCancelled:
	case <-time.After(time.Second):
		t.Fatal("timed-out JWKS request did not cancel the IdP handler")
	}
}

func TestOwnedUpstreamTransportDoesNotUseEnvironmentProxyOrHTTPGlobals(t *testing.T) {
	var proxyRequests atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		proxyRequests.Add(1)
		http.Error(writer, "proxy must not be used", http.StatusBadGateway)
	}))
	defer proxy.Close()
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		t.Setenv(key, proxy.URL)
	}
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	upstreamTimeout := 100 * time.Millisecond
	client := newUpstreamHTTPClient(upstreamTimeout)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.Transport)
	}
	if client.Timeout != upstreamTimeout || transport.Proxy != nil || transport.DialContext == nil ||
		transport.TLSHandshakeTimeout != upstreamTimeout ||
		transport.ResponseHeaderTimeout != upstreamTimeout ||
		transport.ExpectContinueTimeout != upstreamTimeout ||
		transport.IdleConnTimeout != upstreamTimeout ||
		transport.MaxIdleConns != maxIdleConnections ||
		transport.MaxIdleConnsPerHost != maxIdleConnectionsPerHost ||
		transport.MaxConnsPerHost != maxConnectionsPerHost {
		t.Fatalf("owned client is not fully bounded: client=%+v transport=%+v", client, transport)
	}
	response, err := client.Get("http://orquesta-proxy-decoy.invalid/discovery")
	if err == nil {
		response.Body.Close()
		t.Fatal("reserved invalid host unexpectedly resolved")
	}
	if proxyRequests.Load() != 0 {
		t.Fatalf("environment proxy received %d requests", proxyRequests.Load())
	}
}

func TestExplicitHTTPClientCanProvideTestTrustMaterial(t *testing.T) {
	idp := newTestIdentityProvider(t)
	client := idp.server.Client()
	if _, err := New(context.Background(), Options{
		Issuer: idp.issuer(), Audience: "orquesta-v11",
		UpstreamTimeout: 2 * time.Second, HTTPClient: client,
	}); err != nil {
		t.Fatalf("explicit test HTTP client rejected: %v", err)
	}
}

func TestInjectedHTTPClientCannotBypassCanonicalNetworkPolicy(t *testing.T) {
	idp := newTestIdentityProvider(t)
	injected := idp.server.Client()
	injected.Timeout = 0
	injectedTransport, ok := injected.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("injected transport type = %T", injected.Transport)
	}
	injectedTransport = injectedTransport.Clone()
	injectedTransport.Proxy = http.ProxyFromEnvironment
	injectedTransport.MaxIdleConns = 0
	injectedTransport.ResponseHeaderTimeout = 0
	injected.Transport = injectedTransport

	timeout := 2 * time.Second
	bounded, err := boundedUpstreamHTTPClient(injected, timeout)
	if err != nil {
		t.Fatalf("bound injected client: %v", err)
	}
	transport, ok := bounded.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("bounded transport type = %T", bounded.Transport)
	}
	if bounded == injected || transport == injectedTransport || bounded.Timeout != timeout ||
		transport.Proxy != nil || transport.MaxIdleConns != maxIdleConnections ||
		transport.MaxIdleConnsPerHost != maxIdleConnectionsPerHost ||
		transport.MaxConnsPerHost != maxConnectionsPerHost ||
		transport.IdleConnTimeout != timeout || transport.TLSHandshakeTimeout != timeout ||
		transport.ResponseHeaderTimeout != timeout || transport.ExpectContinueTimeout != timeout {
		t.Fatalf("injected policy escaped canonical bounds: client=%+v transport=%+v", bounded, transport)
	}
	injectedTransport.MaxIdleConns = 99
	if transport.MaxIdleConns != maxIdleConnections {
		t.Fatal("bounded transport aliases the mutable injected transport")
	}
	if _, err := New(context.Background(), Options{
		Issuer: idp.issuer(), Audience: "orquesta-v11",
		UpstreamTimeout: timeout, HTTPClient: injected,
	}); err != nil {
		t.Fatalf("bounded injected trust transport rejected: %v", err)
	}
}

func TestProviderConfigurationFailsClosed(t *testing.T) {
	idp := newTestIdentityProvider(t)
	for _, options := range []Options{
		{},
		{Issuer: "http://remote.invalid", Audience: "audience"},
		{Issuer: idp.issuer(), Audience: " audience"},
		{Issuer: idp.issuer(), Audience: "audience", ClockSkew: -time.Second},
		{Issuer: idp.issuer(), Audience: "audience", ClockSkew: 6 * time.Minute},
		{Issuer: idp.issuer(), Audience: "audience", RequiredGroups: []string{"duplicate", "duplicate"}},
	} {
		if _, err := New(context.Background(), options); !IsError(err, CodeConfigInvalid) {
			t.Errorf("invalid options accepted: %+v err=%v", options, err)
		}
	}
	for _, upstreamTimeout := range []time.Duration{0, -time.Millisecond, maxUpstreamTimeout + time.Nanosecond} {
		if _, err := New(context.Background(), Options{
			Issuer: idp.issuer(), Audience: "audience", UpstreamTimeout: upstreamTimeout,
			HTTPClient: idp.server.Client(),
		}); !IsError(err, CodeConfigInvalid) {
			t.Errorf("invalid upstream timeout %s accepted: %v", upstreamTimeout, err)
		}
	}
	if _, err := New(context.Background(), Options{
		Issuer: idp.issuer(), Audience: "audience", UpstreamTimeout: time.Second,
		HTTPClient: &http.Client{},
	}); !IsError(err, CodeConfigInvalid) {
		t.Errorf("default injected transport accepted: %v", err)
	}
	if _, err := New(nil, Options{}); !IsError(err, CodeContextInvalid) {
		t.Fatalf("nil context error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(cancelled, Options{}); !IsError(err, CodeContextInvalid) {
		t.Fatalf("cancelled context error = %v", err)
	}
	badDiscovery := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"issuer": "https://different.invalid"})
	}))
	defer badDiscovery.Close()
	if _, err := New(context.Background(), Options{
		Issuer: badDiscovery.URL, Audience: "audience", UpstreamTimeout: time.Second,
	}); !IsError(err, CodeDiscoveryFailed) {
		t.Fatalf("issuer mismatch discovery error = %v", err)
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
}
