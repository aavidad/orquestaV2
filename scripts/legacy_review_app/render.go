// Este fichero proyecta modelos en HTML; no contiene autoridad ni persiste datos.
// Las pruebas verifican escape, catálogo, teclado y ausencia de señales solo cromáticas.
package main

import (
	"html/template"
	"net/http"
	"net/url"
)

type pageView struct {
	Kind   string
	List   listPage
	Detail detailPage
	Error  errorPage
}

type listPage struct {
	Rows                []listRow
	Families            []string
	DispositionOptions  []selectOption
	Query               string
	SelectedFamily      string
	SelectedDisposition string
}

type listRow struct {
	Item        inventoryItem
	Disposition string
	DetailURL   string
}

type detailPage struct {
	Item               inventoryItem
	Proposals          []proposalView
	DispositionOptions []selectOption
	RuleOptions        []ruleOption
	ExpectedRevision   int
	CSRFToken          string
	FormNonce          string
	IdempotencyKey     string
	ActorRef           string
	ProjectRef         string
	Saved              bool
}

type proposalView struct {
	Proposal         proposal
	DispositionLabel string
	CreatedLabel     string
	Rules            []ruleResult
}

type ruleResult struct {
	Label  string
	Result string
}

type selectOption struct {
	Value string
	Label string
}

type ruleOption struct {
	Value string
	Label string
}

type errorPage struct {
	Message   string
	ItemID    string
	ItemQuery string
}

func (app *application) renderList(response http.ResponseWriter, page listPage) {
	app.render(response, http.StatusOK, pageView{Kind: "list", List: page})
}

func (app *application) renderDetail(response http.ResponseWriter, page detailPage) {
	for index := range page.Proposals {
		for _, rule := range requiredRules {
			result := app.catalog.text("value_no")
			if page.Proposals[index].Proposal.RuleCompliance[rule] {
				result = app.catalog.text("value_yes")
			}
			page.Proposals[index].Rules = append(page.Proposals[index].Rules, ruleResult{
				Label: app.catalog.rule(rule), Result: result,
			})
		}
	}
	app.render(response, http.StatusOK, pageView{Kind: "detail", Detail: page})
}

func (app *application) renderError(response http.ResponseWriter, status int, textKey, itemID string) {
	app.render(response, status, pageView{
		Kind: "error",
		Error: errorPage{
			Message:   app.catalog.text(textKey),
			ItemID:    itemID,
			ItemQuery: url.QueryEscape(itemID),
		},
	})
}

func (app *application) render(response http.ResponseWriter, status int, view pageView) {
	cloned, err := pageTemplate.Clone()
	if err != nil {
		http.Error(response, app.catalog.text("error_generic"), http.StatusInternalServerError)
		return
	}
	cloned = cloned.Funcs(template.FuncMap{"text": app.catalog.text})
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	if err := cloned.ExecuteTemplate(response, "inicio", view); err != nil {
		return
	}
}
