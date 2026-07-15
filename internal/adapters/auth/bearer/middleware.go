package bearer

import (
	"errors"
	"net/http"
	"strings"

	"orquesta/internal/identity"
)

type Middleware struct {
	provider identity.IdentityProvider
}

func New(provider identity.IdentityProvider) (*Middleware, error) {
	if provider == nil || provider.AuthenticationMethod() == "" {
		return nil, errors.New("bearer.identity_provider_required")
	}
	return &Middleware{provider: provider}, nil
}

func (middleware *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if middleware == nil || middleware.provider == nil || request == nil {
			unavailable(writer)
			return
		}
		material, found := requestCredential(request)
		if !found {
			unauthorized(writer)
			return
		}
		credential, err := identity.NewCredential(material)
		clear(material)
		if err != nil {
			unauthorized(writer)
			return
		}
		principal, err := middleware.provider.Authenticate(request.Context(), credential)
		if err != nil {
			unauthorized(writer)
			return
		}
		if next == nil {
			unavailable(writer)
			return
		}
		bound, err := identity.BindPrincipal(request.Context(), principal)
		if err != nil {
			unavailable(writer)
			return
		}
		next.ServeHTTP(writer, request.WithContext(bound))
	})
}

func requestCredential(request *http.Request) ([]byte, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return nil, false
	}
	scheme, material, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || material == "" || strings.ContainsAny(material, " \t\r\n") {
		return nil, false
	}
	return []byte(material), true
}

func unauthorized(writer http.ResponseWriter) {
	writer.Header().Set("WWW-Authenticate", "Bearer")
	writer.WriteHeader(http.StatusUnauthorized)
}

func unavailable(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusServiceUnavailable)
}
