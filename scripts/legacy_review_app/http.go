// Este fichero aplica la frontera HTTP local: autenticación, CSRF y rutas fijas.
// Solo delega propuestas válidas; no sirve rutas de disco ni crea tareas.
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type application struct {
	inventory     *inventory
	store         *proposalStore
	catalog       catalog
	authorization []byte
	actorRef      string
	projectRef    string
	allowedHost   string
	allowedOrigin string
	nonce         func() (string, error)
}

func (app *application) handler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		app.securityHeaders(response)
		if strings.Contains(request.URL.Path, "..") || strings.Contains(request.URL.Path, "\\") {
			app.renderError(response, http.StatusBadRequest, "error_traversal", "")
			return
		}
		if request.Host != app.allowedHost {
			app.renderError(response, http.StatusBadRequest, "error_host", "")
			return
		}
		if !app.authorized(request) {
			response.Header().Set("WWW-Authenticate", `Basic realm="`+app.catalog.text("auth_realm")+`", charset="UTF-8"`)
			app.renderError(response, http.StatusUnauthorized, "error_unauthorized", "")
			return
		}
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/":
			app.handleList(response, request)
		case request.Method == http.MethodGet && request.URL.Path == "/item":
			app.handleDetail(response, request)
		case request.Method == http.MethodPost && request.URL.Path == "/proposal":
			app.handleProposal(response, request)
		default:
			app.renderError(response, http.StatusNotFound, "error_not_found", "")
		}
	})
}

func (app *application) handleList(response http.ResponseWriter, request *http.Request) {
	proposals, err := app.store.list()
	if err != nil {
		app.renderError(response, http.StatusInternalServerError, "error_generic", "")
		return
	}
	query := strings.ToLower(strings.TrimSpace(request.URL.Query().Get("q")))
	family := request.URL.Query().Get("family")
	dispositionFilter := disposition(request.URL.Query().Get("disposition"))
	latest := latestProposalMap(proposals)
	var rows []listRow
	for _, item := range app.inventory.items {
		current := latest[item.ID]
		if query != "" && !containsFold(item.ID+"\n"+item.Title+"\n"+item.Summary, query) {
			continue
		}
		if family != "" && family != item.Family {
			continue
		}
		if dispositionFilter != "" && current.Disposition != dispositionFilter {
			continue
		}
		label := app.catalog.text("disposition_none")
		if current.Disposition != "" {
			label = app.catalog.disposition(current.Disposition)
		}
		rows = append(rows, listRow{
			Item:        item,
			Disposition: label,
			DetailURL:   "/item?id=" + url.QueryEscape(item.ID),
		})
	}
	app.renderList(response, listPage{
		Rows:                rows,
		Families:            app.inventory.families,
		DispositionOptions:  app.dispositionOptions(true),
		Query:               request.URL.Query().Get("q"),
		SelectedFamily:      family,
		SelectedDisposition: string(dispositionFilter),
	})
}

func (app *application) handleDetail(response http.ResponseWriter, request *http.Request) {
	item, found := app.inventory.find(request.URL.Query().Get("id"))
	if !found {
		app.renderError(response, http.StatusNotFound, "error_not_found", "")
		return
	}
	proposals, err := app.store.list()
	if err != nil {
		app.renderError(response, http.StatusInternalServerError, "error_generic", item.ID)
		return
	}
	history := sortedProposalsForItem(proposals, item.ID)
	expected := latestRevision(proposals, item.ID)
	nonce, err := app.newNonce()
	if err != nil {
		app.renderError(response, http.StatusInternalServerError, "error_generic", item.ID)
		return
	}
	app.renderDetail(response, detailPage{
		Item:               item,
		Proposals:          app.proposalViews(history),
		DispositionOptions: app.dispositionOptions(false),
		RuleOptions:        app.ruleOptions(),
		ExpectedRevision:   expected,
		CSRFToken:          app.signedValue("csrf", item.ID),
		FormNonce:          nonce,
		IdempotencyKey:     app.signedValue("idempotency", item.ID, item.Revision, strconv.Itoa(expected), nonce),
		ActorRef:           app.actorRef,
		ProjectRef:         app.projectRef,
		Saved:              request.URL.Query().Get("saved") == "1",
	})
}

func (app *application) handleProposal(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != app.allowedOrigin {
		app.renderError(response, http.StatusForbidden, "error_origin", "")
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		app.renderError(response, http.StatusUnsupportedMediaType, "error_content_type", "")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, 64*1024)
	if err := request.ParseForm(); err != nil {
		app.renderError(response, http.StatusBadRequest, "error_invalid_form", "")
		return
	}
	item, found := app.inventory.find(request.Form.Get("item_ref"))
	if !found {
		app.renderError(response, http.StatusNotFound, "error_not_found", "")
		return
	}
	if !app.validSignedValue(request.Form.Get("csrf_token"), "csrf", item.ID) {
		app.renderError(response, http.StatusForbidden, "error_csrf", item.ID)
		return
	}
	if err := app.inventory.verifyUnchanged(); err != nil {
		app.renderError(response, http.StatusConflict, "error_stale_inventory", item.ID)
		return
	}
	expected, confidence, ok := parseProposalNumbers(request.Form)
	if !ok {
		app.renderError(response, http.StatusBadRequest, "error_invalid_form", item.ID)
		return
	}
	idempotencyKey := request.Form.Get("idempotency_key")
	nonce := request.Form.Get("form_nonce")
	if len(nonce) != 32 || !app.validSignedValue(
		idempotencyKey,
		"idempotency",
		item.ID,
		item.Revision,
		strconv.Itoa(expected),
		nonce,
	) {
		app.renderError(response, http.StatusBadRequest, "error_invalid_form", item.ID)
		return
	}
	rules := map[string]bool{}
	for _, rule := range requiredRules {
		rules[rule] = request.Form.Get("rule_"+rule) == "yes"
	}
	_, err = app.store.submit(item, proposalRequest{
		ItemRef:          item.ID,
		ItemRevision:     request.Form.Get("item_revision"),
		ExpectedRevision: expected,
		Disposition:      disposition(request.Form.Get("disposition")),
		Reason:           request.Form.Get("reason"),
		FoundedSolution:  request.Form.Get("founded_solution"),
		Confidence:       confidence,
		RuleCompliance:   rules,
		RuleNotes:        request.Form.Get("rule_notes"),
		ActorRef:         app.actorRef,
		ProjectRef:       app.projectRef,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		app.renderProposalError(response, err, item.ID)
		return
	}
	http.Redirect(response, request, "/item?id="+url.QueryEscape(item.ID)+"&saved=1", http.StatusSeeOther)
}

func (app *application) renderProposalError(response http.ResponseWriter, err error, itemID string) {
	switch {
	case errors.Is(err, errInventoryChanged):
		app.renderError(response, http.StatusConflict, "error_stale_inventory", itemID)
	case errors.Is(err, errRevisionConflict):
		app.renderError(response, http.StatusConflict, "error_revision_conflict", itemID)
	case errors.Is(err, errIdempotencyConflict):
		app.renderError(response, http.StatusConflict, "error_idempotency_conflict", itemID)
	case errors.Is(err, errInvalidProposal):
		app.renderError(response, http.StatusBadRequest, "error_invalid_form", itemID)
	default:
		app.renderError(response, http.StatusInternalServerError, "error_generic", itemID)
	}
}

func (app *application) authorized(request *http.Request) bool {
	username, password, ok := request.BasicAuth()
	if !ok || username != "local" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), app.authorization) == 1
}

func (app *application) signedValue(parts ...string) string {
	mac := hmac.New(sha256.New, app.authorization)
	for _, part := range parts {
		mac.Write([]byte{0})
		mac.Write([]byte(part))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func (app *application) newNonce() (string, error) {
	if app.nonce != nil {
		return app.nonce()
	}
	content := make([]byte, 16)
	if _, err := rand.Read(content); err != nil {
		return "", err
	}
	return hex.EncodeToString(content), nil
}

func (app *application) validSignedValue(value string, parts ...string) bool {
	expected := app.signedValue(parts...)
	return subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func (app *application) securityHeaders(response http.ResponseWriter) {
	response.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("X-Frame-Options", "DENY")
	response.Header().Set("Cache-Control", "no-store")
}

func parseProposalNumbers(form url.Values) (int, int, bool) {
	expected, errExpected := strconv.Atoi(form.Get("expected_revision"))
	confidence, errConfidence := strconv.Atoi(form.Get("confidence"))
	return expected, confidence, errExpected == nil && errConfidence == nil
}

func containsFold(value, lowerQuery string) bool {
	return strings.Contains(strings.ToLower(value), lowerQuery)
}

func latestProposalMap(proposals []proposal) map[string]proposal {
	result := map[string]proposal{}
	for _, item := range proposals {
		if item.ProposalRevision > result[item.ItemRef].ProposalRevision {
			result[item.ItemRef] = item
		}
	}
	return result
}

func (app *application) dispositionOptions(includeNone bool) []selectOption {
	var result []selectOption
	if includeNone {
		result = append(result, selectOption{Value: "", Label: app.catalog.text("disposition_none")})
	}
	for _, value := range []disposition{
		dispositionAdmitted, dispositionStudy, dispositionRejected, dispositionEvidence, dispositionDuplicate,
	} {
		result = append(result, selectOption{Value: string(value), Label: app.catalog.disposition(value)})
	}
	return result
}

func (app *application) ruleOptions() []ruleOption {
	result := make([]ruleOption, 0, len(requiredRules))
	for _, rule := range requiredRules {
		result = append(result, ruleOption{Value: rule, Label: app.catalog.rule(rule)})
	}
	return result
}

func (app *application) proposalViews(proposals []proposal) []proposalView {
	result := make([]proposalView, 0, len(proposals))
	for _, item := range proposals {
		createdLabel := item.CreatedAt
		if created, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err == nil {
			createdLabel = created.UTC().Format("02/01/2006 15:04:05 UTC")
		}
		result = append(result, proposalView{
			Proposal:         item,
			DispositionLabel: app.catalog.disposition(item.Disposition),
			CreatedLabel:     createdLabel,
		})
	}
	return result
}
