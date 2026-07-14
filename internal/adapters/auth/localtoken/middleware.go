package localtoken

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

// Middleware protects an HTTP handler with the persisted local Bearer token.
func (authenticator *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		candidate, present := requestBearerToken(request)
		if authenticator == nil || !present || !authenticator.matches(candidate) {
			unauthorized(writer)
			return
		}
		if next == nil {
			http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func requestBearerToken(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, candidate, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || candidate == "" || strings.ContainsAny(candidate, " \t\r\n") {
		return "", false
	}
	return candidate, true
}

func (authenticator *Authenticator) matches(candidate string) bool {
	candidateDigest := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(authenticator.digest[:], candidateDigest[:]) == 1
}

func unauthorized(writer http.ResponseWriter) {
	writer.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
