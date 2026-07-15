package oidc

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func TestPublicClientHarnessEnforcesCodePKCEStateNonceAndSingleUse(t *testing.T) {
	idp := newTestIdentityProvider(t)
	ctx := context.Background()
	discovered, err := coreoidc.NewProvider(ctx, idp.issuer())
	if err != nil {
		t.Fatalf("discover public client: %v", err)
	}
	endpoint := discovered.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInParams
	client := oauth2.Config{
		ClientID: "orquesta-v11", RedirectURL: "https://client.v11.invalid/callback",
		Endpoint: endpoint, Scopes: []string{coreoidc.ScopeOpenID},
	}

	t.Run("state mismatch stops before exchange", func(t *testing.T) {
		callback, transaction := requestAuthorization(t, client, "state-issued", "nonce-issued")
		values := callback.Query()
		values.Set("state", "different-state")
		callback.RawQuery = values.Encode()
		if _, err := transaction.callbackCode(callback); err == nil {
			t.Fatal("state mismatch accepted")
		}
	})

	t.Run("wrong PKCE verifier is rejected", func(t *testing.T) {
		callback, transaction := requestAuthorization(t, client, "state-pkce", "nonce-pkce")
		code, err := transaction.callbackCode(callback)
		if err != nil {
			t.Fatalf("callback: %v", err)
		}
		if _, err := client.Exchange(ctx, code, oauth2.VerifierOption(oauth2.GenerateVerifier())); err == nil {
			t.Fatal("wrong PKCE verifier exchanged code")
		}
	})

	callback, transaction := requestAuthorization(t, client, "state-happy", "nonce-happy")
	code, err := transaction.callbackCode(callback)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if _, err := transaction.callbackCode(callback); err == nil {
		t.Fatal("state/callback replay accepted")
	}
	token, err := client.Exchange(ctx, code, oauth2.VerifierOption(transaction.verifier))
	if err != nil {
		t.Fatalf("public code exchange: %v", err)
	}
	if _, err := client.Exchange(ctx, code, oauth2.VerifierOption(transaction.verifier)); err == nil {
		t.Fatal("authorization code replay succeeded")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		t.Fatal("token response lacks id_token")
	}
	verified, err := discovered.Verifier(&coreoidc.Config{
		ClientID: "orquesta-v11", Now: func() time.Time { return idp.now },
	}).Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("client ID token verification: %v", err)
	}
	wrongNonce := &publicClientTransaction{nonce: "different-nonce"}
	if err := wrongNonce.verifyNonce(verified.Nonce); err == nil {
		t.Fatal("nonce mismatch accepted")
	}
	if err := transaction.verifyNonce(verified.Nonce); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	if err := transaction.verifyNonce(verified.Nonce); err == nil {
		t.Fatal("nonce replay accepted")
	}
	if _, err := authenticateToken(idp.adapter(t), rawIDToken); err != nil {
		t.Fatalf("resource server rejected client-verified ID token: %v", err)
	}
}

type publicClientTransaction struct {
	state        string
	nonce        string
	verifier     string
	callbackUsed bool
	nonceUsed    bool
}

func requestAuthorization(t *testing.T, client oauth2.Config, state, nonce string) (*url.URL, *publicClientTransaction) {
	t.Helper()
	transaction := &publicClientTransaction{state: state, nonce: nonce, verifier: oauth2.GenerateVerifier()}
	authorizationURL := client.AuthCodeURL(
		transaction.state, oauth2.S256ChallengeOption(transaction.verifier), coreoidc.Nonce(transaction.nonce),
	)
	httpClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := httpClient.Get(authorizationURL)
	if err != nil {
		t.Fatalf("authorization request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("authorization status = %d", response.StatusCode)
	}
	callback, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse callback: %v", err)
	}
	return callback, transaction
}

func (transaction *publicClientTransaction) callbackCode(callback *url.URL) (string, error) {
	if transaction == nil || transaction.callbackUsed {
		return "", errors.New("oidc_client.state_consumed")
	}
	transaction.callbackUsed = true
	if callback == nil || callback.Query().Get("state") != transaction.state {
		return "", errors.New("oidc_client.state_invalid")
	}
	code := callback.Query().Get("code")
	if code == "" {
		return "", errors.New("oidc_client.code_missing")
	}
	return code, nil
}

func (transaction *publicClientTransaction) verifyNonce(actual string) error {
	if transaction == nil || transaction.nonceUsed {
		return errors.New("oidc_client.nonce_consumed")
	}
	transaction.nonceUsed = true
	if transaction.nonce == "" || actual != transaction.nonce {
		return errors.New("oidc_client.nonce_invalid")
	}
	return nil
}
