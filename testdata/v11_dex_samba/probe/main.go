package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"orquesta/internal/adapters/auth/oidc"
	"orquesta/internal/identity"
)

type result struct {
	Status       string `json:"status"`
	Code         string `json:"code,omitempty"`
	PrincipalRef string `json:"principal_ref,omitempty"`
	ActorRef     string `json:"actor_ref,omitempty"`
}

func main() {
	if len(os.Args) != 4 {
		fail("probe.arguments_invalid")
	}
	provider, err := oidc.New(context.Background(), oidc.Options{
		Issuer:          os.Args[1],
		Audience:        os.Args[2],
		RequiredGroups:  []string{os.Args[3]},
		ClockSkew:       30 * time.Second,
		UpstreamTimeout: 10 * time.Second,
	})
	if err != nil {
		fail("probe.provider_failed")
	}
	raw, err := bufio.NewReader(io.LimitReader(os.Stdin, 64<<10)).ReadString('\n')
	if err != nil && raw == "" {
		fail("probe.token_missing")
	}
	raw = strings.TrimSuffix(raw, "\n")
	credential, err := identity.NewCredential([]byte(raw))
	if err != nil {
		fail("probe.credential_invalid")
	}
	principal, err := provider.Authenticate(context.Background(), credential)
	if err != nil {
		for _, code := range []oidc.ErrorCode{
			oidc.CodeGroupDenied,
			oidc.CodeTokenInvalid,
			oidc.CodeAudienceInvalid,
			oidc.CodeSubjectInvalid,
			oidc.CodeTimeInvalid,
		} {
			if oidc.IsError(err, code) {
				write(result{Status: "rejected", Code: string(code)})
				return
			}
		}
		fail("probe.authentication_failed")
	}
	write(result{
		Status:       "accepted",
		PrincipalRef: principal.Ref.String(),
		ActorRef:     principal.ActorRef.String(),
	})
}

func fail(code string) {
	write(result{Status: "failed", Code: code})
	os.Exit(1)
}

func write(value result) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		os.Exit(1)
	}
}
