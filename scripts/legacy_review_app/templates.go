// Este fichero contiene la estructura HTML accesible; obtiene todo texto del catálogo.
// No lee ni escribe datos y no incorpora JavaScript ni recursos externos.
package main

import (
	"html/template"
)

var pageTemplate = template.Must(template.New("paginas").Funcs(template.FuncMap{
	"text": func(string) string { return "" },
}).Parse(`
{{define "inicio"}}
<!doctype html><html lang="es"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{text "app_title"}}</title><style>
:root{font-family:system-ui,sans-serif;color-scheme:light dark}
body{max-width:76rem;margin:auto;padding:1rem;line-height:1.45}
a{color:LinkText}.skip{position:absolute;left:-10000px}.skip:focus{left:1rem;top:1rem;background:Canvas;padding:.7rem;z-index:2}
:focus-visible{outline:.2rem solid Highlight;outline-offset:.15rem}
header{border-bottom:1px solid GrayText;margin-bottom:1.5rem}
form.filters,.panel{border:1px solid GrayText;border-radius:.4rem;padding:1rem;margin:1rem 0}
label{display:block;font-weight:650;margin-top:.8rem}
input,select,textarea,button{font:inherit;max-width:100%;padding:.45rem}
input[type=search],input[type=text],textarea,select{width:100%;box-sizing:border-box}
textarea{min-height:7rem}.rules label{font-weight:400}
table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid GrayText;padding:.55rem;text-align:left;vertical-align:top}
code,pre{font-family:ui-monospace,monospace}pre{overflow:auto;border:1px solid GrayText;padding:1rem}
.status{font-weight:700}.notice{border-left:.4rem solid Highlight;padding:.8rem}.error{border-left:.4rem solid Mark;padding:.8rem}
.muted{opacity:.8}.actions{margin-top:1rem}
@media(max-width:50rem){table,thead,tbody,tr,th,td{display:block}thead{position:absolute;left:-10000px}td{border:0}tr{border-bottom:1px solid GrayText;padding:.6rem}}
</style></head><body>
<a class="skip" href="#contenido">{{text "skip_content"}}</a>
<header><h1>{{text "app_title"}}</h1><nav><a href="/">{{text "nav_inventory"}}</a></nav></header>
{{template "contenido" .}}
</body></html>
{{end}}

{{define "lista"}}{{template "inicio" .}}{{end}}
{{define "contenido"}}
<main id="contenido" tabindex="-1">
{{if eq .Kind "list"}}
<h2>{{text "list_title"}}</h2>
<form class="filters" method="get" action="/" aria-labelledby="filtros">
<h3 id="filtros">{{text "filter_title"}}</h3>
<label for="q">{{text "filter_search"}}</label><input id="q" name="q" type="search" value="{{.List.Query}}">
<label for="family">{{text "filter_family"}}</label><select id="family" name="family">
<option value="">{{text "filter_all"}}</option>{{range .List.Families}}<option value="{{.}}" {{if eq . $.List.SelectedFamily}}selected{{end}}>{{.}}</option>{{end}}</select>
<label for="disposition">{{text "filter_disposition"}}</label><select id="disposition" name="disposition">
{{range .List.DispositionOptions}}<option value="{{.Value}}" {{if eq .Value $.List.SelectedDisposition}}selected{{end}}>{{.Label}}</option>{{end}}</select>
<div class="actions"><button type="submit">{{text "filter_apply"}}</button></div></form>
{{if .List.Rows}}<table><thead><tr><th>{{text "table_item"}}</th><th>{{text "table_family"}}</th>
<th>{{text "table_revision"}}</th><th>{{text "table_disposition"}}</th><th>{{text "table_action"}}</th></tr></thead><tbody>
{{range .List.Rows}}<tr><td><strong>{{.Item.Title}}</strong><br><code>{{.Item.ID}}</code></td><td>{{.Item.Family}}</td>
<td><code>{{.Item.Revision}}</code></td><td><span class="status">{{.Disposition}}</span></td>
<td><a href="{{.DetailURL}}">{{text "action_review"}}</a></td></tr>{{end}}</tbody></table>
{{else}}<p>{{text "empty_list"}}</p>{{end}}
{{end}}

{{if eq .Kind "detail"}}
<h2>{{text "detail_title"}}</h2>
{{if .Detail.Saved}}<p class="notice" role="status">{{text "saved_notice"}}</p>{{end}}
<dl><dt>{{text "field_identifier"}}</dt><dd><code>{{.Detail.Item.ID}}</code></dd>
<dt>{{text "field_revision"}}</dt><dd><code>{{.Detail.Item.Revision}}</code></dd>
<dt>{{text "field_family"}}</dt><dd>{{.Detail.Item.Family}}</dd>
<dt>{{text "field_summary"}}</dt><dd>{{.Detail.Item.Summary}}</dd></dl>
<section><h3>{{text "sources_title"}}</h3>{{if .Detail.Item.Sources}}<ul>{{range .Detail.Item.Sources}}<li><code>{{.}}</code></li>{{end}}</ul>
{{else}}<p>{{text "empty_sources"}}</p>{{end}}</section>
<section><h3>{{text "attempts_title"}}</h3>{{if .Detail.Item.Attempts}}<ul>{{range .Detail.Item.Attempts}}<li>{{.}}</li>{{end}}</ul>
{{else}}<p>{{text "empty_attempts"}}</p>{{end}}</section>
<section><h3>{{text "uncertainties_title"}}</h3>{{if .Detail.Item.Uncertainties}}<ul>{{range .Detail.Item.Uncertainties}}<li>{{.}}</li>{{end}}</ul>
{{else}}<p>{{text "empty_uncertainties"}}</p>{{end}}</section>
<details><summary>{{text "raw_title"}}</summary><pre>{{.Detail.Item.Raw}}</pre></details>

<section><h3>{{text "proposal_history_title"}}</h3>
{{if .Detail.Proposals}}{{range .Detail.Proposals}}<article class="panel">
<h4>{{.DispositionLabel}} · {{text "field_revision"}} {{.Proposal.ProposalRevision}}</h4>
<dl><dt>{{text "field_reason"}}</dt><dd>{{.Proposal.Reason}}</dd>
<dt>{{text "field_solution"}}</dt><dd>{{.Proposal.FoundedSolution}}</dd>
<dt>{{text "field_confidence"}}</dt><dd>{{.Proposal.Confidence}}</dd>
<dt>{{text "field_rule_notes"}}</dt><dd>{{.Proposal.RuleNotes}}</dd>
<dt>{{text "field_rules_result"}}</dt><dd><ul>{{range .Rules}}<li>{{.Label}}: <strong>{{.Result}}</strong></li>{{end}}</ul></dd>
<dt>{{text "field_actor"}}</dt><dd><code>{{.Proposal.ActorRef}}</code></dd>
<dt>{{text "field_project"}}</dt><dd><code>{{.Proposal.ProjectRef}}</code></dd>
<dt>{{text "field_created"}}</dt><dd>{{.CreatedLabel}}</dd></dl></article>{{end}}
{{else}}<p>{{text "empty_proposals"}}</p>{{end}}</section>

<form class="panel" method="post" action="/proposal">
<h3>{{text "proposal_form_title"}}</h3><p class="notice">{{text "proposal_boundary"}}</p>
<input type="hidden" name="item_ref" value="{{.Detail.Item.ID}}">
<input type="hidden" name="item_revision" value="{{.Detail.Item.Revision}}">
<input type="hidden" name="expected_revision" value="{{.Detail.ExpectedRevision}}">
<input type="hidden" name="csrf_token" value="{{.Detail.CSRFToken}}">
<input type="hidden" name="form_nonce" value="{{.Detail.FormNonce}}">
<input type="hidden" name="idempotency_key" value="{{.Detail.IdempotencyKey}}">
<p><strong>{{text "field_expected_revision"}}:</strong> {{.Detail.ExpectedRevision}}</p>
<p><strong>{{text "field_actor"}}:</strong> <code>{{.Detail.ActorRef}}</code><br>
<strong>{{text "field_project"}}:</strong> <code>{{.Detail.ProjectRef}}</code></p>
<label for="proposal-disposition">{{text "field_disposition"}}</label><select required id="proposal-disposition" name="disposition">
{{range .Detail.DispositionOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select>
<label for="reason">{{text "field_reason"}}</label><textarea required minlength="10" id="reason" name="reason"></textarea>
<label for="solution">{{text "field_solution"}}</label><textarea required minlength="10" id="solution" name="founded_solution"></textarea>
<label for="confidence">{{text "field_confidence"}}</label><input required id="confidence" name="confidence" type="number" min="0" max="100" value="50">
<fieldset class="rules"><legend>{{text "rules_title"}}</legend>{{range .Detail.RuleOptions}}
<label><input type="checkbox" name="rule_{{.Value}}" value="yes"> {{.Label}}</label>{{end}}</fieldset>
<label for="rule-notes">{{text "field_rule_notes"}}</label><textarea required minlength="10" id="rule-notes" name="rule_notes"></textarea>
<div class="actions"><button type="submit">{{text "action_submit"}}</button></div></form>
<p><a href="/">{{text "back_list"}}</a></p>
{{end}}

{{if eq .Kind "error"}}
<section class="error" role="alert"><h2>{{text "error_title"}}</h2><p>{{.Error.Message}}</p></section>
{{if .Error.ItemID}}<p><a href="/item?id={{.Error.ItemQuery}}">{{text "back_detail"}}</a></p>{{else}}<p><a href="/">{{text "back_list"}}</a></p>{{end}}
{{end}}
</main>
{{end}}
`))
