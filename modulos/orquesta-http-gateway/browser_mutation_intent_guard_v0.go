package orquestahttpgateway

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	BrowserMutationIntentDecisionHeaderV0 = "X-Orquesta-Browser-Intent-Decision"
	BrowserMutationIntentReasonHeaderV0   = "X-Orquesta-Browser-Intent-Reason"
	BrowserMutationIntentTokenHeaderV0    = "X-Orquesta-Intent-Token"
	BrowserMutationCSRFTokenHeaderV0      = "X-CSRF-Token"

	BrowserMutationDecisionOriginAllowedV0 = "origin_allowed"
	BrowserMutationDecisionCSRFRequiredV0  = "csrf_required"
	BrowserMutationDecisionIntentMissingV0 = "intent_missing"
	BrowserMutationDecisionOriginBlockedV0 = "origin_blocked"
	BrowserMutationDecisionNonBrowserV0    = "non_browser_client"
	BrowserMutationDecisionReadOnlyV0      = "read_only"
)

type BrowserMutationIntentGuardConfigV0 struct {
	IntentToken string
	Disabled    bool
}

type BrowserMutationIntentDecisionV0 struct {
	Mutable        bool
	BrowserRequest bool
	Allowed        bool
	Decision       string
	Reason         string
}

func NewBrowserMutationIntentGuardV0(
	config BrowserMutationIntentGuardConfigV0,
	next http.Handler,
) http.Handler {
	if next == nil {
		return nil
	}
	if config.Disabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision := EvaluateBrowserMutationIntentV0(config, r)
		if decision.Mutable {
			w.Header().Set(BrowserMutationIntentDecisionHeaderV0, decision.Decision)
			w.Header().Set(BrowserMutationIntentReasonHeaderV0, decision.Reason)
		}
		if !decision.Allowed {
			writeBrowserMutationDeniedV0(w, decision)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func EvaluateBrowserMutationIntentV0(
	config BrowserMutationIntentGuardConfigV0,
	r *http.Request,
) BrowserMutationIntentDecisionV0 {
	decision := BrowserMutationIntentDecisionV0{
		Allowed:  true,
		Decision: BrowserMutationDecisionReadOnlyV0,
		Reason:   "route_read_only",
	}
	if r == nil || r.URL == nil || methodIsReadOnlyBrowserMutationV0(r.Method) ||
		PublicRouteMutabilityV0(r.URL.Path) != PublicRouteMutationV0 {
		return decision
	}
	decision.Mutable = true
	decision.Decision = BrowserMutationDecisionNonBrowserV0
	decision.Reason = "non_browser_client"
	if !browserMutationSignalV0(r) {
		return decision
	}
	decision.BrowserRequest = true
	if browserMutationIntentTokenAllowedV0(config.IntentToken, r) {
		decision.Decision = BrowserMutationDecisionOriginAllowedV0
		decision.Reason = "intent_token"
		return decision
	}
	if browserMutationSameOriginV0(r.Header.Get("Origin"), r) ||
		browserMutationSameOriginV0(r.Header.Get("Referer"), r) {
		decision.Decision = BrowserMutationDecisionOriginAllowedV0
		decision.Reason = "same_origin"
		return decision
	}
	decision.Allowed = false
	decision.Decision = BrowserMutationDecisionIntentMissingV0
	decision.Reason = BrowserMutationDecisionCSRFRequiredV0
	if strings.TrimSpace(r.Header.Get("Origin")) != "" || strings.TrimSpace(r.Header.Get("Referer")) != "" {
		decision.Decision = BrowserMutationDecisionOriginBlockedV0
		decision.Reason = "origin_not_same_origin"
	}
	return decision
}

func browserMutationSignalV0(r *http.Request) bool {
	if r == nil {
		return false
	}
	for _, header := range []string{"Origin", "Referer", "Sec-Fetch-Site", "Sec-Fetch-Mode"} {
		if strings.TrimSpace(r.Header.Get(header)) != "" {
			return true
		}
	}
	return false
}

func browserMutationSameOriginV0(raw string, r *http.Request) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") || r == nil {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, browserMutationRequestSchemeV0(r)) &&
		strings.EqualFold(parsed.Host, r.Host)
}

func browserMutationRequestSchemeV0(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); proto == "http" || proto == "https" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func browserMutationIntentTokenAllowedV0(want string, r *http.Request) bool {
	want = strings.TrimSpace(want)
	if want == "" || r == nil {
		return false
	}
	got := strings.TrimSpace(r.Header.Get(BrowserMutationIntentTokenHeaderV0))
	if got == "" {
		got = strings.TrimSpace(r.Header.Get(BrowserMutationCSRFTokenHeaderV0))
	}
	return len(want) == len(got) && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func methodIsReadOnlyBrowserMutationV0(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func writeBrowserMutationDeniedV0(w http.ResponseWriter, decision BrowserMutationIntentDecisionV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"errores_publicos": []map[string]string{{
			"code":    decision.Decision,
			"message": decision.Reason,
		}},
	})
}
